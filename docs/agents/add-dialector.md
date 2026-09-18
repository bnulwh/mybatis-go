# 添加数据库 Dialector 技能

> 本文档是 mybatis-go 项目新增数据库方言适配的完整操作手册。按步骤执行即可完成一种新数据库的适配、测试、提交与推送。

---

## 1. 术语与分类

| 分类 | 含义 | 已有成员 | 适配难度 |
|------|------|----------|----------|
| **MySQL 兼容族** (`FamilyMySQL`) | MySQL 协议兼容，复用 `go-sql-driver/mysql` 驱动 | MySQL / TiDB / TDSQL / PolarDB-MySQL | 零成本 |
| **PostgreSQL 兼容族** (`FamilyPostgres`) | PG 协议兼容，复用 `lib/pq` 驱动 | PostgreSQL / KingbaseES | 低（KingbaseES 模式） |
| **Oracle 方言族** (`FamilyOracle`) | 自有协议，需独立驱动 | 暂无 | 中高 |
| **Informix 方言族** (`FamilyInformix`) | Informix 兼容 | 暂无 | 中高 |

**零成本适配**：新数据库与某族完全兼容时，只需注册驱动别名 + 创建内嵌 dialector，无需覆写任何方法。

---

## 2. 适配前决策清单

开始写代码前，先确认以下信息并记录（后续步骤会用到）：

| 决策项 | 示例值 | 说明 |
|--------|--------|------|
| 数据库英文名 | `opengauss` | 用于 DatabaseType 常量值、驱动注册名、JDBC URL 协议段 |
| 兼容族 | `FamilyPostgres` | 决定 DSN 格式、占位符风格、NeedsReturning、表结构查询 SQL |
| 底层驱动 | `openGauss-connector-go-pq` | 实际 Go sql.Driver 实现 |
| 默认端口 | `5432` | 文档与 properties 模板使用 |
| JDBC URL 协议前缀 | `opengauss` | `parseAddr` 正则 `jdbc:<prefix>://` 已自动兼容任意前缀 |
| 是否需要额外 DSN 参数 | 否 | 如 MySQL 族需 `parseTime=true&loc=Local`（已内建） |
| 是否覆写 `NeedsReturning()` | PG 族 = true；MySQL/SQLite 族 = false | 控制自增主键回填走 `RETURNING` 还是 `LastInsertId()` |

---

## 3. 逐步操作

### 步骤 1：`orm/dialector/types.go` — 注册类型常量与路由

#### 1a. 添加 `DatabaseType` 常量

在 `DatabaseType` 常量块末尾添加：

```go
const (
    // ... 已有常量 ...
    OpenGaussDb DatabaseType = "opengauss"  // ← 新增
)
```

命名规则：`PascalCase` + `Db` 后缀（如 `TiDBDb`、`OpenGaussDb`、`DamengDb`）。

#### 1b. 添加 `Family()` 映射

在 `DatabaseType.Family()` 方法的 switch 中添加 case：

```go
case OpenGaussDb:
    return FamilyPostgres
```

- MySQL 兼容族 → `FamilyMySQL`（TiDB/TDSQL/PolarDB 等）
- PG 兼容族 → `FamilyPostgres`（openGauss/GaussDB/HighGo/Vastbase 等）
- 独立方言 → 新增 `DatabaseFamily` 常量（如 `FamilyOracle`）

#### 1c. 添加 `ParseDatabaseType()` case

```go
case "opengauss":
    return OpenGaussDb, nil
```

支持多个别名（如 Kingbase 支持 `kingbase5~8`）：

```go
case "opengauss", "opengauss-server":
    return OpenGaussDb, nil
```

#### 1d. 添加 `GetDriverName()` case

```go
case OpenGaussDb:
    return "opengauss"
```

#### 1e. 添加 `NewForType()` case

```go
case OpenGaussDb:
    return NewOpenGaussDialector(cfg), nil
```

---

### 步骤 2：`orm/dialector/<name>.go` — 创建 dialector 文件

文件放 `orm/dialector/` 目录下，命名与数据库英文名一致。

#### 模板 A：MySQL 兼容族（零成本，内嵌 MySqlDialector）

