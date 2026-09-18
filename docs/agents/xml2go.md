# xml2go — XML Mapper 逆向代码生成（AI agent 说明文档）

> 命令 `cmd/xml2go`，引擎 `types.GenerateXml2GoFiles`（`types/xml2go_gen.go`）。
> 从既有 MyBatis / MyBatis-Plus XML Mapper 生成**可直接编译运行**的 Go 代码：
> `<output>/models/`（模型 struct + db tag 元数据 + TableName）与
> `<output>/mapper/`（orm.BaseMapper 代理 struct + init 注册 + 单例 getter）。

## 1. 定位与选型

| 工具 | 输入 | 输出 | 适用场景 |
|------|------|------|----------|
| `cmd/generator` | XML Mapper | Go 代码（旧版，签名粗糙） | 已被 xml2go 取代，不再推荐 |
| `cmd/schema2code` | 数据库表结构 | Go 模型 + Mapper XML（`-mp` 附带 BaseMapper CRUD XML） | 先有库表、后写代码（正向工程） |
| `cmd/sqlc` | XML Mapper（静态 select） | 类型安全 Querier 函数（免反射，走 `QueryToContext`） | 纯静态查询、追求零反射开销 |
| **`cmd/xml2go`** | **XML Mapper（全量语句）** | **模型 + Mapper 代理 struct（反射代理，与手写 Mapper 同语义）** | **存量 MyBatis 工程迁移：动态 SQL、MP 内置 CRUD、resultMap 全覆盖** |

选型判断：语句含 `<if>`/`<foreach>` 等动态标签或依赖 MP 内置 CRUD → xml2go；
纯静态 select 且不需要 Mapper 形态 → sqlc。二者产物可共存于同一工程。

## 2. 如何运行：两种入口

```bash
go build -o D:\Develop\bin\xml2go.exe cmd/xml2go/main.go
```

### 场景一：已有 Java MyBatis / MyBatis-Plus 的 XML Mapper 文件（`-m`）

已知 Mapper XML 所在目录时，直接指向该目录生成 Go 模型与 Mapper：

```bash
# 典型 Java 工程布局：把 -m 指到 src/main/resources/mapper
D:\Develop\bin\xml2go.exe -m <工程>\src\main\resources\mapper -d gen -p github.com/xxx/app

# 本仓库 RuoYi 样本（仓库根目录执行）
D:\Develop\bin\xml2go.exe -m samples -d gen -p github.com/xxx/app -v
```

目录会递归收集全部 `.xml`；产物与手写 Mapper 语义一致，接 orm 初始化即可用（见 §3）。

### 场景二：只有 MyBatis / MyBatis-Plus 配置文件（`-c`）

不知道 XML 在哪、手头只有 Java 侧配置文件时，用 `-c` 直接给配置文件——工具**先从配置解析
Mapper XML 位置（mapper-locations / `<mappers>` / mapperLocations），再按场景一同一路径生成**：

```bash
# Spring Boot application.yml / application.properties
D:\Develop\bin\xml2go.exe -c <工程>\src\main\resources\application.yml -d gen -p github.com/xxx/app

# 经典 MyBatis mybatis-config.xml
D:\Develop\bin\xml2go.exe -c <工程>\src\main\resources\mybatis\mybatis-config.xml -d gen -p github.com/xxx/app

# Spring / MyBatis-Plus Spring XML（SqlSessionFactoryBean 配置）
D:\Develop\bin\xml2go.exe -c <工程>\src\main\resources\spring-datasource.xml -d gen -p github.com/xxx/app
```

支持的配置形态与解析规则（`types/mybatis_config.go`，`types.LoadMappersFromMyBatisConfig`）：

| 配置类型 | 识别的键 / 节点 |
|------|------|
| `.properties` | `mybatis.mapper-locations`、`mybatis-plus.mapper-locations`（兼容驼峰 `mapperLocations`）；`mybatis[-plus].config-location` 链式进入 XML 配置 |
| `.yml` / `.yaml` | `mybatis` / `mybatis-plus` 节点下的 `mapper-locations`（标量 / 行内列表 / 块列表）与 `config-location` |
| `.xml`（mybatis-config） | `<configuration><mappers><mapper resource="..."/>`（classpath 相对）与 `<mapper url="file:///..."/>`（file URL） |
| `.xml`（Spring/MP Spring） | `<property name="mapperLocations" value="..."/>` 及嵌套 `<list><value>` / `<array><value>`；`configLocation` 链式 |

