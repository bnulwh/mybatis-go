# 数据表名前缀（TablePrefix）实现说明

> 功能入口见 README「数据表名前缀」小节；本文记录设计决策与实现细节，修改本功能前必读。

## 解决的问题

同一套 XML Mapper 部署到多个环境时，物理表名可能不同：测试环境用 `test_sys_user`，
生产环境用 `sys_user`（或 `prod_sys_user`）。目标是 **XML Mapper 里的 SQL 完全不动**，
仅靠配置（`mybatis.table-prefix=test_`）在运行期把表引用改写为带前缀的物理表名。

## 配置与优先级

- 配置键（`orm/table_prefix.go` → `orm/database_config.go::parseTablePrefix`），优先级从高到低：
  1. `mybatis.table-prefix`（本项目主键）
  2. `mybatis-plus.global-config.db-config.table-prefix`（MyBatis-Plus 兼容键）
  3. `spring.datasource.table-prefix`（每数据源覆盖键，多数据源场景可用）
- 编程式：`orm.SetTablePrefix("test_")`（全局默认前缀）；`orm.GetTablePrefix()` 查询当前生效值。
- **生效范围**：`DB.tablePrefix()` 先看本 DB 配置（`Config.Setting.TablePrefix`），为空才回退全局。
  `SetTablePrefix` 只改全局；对已初始化且未单独配置前缀的数据源立即生效。

## 执行入口（单一改口）

改写放在数据库执行层 `DB.ExecContext / QueryContext / QueryRowContext`
（`orm/database_connection.go`），在占位符方言转换（`formatSQL`）之前：
`query = db.applyTablePrefix(query)`。

为什么选这里而不是 SQL 生成层（`types`）：

- 一处覆盖所有执行路径：Mapper 代理（`executeMethod`/`executeStream`）、`orm.Execute/Query`、
  流式 `QueryStream`、事务（`DB.x` 内部走 `currentTx` 分支，`Transaction` 直调方法在
  `orm/transaction.go` 同样补了 `applyTablePrefix` + `formatSQL`，与 DB 路径行为一致）。
- 与查找无关：预编译缓存（`PreparedStmtDB`）的 key 是改写后的 SQL，各前缀互不串缓存；
  语句统计记录的是改写后的真实语句。

## 改写算法（rewriteSQLTables）

纯函数，输入输出 SQL 除表名前缀外逐字节一致（改写基于 token 文本拼接还原，空白/大小写/注释零改动）。
核心分两步：

### 1) 词法扫描（tokenizeSQL）

把 SQL 切成 token：`tkWord`（裸标识符/关键字/数字）、`tkQuoted`（`"x"` / `` `x` `` / `[x]`）、
`tkString`（`'x'` 含 `''` 转义、Postgres `$$...$$` / `$tag$...$tag$` 美元引用）、
`tkPunct`、`tkSpace`、`tkComment`（`-- ...`、`# ...`、`/* ... */`）。
字符串/注释整体折叠为单个 token，后续状态机不会进入它们内部。

### 2) 状态机

两个核心状态：

- `expectTable`：刚遇到表关键字，下一个标识符处于「表位置」。
  - 表关键字集合：`FROM / JOIN / INTO / UPDATE / TABLE`（大小写不敏感，整词匹配）。
  - 表位置的处理：
    - `CREATE/DROP/ALTER TABLE IF NOT EXISTS` 的 `IF/NOT/EXISTS` 是 DDL 修饰词，跳过不当作表名；
    - schema 限定名 `schema.table`：仅 `public` / `main`（SQLite 默认模式）给表名部分加前缀，
      其余 schema（`app`、`information_schema`、`pg_catalog` 等）整体跳过，避免跨库/跨 schema 误改；
    - 否则对该标识符判定 `prefixRequired` 后插入前缀。
- `commaList`：`FROM t1, t2` / MySQL `UPDATE t1, t2 SET` 逗号分隔的多表列表。
  `,` 后继续期待表名；遇 `(`/`)` 或 `WHERE/SET/ON/GROUP/ORDER/LIMIT/VALUES/...` 等断句关键字复位
  （`clauseBreakWords`）。`INSERT INTO t (a, b) VALUES (1, 2)` 中列名列表的逗号不会误触发
  （`INTO`/`UPDATE` 后的 `(` 立即复位 commaList）。

### 3) 真实表集合匹配（prefixRequired，v0.1.15 新增）

在纯前缀匹配之前，先从**指定 schema 获取数据库真实表名集合**（`tableNameSet`，从
`information_schema.COLUMNS` / `pg_class(+pg_namespace)` / `sqlite_master` 查询，键为小写），
改写时与集合比对，解决「改了前缀但物理表未改名导致访问错误表」的问题，提高准确率：

1. 系统表 / 已带前缀的表 → 不改写（防叠加，同旧行为）；
2. **带前缀的表名在集合中真实存在** → 按配置前缀改写（配置意图优先）；
3. **SQL 引用的表名（无前缀）在集合中真实存在** → 保持原样（它就是物理表，改写反而出错）——
   这是本特性修复的核心场景：如配置 `mybatis.table-prefix=test_` 但库中只有 `sys_user`（未改名），
   旧实现会改写成不存在的 `test_sys_user`，新实现命中集合后保留 `sys_user`；
4. 两者均未收录（`CREATE TABLE` 新建表、目标表确实不存在）→ 沿用前缀匹配兜底（行为与旧版一致）。

集合缓存与失效：

- 集合按数据源（`Config.cacheStore`，key `tableNames`）惰性缓存，首次启用前缀的 SQL 执行时拉取一次；
- DDL（`CREATE/DROP/ALTER/RENAME/TRUNCATE`，`isDDLStatement`）在 `ExecContext` 执行后使缓存失效，
  下一条 SQL 按最新表集合改写（新建表立即可见）；
