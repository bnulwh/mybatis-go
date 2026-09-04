# 项目待办事项

> 整理时间：2026-08-20（基于 `main` 分支当前工作区状态，含 v0.1.11）
> 验证基线：`go build ./...` ✅ · `go vet ./...` ✅ · `go test -count=1 ./...` ✅（全绿）

---

## ✅ 已完成

### v0.1.15（2026-09-04，表名前缀真实表集合匹配）

- **表名前缀按真实表集合精确改写（提高准确率）**：纯前缀匹配存在「改了前缀但物理表未改名时改写指向不存在的表」的问题。新增 `DB.tableNameSet()` 在改写前从**配置的 schema** 获取真实表名集合（查询 `information_schema.COLUMNS`（MySQL）/ `pg_class join pg_namespace`（PG/金仓，`spring.datasource.schema` 指定）/ `sqlite_master`（SQLite），键小写），按数据源（`Config.cacheStore` key `tableNames`）惰性缓存；改写时与集合比对（`prefixRequired`，`rewriteSQLTablesWithSet`）：① 带前缀表名真实存在→按配置前缀改写（配置意图优先）；② **无前缀表名真实存在→保持原样**（核心修复场景，`Test_TablePrefix_SqliteExistingUnprefixed`）；③ 两者均未收录（`CREATE TABLE` 新建表 / 表确实不存在）→沿用前缀匹配兜底（兼容旧行为）；DDL（`CREATE/DROP/ALTER/RENAME/TRUNCATE`，`isDDLStatement`）在 `ExecContext` 执行后使缓存失效（`Test_TablePrefix_SqliteTableSetRefresh`），新建表立即可见；拉取失败记录 Warn 并缓存空集，降级纯前缀匹配不阻塞查询；集合查询直连底层 ConnPool（不经 `applyTablePrefix`，避免循环依赖），仅访问系统表不会被自我改写；`fetchTables` 复用新 `tableListSQL` 生成器（行为不变）。纯函数单测 `Test_rewriteSQLTables_tableSet`（8 组断言：带前缀/无前缀/并存/未知/已带前缀/大小写/schema 限定/退化）；既有前缀测试全部保持兼容。实现细节见 docs/agents/table-prefix.md

### v0.1.14（schema 配置支持）

- **数据库模式（schema）配置支持**：`DatabaseSetting` 新增 `Schema` 字段；配置来源按优先级：`spring.datasource.schema` 键 > JDBC URL query 参数（`currentSchema` / `search_path` / `schema`，由 `parseSchema`/`schemaFromURL` 解析）。PG/Kingbase 连接串自动追加 `search_path=<schema>`（lib/pq 运行时参数），表结构查询（`fetchTables`/`newTableStruct`）按配置 schema 过滤（非 public schema 时 `attrelid` 全限定 `'schema.table'::regclass`）；MySQL 下 schema 即数据库名，显式配置时覆盖表结构查询库名（DSN 不变）、未配置回退库名；SQLite 忽略。未配置时行为与历史完全一致（PG 默认 `public`）。回归测试 `Test_schemaFromURL` / `Test_parseSchema` / `Test_generateConn_schema` / `Test_effectiveSchema` / `Test_parseDatabaseConfig_schema`

### v0.1.12（2026-08-20）

