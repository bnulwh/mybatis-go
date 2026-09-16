# HighGo DB Demo（瀚高）

mybatis-go 连接 HighGo DB（瀚高）的使用示例。PostgreSQL 兼容，默认端口 5432。

## 配置

本目录的 [`application-highgo.properties`](./application-highgo.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

## 驱动

无需引入——框架自动以 `highgo` 名称注册 `lib/pq` 驱动。

## 运行

在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

```bash
go run ./cmd/highgodemo
```
