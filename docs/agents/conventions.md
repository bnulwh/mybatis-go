# 编码约定（mybatis-go）

> 编写/修改代码时必须遵守。AGENTS.md 只保留硬性摘要，完整约定在本文件。

## 包组织

垂直按职责分包（orm/types/utils/log），orm 内按功能拆文件。

## 错误处理

- 测试中标记失败使用 `t.Error()` 而非 `t.Fatal()`（允许后续断言继续执行）。
- 业务代码返回 `(value, error)` 双返回值。

## 命名

- 导出类型/函数使用 PascalCase。
- 测试函数命名 `Test_函数名` 或 `Test函数名`。
- XML Mapper 的文件名与 `namespace` 对应，放在 `resources/mapper` 目录。

## 日志

通过 `log` 包调用所有日志（`log.Debugf`/`Infof`/`Warnf`/`Errorf`），可替换实现。

## 配置

- 支持 Spring Boot 风格 `.properties` 文件（`spring.datasource.*`）和 `mybatis.mapper-locations`。
- 数据库模式（schema）：可选，PG/Kingbase 默认 `public`。配置来源按优先级：`spring.datasource.schema` 键 > JDBC URL query 参数（`currentSchema` / `search_path` / `schema`）；
  PG/Kingbase 连接时自动追加 `search_path`，表结构查询按配置 schema 过滤；MySQL 下 schema 即数据库名（显式配置时覆盖表结构查询库名，DSN 不受影响）；SQLite 忽略。
- KingbaseES 使用 `jdbc:kingbase8://host:port/dbname` 或 `jdbc:kingbase://host:port/dbname` URL，类型填 `kingbase`。
- 数据表名前缀：`mybatis.table-prefix`（如 `test_`），SQL 执行入口自动改写表名，XML Mapper 语句不变；
  系统表（information_schema / pg_% / sqlite_% / pragma_%）与已带前缀的表不参与改写；也可用 `orm.SetTablePrefix`。
