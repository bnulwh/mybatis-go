# 架构（mybatis-go）

> 需要修改框架代码、排查调用链或理解模块职责时阅读本文件。

## 核心模块

- **`orm/`** — 主框架包。包含初始化、Mapper 代理、SQL 执行、结果转换、数据库连接池管理、Dialector（MySQL/PostgreSQL/SQLite/KingbaseES）。
  - `orm_init.go` — `Initialize()` / `InitializeFromSettings()` 入口，加载配置、解析 XML、连接数据库
  - `base_mapper.go` — `BaseMapper` 是所有 Mapper 的基类，内含 `executeMethod()` 调度 SQL 执行
  - `proxy_value.go` — 核心代理机制：通过反射为 Mapper struct 的函数字段注入代理函数
  - `proxy_arg.go` — `ProxyArg` 封装函数调用参数（tag args 映射 + 位置参数）
  - `return_type.go` / `return_value.go` — 返回值类型推断与结果转换
  - `sql_execute.go` — 实际 `database/sql` 的 Exec/Query 调用
  - `transaction.go` — 事务支持（`Begin`/`BeginTx`/`Commit`/`Rollback`，开启后 Mapper 与 orm.Execute/Query 自动在事务内执行）
  - `multi_datasource.go` — 多数据源注册表（`InitializeDataSources`/`UseDataSource`/`AddDataSource`/`ReConnectDataSource`，默认源键无前缀、附加源键为 `spring.datasource.<name>.*` 并在 `mybatis.datasources` 列出）
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
  - `sql_where.go` — `<where>` 标签支持（`whereSqlFragment`/`sqlWhere`，子片段全空时输出空、否则输出 `where` 并剥离首个条件前导 `AND/OR`）
  - `sql_param.go` — 参数类型解析（`SqlParam` / `SqlParamInput`）
  - `sql_result.go` — 结果映射解析（`SqlResult` / `ResultMapping`）
  - `xml_parse.go` — XML 文件解析入口
  - `common.go` — `GetShortName()` / `ToJson()` / `UpperFirst()` 等通用工具
  - `sql_element.go` / `sql_include.go` / `sql_renderer.go` — SQL 元素与渲染（`SqlElement.Fragments` 支持 `<include>` 内嵌套标签参数替换）
  - `result_map.go` / `result_item.go` — 结果映射
  - `stack.go` — 栈数据结构

- **`utils/`** — 工具包。
  - `change_type.go` — 类型转换（string ↔ int/float/bool 等，带 SQL Null 支持）
  - `env_utils.go` — 环境变量读取
  - `file_utils.go` — 文件操作
  - `list_utils.go` — 切片工具

- **`log/`** — 日志接口。定义 `Logger` 接口（Debugf/Printf/Infof/Warnf/Errorf），默认 ConsoleLogger 实现，可通过 `SetLogger()` 替换。

- **`mapper/`** — 生成的 Mapper 示例文件（`UserInfoModelMapper.go` 等）。

## 核心流程

1. `orm.Initialize("config.properties")` — 加载配置 → 解析 XML Mapper → 连接数据库
2. `orm.RegisterModel(new(Model))` — 注册模型，缓存字段信息
3. `orm.RegisterMapper(new(MapperStruct))` — 注册 Mapper，为函数字段注入代理
4. `orm.NewMapper("Name").(MapperType)` — 创建 Mapper 实例
5. 调用 `Mapper.SelectAll()` 等 → 代理函数执行 → SQL 生成 → DB 查询 → 结果自动映射

## 关键实现备注

- `orm/common_test.go` 中 `Test_newInstance` 已修复（`newInstance` 对 `time.Time`/`sql.NullTime` 返回 `*sql.NullTime`，`convertTimeToTime` 统一处理三种时间指针类型）。
