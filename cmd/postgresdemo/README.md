# PostgreSQL Demo

mybatis-go 连接 PostgreSQL 的使用示例（默认端口 5432）。

## 配置

本目录的 [`application-pg.properties`](./application-pg.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

如需指定 schema，可取消注释 `spring.datasource.schema`，或通过 URL 参数 `?currentSchema=myschema` / `?search_path=myschema` 指定。

## 驱动

示例已引入 `github.com/lib/pq`，无需额外操作。

## 运行

在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

```bash
go run ./cmd/postgresdemo
```
