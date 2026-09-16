# MS SQL Server Demo

mybatis-go 连接 MS SQL Server 的使用示例。MSSQL 方言族，默认端口 1433。

## 配置

本目录的 [`application-mssql.properties`](./application-mssql.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

## 驱动

无需引入——框架 blank import `github.com/microsoft/go-mssqldb`，自动以 `sqlserver` 名称注册。

## 方言说明

- 占位符自动 `?`→`@p1`/`@p2`…
- `NeedsReturning` 返回 false（自增键通过 `SCOPE_IDENTITY()` + `LastInsertId()` 获取）
- 分页使用 `OFFSET N ROWS FETCH NEXT M ROWS ONLY`，要求 SQL 含 `ORDER BY`
- 默认 Schema 为 `dbo`

## 运行

在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

```bash
go run ./cmd/mssqldemo
```