```go
package dialector

import (
    "database/sql"
    "github.com/go-sql-driver/mysql"
)

func init() {
    for _, name := range []string{"opengauss"} {  // ← 改为实际名称
        if isDriverRegistered(name) {
            continue
        }
        sql.Register(name, &mysql.MySQLDriver{})  // ← 改为实际驱动
    }
}

type OpenGaussDialector struct {   // ← 改名
    MySqlDialector
}

func NewOpenGaussDialector(cfg ConfigProvider) *OpenGaussDialector {  // ← 改名
    params := cfg.GetConnectParams()
    dsn := cfg.GetDSN()
    if dsn == "" {
        dsn = generateMySQLDSN(params)   // ← MySQL 族用 generateMySQLDSN
    }
    return &OpenGaussDialector{
        MySqlDialector: MySqlDialector{
            BaseDialector: BaseDialector{
                params:           params,
                name:             "opengauss",           // ← 数据库标识名
                driverName:       GetDriverName(params.Type),
                dsn:              dsn,
                conn:             cfg.GetConnPool(),
                family:           FamilyMySQL,            // ← 所属族
                placeholderStyle: PlaceholderQuestion,    // ← MySQL 族用 ?
                defaultMaxIdle:   DefaultMaxIdle,
                maxTimeout:       cfg.GetMaxTimeout(),
                maxOpen:          cfg.GetMaxOpen(),
            },
        },
    }
}
```

#### 模板 B：PostgreSQL 兼容族（内嵌 PostgresDialector）

```go
package dialector

import (
    "database/sql"
    "github.com/lib/pq"     // ← 或专用驱动 fork
)

func init() {
    for _, name := range []string{"opengauss"} {
        if isDriverRegistered(name) {
            continue
        }
        sql.Register(name, &pq.Driver{})   // ← 或专用驱动
    }
}

type OpenGaussDialector struct {
    PostgresDialector                       // ← 内嵌 PG dialector
}

func NewOpenGaussDialector(cfg ConfigProvider) *OpenGaussDialector {
    params := cfg.GetConnectParams()
    dsn := cfg.GetDSN()
    if dsn == "" {
        dsn = generatePostgresDSN(params)   // ← PG 族用 generatePostgresDSN
    }
    return &OpenGaussDialector{
        PostgresDialector: PostgresDialector{
            BaseDialector: BaseDialector{
                params:           params,
                name:             "opengauss",
                driverName:       GetDriverName(params.Type),
                dsn:              dsn,
                conn:             cfg.GetConnPool(),
                family:           FamilyPostgres,         // ← PG 族
                placeholderStyle: PlaceholderDollar,      // ← PG 族用 $n
                defaultMaxIdle:   DefaultMaxIdle,
                maxTimeout:       cfg.GetMaxTimeout(),
                maxOpen:          cfg.GetMaxOpen(),
            },
        },
    }
}
```

#### 模板 C：独立方言（内嵌 BaseDialector）

```go
package dialector

import (
    "database/sql"
    "github.com/dameng/go-driver"   // ← 专用驱动
)

func init() {
    for _, name := range []string{"dameng"} {
        if isDriverRegistered(name) {
            continue
        }
        sql.Register(name, &dameng.Driver{})
    }
}

type DamengDialector struct {
    BaseDialector
}

func NewDamengDialector(cfg ConfigProvider) *DamengDialector {
    params := cfg.GetConnectParams()
    dsn := cfg.GetDSN()
    if dsn == "" {
        dsn = generateDamengDSN(params)    // ← 需自写 DSN 生成函数
    }
    return &DamengDialector{
        BaseDialector: BaseDialector{
            params:           params,
            name:             "dameng",
            driverName:       GetDriverName(params.Type),
            dsn:              dsn,
            conn:             cfg.GetConnPool(),
            family:           FamilyOracle,             // ← 新族或已有族
            placeholderStyle: PlaceholderColon,         // ← 或 PlaceholderQuestion
            defaultMaxIdle:   DefaultMaxIdle,
            maxTimeout:       cfg.GetMaxTimeout(),
            maxOpen:          cfg.GetMaxOpen(),
        },
    }
}

// 以下方法按需覆写：

func (d *DamengDialector) NeedsReturning() bool {
    return false   // Oracle 风格用自增序列，不用 RETURNING
}

func (d *DamengDialector) SystemTablePrefixes() []string {
    return []string{"SYS", "SYSTEM"}   // ← 系统表前缀
}

func (d *DamengDialector) TableStructureSQL(table string) string {
    schema := d.effectiveSchema()
    return fmt.Sprintf(`SELECT ... FROM ... WHERE OWNER='%s' AND TABLE_NAME='%s'`, schema, table)
}

func (d *DamengDialector) TableListSQL() string {
    schema := d.effectiveSchema()
    return fmt.Sprintf("SELECT TABLE_NAME FROM ALL_TABLES WHERE OWNER='%s'", schema)
}
```

