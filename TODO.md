# 项目待办事项

> 整理时间：2025-07-23（基于 `main` 分支 `c84d5f0` 之后的当前工作区状态）
> 验证基线：`go build ./...` ✅ · `go test ./types/... ./utils/...` ✅ · `go test ./orm/...` ❌（见 P1-1）

---

## P1 — 缺陷修复（阻塞测试全绿）

### P1-1 ✅ 已修复（待提交）
- **修复内容**：
  - `newInstance`：`time.Time`/`sql.NullTime` → `*sql.NullTime`（NULL 友好，与测试期望一致）；`mysql.NullTime` → `*mysql.NullTime`
  - `convertTimeToTime`：统一处理 `*time.Time` / `*sql.NullTime` / `*mysql.NullTime`，避免 DB 扫描路径静默回归
  - `convertInstanceType`：时间类型统一走 `convertTimeToTime`，删除只认 `*mysql.NullTime` 的 `convertMySqlTime2Time`
- **新增回归测试**：`Test_convertTimeToTime`
- **验收**：`go test -count=1 ./orm/... ./types/... ./utils/...` 全绿；`go vet` 通过

---

## P2 — README 路线图未完成项

见 `README.md`「路线图」章节：

- [ ] **SQLite 支持** — 新增 `sqlite3_dialector.go`，依赖 `github.com/mattn/go-sqlite3`；需补充 Demo 与测试
- [ ] **多数据源支持** — 当前 `orm.Initialize()` 仅支持单数据源（配置、连接池、缓存均为全局单例）；需设计按 name 区分的数据源注册/路由机制
- [ ] **其他改进和优化** — 未细化，建议在迭代中拆分为具体条目（可参考 P3 代码质量项）

---

## P3 — 工程化 / 仓库卫生

### P3-1 ✅ 已修复（commit `07f90f5`）
- 42 个文件执行位已统一为 644 并提交；本地已设 `core.filemode false` 防复发

### P3-2 ✅ 已修复（commit `07f90f5`）
- `.gitignore` 已追加 `/generator` `/mysqldemo` `/postgresdemo` `/schema2code` `/temp/`

### P3-3 ✅ 已修复（commit `07f90f5`）
- `orm/test.xml` 确认为 `Test_TableStructure_saveToFile` 的测试输出（自动再生成），已忽略 `/orm/test.xml`
- `reasonix.toml` 已忽略（本机工具配置）
- `temp/` 已忽略（见 P3-2）

### P3-4 ✅ 已修复（commit `07f90f5`）
- 已删除 `.gitignore` 中失效的 `/orm/proxy_value.go` 条目（文件实际已被跟踪提交）

---

## P4 — 代码质量 / 后续改进（非阻塞）

- [ ] **依赖升级**：`go 1.14` 较旧；`mysql v1.6.0`、`pq v1.10.1`、`etree v1.1.0` 均有更新版本（注意 `pq` 已进入维护模式，可评估 `pgx` 迁移）
- [ ] **补充 CI**：无 `.github/workflows`，建议添加 build + test（types/utils/log/orm）+ 覆盖率上传
- [ ] **orm 集成测试去 DB 化**：`orm/` 下数据库相关测试依赖真实 PostgreSQL/MySQL 连接（见 `cmd/postgresdemo`、`cmd/mysqldemo`），可用 `sqlmock` 或接口抽象实现单测化，便于无环境时验证
- [ ] **方言扩展**：`mysql_dialector.go` / `postgres_dialector.go` 已有抽象，可顺势评估 SQLite（P2）与其他驱动接入成本
- [ ] **代理核心文件可追踪性**：`orm/proxy_value.go` 为框架核心（代理注入机制）却在 .gitignore 中声明（见 P3-4），确认无需隐藏后删除该条目，保证代码评审可追踪

---

## 验收命令速查

```bash
go build ./...                                            # 编译
go test -count=1 ./types/... ./utils/... ./log/...        # 无 DB 依赖测试
go test -count=1 ./orm/...                                # orm 测试（P1-1 修复后应全绿）
bash coverage.sh                                          # 覆盖率报告
git status                                                # 应无 P3 相关噪音
```
