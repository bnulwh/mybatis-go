# 项目待办事项

> 整理时间：2026-08-19（基于 `main` 分支当前工作区状态，含 v0.1.7）
> 验证基线：`go build ./...` ✅ · `go vet ./...` ✅ · `go test -count=1 ./orm/... ./types/... ./utils/...` ✅（全绿）

---

## ✅ 已完成（近期收尾）

### P0-1 预编译语句缓存接通 + PG/Kingbase 占位符修复（commit `a6547fa`）
- `DB.ExecContext/QueryContext/QueryRowContext` 直接走 `ConnPool`（`PreparedStmtDB` 包装），不再用 `db.DB()` 解包绕过缓存
- `formatSQL` 按方言把 `?` 转 `$1、$2…`（lib/pq 不支持 `?`），参数化查询在 PG/Kingbase 从「跑不通」变为可用
- 无参数 SQL（DDL/静态查询）直接执行不进缓存（PG 扩展协议不支持 prepare DDL）
- 新增 `spring.datasource.prepared-stmt` 配置（默认 `true`，PgBouncer 等代理场景可关）
- 回归测试：`Test_PreparedStmtCacheWiredUp` / `Test_PreparedStmtBypassedInTx` / `Test_DBFormatSQLDialect` / `Test_PreparedStmtPropertyParsing`

### P1-3 / P1-4 结果集转换优化（commit `7def86a`）
- `fetchRows` 扫描目标（`prepareColumns`）只建一次跨行复用，替代每行每列 `new(sql.NullXxx)`；构建延迟到首次 `rows.Next()`（modernc sqlite 等驱动在首行后才填充 `ScanType`），并对 nil `ScanType` 做 `sql.NullString` 兜底
- 列值转换函数预编译成 `convertFn` 表（一次查询一次），替代每行每列 `ScanType().String()` + switch 分派
- resultMap 的 property→字段索引预编译（`buildFieldIndexMap`），`setColumnValues` 改用 `FieldByIndex`，替代每列每行两次 `FieldByName` O(N) 匹配
- 回归测试：`Test_buildFieldIndexMap` / `Test_setColumnValuesPrepared`

### P2-5 静态 SQL 生成缓存（commit `b18f360`）
- `SqlFunction.generateSqlWithoutParam` 用 `sync.Once` 缓存无参 SQL 拼接结果，替代每次调用重新拼接
- 回归测试：`Test_GenerateSqlWithoutParamCached` / `Test_GenerateSqlWithoutParamConcurrent`

### 历史（2026-08-14 之前）
- ✅ P1-1 时间类型转换修复（`newInstance`/`convertTimeToTime`/`convertInstanceType`）
- ✅ SQLite 支持（纯 Go 驱动 modernc.org/sqlite，`cmd/sqlitedemo`）
- ✅ 人大金仓 KingbaseES 支持（复用 `lib/pq` 注册 `kingbase` 驱动，`cmd/kingbasedemo`）
- ✅ 事务支持（`Begin`/`Commit`/`Rollback`，Mapper 方法自动参与）
- ✅ 多数据源支持（`InitializeDataSources`/`UseDataSource`/`AddDataSource`）
- ✅ JDBC URL IPv6、`LoadProperties` 健壮性、`ReConnect` 重建预编译缓存等可靠性修复
- ✅ P3-1~P3-4 仓库卫生（文件权限、.gitignore、test.xml 忽略、proxy_value.go 条目清理）

---

## 🔴 P0 — 性能（剩余）

### P0-2 ✅ 已修复（commit `489638e`）
- `log` 包新增 `IsDebugEnabled()`（可选接口 `debugEnabler`，未实现者保守返回 true 不丢日志）
- 四个热路径序列化点（`fetchRows`/`convert2Results`/`buildReturnValues`/`executeMethod`）改为 `if log.IsDebugEnabled()` 守卫；顺带删除 `executeMethod` 重复的 ToJson 调试行
- 新增 `log/logger_test.go`
- 解析期（init 阶段）XML 日志优先级低，可后续一并处理

---

## 🟡 P1 — 健壮性 / 并发安全

