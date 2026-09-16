# GBase 8s Demo（南大通用）

mybatis-go 连接 GBase 8s（南大通用）的使用示例。Informix 兼容族，默认端口 9088。

## 配置

本目录的 [`application-gbase8s.properties`](./application-gbase8s.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

## 驱动

需自行引入 Informix 驱动：

```go
import _ "github.com/alexgarrec/go-informix"
```

## 方言说明

- 占位符保持 `?`
- `NeedsReturning` 返回 false
- `EffectiveSchema` 返回 `UPPER(Username)`
- DSN 格式 `user:password@host:port/dbname`

## 运行

在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

```bash
go run ./cmd/gbase8sdemo
```
