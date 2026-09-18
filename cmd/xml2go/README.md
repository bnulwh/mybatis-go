# xml2go

从既有 MyBatis / MyBatis-Plus XML Mapper 生成可直接编译运行的 Go 代码：
`<output>/models/`（模型 struct + db tag 元数据 + TableName）与
`<output>/mapper/`（orm.BaseMapper 代理 struct + init 注册 + 单例 getter）。

与 `cmd/sqlc` 的分工：sqlc 面向纯静态 select 生成类型安全 Querier 函数；
xml2go 面向全量语句（含动态 SQL / MP 内置 CRUD）生成反射代理 Mapper，二者可共存。

完整生成规则、签名推导表、已知边界与验证步骤见 **docs/agents/xml2go.md**。

## 用法

### 已知 XML Mapper 目录（`-m`）

```bash
go build -o xml2go main.go
./xml2go -m <工程>/src/main/resources/mapper -d gen -p github.com/xxx/app
```

### 只有 MyBatis / MyBatis-Plus 配置文件（`-c`）

先从配置解析 Mapper XML 位置（mapper-locations / `<mappers>` / mapperLocations），再生成：

```bash
./xml2go -c <工程>/src/main/resources/application.yml -d gen -p github.com/xxx/app   # .yml/.yaml/.properties
./xml2go -c <工程>/src/main/resources/mybatis/mybatis-config.xml -d gen -p github.com/xxx/app
./xml2go -c <工程>/src/main/resources/spring-datasource.xml -d gen -p github.com/xxx/app  # Spring XML
```

支持 `mybatis[-plus].mapper-locations`（classpath*:/classpath: 前缀、`*`/`**` 通配、逗号多值、
config-location 链式解析）；相对路径以配置文件所在目录为 classpath 根，未命中回退父/祖父目录。

| 参数 | 默认 | 说明 |
|------|------|------|
| `-c` | 空 | mybatis/mybatis-plus 配置文件（.properties/.yml/.yaml/.xml），设置时 `-m` 被忽略 |
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
