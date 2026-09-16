# MySQL Demo

mybatis-go 连接 MySQL 的使用示例（默认端口 3306）。

## 配置

本目录的 [`application-mysql.properties`](./application-mysql.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

## 驱动

示例已引入 `github.com/go-sql-driver/mysql`，无需额外操作。

## 运行

在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

```bash
go run ./cmd/mysqldemo
```

示例先执行 `SelectAll` 打印全部行，再并发 10 个 goroutine 各执行 1000 次 `SelectByPrimaryKey`，演示连接池与 Mapper 代理的并发能力。