### P1-1 ✅ 已修复（全局缓存 map 加锁）
- `modelCache` / `mapperCache` 加 `sync.RWMutex`：注册写锁、创建读锁；`bindSqls`（修改 `funcInfo.SqlFunc`）持写锁
- 新增并发冒烟测试 `Test_ModelCacheConcurrent` / `Test_MapperCacheConcurrent`

### P1-2 ✅ 已修复（PreparedStmtDB 缓存上限）
- 新增 `maxPreparedStmts = 100` 上限；缓存满后参数化 SQL 降级为直接执行（不再缓存、不钉连接、无泄漏）
- 新增回归测试 `Test_PreparedStmtCacheCap`

### P1-3 ✅ 已修复（默认日志开启）
- `ConsoleLogger` 引入 `Level` 分级（Debug/Info/Warn/Error），默认 `Enable=true, Level=WarnLevel`：不调用 `SetLogger` 时 Warn/Error 可见
- 新增 `log.DebugEnabled/InfoEnabled/WarnEnabled/ErrorEnabled` 级别查询 + `Test_DefaultLevel`

---

## 🟢 P2 — 代码质量 / 现代化

- [x] **P2-1 ✅ 已修复 ioutil 弃用**：`orm/orm_init.go`、`types/result_map.go`、`types/sql_mapper.go`、`types/table_struct.go`、`types/xml_parse.go` 的 `ioutil.ReadFile/WriteFile` 全部改为 `os.ReadFile/WriteFile`
- [x] **P2-2 ✅ 已修复 死代码清理**：删除 `orm/base_mapper.go` 的 `convert2Interfaces` 与 `orm/interfaces.go` 的 `Rows` 接口（均无引用）
- [ ] **P2-3 依赖升级**：`go-sql-driver/mysql v1.6.0` → v1.8.x；`lib/pq v1.10.1` 已进维护模式（评估 `pgx/v5` 迁移）；`beevik/etree v1.1.0` 有更新版；go.mod 已为 `go 1.21`（旧 TODO 中「go 1.14」已过时）
- [x] **P2-4 ✅ 已修复 .gitignore 去重**：`/generator /mysqldemo /postgresdemo /schema2code /temp/ /orm/test.xml /reasonix.toml` 重复块已合并
- [x] **P2-5 ✅ 已修复 MinDuration 初始化语义**：`MinDuration` 初始改为 0（无数据语义）；`UpdateUsage` 用 CAS 循环原子更新 Max/Min（修复并发下 `SwapInt64` 互相覆盖的统计 bug），`String()` 统计字段改原子读。**跟进修复（2026-08-20）**：0 哨兵与真实 0ms 测量值冲突 —— 以 `MinDuration==0` 兼作「未初始化」标记时，真实最小值 0 会被后续较大值误覆盖（`updateMinDuration` CAS 分支 bug）；新增 `minDurationInit` 原子标志区分「未初始化」与「已初始化为 0」，回归测试 `Test_updateMinDuration_ZeroCollision` + 并发测试 `Test_UpdateUsageConcurrent` 改为与观测样本极值比对（消除绝对毫秒阈值的调度抖动假阳性）

---

## 🔵 P3 — 测试 / CI

### P3-1 ✅ 已修复（核心代理机制测试）
- `orm/proxy_value_test.go` 补齐：`proxyValue` 注入与调用（含 `args` tag 映射）、`makeReturnType`/`makeParamType`/`methodFieldCheck` 校验与 panic 分支
- **顺带修复真实 bug**：文档化的 `args:name` tag 格式因 Go `reflect.StructTag.Get` 解析不到未加引号的值而失效；新增 `getTagArgNames` 兼容 `args:name`（legacy）与 `args:"name"`（标准）两种写法，应用于 `proxy_value.go` / `param_type.go`

### P3-2 ✅ 已修复（覆盖率）
- 新增 `utils/file_utils_test.go`、`utils/list_utils_test.go`（此前无测试）；utils 覆盖率 81.7%，orm 67.9%，log 由 0% 提升（P0-2/P1-3 时已加测试）

