# ClickHouse Demo

mybatis-go 连接 ClickHouse 的使用示例。ClickHouse 方言族，默认端口 9000（原生 TCP）/ 8123（HTTP）。

## 配置

本目录的 [`application-clickhouse.properties`](./application-clickhouse.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

## 驱动

无需引入——框架 blank import `github.com/ClickHouse/clickhouse-go/v2`，自动以 `clickhouse` 名称注册。

## 方言说明

- 占位符保持 `?`
- `NeedsReturning` 返回 false（ClickHouse 无 RETURNING 支持）
- 分页使用 `LIMIT n OFFSET m`
- ClickHouse 是列式 OLAP 引擎：**不支持事务**，UPDATE / DELETE 为异步 mutation

## 运行

在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

```bash
go run ./cmd/clickhousedemo
```
