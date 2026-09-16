# SQLite Demo

mybatis-go 连接 SQLite 的使用示例。**无需外部数据库**——文件型数据库，运行时自动建表并完成 Mapper 全流程演示。

## 配置

本目录的 [`application-sqlite.properties`](./application-sqlite.properties) 中 `jdbc:sqlite:test.db` 的 `test.db` 为数据库文件路径（相对运行目录），无需用户名 / 密码。

## 驱动

示例已引入 `modernc.org/sqlite`（纯 Go 实现，无 CGo），无需额外操作。

## 运行

在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

```bash
go run ./cmd/sqlitedemo
```

示例流程：自动建表 `user_info` → 清空数据 → `Insert` 一条记录 → `SelectAll` 打印全部行，运行后根目录生成 `test.db`。