- **M-03 PG/金仓 useGeneratedKeys RETURNING 支持（P4）**：`backfillGeneratedKey` 依赖 `sql.Result.LastInsertId()`，lib/pq 返回 error → 回填跳过（MySQL/SQLite 正常）。新增 RETURNING 路径：当数据库为 PostgreSQL / KingbaseES 且 `useGeneratedKeys` + `keyProperty` 已指定时，自动追加 `RETURNING col`（keyColumn 显式指定或 keyProperty 驼峰转下划线），改用 `QueryContext` + `Scan` 读取生成的 ID 并回填；MySQL / SQLite 仍走 `LastInsertId()` 路径，行为不变。回归测试 `Test_keyColumnToSnake` / `Test_toInt64` / `Test_needsReturning_noDb`（`orm/base_mapper_test.go`）+ `Test_SqliteGeneratedKeysBackfill`（原有，验证无回归）
- **P2-3 依赖升级**：`go-sql-driver/mysql` v1.6.0 → v1.10.0；`beevik/etree` v1.1.0 → v1.7.1；`lib/pq` v1.10.1 → v1.12.3；`go.mod` go 版本升至 1.24.0（新依赖要求）；`lib/pq` 迁移 `pgx/v5` 已评估，当前 lib/pq v1.12.3 仍可维护，迁移收益有限暂缓

### v0.1.11（2026-08-20）

- **P4-2 大结果集流式读取**：`orm.QueryStream(ctx, sql, args...)` 返回 `*RowStream`（`orm/row_stream.go`），`Next()`/`Row()` 逐行消费（内存 O(1)），`Scan(&dest)` 填充结构体或 map，`Err()`/`Count()`/`Close()` 齐备；Mapper 代理支持 select 方法返回 `(*RowStream, error)`（`BaseMapper.executeStream`，结果类型与 XML resultType 解耦）；行数上限遵循 P4-3、ctx 超时遵循 P4-1、扫描失败不静默丢行（M-05 精神）。回归测试 `Test_QueryStreamBasic/Parity/Scan/EarlyClose/RowLimit/Err` + `Test_SqliteStreamMapper`
- **P2-5 跟进修复 MinDuration 0 哨兵冲突**：`updateMinDuration` 以 `MinDuration==0` 兼作「未初始化」标记时，真实 0ms 最小值会被后续较大值误覆盖；新增 `minDurationInit` 原子标志区分，回归测试 `Test_updateMinDuration_ZeroCollision` + `Test_UpdateUsageConcurrent` 改为与观测样本极值比对

### v0.1.10（2026-08-19）

- **P4-3 全局查询行数上限**：`fetchRows` 默认最多返回 10000 行；`orm.SetDefaultRowLimit(n)` / `orm.DefaultRowLimit()`（负数不限制、0 不返回行），达到上限 Warn 提示。回归测试 `Test_DefaultRowLimit` / `Test_QueryRowLimit`

### v0.1.9（2026-08-19）

- **P4-1 带超时/上下文的执行 API**：`ExecuteContext`/`QueryContext` + `orm.SetDefaultTimeout(d)`（默认 5 分钟），Mapper 代理同步接入。回归测试 `Test_DefaultTimeout` / `Test_withExecTimeout` / `Test_ExecuteQueryContext` / `Test_ExecuteContextCanceled` / `Test_ExecuteContextShortTimeout` / `Test_ExecuteNoTimeout`
- **M-05 scan error 聚合（P14）**：`convert2Results` 转换失败不再静默丢行，新增 `ResultConvertReport`/`ResultConvertError` 按行号/列名聚合。回归测试 `Test_Convert2Results_Report` / `Test_Convert2Results_ResultMapColumnErr`

### v0.1.8（2026-08-19）

- **MP 内置 CRUD 内存自动生成**：XML 含 resultMap 但缺 MP 方法时，加载期自动补生成 10 个 CRUD（不落盘、不覆盖手写），逻辑删除支持 deleted/del_flag 双约定
- **MyBatis-Plus codegen 增强**：`schema2code -mp` 生成 BaseMapper 标准方法名 XML；codegen 检测 `<foreach>` 自动为批量方法生成切片签名；`SelectCount` 返回 `[]int64`
- **M-02 `<if>` 数值比较（P10）**：`!= 0`/`> 0`/`== 0` 及 `x.length > 0` 集合长度。回归测试 `Test_parseIfConditionsFromText_Compare` / `Test_IfCondition_CheckCompare` / `Test_IfCompare_Samples`
- **M-01 convert2Map nil 指针 panic（P8）**：`safeIndirectInterface` 统一安全解引用。回归测试 `Test_Convert2Map_NilPtrField` / `Test_Convert2Map_NilPtrMapValue`

