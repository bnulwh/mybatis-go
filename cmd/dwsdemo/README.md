# DWS Demo（华为云）

mybatis-go 连接 DWS（华为云）的使用示例。PostgreSQL libpq 协议，默认端口 8000。**须使用 MD5 认证账户**（lib/pq 不支持 DWS SHA256 认证）。

## 配置

本目录的 [`application-dws.properties`](./application-dws.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

## 驱动

无需引入——框架自动以 `dws` 名称注册 `lib/pq` 驱动。

## 运行

在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

```bash
go run ./cmd/dwsdemo
```
