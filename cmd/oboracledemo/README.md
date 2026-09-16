# OceanBase Oracle 模式 Demo（蚂蚁集团）

mybatis-go 连接 OceanBase Oracle 模式的使用示例。Oracle 方言族，默认端口 2881。

## 配置

本目录的 [`application-oboracle.properties`](./application-oboracle.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

## 驱动

需自行引入 OceanBase Oracle 模式驱动：

```go
import _ "github.com/oceanbase/oceanbase-driver-go"
```

## 方言说明

- 占位符自动 `?`→`:1`/`:2`…
- `NeedsReturning` 返回 false（Oracle 风格使用自增序列）

## 运行

在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

```bash
go run ./cmd/oboracledemo
```