**独立方言需额外添加：**

- `generateXxxDSN(params ConnectParams) string` — 在 `types.go` 中添加并在 `GenerateDSN()` switch 中注册
- 如需新 `PlaceholderStyle`，在 `PlaceholderStyle` const 块添加并在 `formatPlaceholders()` 中处理
- 如需新 `DatabaseFamily`，在 `DatabaseFamily` const 块添加并在 `EffectiveSchema()` 中处理

---

### 步骤 3：`orm/interfaces.go` — 导出新常量

```go
const (
    // ... 已有常量 ...
    OpenGaussDb = dialector.OpenGaussDb   // ← 新增
)
```

---

### 步骤 4：`cmd/<name>demo/main.go` — 创建可运行 demo

目录：`cmd/<name>demo/main.go`

```go
package main

import (
    log "github.com/bnulwh/logrus"
    "github.com/bnulwh/mybatis-go/orm"
    "github.com/bnulwh/mybatis-go/types"
    _ "github.com/go-sql-driver/mysql"   // ← 改为实际驱动引入（框架自动注册的可省略）
    "time"
)

type UserInfoModel struct {
    Id          int
    CreatedBy   string
    UpdatedBy   string
    CreateTime  time.Time
    UpdateTime  time.Time
    GroupId     int
    Username    string
    PassMd5     string
    Roles       string
    Description string
    Avatar      string
}

type UserInfoModelMapper struct {
    orm.BaseMapper
    DeleteByPrimaryKey func(id int) (int64, error)
    Insert             func(model UserInfoModel) (int64, error)
    UpdateByPrimaryKey func(model UserInfoModel) (int64, error)
    SelectByPrimaryKey func(id int) ([]UserInfoModel, error)
    SelectAll          func() ([]UserInfoModel, error)
}

func init() {
    log.ConfigLocalFileSystemLogger("logs", "opengaussdemo")  // ← 改名
    orm.SetLogger(log.StandardLogger())
    orm.Initialize("cmd/opengaussdemo/application-opengauss.properties")  // ← 改名
    orm.RegisterModel(new(UserInfoModel))
    orm.RegisterMapper(new(UserInfoModelMapper))
}

func main() {
    defer orm.Close()
    mp := orm.NewMapper("UserInfoModelMapper").(UserInfoModelMapper)
    rs, err := mp.SelectAll()
    if err != nil {
        log.Errorf("select failed: %v", err)
    } else {
        for _, row := range rs {
            log.Infof("row: %v", types.ToJson(row))
        }
    }
}
```

> 框架自动注册驱动的数据库（KingbaseES/TiDB/TDSQL/PolarDB 等），demo 中 `_ "driver"` 引入行可省略。

---

### 步骤 5：`application-<name>.properties` + `README.md` — 配置模板与说明

与 demo 同目录，放在 `cmd/<name>demo/` 下，并为 demo 目录编写 README.md（数据库描述、配置说明、驱动引入方式、运行命令，参考既有 demo 目录）。

#### 统一模板

```properties
# <数据库中文名>（<方言族/兼容说明>，默认端口 <PORT>）
spring.datasource.url= jdbc:<name>://localhost:<PORT>/<dbname>
spring.datasource.username= root
spring.datasource.password= 123456
# 连接池设置（可选，省略则使用框架默认值）
spring.datasource.max-idle= 100
spring.datasource.max-open= 100
spring.datasource.max-timeout= 100
mybatis.mapper-locations= resources/mapper
```

MySQL 兼容族的 URL 可追加 `?useUnicode=true&characterEncoding=utf-8&useSSL=false`；SQLite 等文件型数据库省略 username/password/池设置。

---

### 步骤 6：测试

#### 6a. Dialector 单元测试

在 `orm/dialector/kingbase_test.go` 中追加（该文件是 dialector 测试的统一位置）：

```go
func Test_OpenGaussDriverRegistered(t *testing.T) {
    if !isDriverRegistered("opengauss") {
        t.Error("opengauss driver should be registered by init")
    }
}

func Test_OpenGaussFormatPrepareSQL(t *testing.T) {
    d := NewOpenGaussDialector(&testConfig{dbType: OpenGaussDb})
    src := "select * from t where a = ? and b = ?"
    got := d.FormatPrepareSQL(src)
    want := "select * from t where a = $1 and b = $2"   // PG 族: ? → $n
    // MySQL 族: got 应等于 src（保持 ?）
    if got != want {
        t.Errorf("opengauss format prepare sql failed, got: %q want: %q", got, want)
    }
}

func Test_OpenGaussNeedsReturning(t *testing.T) {
    d := NewOpenGaussDialector(&testConfig{dbType: OpenGaussDb})
    if !d.NeedsReturning() {   // PG 族期望 true；MySQL 族期望 false
        t.Error("opengauss should need RETURNING")
    }
}
```

