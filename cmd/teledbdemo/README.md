# TeleDB Demo（天翼云）

mybatis-go 连接 TeleDB（天翼云）的使用示例。PostgreSQL 协议兼容；端口为示例值（官方未公布默认端口），请按实际环境修改。

## 配置

本目录的 [`application-teledb.properties`](./application-teledb.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

## 驱动

无需引入——框架自动以 `teledb` 名称注册 `lib/pq` 驱动。

## 运行

在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

```bash
go run ./cmd/teledbdemo
```
