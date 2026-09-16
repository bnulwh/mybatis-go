# IBM DB2 Demo

mybatis-go 连接 IBM DB2 的使用示例。DB2 方言族，默认端口 50000。

## 配置

本目录的 [`application-db2.properties`](./application-db2.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

## 驱动

需自行引入 DB2 驱动，并**安装 DB2 ODBC/CLI 客户端库**：

```go
import _ "github.com/ibmdb/go_ibm_db"
```

## 方言说明

- 占位符自动 `?`→`:1`/`:2`…
- `NeedsReturning` 返回 false（IDENTITY 列 + `IDENTITY_VAL_LOCAL()`）
- 分页使用 `OFFSET N ROWS FETCH NEXT M ROWS ONLY`（需 DB2 10.1+）
- `EffectiveSchema` 返回 `UPPER(Username)`
- DSN 格式 `HOSTNAME=host;PORT=port;DATABASE=dbname;UID=username;PWD=password`

## 运行

在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

```bash
go run ./cmd/db2demo
```
