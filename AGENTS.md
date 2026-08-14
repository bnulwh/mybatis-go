# mybatis-go

Go 语言实现的 MyBatis 风格 ORM 框架。通过 XML Mapper 文件定义 SQL，利用反射为 struct 的函数字段注入代理实现，支持 PostgreSQL、MySQL、SQLite 和人大金仓 KingbaseES。

## 项目

- **语言/版本**: Go 1.21（go.mod；旧版声明为 1.14）
- **模块**: `github.com/bnulwh/mybatis-go`
- **核心依赖**: `github.com/beevik/etree` (XML 解析), `github.com/go-sql-driver/mysql`, `github.com/lib/pq` (PostgreSQL/KingbaseES), `modernc.org/sqlite` (纯 Go SQLite), `github.com/bnulwh/logrus`
- **入口点**:
  - `cmd/generator/main.go` — 从 XML Mapper 文件生成 Go 模型/Mapper 代码
  - `cmd/schema2code/main.go` — 从数据库表结构生成 Go 模型/Mapper 代码
  - `cmd/postgresdemo/main.go` — PostgreSQL 使用示例
  - `cmd/mysqldemo/main.go` — MySQL 使用示例
  - `cmd/kingbasedemo/main.go` — 人大金仓 KingbaseES 使用示例
  - `cmd/sqlitedemo/main.go` — SQLite 使用示例（纯 Go 驱动 modernc.org/sqlite，无需 CGO）
  - `cmd/demo/main.go` — 通用使用示例

## 命令

| 命令 | 说明 |
|------|------|
| `go build ./...` | 编译所有包 |
| `go test ./types/... ./utils/...` | 运行 types + utils 测试（不依赖数据库） |
| `go test ./orm/... -v` | 运行 orm 测试（含 SQLite 端到端测试 `Test_Sqlite*`，无需外部数据库） |
| `go test -v -count=1 ./... -coverprofile=cover.out` | 全量测试 + 覆盖率 |
| `bash coverage.sh` | 生成覆盖率 HTML 报告 |
| `bash scripts/auto-commit.sh [秒]` | 自动提交监视：检测到修改后自动 commit + push（见「自动提交」） |
| `go build -o generator cmd/generator/main.go` | 编译 generator 工具 |
| `go build -o schema2code cmd/schema2code/main.go` | 编译 schema2code 工具 |
| `go run ./cmd/sqlitedemo` | 运行 SQLite 示例（自动建表 + Mapper 全流程，生成 test.db） |

## 架构

### 核心模块

- **`orm/`** — 主框架包。包含初始化、Mapper 代理、SQL 执行、结果转换、数据库连接池管理、Dialector（MySQL/PostgreSQL/SQLite/KingbaseES）。
  - `orm_init.go` — `Initialize()` / `InitializeFromSettings()` 入口，加载配置、解析 XML、连接数据库
  - `base_mapper.go` — `BaseMapper` 是所有 Mapper 的基类，内含 `executeMethod()` 调度 SQL 执行
  - `proxy_value.go` — 核心代理机制：通过反射为 Mapper struct 的函数字段注入代理函数
  - `proxy_arg.go` — `ProxyArg` 封装函数调用参数（tag args 映射 + 位置参数）
  - `return_type.go` / `return_value.go` — 返回值类型推断与结果转换
  - `sql_execute.go` — 实际 `database/sql` 的 Exec/Query 调用
  - `transaction.go` — 事务支持（`Begin`/`BeginTx`/`Commit`/`Rollback`，开启后 Mapper 与 orm.Execute/Query 自动在事务内执行）
  - `prepared_stmt.go` — 预编译语句缓存（`PreparedStmtDB`）
  - `database_config.go` — `Config` / `MyBatisSetting` 配置结构体与解析
  - `database_connection.go` — `DB` 结构体与 `Open()` 函数
  - `mysql_dialector.go` / `postgres_dialector.go` / `sqlite_dialector.go` / `kingbase_dialector.go` — 数据库方言实现（KingbaseES 与 PostgreSQL 同线协议，复用 `lib/pq` 驱动以 `kingbase` 名称注册）
  - `statement.go` — 执行统计（Query/Execute 次数、时长、错误数）
  - `common.go` — 大量工具函数（类型转换、SQL Null 处理、结构体扫描）
  - `table_structure.go` / `database_structure.go` — 从 information_schema 获取表结构
  - `orm_cache.go` / `mapper_cache.go` / `model_cache.go` — 缓存层