### P3-3 ✅ 已修复（性能基准）
- 新增 `BenchmarkGenerateSQL_NoParam`（~500 ns/op，静态 SQL 缓存生效）与 `BenchmarkConvert2Results`（100 行 ~232 µs/op）

### P3-4 ✅ 已修复（CI）
- 新增 `.github/workflows/ci.yml`：Ubuntu + Go 1.21，build / vet / 全量测试 / 覆盖率

### P3-5 ✅ 已关闭（无需 sqlmock）
- SQLite 端到端测试已覆盖完整执行路径（Mapper/事务/表结构，无外部 DB）；PG/Kingbase/MySQL 方言特有逻辑（占位符转换、DSN、驱动注册）已有单测（`kingbase_dialector_test`、`Test_DBFormatSQLDialect` 等）；引入 sqlmock 收益有限，不新增依赖

---

## 🟣 P4 — 功能建议

- [x] **P4-1 ✅ 已实现 带超时/上下文的执行 API**：`Execute`/`Query` 内部原来固定 `context.Background()`，慢 SQL 会无限挂起占住连接。现在（`orm/sql_execute.go`）：① 新增 `DefaultTimeout()`/`SetDefaultTimeout(d)` 全局超时设置（默认 5 分钟，`Config.MaxTimeout` 已有同值常量，但两者独立——`MaxTimeout` 控制连接池生命周期，`DefaultTimeout` 控制语句执行超时）；② 新增 `ExecuteContext(ctx, sql, args...)` / `QueryContext(ctx, sql, args...)` 支持调用方 context；③ 内部 `executeWithResult`/`queryRows` 通过 `withExecTimeout` 在 ctx 无 deadline 时自动叠加全局默认超时（有 deadline 时保持调用方原样，防止误叠加）；④ Mapper 代理执行路径（`base_mapper.go executeMethod`）同步通过 `context.Background()` 接入，慢 SQL 同样受全局超时保护。回归测试：`Test_DefaultTimeout` / `Test_withExecTimeout` / `Test_ExecuteQueryContext` / `Test_ExecuteContextCanceled` / `Test_ExecuteContextShortTimeout` / `Test_ExecuteNoTimeout`（`orm/sql_execute_test.go`）。
- [x] **P4-2 ✅ 已实现 大结果集流式读取**：`fetchRows` 全量进内存，10 万行以上内存压力明显。新增流式读取：① 顶层 `QueryStream(ctx, sql, args...)` 返回 `*RowStream`（`orm/row_stream.go`）：`Next()`/`Row()` 逐行消费（游标保持打开、任意时刻只保留当前一行，内存 O(1)），`Scan(&dest)` 填充结构体或 map（列名→字段名匹配：原名/首字母大写/下划线转驼峰/大小写不敏感，`utils.ChangeType` 转换），`Err()`/`Count()`/`Close()` 齐备；② Mapper 代理支持 select 方法返回 `(*RowStream, error)`（`BaseMapper.executeStream` 分流，`orm_cache.go` 按返回类型走流式路径；`makeReturnType`/`checkSql` 放行 `*RowStream`，结果类型与 XML resultType 解耦，仅限 select）；③ 语义对齐：行数上限遵循 P4-3 全局 `DefaultRowLimit`（打开时快照，达到上限停止读取并 Warn）、ctx 无 deadline 叠加 P4-1 全局超时（cancel 延至 Close 释放）、扫描失败不静默丢行（`Err()` 返回行号明细，M-05 精神）、`Close()` 幂等（未读完也必须 Close 释放连接）。回归测试：`Test_QueryStreamBasic/Parity/Scan/EarlyClose/RowLimit/Err`（`orm/row_stream_test.go`）+ `Test_SqliteStreamMapper`（`orm/sqlite_test.go`，SQLite 端到端 Mapper 流式 select）。分页选项仍待实现（见 M-07）。
- [x] **P4-3 ✅ 已实现 全局查询行数上限**：`fetchRows`（`orm/sql_execute.go`）默认最多返回 10000 行，防止大结果集 OOM/拖垮连接；新增 `DefaultRowLimit()`/`SetDefaultRowLimit(n)` 全局系统设置——负数不限制（返回全部）、0 不返回任何行；达到上限停止读取并 Warn 日志提示（`SetDefaultRowLimit(-1)` 可恢复全部返回）。所有查询路径（`Query`/`QueryContext`/Mapper 代理）共用同一 `fetchRows`，自动生效。回归测试：`Test_DefaultRowLimit` / `Test_QueryRowLimit`（上限截断/0 不返回/负数全量/上限大于总行数）（`orm/sql_execute_test.go`）。