- 拉取失败（如连不上库）记录 Warn 并缓存空集，退化为纯前缀匹配（不阻塞查询）；
- 集合查询直连底层 `ConnPool`（不经 `applyTablePrefix`），避免循环依赖；只查系统表，不会自我改写。

其它保护规则：

- **字符串/注释/占位符不被触及**：`'sys_user'`、`-- ...`、`?`、`$1` 原样保留；
- **列名/别名不误伤**：`SELECT sys_user FROM sys_user` 只有 FROM 后那个被改；
- **系统表/目录永不参与**（`isSystemTable`）：`information_schema`、`pg_catalog`、
  `pg_%`、`sqlite_%`、`pragma_%` —— 框架内置的表结构探查（`info_schema` 三兄弟、
  `sqlite_master`、`pragma_table_info`）因此不受前缀影响；
- **已带前缀不叠加**：`test_sys_user` 不会再变 `test_test_sys_user`（大小写不敏感判断）；
- **CTE 名跳过**（`cteModeCollect`）：`WITH cte AS (SELECT ...) SELECT * FROM cte`
  收集顶层 CTE 名（支持 `RECURSIVE`、`NOT MATERIALIZED` 修饰与多 CTE 逗号分隔），
  表位置命中 CTE 名时不加前缀；
- **别名不破坏逗号列表**：`FROM t1 a, t2 b` 中的别名 `a`/`b` 只当普通词，不中断 `commaList`，
  别名本身也不是表关键字或断句词时保持原样。

## 多数据源

`parseMultiDatabaseConfig` 解析附加数据源后，若其未单独配置前缀（如通过
`spring.datasource.<name>.table-prefix`），**继承默认数据源的前缀**
（`orm/multi_datasource.go`），保证一套 `mybatis.table-prefix` 全局生效、按源可覆盖。

## 测试覆盖（orm/table_prefix_test.go）

- 单元（不依赖数据库）：
  - `Test_rewriteSQLTables_basic` —— SELECT/INSERT/UPDATE/DELETE/JOIN/逗号列表/DDL/schema 限定/
    引号标识符/CTE 共 20 组正向断言；
  - `Test_rewriteSQLTables_noFalsePositive` —— 列名同名、字符串字面量、`$1`、`LIMIT/GROUP BY`、
    已带前缀、三种注释、美元引用、批量 VALUES、子查询、系统表、引号 schema 等 24 组反向断言；
  - `Test_rewriteSQLTables_tableSet` —— 真实表集合匹配：带前缀表存在→改写；无前缀表存在→保持原样
    （修复核心场景）；两者并存→带前缀者优先；未知表→前缀兜底；已带前缀/大小写不敏感/schema 限定名/
    nil 与空集合退化，共 8 组断言；
  - `Test_parseTablePrefix` / `Test_parseDatabaseConfig_tablePrefix` / `Test_parseMultiDatabaseConfig_tablePrefixInherit` —— 配置解析与多源继承。
- 端到端（SQLite 真实执行，物理表带 `test_` 前缀，SQL 保持不带前缀）：
  - `Test_TablePrefix_SqliteQuery` —— `Execute/Query` 直调路径改写生效；
  - `Test_TablePrefix_SqliteMapper` —— Mapper 代理 Insert/Select/Count + `useGeneratedKeys` 回填；
  - `Test_TablePrefix_GlobalSetter` —— 编程式 `SetTablePrefix`；
  - `Test_TablePrefix_GetAndTransactions` —— 事务内 SQL 同样生效 + `GetTablePrefix`；
  - `Test_TablePrefix_SqliteTableStructure` —— 带前缀时内置表结构探查（`sqlite_master`/
    `pragma_table_info`）不受影响；
  - `Test_TablePrefix_SqliteExistingUnprefixed` —— 端到端验证修复场景：物理表未带前缀时，
    启用前缀后 SQL 引用保持原样（不断言改写），同库新表仍按前缀建表；
  - `Test_TablePrefix_SqliteTableSetRefresh` —— DDL 成功后表名集合缓存失效并重新获取
    （新建表对后续改写可见）。

## 已知边界（有意简化）

- 表关键字仅识别 `FROM/JOIN/INTO/UPDATE/TABLE`；`MERGE INTO`、`CREATE INDEX ... ON`（ON 是
  JOIN 条件关键字，不能当表关键字）、MySQL `RENAME TABLE a TO b` 的目标名等罕见写法不加前缀；
- 非 `public`/`main` schema 的限定名整段跳过（跨库引用由各自环境自行管理）；
- 嵌套 CTE（子查询内部再 `WITH`）名称未收集，若 Mapper 使用会报 SQL 错误（显式失败而非静默错数据）。

## 表集合匹配边界

- **带前缀与无前缀同名表并存时**，判定为「带前缀表存在」→ 按配置前缀改写，以配置意图为准；
  若需要无前缀表优先，应去掉前缀配置（保持行为：SQL 引用即物理表）；
- **表集合缓存按库内容刷新**：在本 DB 上执行 DDL 后自动失效重取；其他会话/进程改表（如外部改名、
  手工建无前缀表）不会主动触发刷新——此时若新表尚未出现在集合中，仍按前缀匹配兜底处理，
  可执行任意 DDL 或重启进程强制刷新；
- **集合查询失败**（如临时网络抖动）时缓存空集、降级为纯前缀匹配并 Warn 日志，不阻塞业务查询；
- **大小写归一**：集合键与 SQL 表名均转小写后比对（与「已带前缀不叠加」的判断口径一致），
  数据库实际表名的大写形式不影响匹配。