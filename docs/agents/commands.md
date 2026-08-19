# 命令速查（mybatis-go）

> 完整命令表。AGENTS.md 只保留最常用的 4 条。

## 构建与测试

| 命令 | 说明 |
|------|------|
| `go build ./...` | 编译所有包 |
| `go vet ./...` | 静态检查 |
| `go test ./types/... ./utils/...` | 运行 types + utils 测试（不依赖数据库） |
| `go test ./orm/... -v` | 运行 orm 测试（含 SQLite 端到端测试 `Test_Sqlite*`，无需外部数据库） |
| `go test -v -count=1 ./... -coverprofile=cover.out` | 全量测试 + 覆盖率 |
| `bash coverage.sh` | 生成覆盖率 HTML 报告 |
| `bash scripts/auto-commit.sh [秒]` | 自动提交监视（见 docs/agents/auto-commit.md） |

## 工具编译

| 命令 | 说明 |
|------|------|
| `go build -o generator cmd/generator/main.go` | 编译 generator 工具 |
| `go build -o schema2code cmd/schema2code/main.go` | 编译 schema2code 工具 |
| `go run ./cmd/sqlitedemo` | 运行 SQLite 示例（自动建表 + Mapper 全流程，生成 test.db） |

## 入口点

- `cmd/generator/main.go` — 从 XML Mapper 文件生成 Go 模型/Mapper 代码
- `cmd/schema2code/main.go` — 从数据库表结构生成 Go 模型/Mapper 代码（`-mp` 生成 MyBatis-Plus 内置 CRUD：BaseMapper 标准方法名 insert/deleteById/updateById/selectById/selectList/selectOne/selectPage/selectCount/selectBatchIds/deleteBatchIds）
- `cmd/postgresdemo/main.go` — PostgreSQL 使用示例
- `cmd/mysqldemo/main.go` — MySQL 使用示例
- `cmd/kingbasedemo/main.go` — 人大金仓 KingbaseES 使用示例
- `cmd/sqlitedemo/main.go` — SQLite 使用示例（纯 Go 驱动 modernc.org/sqlite，无需 CGO）
- `cmd/demo/main.go` — 通用使用示例

## 项目信息

- **语言/版本**: Go 1.21（go.mod；旧版声明为 1.14）
- **模块**: `github.com/bnulwh/mybatis-go`
- **核心依赖**: `github.com/beevik/etree` (XML 解析), `github.com/go-sql-driver/mysql`, `github.com/lib/pq` (PostgreSQL/KingbaseES), `modernc.org/sqlite` (纯 Go SQLite), `github.com/bnulwh/logrus`