---

## 📁 samples 目录（RuoYi Mapper）兼容性缺陷

> **来源**：`samples/` 目录（从 `threedb.jar` 提取的 23 个 XML / 166 条 SQL，RuoYi 标准表结构，方言 KingbaseES）
> **验证方式**：`types.NewSqlMappers("samples")` 真实解析全部文件 + 代表性函数 `GenerateSQL`/`PrepareSQL` 生成 SQL 复现实际影响
> **整理时间**：2026-08-19

### ✅ 已修复

- **S-01 `<where>` 标签支持（11 处）**：解析器原先只认 `if/include/for/foreach/choose`，`<where>` 整体被丢弃 → WHERE 子句消失、返回全表。新增 `whereSqlFragment`/`sqlWhere`（`types/sql_where.go`），子片段全空时输出空、否则输出 `where` 并剥离首个条件前导 `AND/OR`（大小写不敏感）；`<if>` 内嵌套 `<where>` 亦支持。回归测试 `Test_WhereFragment_*`（7 个用例，`types/sql_where_test.go`）。
- **S-02 `<set>` 标签支持（11 处）**：UPDATE 语句缺失 set 子句 → 语法错误（如 `updateConfig`）。新增 `setSqlFragment`/`sqlSet`（`types/sql_set.go`），子片段非空时输出 `set` 并剥离前导/尾随逗号（`<if>` 片段常见尾部 `,`），空时输出空；`<include>` 内嵌套 `<set>` 亦支持。回归测试 `Test_SetFragment_*`（7 个用例，`types/sql_set_test.go`，含 samples `updateConfig` 真实回归）。
- **S-03 点号参数 `#{a.b}` 支持（48 处）**：`#{params.beginTime}`、`#{item.deptId}` 等不替换 → SQL 残留 `#{...}` 字面量报错。`parseSqlFragmentParamFromText`/`parseIfConditionsFromText` 参数名正则扩展为 `[\w.]+`，并新增 `lookupParam`（`types/sql_fragments.go`）按 `.` 分段遍历嵌套 map/struct（含指针）；`buildParams` 补 `reflect.Map` 分支并保留 `item` 本身；顺带修复 `SliceSqlParam` 参数双重包裹（`sliceArgsFrom`，`types/common.go`，`GenerateSQL`/`PrepareSQL` 取 `args[0]`）使 foreach 切片参数可用。回归测试 `Test_DotParam_*`（7 个用例，`types/sql_param_dot_test.go`，含 samples `selectConfigList`/`updateDeptChildren` 真实回归）。
- **S-04 原始替换 `${params.dataScope}`/`${sql}` 支持（6 处）**：dataScope 数据权限逻辑丢失、`createTable` 的 `${sql}` 残留字面量。`sqlFragmentParam` 新增 `Raw` 标记（`parseSqlFragmentParamFromText` 按 `${` 前缀识别），`simpleSql` 渲染（`types/sql_fragments.go`）对 `${}` 直接内联注入 `rawFormatValue`（不加引号、不入占位符参数，Prepare 亦不产生 args）；`GenerateSQL`/`PrepareSQL` 在无 `parameterType` 但传入参数时改走参数渲染，使 `createTable` 可用。回归测试 `Test_RawParam_*`（6 个用例，`types/sql_param_raw_test.go`，含 samples `selectDeptList`/`createTable` 真实回归）。
- **S-05 `parameterType="Long"` 基础类型映射（55 处）**：`Long` 落入 StructSqlParam → 标量查询 `#{configId}` 残留字面量、`collection="array"` 批量删除生成空 `in ()` 子句。`parseSqlParamTypeFrom` 补充 `LONG`/`BIGINT` → BaseSqlParam，`toGolangType` 补充 `LONG`/`BIGINT` → `int64`（codegen 不再生成 `models.Long`）；`GenerateSQL`/`PrepareSQL` 新增 `effectiveParamType`（`types/sql_function.go`）按实际参数类型分派（切片→Slice、标量→Base、map/struct→Map，time.Time 除外），`validParam` 的 BaseSqlParam 容忍切片；顺带修复 `buildParams` 标量分支二次格式化（`types/sql_fragments.go`，foreach 值不再被重复加引号）。回归测试 `Test_LongParam_*` 等（9 个用例，`types/sql_param_long_test.go`，含 samples `deleteConfigByIds`/`deleteConfigById` 真实回归）。
- **S-06 resultMap 的 `<association>`/`<collection>` 被当普通 `<result>`**：生成模型出现 `Dept string`/`Roles string` 假字段，嵌套映射丢失。`ResultItem` 新增 `Kind`（id/result/association/collection）+ `JavaType`/`OfType`/`ResultMap` 引用（`types/result_item.go`），`parseResultItemFromXmlNode` 按节点名分发解析；`golangType` 依据 javaType/ofType/嵌套 resultMap（`SysUserResult` 的 `dept`→`*SysDept`、`roles`→`[]SysRole`，`GenTableResult` 的 `columns`→`[]GenTableColumn`）建立真实关联类型，不再生成 `Dept string`/`Roles string` 假字段；`makeColumnMap` 跳过无 column 的关联项（避免 `mp[""]` 污染），`hasTimeItem` 跳过关联项防 nil Type panic。回归测试 `Test_ResultItemGolangType_Samples`/`Test_ResultMapGenerateContent_Samples`（`types/result_item_test.go`，samples `SysUserMapper` 真实回归）。
- **S-07 `<if test="deptCheckStrictly">` 裸标识符恒为 true**：无 `null`/`''` 比较的裸标识符（`deptCheckStrictly`/`menuCheckStrictly`）解析不出任何条件 → `<if>` 恒 true，布尔 false 时也不剔除（MyBatis 语义丢失）。`parseIfConditionsFromText`（`types/sql_fragments.go`）新增 `boolCheckCond` 分支：`^[\w.]+$` 裸标识符按布尔条件解析；`ifCondition.checkValue` 对 `boolCheckCond` 取真实布尔值（false/缺失/nil/非布尔一律不通过），不再无条件通过。回归测试 `Test_parseIfConditionsFromText_Bool`/`Test_IfCondition_CheckBool`/`Test_IfBool_Samples`（`types/sql_fragments_test.go`，samples `selectDeptListByRoleId` 真实回归：`deptCheckStrictly=true` 含 `not in` 子句、false/缺失剔除）。
- **S-08 `<include>` 内 `#{}`/`${}` 参数不替换（P1）**：`<sql>` 块原来只取纯文本、嵌套标签被忽略，include 复制后参数替换失效。`SqlElement` 新增 `Fragments []*sqlFragment`，嵌套标签（`<where>/<if>/<foreach>` 等）解析为片段；`sqlInclude` 渲染片段并正确收集占位符参数（随 S-01 一并修复）。
- **S-09 `validParam` 对 `nil` 参数反射零值 panic**：`PrepareSQL(nil)`/`GenerateSQL(nil)` 崩溃（`reflect.Indirect(reflect.ValueOf(nil)).Type()` / `convert2Map` / `getFormatValue` / `validValue` / foreach nil 元素多处反射零值 panic）。`validParam`（`types/sql_param.go`）Base/Slice/Map/Struct 各分支拒绝 nil 与类型化 nil 指针（返回错误而非 panic）；`GenerateSQL`/`PrepareSQL`（`types/sql_function.go`）对无 parameterType 函数传 nil 也返回错误；`convert2Map`/`validValue`/`getFormatValue`（`types/common.go`）零值与 nil 防御（`getFormatValue(nil)` 按 SQL `null` 渲染）；`buildParams`（`types/sql_fragments.go`）foreach nil 元素直接保留 item 键不反射。回归测试 `Test_ValidParam_Nil`/`Test_GenerateSQL_NilParam`/`Test_ValidValue_Nil`/`Test_Convert2Map_InvalidValue`（`types/sql_param_nil_test.go`，覆盖 Base/Struct/Slice 各类型 nil、类型化 nil 指针、切片含 nil 元素、无 parameterType 函数）。
- **S-10 `filterMapperFiles` 全目录扫描 `.xml`**：`mybatis-config.xml`（根标签 `<configuration>`、无 namespace）被误加载为 Mapper → 空 namespace 污染 `NamedMappers[""]`、codegen 生成空 `.go` 文件。`loadMapper`（`types/sql_mapper.go`）新增根标签校验（非 `<mapper>` 一律跳过）+ namespace 非空校验；`filterMapperFiles`（`types/sql_mappers.go`）改用 `strings.HasSuffix` 判定 `.xml` 并跳过目录（顺带修复短文件名 `path[len-4:]` 越界 panic）。回归测试 `Test_filterMapperFiles_Suffix`/`Test_LoadMapper_SkipNonMapper`/`Test_NewSqlMappers_SkipConfig`（`types/sql_mappers_test.go`，samples 真实回归：`mybatis-config.xml` 不再作为 Mapper 加载，`NamedMappers` 无空键）。
- **S-11 `useGeneratedKeys`/`keyProperty` 属性被忽略**：自增主键不回填。`SqlFunction`（`types/sql_function.go`）新增 `UseGeneratedKeys`/`KeyProperty`/`KeyColumn` 字段并在 `parseSqlFunctionFromXmlNode` 解析暴露（`useGeneratedKeys="true"` 大小写不敏感）；`executeWithResult`（`orm/sql_execute.go`）返回原始 `sql.Result` 供取 `LastInsertId`；`executeMethod`（`orm/base_mapper.go`）对 Insert + useGeneratedKeys + keyProperty 调用 `backfillGeneratedKey` 回填（struct 指针/ map 入参生效，值传递 struct 无法写回属 Go 语义限制）。回归测试 `Test_parseSqlFunctionFromXmlNode_GeneratedKeys`/`Test_GeneratedKeys_Samples`（`types/sql_function_test.go`，samples 5 个 `insertJob`/`insertPost`/`insertRole`/`insertGenTable`/`insertGenTableColumn` 真实回归）与 `Test_SqliteGeneratedKeysBackfill`（`orm/generated_key_test.go`，SQLite 端到端：两次 insert 后 `Id` 回填为 1/2）。

