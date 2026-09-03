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
    - 否则对该标识符判定 `prefixable` 后插入前缀。
- `commaList`：`FROM t1, t2` / MySQL `UPDATE t1, t2 SET` 逗号分隔的多表列表。
  `,` 后继续期待表名；遇 `(`/`)` 或 `WHERE/SET/ON/GROUP/ORDER/LIMIT/VALUES/...` 等断句关键字复位
  （`clauseBreakWords`）。`INSERT INTO t (a, b) VALUES (1, 2)` 中列名列表的逗号不会误触发
  （`INTO`/`UPDATE` 后的 `(` 立即复位 commaList）。

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
  - `Test_parseTablePrefix` / `Test_parseDatabaseConfig_tablePrefix` / `Test_parseMultiDatabaseConfig_tablePrefixInherit` —— 配置解析与多源继承。
- 端到端（SQLite 真实执行，物理表带 `test_` 前缀，SQL 保持不带前缀）：
  - `Test_TablePrefix_SqliteQuery` —— `Execute/Query` 直调路径改写生效；
  - `Test_TablePrefix_SqliteMapper` —— Mapper 代理 Insert/Select/Count + `useGeneratedKeys` 回填；
  - `Test_TablePrefix_GlobalSetter` —— 编程式 `SetTablePrefix`；
  - `Test_TablePrefix_GetAndTransactions` —— 事务内 SQL 同样生效 + `GetTablePrefix`；
  - `Test_TablePrefix_SqliteTableStructure` —— 带前缀时内置表结构探查（`sqlite_master`/
    `pragma_table_info`）不受影响。

## 已知边界（有意简化）

- 表关键字仅识别 `FROM/JOIN/INTO/UPDATE/TABLE`；`MERGE INTO`、`CREATE INDEX ... ON`（ON 是
  JOIN 条件关键字，不能当表关键字）、MySQL `RENAME TABLE a TO b` 的目标名等罕见写法不加前缀；
- 非 `public`/`main` schema 的限定名整段跳过（跨库引用由各自环境自行管理）；
- 嵌套 CTE（子查询内部再 `WITH`）名称未收集，若 Mapper 使用会报 SQL 错误（显式失败而非静默错数据）。