# mybatis-go

Go 语言实现的 MyBatis 风格 ORM 框架。通过 XML Mapper 文件定义 SQL，利用反射为 struct 的函数字段注入代理实现，支持 PostgreSQL、MySQL、SQLite 和人大金仓 KingbaseES。

## 快速命令

| 命令 | 说明 |
|------|------|
| `go build ./...` | 编译所有包 |
| `go test ./types/... ./utils/...` | types + utils 测试（不依赖数据库） |
| `go test ./orm/... -v` | orm 测试（含 SQLite 端到端 `Test_Sqlite*`，无需外部数据库） |
| `go test -v -count=1 ./...` | 全量测试 |
| `go run ./cmd/sqlitedemo` | SQLite 示例（自动建表 + Mapper 全流程） |

完整命令表（工具编译、入口点、依赖）见 **docs/agents/commands.md**。

## 目录速览

- `orm/` — 主框架包（初始化、Mapper 代理、SQL 执行、事务、多数据源、方言）
- `types/` — XML 解析引擎（Mapper 定义、SQL 片段、结果映射、代码生成）
- `utils/` `log/` `mapper/` — 工具包 / 日志接口 / 生成的 Mapper 示例
- `cmd/` — 各数据库 demo 与 generator/schema2code 工具
- `samples/` — RuoYi Mapper 兼容性测试样本（KingbaseES 方言）

## 硬性约定（每次改动必须遵守）

- 测试失败用 `t.Error()` 而非 `t.Fatal()`（允许后续断言继续执行）。
- 业务代码返回 `(value, error)` 双返回值。
- 日志一律走 `log` 包（`log.Debugf`/`Infof`/`Warnf`/`Errorf`），可替换实现。
- XML Mapper 文件名与 `namespace` 对应，放 `resources/mapper`。
- 导出类型/函数 PascalCase；测试函数命名 `Test_函数名`。

完整约定见 **docs/agents/conventions.md**。

## 按需阅读的文档

| 场景 | 文档 |
|------|------|
| 修改框架代码 / 排查调用链 | docs/agents/architecture.md（核心模块 + 核心流程） |
| 编写代码时的完整规范 | docs/agents/conventions.md |
| 完整命令表 / 工具编译 / 入口点 | docs/agents/commands.md |
| samples 兼容性缺陷（S-01~S-11） | TODO.md「📁 samples 目录」段落（修复时务必同步更新） |
| 自动提交 / hooks 配置 | docs/agents/auto-commit.md |

## 备注

- `orm/common_test.go` 中 `Test_newInstance` 已修复（`newInstance` 对 `time.Time`/`sql.NullTime` 返回 `*sql.NullTime`，`convertTimeToTime` 统一处理三种时间指针类型）。