### 🔴 P0 待修复（崩溃 / 错误 SQL）

（无待修复项 — 已全部完成）

### 🟡 P1 待修复（功能缺失）

（无待修复项 — 已全部完成）

### 🟢 P2 待修复（健壮性）

（无待修复项 — S-01~S-11 已全部完成）

---

## 📄 实战使用问题（源自 docs/mybatis-go使用问题与解决方案.md）

> **来源**：中国化学分包安全监管数据平台 Go 后端改造（RuoYi + MDM 的 24 个 Mapper XML，金仓 KingbaseES，版本 v0.1.7）
> **文档**：`docs/mybatis-go使用问题与解决方案.md`（P1~P24 全记录）
> **整理时间**：2026-08-19

### ✅ 框架已解决（含版本）

- **P2 MySQL DATETIME 丢行（v0.1.7）**：DSN 自动追加 `?parseTime=true&loc=Local`，DATETIME/TIMESTAMP 列可直接 Scan 为 `time.Time`；另 `newInstance`/`resolveConverter` 兼容 `[]uint8` 扫描类型兜底。
- **P3 金仓连接（0.1.x）**：复用 lib/pq 以 `kingbase/kingbase8/...` 名称注册驱动，`jdbc:kingbase8://` 直接可用；无金仓环境可用 Docker PostgreSQL 模拟。
- **P5 多参数 `args` tag（0.1.x）**：`func(a int64, b string) ...` + `` `args:"a,b"` `` 对应 Java @Param；tag 长度须等于参数个数（P3-1 修复 `args:name` 未加引号解析失效的 bug）。
- **P15 非 Mapper XML 排除（S-10，v0.1.6）**：`loadMapper` 校验根标签为 `<mapper>` + namespace 非空，`mybatis-config.xml` 不再误加载（递归目录扫描为预期行为）。
- **P18 foreach 切片参数（S-05，v0.1.6）**：`parameterType="Long"` + `collection="array"` 批量删除走 SliceSqlParam，不再生成空 `in ()`；`collection="list"/"array"` 均可。
- **P23 ScanType 回退（P1-3）**：驱动首行前 ScanType 为空时回退 `sql.NullString`。

