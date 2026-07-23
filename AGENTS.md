# mybatis-go

Go 语言实现的 MyBatis 风格 ORM 框架。通过 XML Mapper 文件定义 SQL，利用反射为 struct 的函数字段注入代理实现，支持 PostgreSQL 和 MySQL。

## 项目

- **语言/版本**: Go 1.14
- **模块**: `github.com/bnulwh/mybatis-go`
- **核心依赖**: `github.com/beevik/etree` (XML 解析), `github.com/go-sql-driver/mysql`, `github.com/lib/pq`, `github.com/bnulwh/logrus`
- **入口点**:
  - `cmd/generator/main.go` — 从 XML Mapper 文件生成 Go 模型/Mapper 代码
  - `cmd/schema2code/main.go` — 从数据库表结构生成 Go 模型/Mapper 代码
  - `cmd/postgresdemo/main.go` — PostgreSQL 使用示例
  - `cmd/mysqldemo/main.go` — MySQL 使用示例
  - `cmd/demo/main.go` — 通用使用示例

## 命令

| 命令 | 说明 |
|------|------|
| `go build ./...` | 编译所有包 |
| `go test ./types/... ./utils/...` | 运行 types + utils 测试（不依赖数据库） |
| `go test ./orm/... -v` | 运行 orm 测试（`Test_newInstance` 为预先存在的问题） |
| `go test -v -count=1 ./... -coverprofile=cover.out` | 全量测试 + 覆盖率 |
| `bash coverage.sh` | 生成覆盖率 HTML 报告 |
| `go build -o generator cmd/generator/main.go` | 编译 generator 工具 |
| `go build -o schema2code cmd/schema2code/main.go` | 编译 schema2code 工具 |

## 架构

### 核心模块

- **`orm/`** — 主框架包。包含初始化、Mapper 代理、SQL 执行、结果转换、数据库连接池管理、Dialector（MySQL/PostgreSQL）。
  - `orm_init.go` — `Initialize()` / `InitializeFromSettings()` 入口，加载配置、解析 XML、连接数据库
  - `base_mapper.go` — `BaseMapper` 是所有 Mapper 的基类，内含 `executeMethod()` 调度 SQL 执行
  - `proxy_value.go` — 核心代理机制：通过反射为 Mapper struct 的函数字段注入代理函数
  - `proxy_arg.go` — `ProxyArg` 封装函数调用参数（tag args 映射 + 位置参数）
  - `return_type.go` / `return_value.go` — 返回值类型推断与结果转换
  - `sql_execute.go` — 实际 `database/sql` 的 Exec/Query 调用
  - `prepared_stmt.go` — 预编译语句缓存（`PreparedStmtDB`）
  - `database_config.go` — `Config` / `MyBatisSetting` 配置结构体与解析
  - `database_connection.go` — `DB` 结构体与 `Open()` 函数
  - `mysql_dialector.go` / `postgres_dialector.go` — 数据库方言实现
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
- **配置**: 支持 Spring Boot 风格 `.properties` 文件（`spring.datasource.*`）和 `mybatis.mapper-locations`。
## 备注

- `orm/common_test.go` 中 `Test_newInstance` 在 `time.Time` 类型上失败，为预先存在的测试问题。
