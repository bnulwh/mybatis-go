# KingbaseES Demo（人大金仓）

mybatis-go 连接 KingbaseES（人大金仓）的使用示例。PostgreSQL 兼容，默认端口 54321。

## 配置

本目录的 [`application-kingbase.properties`](./application-kingbase.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

如需指定 schema，可取消注释 `spring.datasource.schema`，或通过 URL 参数 `?currentSchema=myschema` / `?search_path=myschema` 指定。

## 驱动

无需引入——框架自动以 `kingbase` 名称注册 `lib/pq` 驱动（兼容 kingbase5~8 各版本 URL 前缀）。

## 运行

在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

```bash
go run ./cmd/kingbasedemo
```
