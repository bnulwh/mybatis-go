# TDSQL Demo（腾讯云）

mybatis-go 连接 TDSQL（腾讯云）的使用示例。MySQL 兼容，默认端口 3306。

## 配置

本目录的 [`application-tdsql.properties`](./application-tdsql.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

## 驱动

无需引入——框架自动以 `tdsql` 名称注册 `go-sql-driver/mysql` 驱动。

## 运行

在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

```bash
go run ./cmd/tdsqldemo
```
