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

## 验收命令速查

```bash
go build ./...                                            # 编译
go vet ./...                                              # 静态检查
go test -count=1 ./orm/... ./types/... ./utils/...        # 全量测试（SQLite 端到端无需外部 DB）
go run ./cmd/sqlitedemo                                   # SQLite 全流程示例
bash coverage.sh                                          # 覆盖率报告
```