#### 6b. 扩展现有通用测试

在 `kingbase_test.go` 的以下测试函数中追加新数据库的 case：

| 测试函数 | 追加内容 |
|----------|----------|
| `Test_ParseDatabaseType` | `{"opengauss", OpenGaussDb}` |
| `Test_DatabaseTypeFamily` | `OpenGaussDb.Family() != FamilyPostgres` 断言 |
| `Test_GetDriverName` | `{OpenGaussDb, "opengauss"}` |
| `Test_EffectiveSchema` | `{ConnectParams{DBName: "mydb", Type: OpenGaussDb}, "public"}` （PG 族）/ `"mydb"`（MySQL 族） |
| `Test_GenerateDSN` | 新增 DSN 生成断言 |
| `Test_NewForType` | `NewForType(OpenGaussDb, ...)` 验证 `d.Name() == "opengauss"` |

#### 6c. ORM 层 JDBC URL 解析测试

在 `orm/database_config_test.go` 的 `Test_parseAddr` 中追加：

```go
mp["spring.datasource.url"] = "jdbc:opengauss://10.1.2.3:5432/testdb"
tp, host, port, db, err := parseAddr(mp)
if tp != "opengauss" || host != "10.1.2.3" || port != 5432 || db != "testdb" || err != nil {
    t.Error("test parseAddr opengauss failed.")
}
```

在 `Test_parseDatabaseType` 中追加：

```go
r, err := dialector.ParseDatabaseType("opengauss")
if r != OpenGaussDb || err != nil {
    t.Error("test parseDatabaseType opengauss failed.")
}
```

#### 6d. 运行测试

```bash
go build ./...
go test -count=1 ./... 
```

**必须全部通过才能继续。**

---

### 步骤 7：更新 README.md

#### 7a. 特性描述

将新数据库加入「多数据库」特性行（如从"进行中"列表移到已支持列表）。

#### 7b. P0/P1/P2 清单

将对应条目从 `[ ]` 改为 `[x]`，补充示例目录名。

#### 7c. 驱动引入表

新增一行（框架自动注册的注明"无需引入"）：

```markdown
| <中文名> | 无需引入（框架自动以 `<name>` 名称注册 `<driver>`） |
```

#### 7d. 配置章节

在 docs/configuration.md「配置文件」的「KingbaseES（人大金仓）」小节之后新增 `<中文名>` 配置小节，含 properties 示例。

#### 7e. 运行示例

追加 `go run ./cmd/<name>demo` 行。

#### 7f. 项目结构

- `cmd/` 下追加 `<name>demo/` 条目
- `orm/dialector/` 下追加 `<name>.go` 条目

#### 7g. 编程式配置

在 docs/configuration.md「编程式配置」追加 `orm.InitializeDatabase("<name>", ...)` 示例行。

#### 7h. 更新日志

在 CHANGELOG.md 顶部追加新版本条目（README「更新日志」仅保留指向 CHANGELOG.md 的链接）。

#### 7i. schema2code

如适用，更新 `-type` 参数说明（如 `mysql/postgres/kingbase/sqlite/tidb/tdsql/polardb/opengauss`）。

---

### 步骤 8：提交与推送

```bash
git add orm/dialector/types.go orm/dialector/<name>.go orm/dialector/kingbase_test.go \
        orm/interfaces.go orm/database_config_test.go \
        cmd/<name>demo/ \
        cmd/schema2code/main.go README.md docs/configuration.md CHANGELOG.md
git commit -m "feat: <中文名>数据库适配 — <英文名> (<族>兼容族)

- 新增 DatabaseType 常量: <Name>Db，Family() 映射 Family<Family>
- ParseDatabaseType 支持 <name>/<aliases>
- GetDriverName 返回 <name>，框架 init 自动注册 <driver> 别名
- 新增 orm/dialector/<name>.go（内嵌 <Parent>Dialector）
- NewForType 支持 <Name>Db
- JDBC URL 解析支持 jdbc:<name>:// 格式
- 新增 cmd/<name>demo 示例 + application-<name>.properties 配置模板 + demo README
- schema2code -type 支持 <name>
- dialector + orm 单测覆盖
- README 更新: [x] 标记/驱动表/示例/结构；docs/configuration.md 配置+编程式示例；CHANGELOG.md 更新日志"
git push
```

