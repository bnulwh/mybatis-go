# TaurusDB Demo（华为云）

mybatis-go 连接 TaurusDB（华为云）的使用示例。MySQL 100% 兼容，默认端口 3306。

## 配置

本目录的 [`application-taurusdb.properties`](./application-taurusdb.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

## 驱动

无需引入——框架自动以 `taurusdb` 名称注册 `go-sql-driver/mysql` 驱动。

## 运行

在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

```bash
go run ./cmd/taurusdbdemo
```
