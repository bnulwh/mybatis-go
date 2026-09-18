# 代码生成

## 工具安装

四款工具均为独立 main 包，支持 `go install` 一键安装（二进制落入 `$GOBIN` / `$GOPATH/bin`，无需克隆仓库）；也可在仓库内 `go build -o <name> cmd/<name>/main.go` 或 `go run ./cmd/<name>`：

```bash
go install github.com/bnulwh/mybatis-go/cmd/xml2go@latest      # XML Mapper → Go 模型 + Mapper 代理
go install github.com/bnulwh/mybatis-go/cmd/schema2code@latest # 数据库表结构 → Go
go install github.com/bnulwh/mybatis-go/cmd/sqlc@latest        # 静态 select → 类型安全 Querier
go install github.com/bnulwh/mybatis-go/cmd/generator@latest   # 已弃用，由 xml2go 取代
```

> `go install <pkg>@latest` 走 module proxy 解析依赖并编译，安装后的二进制用法与下文各工具的 `./<name>` 形式一致。

## generator（已弃用）— 从 XML Mapper 生成 Go 代码

> **Deprecated**：generator 是 v0.1 早期产物，已被 **xml2go** 全面取代，仅为兼容保留；新项目请直接使用 [xml2go](#xml2go--从-xml-mapper-逆向生成-go-代码)。

```bash
go build -o generator cmd/generator/main.go
./generator -p mypackage -d temp -m resources/mapper
```

参数说明：
- `-p` 包名（默认 `temp`；实际未生效，产物固定 `package mapper` / `package models`）
- `-d` 输出目录（默认 `temp`）
- `-m` XML Mapper 文件目录（默认 `resources/mapper`）

### generator 与 xml2go 对比

| 能力 | generator（弃用） | xml2go |
|------|------------------|--------|
| 包名 / 导入 | 硬编码 `package mapper`、裸 `import "models"`（`-p` 被忽略），仅 GOPATH 布局可用 | `-p` 模块前缀生成 `<prefix>/models` 导入，标准 module 工程直接编译 |
| 模型元数据 | 仅 json tag；`deleted` / `delete_time` 列粗暴跳过 | `db:"column,pk/logic/version"` + json + `TableName()`；逻辑删除按 MP 约定（deleted / del_flag） |
| MyBatis-Plus 内置 CRUD | 无 | 自动识别 / 补生成，`-skip-mp` 可关 |
| 跨 Mapper 模型去重 / 冲突处理 | 无 | 首个 resultMap 生效，Mapper 短名 / 字段名冲突跳过并记录原因 |
| 产物可编译保证 | 无（引用未生成模型即编译失败） | 引用未生成模型的语句跳过，产物恒可编译；生成后 gofmt + `go/parser` 校验 |
| 配置文件入口 | 无 | `-c` 直接读 Java 侧 .properties / .yml / .yaml / .xml 自动定位 XML |
| 签名推导 | 旧版推导，与 orm 注册期校验未对齐 | 与 orm 注册期校验逐条对齐（指针参数 / 分页 / foreach / 批量切片） |
| 单例 getter | `Get<短名>()` | `GetXxxMapper()` |
| 跳过原因 / 统计输出 | 无 | 逐 Mapper 统计 + `-v` 打印跳过原因 |

## schema2code — 从数据库表结构生成代码

```bash
go install github.com/bnulwh/mybatis-go/cmd/schema2code@latest
schema2code -type mysql -host localhost -port 3306 -username root -password 123456 -db mydb -output temp
# MyBatis-Plus 内置 CRUD：加 -mp 生成 BaseMapper 标准方法名（insert/deleteById/updateById/selectById/selectList/selectOne/selectPage/selectCount/selectBatchIds/deleteBatchIds）
schema2code -type postgres -host localhost -port 5432 -username root -password 123456 -db mydb -prefix sys_ -tables sys_user -output temp -mp
```

参数说明：
- `-type` 数据库类型：`mysql` / `postgres` / `kingbase` / `sqlite` / `tidb` / `tdsql` / `polardb` / `opengauss` / `gaussdb` / `highgo` / `vastbase` / `oceanbase` / `oceanbase-oracle` / `dameng` / `gbase8s` / `mssql` / `oracle` / `db2`
- `-host` 数据库地址
- `-port` 端口
- `-username` / `-password` 认证信息（SQLite 无需填写）
- `-db` 数据库名（SQLite 为 `.db` 文件路径）
- `-output` 输出目录
- `-prefix` 可选，表名前缀
- `-tables` 可选，指定表名（逗号分隔），为空则生成全部表
- `-mp` 可选，生成 MyBatis-Plus 内置 CRUD XML（BaseMapper 标准方法名；批量方法自动生成切片签名，如 `DeleteBatchIds func([]int64)`；`SelectCount` 返回 `[]int64`）

## xml2go — 从 XML Mapper 逆向生成 Go 代码

```bash
go install github.com/bnulwh/mybatis-go/cmd/xml2go@latest
# 入口一：已知 Mapper XML 目录
xml2go -m resources/mapper -d gen -p github.com/xxx/app
# 入口二：直接给 MyBatis/MyBatis-Plus 配置文件（先解析 mapper 位置再生成）
xml2go -c src/main/resources/application.yml -d gen -p github.com/xxx/app
```

参数说明：
- `-c` mybatis/mybatis-plus 配置文件（.properties/.yml/.yaml/.xml），先从配置解析 Mapper XML 位置（`mybatis[-plus].mapper-locations`、`<mappers><mapper resource/>`、`<property name="mapperLocations">`；支持 `classpath*:` 前缀、`*`/`**` 通配、逗号多值、`config-location` 链式解析）再生成；设置时 `-m` 被忽略
- `-m` XML Mapper 目录（默认 `resources/mapper`）
- `-d` 输出目录（默认 `gen`，内含 `models/`、`mapper/` 两个子目录）
- `-p` 输出目录自身的模块导入路径前缀（生成 import 为 `<prefix>/models` 等；留空退化为裸导入，GOPATH 兼容）
- `-skip-mp` 可选，跳过 MyBatis-Plus 内置 CRUD 字段
- `-v` 可选，打印跳过语句及原因

生成代码用法与手写 Mapper 一致（`mapper.GetSysUserMapper().SelectUserByUserName("admin")`）；完整签名推导规则与已知边界见 **docs/agents/xml2go.md**。

## sqlc — 从静态 select 生成类型安全 Querier

```bash
go install github.com/bnulwh/mybatis-go/cmd/sqlc@latest
sqlc -m resources/mapper -d querier -p querier
```

参数说明：
- `-p` 输出包名（默认 `querier`）
- `-d` 输出目录（默认 `querier`）
- `-m` XML Mapper 文件目录（默认 `resources/mapper`）
- `-v` 可选，打印跳过语句及原因

扫描 XML Mapper，把静态 `<select>`（无动态标签、无 `${}`）抽取为类型安全的 Querier 接口 + 实现（model / `XxxQuerier` 接口 / `NewXxxQuerier(db)` 构造，一个 mapper 一个文件）；免连库——resultMap 按 jdbcType、resultType 按声明类型定型，标量参数按占位符名具名定型；动态语句自动跳过（`-v` 查看原因）；生成代码走 `QueryToContext`（方言占位符转换 / 表名前缀 / ctx 事务全部由 orm 层承担）。与 xml2go 互补——动态语句/MP CRUD 选 xml2go，纯静态查询选 sqlc。