- **`types/`** — 数据类型与 XML 解析引擎。
  - `sql_mapper.go` / `sql_mappers.go` — Mapper 定义（`SqlMapper` / `SqlMappers`），`GenerateFiles()` 生成代码
  - `sql_function.go` — `SqlFunction` 表示一个 SQL 操作（id/type/param/result/items）
  - `sql_fragment.go` / `sql_fragments.go` — SQL 片断解析（`#{}` `${}` `<if>` `<where>` 等标签处理）
  - `sql_param.go` — 参数类型解析（`SqlParam` / `SqlParamInput`）
  - `sql_result.go` — 结果映射解析（`SqlResult` / `ResultMapping`）
  - `xml_parse.go` — XML 文件解析入口
  - `common.go` — `GetShortName()` / `ToJson()` / `UpperFirst()` 等通用工具
  - `sql_element.go` / `sql_include.go` / `sql_renderer.go` — SQL 元素与渲染
  - `result_map.go` / `result_item.go` — 结果映射
  - `stack.go` — 栈数据结构

- **`utils/`** — 工具包。
  - `change_type.go` — 类型转换（string ↔ int/float/bool 等，带 SQL Null 支持）
  - `env_utils.go` — 环境变量读取
  - `file_utils.go` — 文件操作
  - `list_utils.go` — 切片工具

- **`log/`** — 日志接口。定义 `Logger` 接口（Debugf/Printf/Infof/Warnf/Errorf），默认 ConsoleLogger 实现，可通过 `SetLogger()` 替换。

- **`mapper/`** — 生成的 Mapper 示例文件（`UserInfoModelMapper.go` 等）。

### 核心流程

1. `orm.Initialize("config.properties")` — 加载配置 → 解析 XML Mapper → 连接数据库
2. `orm.RegisterModel(new(Model))` — 注册模型，缓存字段信息
3. `orm.RegisterMapper(new(MapperStruct))` — 注册 Mapper，为函数字段注入代理
4. `orm.NewMapper("Name").(MapperType)` — 创建 Mapper 实例
5. 调用 `Mapper.SelectAll()` 等 → 代理函数执行 → SQL 生成 → DB 查询 → 结果自动映射

## 约定

- **包组织**: 垂直按职责分包（orm/types/utils/log），orm 内按功能拆文件。
- **错误处理**: 使用 `t.Error()` 而非 `t.Fatal()` 在测试中标记失败（允许后续断言继续执行）。业务代码返回 `(value, error)` 双返回值。
- **命名**: 
  - 导出类型/函数使用 PascalCase。
  - 测试函数命名 `Test_函数名` 或 `Test函数名`。
  - XML Mapper 的文件名与 `namespace` 对应，放在 `resources/mapper` 目录。
- **日志**: 通过 `log` 包调用所有日志（`log.Debugf`/`Infof`/`Warnf`/`Errorf``），可替换实现。
- **配置**: 支持 Spring Boot 风格 `.properties` 文件（`spring.datasource.*`）和 `mybatis.mapper-locations`。KingbaseES 使用 `jdbc:kingbase8://host:port/dbname` 或 `jdbc:kingbase://host:port/dbname` URL，类型填 `kingbase`。
## 自动提交（项目配置）

项目已启用「修改后自动提交仓库」机制，配置全部存于仓库内（可版本化）：

- **hooks 目录**：`.githooks/`，通过 `git config core.hooksPath .githooks` 指向（本机已配置）
  - `.githooks/post-commit` — 每次 commit 成功后自动 `git push` 到远程（失败静默降级，不影响本地提交）
- **监视脚本**：`scripts/auto-commit.sh` — 检测工作区变更后自动 `git add -A` + commit（提交信息按变更文件名自动生成）
  - `bash scripts/auto-commit.sh` — 后台监视，默认每 60s 检查一次（Ctrl+C 停止）
  - `bash scripts/auto-commit.sh 10` — 每 10s 检查一次
  - `bash scripts/auto-commit.sh 0` — 单次检查后立即退出
  - commit 后自动触发 post-commit hook 完成 push
- 新克隆仓库需执行一次启用 hooks：`git config core.hooksPath .githooks`

## 备注

- `orm/common_test.go` 中 `Test_newInstance` 已修复（`newInstance` 对 `time.Time`/`sql.NullTime` 返回 `*sql.NullTime`，`convertTimeToTime` 统一处理三种时间指针类型）。
