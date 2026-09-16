# OceanBase MySQL 模式 Demo（蚂蚁集团）

mybatis-go 连接 OceanBase MySQL 模式的使用示例。MySQL 兼容，默认端口 2881。

## 配置

本目录的 [`application-oceanbase.properties`](./application-oceanbase.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

## 驱动

无需引入——框架自动以 `oceanbase` 名称注册 `go-sql-driver/mysql` 驱动。

## 运行

在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

```bash
go run ./cmd/oceanbasedemo
```