位置解析约定：

- `classpath*:` / `classpath:` 前缀去除后，相对路径以**配置文件所在目录为 classpath 根**；
- 模式支持 `*` / `?` / `**`（`**` 跨目录层级，如 `classpath*:mapper/**/*.xml`）；逗号 / 分号多值；
- 单条模式在配置目录未命中时，依次回退其**父 / 祖父目录**（兼容「配置在 `mybatis/` 子目录、XML 在资源根」布局）；
- 仅 `config-location` 而无 `mapper-locations` 时沿链解析（最深 3 层），链内 resource 仍按最外层配置的 classpath 根解析；
- 展开结果只保留 `.xml`，去重排序；解析 0 个 XML 时报错退出（`-c` 优先级高于 `-m`）。

程序化调用（生成器 / agent 内嵌场景）：

```go
mps, info, err := types.LoadMappersFromMyBatisConfig("src/main/resources/application.yml")
// info.Locations = 解析出的位置模式；info.XmlFiles = 展开后的 XML 清单
result, err := mps.GenerateXml2GoFiles(&types.Xml2GoOptions{OutputDir: "gen", ModulePrefix: "github.com/xxx/app"})
```

### 参数表

| 参数 | 默认 | 说明 |
|------|------|------|
| `-c` | 空 | mybatis/mybatis-plus 配置文件（.properties/.yml/.yaml/.xml），先从配置定位 XML 再生成；设置时 `-m` 被忽略 |
| `-m` | `resources/mapper` | XML Mapper 目录（场景一） |
| `-d` | `gen` | 输出根目录（内含 `models/`、`mapper/` 两个子目录） |
| `-p` | 空 | **输出目录自身**的模块导入路径前缀；生成代码内 import 为 `<prefix>/models`、`<prefix>/mapper`（若输出目录不在 Go module 内则留空，退化为裸导入 `models`） |
| `-skip-mp` | false | 跳过 MyBatis-Plus 内置 CRUD 字段（insert/deleteById/…/insertBatch） |
| `-v` | false | 打印跳过语句及原因 |

输出逐 Mapper 一行统计 + 总计；samples 全量（`-m samples` 或 `-c` 配置文件指到 samples 均可）：20 个模型文件、22 个 mappers、331 个函数、0 跳过。

## 3. 产物结构

```
<output>/
├── models/                 # package models
│   ├── SysUser.go          # struct SysUser + db tag + json tag + TableName() + init() RegisterModel
│   └── ...
└── mapper/                 # package mapper
    ├── SysUserMapper.go    # struct SysUserMapper{ orm.BaseMapper; 函数字段... }
    │                       # + init()（RegisterModel + RegisterMapper）+ sync.Once 单例 GetSysUserMapper()
    └── ...
```

生成代码的使用方式与手写 Mapper 完全一致：

```go
import (
    "github.com/bnulwh/mybatis-go/orm"
    mapper "github.com/xxx/app/gen/mapper"
    models "github.com/xxx/app/gen/models"
)

orm.InitializeFromSettings("application.properties") // 或任一初始化入口
user, err := mapper.GetSysUserMapper().SelectUserByUserName("admin") // []models.SysUser, error
```

注意：`GetXxxMapper()` 首次调用会执行 SQL 绑定，必须在 orm 初始化完成之后调用；
import 本身只做模型/Mapper 注册，不触发绑定。

## 4. 生成规则（签名推导）

与 orm 注册期校验（`ParamType.checkSql` / `ReturnType.checkSql` / M-04）逐条对齐，保证产物可编译且注册不报错：