### v0.1.7（2026-08-19）

- **MySQL DATETIME 修复（P2）**：DSN 自动追加 `?parseTime=true&loc=Local`；`resolveConverter`/`newInstance` 兼容 `[]uint8` 兜底
- **自定义 DB/DSN 注入**：`Config.ConnPool`/`Config.DSN` 注入，`orm.Open(cfg)` 优先使用

### v0.1.6（2026-08-19）

- **S-01~S-11 RuoYi Mapper 兼容性修复**（11 项）：`<where>`/`<set>` 标签、点号参数 `#{a.b}`、原始替换 `${...}`、`parameterType="Long"` 基础类型映射、resultMap `<association>`/`<collection>` 真实关联类型、裸标识符布尔 `<if>` 求值、`<include>` 内参数替换、nil 参数反射零值防御、排除非 Mapper XML、`useGeneratedKeys` 自增主键回填

### v0.1.5 及更早（2026-08-14 之前）

- **P0-1 预编译语句缓存接通 + PG/Kingbase 占位符修复**：`DB.ExecContext/QueryContext` 走 `PreparedStmtDB`，`formatSQL` 按方言 `?`→`$n`，`spring.datasource.prepared-stmt` 配置
- **P1-3/P1-4 结果集转换优化**：扫描目标跨行复用、列转换函数预编译、resultMap 字段索引预编译
- **P2-5 静态 SQL 生成缓存**：`sync.Once` 缓存无参 SQL 拼接
- **P0-2 调试日志零开销**：`log.IsDebugEnabled()` 守卫热路径序列化
- **P1-1 全局缓存 map 加锁**：`modelCache`/`mapperCache` 加 `sync.RWMutex`
- **P1-2 PreparedStmtDB 缓存上限**：`maxPreparedStmts=100`，满后降级直接执行
- **P1-3 默认日志开启**：`ConsoleLogger.Level` 分级，默认 Warn
- **P3-1 核心代理机制测试**：顺带修复 `args:name` tag 解析 bug
- **P3-2 覆盖率**：utils 81.7%，orm 67.9%
- **P3-3 性能基准**：`BenchmarkGenerateSQL_NoParam` / `BenchmarkConvert2Results`
- **P3-4 CI**：`.github/workflows/ci.yml`
- **P3-5 无需 sqlmock**（已关闭）
- **P1-1 时间类型转换修复**：`newInstance`/`convertTimeToTime`/`convertInstanceType`
- **SQLite 支持**：纯 Go 驱动 modernc.org/sqlite，`cmd/sqlitedemo`
- **KingbaseES 支持**：复用 `lib/pq` 注册 `kingbase` 驱动，`cmd/kingbasedemo`
- **事务支持**：`Begin`/`Commit`/`Rollback`，Mapper 方法自动参与
- **多数据源支持**：`InitializeDataSources`/`UseDataSource`/`AddDataSource`
- **JDBC URL IPv6、`LoadProperties` 健壮性、`ReConnect` 重建预编译缓存**
- **P3-1~P3-4 仓库卫生**：文件权限、.gitignore、test.xml 忽略、proxy_value.go 条目清理
- **P2-1 ioutil 弃用替换**：`ioutil.ReadFile/WriteFile` → `os.ReadFile/WriteFile`
- **P2-2 死代码清理**：删除 `convert2Interfaces`、`Rows` 接口
- **P2-4 .gitignore 去重**
- **P2-5 MinDuration 初始化语义**：CAS 循环原子更新 Max/Min（修复 SwapInt64 互相覆盖）

---

## 🔴 待完成

### 框架功能（M 系列）