### 🟡 框架待修复（新增 TODO，M 系列）

- **M-01 ✅ 已修复 convert2Map 对 struct nil 指针字段 panic（P8，崩溃）**：`convert2Map`（`types/common.go`）对 struct 字段执行 `reflect.Indirect(fval).Interface()`，nil 指针字段（如 `*time.Time`）对零值调 `Interface()` → panic（`reflect: call of reflect.Value.Interface on zero Value`）。新增 `safeIndirectInterface` 统一安全解引用（nil 指针/接口输出 nil，`getFormatValue(nil)` 渲染 SQL null；顺带修复 map 值为 nil 指针的同类 panic，以及接口字段持有指针时不再残留 `*T` 导致格式化丢失）。回归测试 `Test_Convert2Map_NilPtrField` / `Test_Convert2Map_NilPtrMapValue`（`types/sql_param_nil_test.go`）。
- **M-02 ✅ 已修复 `<if>` 数值比较 `!= 0` / `> 0` 不支持（P10，功能缺失）**：`parseIfConditionsFromText` 只识别 `!= null` / `!= ''` / 裸布尔，`userId != 0` 被静默丢弃 → 恒渲染（0 值也生成 `AND u.user_id = 0`）。新增 `compareCheckCond` 条件类型（`types/sql_fragments.go`）：解析 `==`/`!=`/`>`/`>=`/`<`/`<=` + 数值字面量（含负数/小数、点号参数 `params.userId != 0`）；`checkValue` 按真实数值求值（缺失/nil/非数值一律不满足，字符串数字如 `"5"` 亦可）；顺带支持 OGNL 集合长度 `businessTypes.length > 0`（切片/数组/map/字符串按 len 比较）。回归测试 `Test_parseIfConditionsFromText_Compare` / `Test_IfCondition_CheckCompare` / `Test_IfCompare_Samples`（`types/sql_fragments_test.go`，samples `selectUserList` 真实回归：`userId != null and userId != 0` 在 userId=0 时剔除 `AND u.user_id` 子句、userId=5 保留）。
- **M-03 PG/金仓 `LastInsertId()` 不支持 → useGeneratedKeys 回填失效（P4，功能缺失）**：`backfillGeneratedKey` 依赖 `sql.Result.LastInsertId()`，lib/pq 返回 error → 回填跳过（MySQL/SQLite 正常）。建议支持 `RETURNING` 子句（由 `keyProperty`/`keyColumn` 自动追加 `... RETURNING col` 并回读）或序列回读兜底。
- **M-04 自定义 `resultType` 短类名不解析（P12，功能缺失）**：`parseResultTypeFrom`（`types/common.go`）只认 JDBC 基础类型，未知类型返回 `map[string]interface{}` → 注册校验失败（`'map[string]interface{}' != 'SysNotice'`）。应在已注册 model 中按短类名解析。
- **M-05 ✅ 已修复 scan error 静默丢行（P14，健壮性）**：`convert2Results` 对转换失败的行 `continue` 丢弃，仅 Warn 日志 → 「0 行但 SQL 有数据」难排查。新增 `ResultConvertReport`/`ResultConvertError` 聚合错误明细（`orm/result_convert.go`）：行级丢弃（如 ResultT 基本类型转换失败）记 `Row` 行号 + 错误信息并计入 `Skipped`；resultMap 列级类型不匹配按 `Column` 列名聚合（行不丢弃、字段保持零值）。`convert2Results` 改为返回 `(reflect.Value, *ResultConvertReport)`，`executeMethod`（`orm/base_mapper.go`）在 `Skipped>0` 或 `Errors` 非空时输出聚合错误日志（含 namespace/function、total/converted/skipped）。回归测试 `Test_Convert2Results_Report` / `Test_Convert2Results_ResultMapColumnErr`（`orm/result_convert_test.go`）。
- **M-06 XML 无 parameterType 但有入参时注册校验失败（P6，健壮性）**：`methodFieldCheck`（`orm/param_type.go`）在 `ArgsLen > 0 && !Param.Need` 时报 `not need func args`；Java @Param 多参数在 XML 不写 parameterType 时无法注册。GenerateSQL 已支持无 parameterType 走参数渲染（S-04），注册期校验应同步放宽（按方法签名推断 Need）。
- **M-07 分页支持（P22，功能建议）**：无 PageHelper 等价物，`selectList` 类操作无 limit；现仅内存分页。建议提供分页参数约定或 SQL 层分页助手（与 P4-2 大结果集流式读取相关）。

