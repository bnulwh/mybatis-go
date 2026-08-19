# 项目待办事项

> 整理时间：2026-08-14（基于 `main` 分支 `b18f360` 之后的当前工作区状态）
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
- [x] **P2-5 ✅ 已修复 MinDuration 初始化语义**：`MinDuration` 初始改为 0（无数据语义）；`UpdateUsage` 用 CAS 循环原子更新 Max/Min（修复并发下 `SwapInt64` 互相覆盖的统计 bug），`String()` 统计字段改原子读

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

- [ ] **P4-1 带超时/上下文的执行 API**：`Execute`/`Query` 内部固定 `context.Background()`，慢 SQL 会无限挂起占住连接；建议暴露 `ExecuteContext`/`QueryContext`（或配置默认超时）
- [ ] **P4-2 大结果集流式读取**：`fetchRows` 全量进内存，10 万行以上内存压力明显；建议提供流式回调或分页选项

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

### 🔴 P0 待修复（崩溃 / 错误 SQL）

（无待修复项 — 已全部完成）

### 🟡 P1 待修复（功能缺失）

（无待修复项 — 已全部完成）

### 🟢 P2 待修复（健壮性）

- **S-10 `filterMapperFiles` 全目录扫描 `.xml`**：`mybatis-config.xml` 被误加载为 Mapper（空 namespace），需排除非 Mapper XML（按 `mybatis-config` 根标签或 namespace 判定）。
- **S-11 `useGeneratedKeys`/`keyProperty` 属性被忽略**：自增主键不回填，`parseSqlFunctionFromXmlNode` 需解析并暴露这两个属性。

---

## 验收命令速查

```bash
go build ./...                                            # 编译
go vet ./...                                              # 静态检查
go test -count=1 ./orm/... ./types/... ./utils/...        # 全量测试（SQLite 端到端无需外部 DB）
go run ./cmd/sqlitedemo                                   # SQLite 全流程示例
bash coverage.sh                                          # 覆盖率报告
```