- **M-04 自定义 `resultType` 短类名不解析（P12）**：`parseResultTypeFrom`（`types/common.go`）只认 JDBC 基础类型，未知类型返回 `map[string]interface{}` → 注册校验失败。应在已注册 model 中按短类名解析
- **M-06 XML 无 parameterType 但有入参时注册校验失败（P6）**：`methodFieldCheck` 在 `ArgsLen > 0 && !Param.Need` 时报错；Java @Param 多参数在 XML 不写 parameterType 时无法注册。GenerateSQL 已支持无 parameterType 走参数渲染（S-04），注册期校验应同步放宽（按方法签名推断 Need）
- **M-07 分页支持（P22）**：无 PageHelper 等价物，`selectList` 类操作无 limit；现仅内存分页。建议提供分页参数约定或 SQL 层分页助手（与 P4-2 流式读取互补）

---

## 📌 业务侧约定（非框架改动，生成器/业务层处理）

- **P1**：JDBC URL 的 query 参数整体不解析（解析正则 `([\w._-]+)` 不含 `?`）→ 需自定义 DSN 时用 `Config.DSN`（v0.1.7 新增）。v0.1.13 起 schema 相关 query 参数（`currentSchema` / `search_path` / `schema`）可直接解析并生效，其余 query 参数仍走 `Config.DSN`
- **P6/P7**：XML 副本由生成器补 `parameterType`（多参数→`java.util.Map`、List/数组→`java.util.List`、反向删除多余）；`List<SysRoleDept>` 等泛型值需规范化为 `java.util.List`（`<`/`>` 未转义会截断标签）
- **P8/P10**：model 字段全部值类型（`time.Time`/`int64`，杜绝 `*T`）；查询参数统一 `map[string]interface{}` + `QueryMap()` 排除零值（规避 M-02；M-01 已修复，`*T` 字段不再 panic，但值类型仍是更稳妥的约定）
- **P9**：Java 泛型映射（`Set<X>`/`X[]`/`Map<String,Object>`）由生成器 j2g 处理
- **P11**：嵌套 association/collection 的联表列由生成器补平铺映射（S-06 已修 codegen 类型，运行时平铺靠生成器）
- **P13**：select 一律 `([]T, error)`（单对象取 `rs[0]`），insert/update/delete 为 `(int64, error)`；流式 select 可返回 `(*orm.RowStream, error)`
- **P16**：MyBatis-Plus 内置操作由 `GoExtraMapper` 手写补充。**框架已支持两条免手写路径**：① `schema2code -mp` / `TableStructure.SaveMPToFile` 产出 BaseMapper 标准方法名 XML；② XML 含 resultMap 且缺 MP 内置方法时加载期内存自动补生成（`ensureMPBuiltinCRUD`，不落盘、不覆盖手写）。仅当 Java 端存在无 XML 的自定义方法时才需手写 GoExtraMapper
- **P17**：所有 RuoYi 数据权限查询入参带 `params:{"dataScope":""}` 默认值（`${...}` 字符串原样替换，注意 SQL 注入面）
- **P19**：if 表达式支持 null/empty/bool/数值比较四类（M-02 已加 `!= 0`/`> 0`/`== 0` 及 `x.length > 0` 集合长度；不期望任意 OGNL：三元/方法调用/字符串比较均不支持）
- **P20/P21**：MySQL 方言函数（反引号/ifnull/find_in_set/status 比较）由生成器改为 PG/金仓通用语法；本地 PG 模拟可补 `find_in_set(int, text)` 重载
- **P24**：定位三步——① `orm.Query("SELECT DATABASE()")` 确认连接 → ② `orm.Query` 执行同 SQL 确认 SQL → ③ 看 scan error 日志

---

## 验收命令速查

```bash
go build ./...                                            # 编译
go vet ./...                                              # 静态检查
go test -count=1 ./...                                    # 全量测试（SQLite 端到端无需外部 DB）
go run ./cmd/sqlitedemo                                   # SQLite 全流程示例
bash coverage.sh                                          # 覆盖率报告
```