### 📌 业务侧约定（非框架改动，生成器/业务层处理）

- **P1**：JDBC URL 不支持 query 参数（解析正则 `([\w._-]+)` 不含 `?`）→ 需自定义 DSN 时用 `Config.DSN`（v0.1.7 新增）。
- **P6/P7**：XML 副本由生成器补 `parameterType`（多参数→`java.util.Map`、List/数组→`java.util.List`、反向删除多余）；`List<SysRoleDept>` 等泛型值需规范化为 `java.util.List`（`<`/`>` 未转义会截断标签）。
- **P8/P10**：model 字段全部值类型（`time.Time`/`int64`，杜绝 `*T`）；查询参数统一 `map[string]interface{}` + `QueryMap()` 排除零值（规避 M-02；M-01 已修复，`*T` 字段不再 panic，但值类型仍是更稳妥的约定）。
- **P9**：Java 泛型映射（`Set<X>`/`X[]`/`Map<String,Object>`）由生成器 j2g 处理。
- **P11**：嵌套 association/collection 的联表列由生成器补平铺映射（S-06 已修 codegen 类型，运行时平铺靠生成器）。
- **P13**：select 一律 `([]T, error)`（单对象取 `rs[0]`），insert/update/delete 为 `(int64, error)`。
- **P16**：MyBatis-Plus 内置操作（insertUser/selectUserById 等无 XML）由 `GoExtraMapper` 手写补充（namespace 不同，Java 端不加载）。**框架已支持两条免手写路径**：① `schema2code -mp` / `TableStructure.SaveMPToFile` 产出 BaseMapper 标准方法名 XML；② **XML 含 resultMap（基本类型列 + `<id>` 主键）且缺 MP 内置方法时，加载期内存自动补生成**（`ensureMPBuiltinCRUD`，不落盘、不覆盖手写，表名由 type 推导，逻辑删除支持 deleted/del_flag 双约定）。仅当 Java 端存在无 XML 的自定义方法（如 `selectUserRoleGroup`）时才需手写 GoExtraMapper。
- **P17**：所有 RuoYi 数据权限查询入参带 `params:{"dataScope":""}` 默认值（`${...}` 字符串原样替换，注意 SQL 注入面）。
- **P19**：if 表达式支持 null/empty/bool/数值比较四类（M-02 已加 `!= 0`/`> 0`/`== 0` 及 `x.length > 0` 集合长度；不期望任意 OGNL：三元/方法调用/字符串比较均不支持）。
- **P20/P21**：MySQL 方言函数（反引号/ifnull/find_in_set/status 比较）由生成器改为 PG/金仓通用语法；本地 PG 模拟可补 `find_in_set(int, text)` 重载。
- **P24**：定位三步——① `orm.Query("SELECT DATABASE()")` 确认连接 → ② `orm.Query` 执行同 SQL 确认 SQL → ③ 看 scan error 日志。

---

## 验收命令速查

```bash
go build ./...                                            # 编译
go vet ./...                                              # 静态检查
go test -count=1 ./orm/... ./types/... ./utils/...        # 全量测试（SQLite 端到端无需外部 DB）
go run ./cmd/sqlitedemo                                   # SQLite 全流程示例
bash coverage.sh                                          # 覆盖率报告
```