| XML 语句特征 | 生成的函数字段签名 |
|------|------|
| `parameterType` 为业务模型（有 resultMap 支撑） | `model *models.X`（指针：ID 回填/自动填充要求可寻址） |
| `parameterType="map"` | `param map[string]interface{}` |
| `<foreach>` 且 parameterType 为模型/list | `items []interface{}`；MP insertBatch → `models []*models.X` |
| 无 `parameterType`（AutoDerive），无 foreach | 按占位符逐个生成具名 `interface{}` 参数（与运行期按位绑定 1.2 对齐） |
| 无 `parameterType` + foreach | `param map[string]interface{}`（map 键名解析 foreach + 槽位） |
| `parameterType` 为基础类型 + foreach | `ids []T` |
| `parameterType` 为基础类型（无 foreach） | 具名标量参数 `T` |
| selectPage | 入参 `page *orm.PageParam`，返回 `*orm.Page, error` |
| resultMap 返回 | `[]models.X, error`（JDK 类型 resultMap 语句跳过） |
| resultType=map / 未注册 pojo | `[]map[string]interface{}, error` |
| resultType=业务模型且已生成 | `[]models.X, error`（M-04 运行期模型转换路径） |
| resultType 标量（long/int/string…） | `[]<Go标量>, error` |
| DML（insert/update/delete） | `int64, error` |

模型字段：
- `db` tag：resultMap id/result → `db:"<列名>,pk"`；逻辑删除列（deleted/del_flag）→ `,logic`；
  `<version>` 列 → `,version`；association/collection → `db:"-"`；其余 → `db:"<列名>"`。
- 表名：`TableName()` 返回 `camelToSnake(短名去 Model 后缀)`，与 orm 默认推导一致；
  **模型自身含 `TableName` 字段时跳过方法生成**（避免与 GenTable 等元数据表冲突，orm 默认推导结果相同，无语义损失）。
- 跨 Mapper 同名模型去重（首个 resultMap 生效，告警日志）；JDK 类型（String/Long…）不生成模型。
- Mapper 短名冲突、函数字段名冲突（大小写不敏感）→ 跳过并记原因。

## 5. 验证步骤（agent 改动 xml2go 后必做）

1. `go build ./...` + `go vet ./...`
2. `go test ./types/... -run "Xml2Go" -v`（4 个用例：Basic / SkipMP / SkipUnknownModel / Samples 真实回归）
3. `go test ./types/... -run "MyBatisConfig|ExpandMapperLocations" -v`（8 个用例：properties / yaml 块列表+`**` / mybatis-config.xml resource+file url / Spring XML list / config-location 链 / 错误分支 / 父目录回退）
4. 全量回归：各包测试二进制编译到 `D:\Develop\bin\<pkg>.test.exe` 后运行（经 `cmd /c` 执行，`-test.run` 用 `=` 形式；types 用例 workdir 必须是 `types/`，orm 用例 workdir 必须是 `orm/`）
5. 端到端类型检查：把生成产物放进临时模块（go.mod + `replace github.com/bnulwh/mybatis-go => <repo>`），`go build ./...` 须 exit=0；
   冒烟：`import _ ".../mapper"` + `orm.GetModelInfo(new(models.SysUser))` 验证注册链路不 panic
6. `-c` 入口冒烟：构造 `src/main/resources/application.yml`（`mybatis-plus.mapper-locations: classpath*:mapper/**/*.xml`）+ samples XML，运行 `xml2go.exe -c ... -d gen -p example.com/x`，产物应与 `-m samples` 一致（22 mappers / 331 functions）

## 6. 已知边界（与运行时一致，非缺陷）

- **跨 Mapper 嵌套 resultMap 引用**（如 `resultMap="SysRoleMapper.RoleResult"`）静态不可解析，字段退化为 `interface{}`；运行期仍按全局 resultMap 正常转换。
- **`${}` 注入占位符**计入参数槽，生成具名 `interface{}` 参数；注意 SQL 注入风险由使用方承担。
- **引用未生成模型的语句被跳过**（如 `parameterType` 指向无 resultMap 的 pojo）——保证产物恒可编译；`-v` 查看原因，需补 resultMap 后重跑。
- `-p` 必须是**输出目录的导入路径**（不是 models 子目录的路径）；产物目录需在 Go module 内才能编译。
