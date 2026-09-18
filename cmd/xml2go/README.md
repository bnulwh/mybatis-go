# xml2go

从既有 MyBatis / MyBatis-Plus XML Mapper 生成可直接编译运行的 Go 代码：
`<output>/models/`（模型 struct + db tag 元数据 + TableName）与
`<output>/mapper/`（orm.BaseMapper 代理 struct + init 注册 + 单例 getter）。

与 `cmd/sqlc` 的分工：sqlc 面向纯静态 select 生成类型安全 Querier 函数；
xml2go 面向全量语句（含动态 SQL / MP 内置 CRUD）生成反射代理 Mapper，二者可共存。

完整生成规则、签名推导表、已知边界与验证步骤见 **docs/agents/xml2go.md**。

## 用法

```bash
go build -o xml2go main.go
./xml2go -m resources/mapper -d gen -p github.com/xxx/app
```

| 参数 | 默认 | 说明 |
|------|------|------|
| `-m` | `resources/mapper` | XML Mapper 目录 |
| `-d` | `gen` | 输出目录（内含 `models/`、`mapper/`） |
| `-p` | 空 | 输出目录自身的模块导入路径前缀（生成 import 为 `<prefix>/models` 等；留空退化为裸导入） |
| `-skip-mp` | false | 跳过 MyBatis-Plus 内置 CRUD 字段 |
| `-v` | false | 打印跳过语句及原因 |

## 使用生成代码

```go
orm.InitializeFromSettings("application.properties")
users, err := mapper.GetSysUserMapper().SelectUserByUserName("admin") // []models.SysUser, error
```

`GetXxxMapper()` 须在 orm 初始化后调用（首次调用绑定 SQL）；import 只做注册，不触发绑定。
