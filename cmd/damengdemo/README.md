# 达梦 DM8 Demo

mybatis-go 连接达梦 DM8 的使用示例。Oracle 方言族，默认端口 5236。

## 配置

本目录的 [`application-dameng.properties`](./application-dameng.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

## 驱动

需自行引入达梦驱动：

```go
import _ "dmdb.com/dm"
```

## 方言说明

- 占位符自动 `?`→`:1`/`:2`…
- `NeedsReturning` 返回 false

## 运行

在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

```bash
go run ./cmd/damengdemo
```
