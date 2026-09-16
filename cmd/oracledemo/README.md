# Oracle Demo

mybatis-go 连接 Oracle 的使用示例。Oracle 方言族，默认端口 1521，要求 Oracle 12c+。

## 配置

本目录的 [`application-oracle.properties`](./application-oracle.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

## 驱动

无需引入——框架 blank import `github.com/sijms/go-ora/v2`，自动以 `oracle` 名称注册。

## 方言说明

- 占位符自动 `?`→`:1`/`:2`…
- `NeedsReturning` 返回 false（Oracle 使用自增序列）
- 分页使用 `OFFSET N ROWS FETCH NEXT M ROWS ONLY`（需 Oracle 12c+ 且 SQL 含 `ORDER BY`）
- `EffectiveSchema` 返回 `UPPER(Username)`
- DSN 为 EZConnect 格式 `user/password@host:port/dbname`，`dbname` 对应服务名；如需 SID 连接请通过 `spring.datasource.url` 自定义 DSN

## 运行

在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

```bash
go run ./cmd/oracledemo
```