---

## 4. 各文件改动汇总表

| 文件 | 改动类型 | 改动内容 |
|------|----------|----------|
| `orm/dialector/types.go` | 修改 | DatabaseType 常量 + Family + ParseDatabaseType + GetDriverName + NewForType + (可选) GenerateDSN/EffectiveSchema |
| `orm/dialector/<name>.go` | **新增** | dialector 实现（init 注册驱动 + struct + 构造函数 + 可选方法覆写） |
| `orm/dialector/kingbase_test.go` | 修改 | 追加新数据库的单元测试 |
| `orm/interfaces.go` | 修改 | 导出新 DatabaseType 常量 |
| `orm/database_config_test.go` | 修改 | 追加 JDBC URL 解析 + 类型解析测试 |
| `cmd/<name>demo/main.go` | **新增** | 可运行 demo |
| `cmd/<name>demo/application-<name>.properties` | **新增** | 配置模板（与 demo 同目录） |
| `cmd/<name>demo/README.md` | **新增** | demo 说明（配置/驱动/运行） |
| `cmd/schema2code/main.go` | 修改 | usage 字符串追加新类型 |
| `README.md` | 修改 | 特性/清单/驱动表/示例/结构 |
| `docs/configuration.md` | 修改 | 配置小节 + 编程式配置示例行 |
| `CHANGELOG.md` | 修改 | 顶部追加新版本条目 |

---

## 5. 特殊场景

### 5a. 驱动需要额外依赖

在 `go.mod` 中添加新驱动依赖：

```bash
go get github.com/opengauss-international/opengauss-connector-go-pq
```

然后在新 dialector 文件中 import 该驱动。

### 5b. 新 DatabaseFamily

独立方言（如达梦 DM8）不属于现有任何族时：

1. `types.go` 新增 `FamilyOracle DatabaseFamily = "oracle"`
2. `DatabaseType.Family()` 添加 `case DamengDb: return FamilyOracle`
3. `EffectiveSchema()` 添加该族的 schema 默认值
4. `GenerateDSN()` 添加该族的 DSN 生成函数
5. `orm/database_connection.go` 的 `Open()` 可能需要特殊处理

### 5c. 新 PlaceholderStyle

如达梦使用 `:1`/`:2` 风格，已有 `PlaceholderColon`，无需新增。如需新风格，在 `PlaceholderStyle` const 块添加并在 `formatPlaceholders()` switch 中处理。

### 5d. 驱动自动注册 vs 用户引入

| 方式 | 适用场景 | 实现位置 |
|------|----------|----------|
| **框架 init 自动注册** | 驱动已有项目依赖（`go-sql-driver/mysql`、`lib/pq`） | dialector 文件的 `init()` 函数，用 `sql.Register` 注册别名 |
| **用户 blank import** | 驱动不在项目依赖中，或用户需要显式控制 | demo 中 `_ "driver/path"` 引入，dialector 不做 init 注册 |

框架自动注册时，需在 dialector 文件中做 `isDriverRegistered()` 防重复注册。

---

## 6. 验证清单

适配完成后，逐项确认：

- [ ] `go build ./...` 编译通过
- [ ] `go test -count=1 ./...` 全部通过
- [ ] `ParseDatabaseType` 支持目标名称与别名
- [ ] `Family()` 返回正确的族
- [ ] `GetDriverName` 返回驱动注册名
- [ ] `NewForType` 返回正确 dialector 且 `Name()` 正确
- [ ] 驱动已在 `init()` 中注册（`isDriverRegistered` 验证）
- [ ] `FormatPrepareSQL` 占位符转换正确（`?` → `$n` 或保持 `?`）
- [ ] `NeedsReturning` 符合族约定
- [ ] `GenerateDSN` 生成正确 DSN
- [ ] JDBC URL `jdbc:<name>://host:port/db` 可被 `parseAddr` 正确解析
- [ ] `cmd/<name>demo/main.go` 可编译
- [ ] `cmd/<name>demo/application-<name>.properties` 配置模板正确
- [ ] `README.md` 已更新（清单/驱动表/示例/结构）；docs/configuration.md 已加配置小节；CHANGELOG.md 已加条目
- [ ] `git push` 推送成功
