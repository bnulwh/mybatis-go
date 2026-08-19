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
- KingbaseES 使用 `jdbc:kingbase8://host:port/dbname` 或 `jdbc:kingbase://host:port/dbname` URL，类型填 `kingbase`。
