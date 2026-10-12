# 信创数据库适配实施计划（GoldenDB / GreatDB / AntDB / 神通 Oscar）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 mybatis-go 补齐 4 款信创数据库支持：中兴 GoldenDB、万里 GreatDB、亚信 AntDB（三款零成本协议兼容适配），并完成神通 Oscar 的驱动可行性调研与决策。

**Architecture:** GoldenDB/GreatDB 走 MySQL 兼容族零成本路径（内嵌 `MySqlDialector`，注册 `go-sql-driver/mysql` 驱动别名）；AntDB 走 PostgreSQL 兼容族路径（内嵌 `PostgresDialector`，注册 `lib/pq` 驱动别名）；Oscar（Oracle 兼容、自有协议，无公开 Go 驱动）先做调研 spike，产出决策文档后再立项实现。每款数据库遵循 `docs/agents/add-dialector.md` 的完整流程：TDD 测试 → dialector 实现 → demo → 文档 → 提交。

**Tech Stack:** Go 1.24、`database/sql`、`go-sql-driver/mysql v1.10.0`、`lib/pq v1.12.3`（均已在 go.mod 中，零新增依赖）。

---

## 适配策略总览

| 数据库 | 英文名 | 兼容族 | 驱动 | 默认端口 | 成本 | 批次 |
|--------|--------|--------|------|----------|------|------|
| 中兴 GoldenDB | `goldendb` | `FamilyMySQL` | `go-sql-driver/mysql` 别名 | 3306 | 零成本 | Task 1 |
| 万里 GreatDB | `greatdb` | `FamilyMySQL` | `go-sql-driver/mysql` 别名 | 3306 | 零成本 | Task 2 |
| 亚信 AntDB | `antdb` | `FamilyPostgres` | `lib/pq` 别名 | 6600 | 低成本 | Task 3 |
| 神通 Oscar | `oscar` | 独立（Oracle 语义） | **未知，需调研** | 2003 | 中高 | Task 0（调研） |

## 任务依赖与执行顺序

- Task 1 → Task 2 → Task 3 **必须串行执行**（三者修改同一批文件：`orm/dialector/types.go`、`orm/dialector/kingbase_test.go`、`orm/interfaces.go`、`orm/database_config_test.go`，并行会产生编辑冲突；且测试内变量编号 r24/r25/r26、tp20/tp21/tp22、d19/d20/d21 依赖前一任务已占用编号）。
- Task 0（Oscar 调研）与 Task 1-3 无文件交集，可任意穿插或并行执行。
- Task 4（全量回归）在 Task 1-3 全部完成后执行。

## 硬性约定（每个任务必须遵守）

- 测试失败用 `t.Error()` 而非 `t.Fatal()`。
- 业务代码返回 `(value, error)` 双返回值。
- 日志一律走 `log` 包。
- Go 1.24；禁止引入要求 go > 1.24.0 的依赖。
- 临时 go 可执行文件（`go test -c` 等）一律 `-o D:\Develop\bin\<name>.exe`。
- 每个任务完成即 commit（遵循项目自动提交约定），**push 需用户确认**。

---

### Task 0: 神通 Oscar 驱动可行性调研（spike）

**Files:**
- Create: `docs/agents/oscar-driver-research.md`（调研决策文档）

**背景：** 神通数据库（Oscar，天津神舟通用）兼容 Oracle SQL 语义但使用自有通信协议，`sijms/go-ora` 无法直连。官方是否提供 Go 语言驱动、协议是否开放，是能否适配的前置条件。本任务只调研、不写业务代码。

- [ ] **Step 1: 调研官方驱动渠道**

  依次检索以下渠道并记录结果（每条记录来源 URL 与结论）：
  1. 神舟通用官网（shentongdata.com.cn）下载/文档中心：查找 Go 驱动、ODBC/C SDK；
  2. GitHub / Gitee 搜索：`oscar golang driver`、`神通数据库 golang`、`shentong go driver`；
  3. pkg.go.dev 搜索：`oscar`、`shentong`；
  4. 确认 Oscar 通信协议是否兼容 Oracle TNS 线协议（若兼容则 `go-ora` 可复用，零成本）。

- [ ] **Step 2: 可行则做驱动连通性验证**

  仅当 Step 1 找到候选驱动时执行。在 `C:\Users\qdn16\AppData\Local\Temp\deveco` 下建临时模块验证：

  ```powershell
  New-Item -ItemType Directory -Force C:\Users\qdn16\AppData\Local\Temp\deveco\oscar-spike
  # 临时 main.go：database/sql.Open("<驱动名>", "<DSN>") + Ping，编译产物输出 D:\Develop\bin
  go build -o D:\Develop\bin\oscar-spike.exe .
  ```

  无 Oscar 实例时，以「能否编译 + `sql.Driver` 接口是否完整」为最低验证标准，连通性标注「待真机验证」。

- [ ] **Step 3: 写决策文档**

  创建 `docs/agents/oscar-driver-research.md`，内容必须包含以下章节（结论按实际调研结果填写，证据留来源链接）：

  ```markdown
  # 神通 Oscar Go 驱动可行性调研

  ## 结论（A / B / C 三选一）
  - A：存在可用的 database/sql 兼容 Go 驱动 → 建议立项，按独立方言适配（参照 oceanbase_oracle.go 模式）
  - B：无 Go 驱动，仅有 ODBC/C SDK → 记录 cgo + ODBC 方案评估结论（不建议进主库，可作为可选构建标签）
  - C：无任何可行驱动路径 → README 文档化为「规划中」，关闭本项

  ## 调研证据
  | 渠道 | 来源 | 结果 |
  |------|------|------|

  ## 关键技术参数（若结论为 A）
  - 驱动 import path / 驱动注册名
  - DSN 格式
  - 占位符风格（? / :n / 其他）
  - 默认端口 / 系统表前缀 / 表结构查询 SQL 样例

  ## 后续行动
  ```

- [ ] **Step 4: Commit**

  ```bash
  git add docs/agents/oscar-driver-research.md
  git commit -m "docs: 神通 Oscar Go 驱动可行性调研（信创适配前置 spike）"
  ```

---

### Task 1: 中兴 GoldenDB 适配（MySQL 兼容族，零成本）

**Files:**
- Modify: `orm/dialector/types.go`
- Create: `orm/dialector/goldendb.go`
- Modify: `orm/dialector/kingbase_test.go`
- Modify: `orm/interfaces.go`
- Modify: `orm/database_config_test.go`
- Create: `cmd/goldendbdemo/main.go`、`cmd/goldendbdemo/application-goldendb.properties`、`cmd/goldendbdemo/README.md`
- Modify: `cmd/schema2code/main.go`、`README.md`、`docs/configuration.md`、`docs/code-generation.md`、`CHANGELOG.md`

- [ ] **Step 1: 写失败测试 — `orm/dialector/kingbase_test.go`**

  在文件末尾追加 4 个新测试函数：

  ```go
  func Test_GoldenDBDriverRegistered(t *testing.T) {
  	if !isDriverRegistered("goldendb") {
  		t.Error("goldendb driver should be registered by init")
  	}
  }

  func Test_GoldenDBFormatPrepareSQL(t *testing.T) {
  	d := NewGoldenDBDialector(&testConfig{dbType: GoldenDBDb})
  	src := "select * from t where a = ? and b = ?"
  	got := d.FormatPrepareSQL(src)
  	if got != src {
  		t.Errorf("goldendb format prepare sql should keep ?, got: %q", got)
  	}
  }

  func Test_GoldenDBNeedsReturning(t *testing.T) {
  	d := NewGoldenDBDialector(&testConfig{dbType: GoldenDBDb})
  	if d.NeedsReturning() {
  		t.Error("goldendb should not need RETURNING")
  	}
  }

  func Test_GoldenDBDialectorName(t *testing.T) {
  	d := NewGoldenDBDialector(&testConfig{dbType: GoldenDBDb})
  	if d.Name() != "goldendb" {
  		t.Errorf("goldendb dialector name failed, got: %q", d.Name())
  	}
  }
  ```

  并在既有表驱动测试中追加条目：

  1. `Test_ParseDatabaseType` 的 `tests` 切片中，`{"clickhouse", ClickHouseDb},` 之后追加：

  ```go
  		{"goldendb", GoldenDBDb},
  		{"golden-db", GoldenDBDb},
  ```

  2. `Test_DatabaseTypeFamily` 末尾追加：

  ```go
  	if GoldenDBDb.Family() != FamilyMySQL {
  		t.Errorf("GoldenDBDb.Family() = %q, want %q", GoldenDBDb.Family(), FamilyMySQL)
  	}
  ```

  3. `Test_GetDriverName` 的 `tests` 切片中 `{ClickHouseDb, "clickhouse"},` 之后追加：

  ```go
  		{GoldenDBDb, "goldendb"},
  ```

  4. `Test_EffectiveSchema` 的 `tests` 切片末尾追加：

  ```go
  		{ConnectParams{DBName: "mydb", Type: GoldenDBDb}, "mydb"},
  ```

  5. `Test_GenerateDSN` 函数体末尾（`}` 之前）追加：

  ```go
  	goldendbParams := ConnectParams{Host: "10.0.0.1", Port: 3306, Username: "root", Password: "123456", DBName: "testdb", Type: GoldenDBDb}
  	if got := GenerateDSN(goldendbParams); got != "root:123456@tcp(10.0.0.1:3306)/testdb?parseTime=true&loc=Local" {
  		t.Errorf("GenerateDSN goldendb = %q", got)
  	}
  ```

  6. `Test_NewForType` 中 `if _, err := NewForType(DatabaseType("unknown"), ...)` 之前追加：

  ```go
  	d19, err := NewForType(GoldenDBDb, &testConfig{dbType: GoldenDBDb})
  	if err != nil || d19.Name() != "goldendb" {
  		t.Errorf("NewForType goldendb failed: %v, name=%q", err, d19.Name())
  	}
  ```

- [ ] **Step 2: 写失败测试 — `orm/database_config_test.go`**

  1. `Test_parseDatabaseType` 末尾（`}` 之前）追加：

  ```go
  	r24, err := dialector.ParseDatabaseType("goldendb")
  	if r24 != GoldenDBDb || err != nil {
  		t.Error("test parseDatabaseType goldendb failed.")
  	}
  ```

  2. `Test_parseAddr` 中 clickhouse 断言块之后追加：

  ```go
  	mp["spring.datasource.url"] = "jdbc:goldendb://10.1.2.3:3306/testdb"
  	tp20, host20, port20, db20, err := parseAddr(mp)
  	if tp20 != "goldendb" || host20 != "10.1.2.3" || port20 != 3306 || db20 != "testdb" || err != nil {
  		t.Error("test parseAddr goldendb failed.")
  	}
  ```

- [ ] **Step 3: 运行测试确认失败**

  ```bash
  go test ./orm/dialector/ -run "GoldenDB" -count=1
  go test ./orm/ -run "Test_parseDatabaseType|Test_parseAddr" -count=1
  ```

  预期：FAIL，报 `undefined: GoldenDBDb` / `undefined: NewGoldenDBDialector`。

- [ ] **Step 4: 实现 — `orm/dialector/types.go`（5 处修改）**

  1. 常量块 `ClickHouseDb    DatabaseType = "clickhouse"` 之后追加：

  ```go
  	GoldenDBDb       DatabaseType = "goldendb"
  ```

  2. `Family()` 中 `case MySqlDb, TiDBDb, TDSQLDb, PolarDBMyDb, OceanBaseDb:` 追加成员，改为：

  ```go
  	case MySqlDb, TiDBDb, TDSQLDb, PolarDBMyDb, OceanBaseDb, GoldenDBDb:
  ```

  3. `ParseDatabaseType` 中 `case "clickhouse", "click-house":` 块之后追加：

  ```go
  	case "goldendb", "golden-db":
  		return GoldenDBDb, nil
  ```

  4. `GetDriverName` 中 `case ClickHouseDb:` 块之后追加：

  ```go
  	case GoldenDBDb:
  		return "goldendb"
  ```

  5. `NewForType` 中 `case ClickHouseDb:` 块之后追加：

  ```go
  	case GoldenDBDb:
  		return NewGoldenDBDialector(cfg), nil
  ```

- [ ] **Step 5: 实现 — 新建 `orm/dialector/goldendb.go`**

  ```go
  package dialector

  import (
  	"database/sql"

  	"github.com/go-sql-driver/mysql"
  )

  func init() {
  	for _, name := range []string{"goldendb"} {
  		if isDriverRegistered(name) {
  			continue
  		}
  		sql.Register(name, &mysql.MySQLDriver{})
  	}
  }

  type GoldenDBDialector struct {
  	MySqlDialector
  }

  func NewGoldenDBDialector(cfg ConfigProvider) *GoldenDBDialector {
  	params := cfg.GetConnectParams()
  	dsn := cfg.GetDSN()
  	if dsn == "" {
  		dsn = generateMySQLDSN(params)
  	}
  	return &GoldenDBDialector{
  		MySqlDialector: MySqlDialector{
  			BaseDialector: BaseDialector{
  				params:           params,
  				name:             "goldendb",
  				driverName:       GetDriverName(params.Type),
  				dsn:              dsn,
  				conn:             cfg.GetConnPool(),
  				family:           FamilyMySQL,
  				placeholderStyle: PlaceholderQuestion,
  				defaultMaxIdle:   DefaultMaxIdle,
  				maxTimeout:       cfg.GetMaxTimeout(),
  				maxOpen:          cfg.GetMaxOpen(),
  			},
  		},
  	}
  }
  ```

- [ ] **Step 6: 实现 — `orm/interfaces.go`**

  常量块 `ClickHouseDb     = dialector.ClickHouseDb` 之后追加：

  ```go
  	GoldenDBDb       = dialector.GoldenDBDb
  ```

- [ ] **Step 7: 运行测试确认通过**

  ```bash
  go test ./orm/dialector/ -run "GoldenDB|ParseDatabaseType|DatabaseTypeFamily|GetDriverName|EffectiveSchema|GenerateDSN|NewForType" -count=1
  go test ./orm/ -run "Test_parseDatabaseType|Test_parseAddr" -count=1
  ```

  预期：全部 PASS。

- [ ] **Step 8: 创建 demo — `cmd/goldendbdemo/`（3 个文件）**

  `cmd/goldendbdemo/main.go`：

  ```go
  package main

  import (
  	log "github.com/bnulwh/logrus"
  	"github.com/bnulwh/mybatis-go/orm"
  	"github.com/bnulwh/mybatis-go/types"
  	_ "github.com/go-sql-driver/mysql"
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
  	log.ConfigLocalFileSystemLogger("logs", "goldendbdemo")
  	orm.SetLogger(log.StandardLogger())
  	orm.Initialize("cmd/goldendbdemo/application-goldendb.properties")
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

  `cmd/goldendbdemo/application-goldendb.properties`：

  ```properties
  # GoldenDB（中兴，MySQL 兼容，默认端口 3306）
  spring.datasource.url= jdbc:goldendb://localhost:3306/testdb?useUnicode=true&characterEncoding=utf-8&useSSL=false
  spring.datasource.username= root
  spring.datasource.password= 123456
  # 连接池设置（可选，省略则使用框架默认值）
  spring.datasource.max-idle= 100
  spring.datasource.max-open= 100
  spring.datasource.max-timeout= 100
  mybatis.mapper-locations= resources/mapper
  ```

  `cmd/goldendbdemo/README.md`：

  ```markdown
  # GoldenDB Demo（中兴）

  mybatis-go 连接 GoldenDB（中兴）的使用示例。MySQL 兼容，默认端口 3306。

  ## 配置

  本目录的 [`application-goldendb.properties`](./application-goldendb.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

  ## 驱动

  无需引入——框架自动以 `goldendb` 名称注册 `go-sql-driver/mysql` 驱动。

  ## 运行

  在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

  ```bash
  go run ./cmd/goldendbdemo
  ```
  ```

- [ ] **Step 9: 更新文档（5 个文件）**

  1. `cmd/schema2code/main.go:23` usage 字符串末尾 `/clickhouse` 后追加 `/goldendb`；
  2. `docs/code-generation.md:54` `-type` 类型列表末尾 `/clickhouse` 后追加 `/goldendb`；
  3. `README.md`：
     - 「多数据库」特性行（line 17）：`GBase 8s（南大通用）、` 之后插入 `GoldenDB（中兴）、`；
     - P0/P1/P2 清单：`- [x] GBase 8s（南大通用）` 条目之后追加 `- [x] GoldenDB（中兴）— MySQL 兼容，内嵌 MySqlDialector（MySQL 兼容族），cmd/goldendbdemo 示例`；
     - 驱动引入表：ClickHouse 行之后追加 `| GoldenDB（中兴） | 无需引入（框架自动以 goldendb 名称注册 go-sql-driver/mysql 驱动） |`；
     - 「完整示例见」行末尾句号前追加 ``、`cmd/goldendbdemo/main.go` ``；
     - 运行示例命令区：`go run ./cmd/clickhousedemo` 行之后追加 `go run ./cmd/goldendbdemo   # GoldenDB（中兴，MySQL 兼容族）`；
     - 项目结构：`│   ├── clickhousedemo/` 之后追加 `│   ├── goldendbdemo/   # GoldenDB（中兴）使用示例`；`│   │   ├── clickhouse.go` 之后追加 `│   │   ├── goldendb.go   # GoldenDB（MySQL 兼容族）`；
  4. `docs/configuration.md`：ClickHouse 配置小节之后新增小节：

  ````markdown
  ### GoldenDB（中兴）

  ```properties
  # GoldenDB（中兴，MySQL 兼容，默认端口 3306）
  spring.datasource.url= jdbc:goldendb://localhost:3306/testdb?useUnicode=true&characterEncoding=utf-8&useSSL=false
  spring.datasource.username= root
  spring.datasource.password= 123456
  mybatis.mapper-locations= resources/mapper
  ```

  GoldenDB（中兴）为 MySQL 兼容族：占位符保持 `?`（`PlaceholderQuestion`）；`NeedsReturning` 返回 false（自增主键回填走 `LastInsertId()`）；`EffectiveSchema` 返回数据库名。框架自动以 `goldendb` 名称注册 `go-sql-driver/mysql` 驱动，无需额外引入。
  ````

  并在「编程式配置」示例中 `orm.InitializeDatabase("clickhouse", ...)` 行之后追加：

  ```go
  // 或 orm.InitializeDatabase("goldendb", "localhost", 3306, "root", "123456", "testdb")
  ```

  5. `CHANGELOG.md` 顶部（`# 更新日志` 之后）插入：

  ```markdown
  - **v0.3.14（信创数据库 GoldenDB 适配，2026-10-12）**：中兴 GoldenDB — MySQL 协议兼容，内嵌 `MySqlDialector`（MySQL 兼容族），零成本适配 — 新增 `DatabaseType` 常量 `GoldenDBDb`（`Family()` 映射 `FamilyMySQL`）；`ParseDatabaseType` 支持 `goldendb`/`golden-db` 别名；`GetDriverName` 返回 `goldendb`，框架 init 自动以 `goldendb` 名称注册 `go-sql-driver/mysql` 驱动别名（无需新增依赖）；`NewForType` 支持 `GoldenDBDb`；JDBC URL 支持 `jdbc:goldendb://` 格式；新增 `cmd/goldendbdemo` 示例 + `application-goldendb.properties` 配置模板；schema2code `-type goldendb`；dialector + orm 单测覆盖（驱动注册、占位符、NeedsReturning、DSN 生成、ParseDatabaseType 别名、Family 映射、GetDriverName、EffectiveSchema、NewForType、parseAddr）
  ```

- [ ] **Step 10: 编译 + 全量测试**

  ```bash
  go build ./...
  go vet ./...
  go test -count=1 ./...
  ```

  预期：全部 PASS，零回归。

- [ ] **Step 11: Commit**

  ```bash
  git add orm/dialector/types.go orm/dialector/goldendb.go orm/dialector/kingbase_test.go \
          orm/interfaces.go orm/database_config_test.go \
          cmd/goldendbdemo/ cmd/schema2code/main.go README.md docs/configuration.md docs/code-generation.md CHANGELOG.md
  git commit -m "feat: 中兴 GoldenDB 数据库适配 — goldendb (MySQL 兼容族)

  - 新增 DatabaseType 常量: GoldenDBDb，Family() 映射 FamilyMySQL
  - ParseDatabaseType 支持 goldendb/golden-db
  - GetDriverName 返回 goldendb，框架 init 自动注册 go-sql-driver/mysql 别名
  - 新增 orm/dialector/goldendb.go（内嵌 MySqlDialector）
  - NewForType 支持 GoldenDBDb
  - JDBC URL 解析支持 jdbc:goldendb:// 格式
  - 新增 cmd/goldendbdemo 示例 + application-goldendb.properties 配置模板 + demo README
  - schema2code -type 支持 goldendb
  - dialector + orm 单测覆盖
  - README/docs/CHANGELOG 更新"
  ```

---

### Task 2: 万里 GreatDB 适配（MySQL 兼容族，零成本）

**Files:**
- Modify: `orm/dialector/types.go`
- Create: `orm/dialector/greatdb.go`
- Modify: `orm/dialector/kingbase_test.go`
- Modify: `orm/interfaces.go`
- Modify: `orm/database_config_test.go`
- Create: `cmd/greatdbdemo/main.go`、`cmd/greatdbdemo/application-greatdb.properties`、`cmd/greatdbdemo/README.md`
- Modify: `cmd/schema2code/main.go`、`README.md`、`docs/configuration.md`、`docs/code-generation.md`、`CHANGELOG.md`

- [ ] **Step 1: 写失败测试 — `orm/dialector/kingbase_test.go`**

  文件末尾追加：

  ```go
  func Test_GreatDBDriverRegistered(t *testing.T) {
  	if !isDriverRegistered("greatdb") {
  		t.Error("greatdb driver should be registered by init")
  	}
  }

  func Test_GreatDBFormatPrepareSQL(t *testing.T) {
  	d := NewGreatDBDialector(&testConfig{dbType: GreatDBDb})
  	src := "select * from t where a = ? and b = ?"
  	got := d.FormatPrepareSQL(src)
  	if got != src {
  		t.Errorf("greatdb format prepare sql should keep ?, got: %q", got)
  	}
  }

  func Test_GreatDBNeedsReturning(t *testing.T) {
  	d := NewGreatDBDialector(&testConfig{dbType: GreatDBDb})
  	if d.NeedsReturning() {
  		t.Error("greatdb should not need RETURNING")
  	}
  }

  func Test_GreatDBDialectorName(t *testing.T) {
  	d := NewGreatDBDialector(&testConfig{dbType: GreatDBDb})
  	if d.Name() != "greatdb" {
  		t.Errorf("greatdb dialector name failed, got: %q", d.Name())
  	}
  }
  ```

  既有表驱动测试追加条目：

  1. `Test_ParseDatabaseType` 的 `tests` 切片，Task 1 追加的 `{"golden-db", GoldenDBDb},` 之后：

  ```go
  		{"greatdb", GreatDBDb},
  		{"great-db", GreatDBDb},
  ```

  2. `Test_DatabaseTypeFamily` 末尾：

  ```go
  	if GreatDBDb.Family() != FamilyMySQL {
  		t.Errorf("GreatDBDb.Family() = %q, want %q", GreatDBDb.Family(), FamilyMySQL)
  	}
  ```

  3. `Test_GetDriverName` 的 `tests` 切片，`{GoldenDBDb, "goldendb"},` 之后：

  ```go
  		{GreatDBDb, "greatdb"},
  ```

  4. `Test_EffectiveSchema` 的 `tests` 切片末尾：

  ```go
  		{ConnectParams{DBName: "mydb", Type: GreatDBDb}, "mydb"},
  ```

  5. `Test_GenerateDSN` 函数体末尾：

  ```go
  	greatdbParams := ConnectParams{Host: "10.0.0.1", Port: 3306, Username: "root", Password: "123456", DBName: "testdb", Type: GreatDBDb}
  	if got := GenerateDSN(greatdbParams); got != "root:123456@tcp(10.0.0.1:3306)/testdb?parseTime=true&loc=Local" {
  		t.Errorf("GenerateDSN greatdb = %q", got)
  	}
  ```

  6. `Test_NewForType` 中 unknown 断言之前：

  ```go
  	d20, err := NewForType(GreatDBDb, &testConfig{dbType: GreatDBDb})
  	if err != nil || d20.Name() != "greatdb" {
  		t.Errorf("NewForType greatdb failed: %v, name=%q", err, d20.Name())
  	}
  ```

- [ ] **Step 2: 写失败测试 — `orm/database_config_test.go`**

  1. `Test_parseDatabaseType` 末尾（Task 1 的 r24 块之后）：

  ```go
  	r25, err := dialector.ParseDatabaseType("greatdb")
  	if r25 != GreatDBDb || err != nil {
  		t.Error("test parseDatabaseType greatdb failed.")
  	}
  ```

  2. `Test_parseAddr` 中 Task 1 的 goldendb 断言块之后：

  ```go
  	mp["spring.datasource.url"] = "jdbc:greatdb://10.1.2.3:3306/testdb"
  	tp21, host21, port21, db21, err := parseAddr(mp)
  	if tp21 != "greatdb" || host21 != "10.1.2.3" || port21 != 3306 || db21 != "testdb" || err != nil {
  		t.Error("test parseAddr greatdb failed.")
  	}
  ```

- [ ] **Step 3: 运行测试确认失败**

  ```bash
  go test ./orm/dialector/ -run "GreatDB" -count=1
  ```

  预期：FAIL，`undefined: GreatDBDb` / `undefined: NewGreatDBDialector`。

- [ ] **Step 4: 实现 — `orm/dialector/types.go`（5 处修改）**

  1. 常量块 `GoldenDBDb` 行之后追加：

  ```go
  	GreatDBDb        DatabaseType = "greatdb"
  ```

  2. `Family()` 中 MySQL case 追加成员：

  ```go
  	case MySqlDb, TiDBDb, TDSQLDb, PolarDBMyDb, OceanBaseDb, GoldenDBDb, GreatDBDb:
  ```

  3. `ParseDatabaseType` 中 `case "goldendb", "golden-db":` 块之后追加：

  ```go
  	case "greatdb", "great-db":
  		return GreatDBDb, nil
  ```

  4. `GetDriverName` 中 `case GoldenDBDb:` 块之后追加：

  ```go
  	case GreatDBDb:
  		return "greatdb"
  ```

  5. `NewForType` 中 `case GoldenDBDb:` 块之后追加：

  ```go
  	case GreatDBDb:
  		return NewGreatDBDialector(cfg), nil
  ```

- [ ] **Step 5: 实现 — 新建 `orm/dialector/greatdb.go`**

  ```go
  package dialector

  import (
  	"database/sql"

  	"github.com/go-sql-driver/mysql"
  )

  func init() {
  	for _, name := range []string{"greatdb"} {
  		if isDriverRegistered(name) {
  			continue
  		}
  		sql.Register(name, &mysql.MySQLDriver{})
  	}
  }

  type GreatDBDialector struct {
  	MySqlDialector
  }

  func NewGreatDBDialector(cfg ConfigProvider) *GreatDBDialector {
  	params := cfg.GetConnectParams()
  	dsn := cfg.GetDSN()
  	if dsn == "" {
  		dsn = generateMySQLDSN(params)
  	}
  	return &GreatDBDialector{
  		MySqlDialector: MySqlDialector{
  			BaseDialector: BaseDialector{
  				params:           params,
  				name:             "greatdb",
  				driverName:       GetDriverName(params.Type),
  				dsn:              dsn,
  				conn:             cfg.GetConnPool(),
  				family:           FamilyMySQL,
  				placeholderStyle: PlaceholderQuestion,
  				defaultMaxIdle:   DefaultMaxIdle,
  				maxTimeout:       cfg.GetMaxTimeout(),
  				maxOpen:          cfg.GetMaxOpen(),
  			},
  		},
  	}
  }
  ```

- [ ] **Step 6: 实现 — `orm/interfaces.go`**

  `GoldenDBDb` 行之后追加：

  ```go
  	GreatDBDb        = dialector.GreatDBDb
  ```

- [ ] **Step 7: 运行测试确认通过**

  ```bash
  go test ./orm/dialector/ -run "GreatDB|ParseDatabaseType|DatabaseTypeFamily|GetDriverName|EffectiveSchema|GenerateDSN|NewForType" -count=1
  go test ./orm/ -run "Test_parseDatabaseType|Test_parseAddr" -count=1
  ```

  预期：全部 PASS。

- [ ] **Step 8: 创建 demo — `cmd/greatdbdemo/`（3 个文件）**

  `cmd/greatdbdemo/main.go`：与 Task 1 Step 8 的 `cmd/goldendbdemo/main.go` 完全同构，仅 3 处不同——
  - `log.ConfigLocalFileSystemLogger("logs", "greatdbdemo")`
  - `orm.Initialize("cmd/greatdbdemo/application-greatdb.properties")`

  `cmd/greatdbdemo/application-greatdb.properties`：

  ```properties
  # GreatDB 万里安全数据库（MySQL 兼容，默认端口 3306）
  spring.datasource.url= jdbc:greatdb://localhost:3306/testdb?useUnicode=true&characterEncoding=utf-8&useSSL=false
  spring.datasource.username= root
  spring.datasource.password= 123456
  # 连接池设置（可选，省略则使用框架默认值）
  spring.datasource.max-idle= 100
  spring.datasource.max-open= 100
  spring.datasource.max-timeout= 100
  mybatis.mapper-locations= resources/mapper
  ```

  `cmd/greatdbdemo/README.md`：

  ```markdown
  # GreatDB Demo（万里安全数据库）

  mybatis-go 连接 GreatDB（万里安全数据库）的使用示例。MySQL 兼容，默认端口 3306。

  ## 配置

  本目录的 [`application-greatdb.properties`](./application-greatdb.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

  ## 驱动

  无需引入——框架自动以 `greatdb` 名称注册 `go-sql-driver/mysql` 驱动。

  ## 运行

  在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

  ```bash
  go run ./cmd/greatdbdemo
  ```
  ```

- [ ] **Step 9: 更新文档（5 个文件）**

  1. `cmd/schema2code/main.go:23` usage 字符串 `/goldendb` 后追加 `/greatdb`；
  2. `docs/code-generation.md:54` `/goldendb` 后追加 `/greatdb`；
  3. `README.md`：
     - 「多数据库」特性行：`GoldenDB（中兴）、` 之后插入 `GreatDB（万里）、`；
     - P0/P1/P2 清单：`- [x] GoldenDB（中兴）` 条目（P0 段）之后追加 `- [x] GreatDB（万里安全数据库）— MySQL 兼容，内嵌 MySqlDialector（MySQL 兼容族），cmd/greatdbdemo 示例`（注意：GoldenDB 条目位于「P0 — MySQL 兼容族」段，勿放入其他段落）；
     - 驱动引入表：GoldenDB 行之后追加 `| GreatDB（万里） | 无需引入（框架自动以 greatdb 名称注册 go-sql-driver/mysql 驱动） |`；
     - 「完整示例见」行末尾句号前追加 ``、`cmd/greatdbdemo/main.go` ``；
     - 运行示例命令区：goldendb 行之后追加 `go run ./cmd/greatdbdemo    # GreatDB（万里安全数据库，MySQL 兼容族）`；
     - 项目结构：`│   ├── goldendbdemo/` 之后追加 `│   ├── greatdbdemo/    # GreatDB（万里）使用示例`；`│   │   ├── goldendb.go` 之后追加 `│   │   ├── greatdb.go   # GreatDB（MySQL 兼容族）`；
  4. `docs/configuration.md`：GoldenDB 小节之后新增小节（结构同 Task 1 Step 9 第 4 点）：

  ````markdown
  ### GreatDB（万里安全数据库）

  ```properties
  # GreatDB 万里安全数据库（MySQL 兼容，默认端口 3306）
  spring.datasource.url= jdbc:greatdb://localhost:3306/testdb?useUnicode=true&characterEncoding=utf-8&useSSL=false
  spring.datasource.username= root
  spring.datasource.password= 123456
  mybatis.mapper-locations= resources/mapper
  ```

  GreatDB（万里安全数据库）为 MySQL 兼容族：占位符保持 `?`（`PlaceholderQuestion`）；`NeedsReturning` 返回 false；`EffectiveSchema` 返回数据库名。框架自动以 `greatdb` 名称注册 `go-sql-driver/mysql` 驱动，无需额外引入。
  ````

  「编程式配置」中 goldendb 行之后追加：

  ```go
  // 或 orm.InitializeDatabase("greatdb", "localhost", 3306, "root", "123456", "testdb")
  ```

  5. `CHANGELOG.md` 顶部插入 v0.3.15 条目（格式与 Task 1 的 v0.3.14 完全一致，替换为 GreatDB 相关内容）：

  ```markdown
  - **v0.3.15（信创数据库 GreatDB 适配，2026-10-12）**：万里安全数据库 GreatDB — MySQL 协议兼容，内嵌 `MySqlDialector`（MySQL 兼容族），零成本适配 — 新增 `DatabaseType` 常量 `GreatDBDb`（`Family()` 映射 `FamilyMySQL`）；`ParseDatabaseType` 支持 `greatdb`/`great-db` 别名；`GetDriverName` 返回 `greatdb`，框架 init 自动以 `greatdb` 名称注册 `go-sql-driver/mysql` 驱动别名（无需新增依赖）；`NewForType` 支持 `GreatDBDb`；JDBC URL 支持 `jdbc:greatdb://` 格式；新增 `cmd/greatdbdemo` 示例 + `application-greatdb.properties` 配置模板；schema2code `-type greatdb`；dialector + orm 单测覆盖
  ```

- [ ] **Step 10: 编译 + 全量测试**

  ```bash
  go build ./...
  go vet ./...
  go test -count=1 ./...
  ```

  预期：全部 PASS，零回归。

- [ ] **Step 11: Commit**

  ```bash
  git add orm/dialector/types.go orm/dialector/greatdb.go orm/dialector/kingbase_test.go \
          orm/interfaces.go orm/database_config_test.go \
          cmd/greatdbdemo/ cmd/schema2code/main.go README.md docs/configuration.md docs/code-generation.md CHANGELOG.md
  git commit -m "feat: 万里 GreatDB 数据库适配 — greatdb (MySQL 兼容族)

  - 新增 DatabaseType 常量: GreatDBDb，Family() 映射 FamilyMySQL
  - ParseDatabaseType 支持 greatdb/great-db
  - GetDriverName 返回 greatdb，框架 init 自动注册 go-sql-driver/mysql 别名
  - 新增 orm/dialector/greatdb.go（内嵌 MySqlDialector）
  - NewForType 支持 GreatDBDb
  - JDBC URL 解析支持 jdbc:greatdb:// 格式
  - 新增 cmd/greatdbdemo 示例 + application-greatdb.properties 配置模板 + demo README
  - schema2code -type 支持 greatdb
  - dialector + orm 单测覆盖
  - README/docs/CHANGELOG 更新"
  ```

---

### Task 3: 亚信 AntDB 适配（PostgreSQL 兼容族）

**Files:**
- Modify: `orm/dialector/types.go`
- Create: `orm/dialector/antdb.go`
- Modify: `orm/dialector/kingbase_test.go`
- Modify: `orm/interfaces.go`
- Modify: `orm/database_config_test.go`
- Create: `cmd/antdbdemo/main.go`、`cmd/antdbdemo/application-antdb.properties`、`cmd/antdbdemo/README.md`
- Modify: `cmd/schema2code/main.go`、`README.md`、`docs/configuration.md`、`docs/code-generation.md`、`CHANGELOG.md`

- [ ] **Step 1: 写失败测试 — `orm/dialector/kingbase_test.go`**

  文件末尾追加：

  ```go
  func Test_AntDBDriverRegistered(t *testing.T) {
  	if !isDriverRegistered("antdb") {
  		t.Error("antdb driver should be registered by init")
  	}
  }

  func Test_AntDBFormatPrepareSQL(t *testing.T) {
  	d := NewAntDBDialector(&testConfig{dbType: AntDBDb})
  	src := "select * from t where a = ? and b = ?"
  	got := d.FormatPrepareSQL(src)
  	want := "select * from t where a = $1 and b = $2"
  	if got != want {
  		t.Errorf("antdb format prepare sql failed, got: %q want: %q", got, want)
  	}
  }

  func Test_AntDBNeedsReturning(t *testing.T) {
  	d := NewAntDBDialector(&testConfig{dbType: AntDBDb})
  	if !d.NeedsReturning() {
  		t.Error("antdb should need RETURNING")
  	}
  }

  func Test_AntDBDialectorName(t *testing.T) {
  	d := NewAntDBDialector(&testConfig{dbType: AntDBDb})
  	if d.Name() != "antdb" {
  		t.Errorf("antdb dialector name failed, got: %q", d.Name())
  	}
  }
  ```

  既有表驱动测试追加条目：

  1. `Test_ParseDatabaseType` 的 `tests` 切片，`{"great-db", GreatDBDb},` 之后：

  ```go
  		{"antdb", AntDBDb},
  		{"ant-db", AntDBDb},
  ```

  2. `Test_DatabaseTypeFamily` 末尾：

  ```go
  	if AntDBDb.Family() != FamilyPostgres {
  		t.Errorf("AntDBDb.Family() = %q, want %q", AntDBDb.Family(), FamilyPostgres)
  	}
  ```

  3. `Test_GetDriverName` 的 `tests` 切片，`{GreatDBDb, "greatdb"},` 之后：

  ```go
  		{AntDBDb, "antdb"},
  ```

  4. `Test_EffectiveSchema` 的 `tests` 切片末尾：

  ```go
  		{ConnectParams{DBName: "mydb", Type: AntDBDb}, "public"},
  ```

  5. `Test_GenerateDSN` 函数体末尾：

  ```go
  	antdbParams := ConnectParams{Host: "10.0.0.1", Port: 6600, Username: "root", Password: "123456", DBName: "testdb", Type: AntDBDb}
  	if got := GenerateDSN(antdbParams); got != "host=10.0.0.1 port=6600 user=root password=123456 dbname=testdb sslmode=disable" {
  		t.Errorf("GenerateDSN antdb = %q", got)
  	}
  ```

  6. `Test_NewForType` 中 unknown 断言之前：

  ```go
  	d21, err := NewForType(AntDBDb, &testConfig{dbType: AntDBDb})
  	if err != nil || d21.Name() != "antdb" {
  		t.Errorf("NewForType antdb failed: %v, name=%q", err, d21.Name())
  	}
  ```

- [ ] **Step 2: 写失败测试 — `orm/database_config_test.go`**

  1. `Test_parseDatabaseType` 末尾（r25 块之后）：

  ```go
  	r26, err := dialector.ParseDatabaseType("antdb")
  	if r26 != AntDBDb || err != nil {
  		t.Error("test parseDatabaseType antdb failed.")
  	}
  ```

  2. `Test_parseAddr` 中 greatdb 断言块之后：

  ```go
  	mp["spring.datasource.url"] = "jdbc:antdb://10.1.2.3:6600/testdb"
  	tp22, host22, port22, db22, err := parseAddr(mp)
  	if tp22 != "antdb" || host22 != "10.1.2.3" || port22 != 6600 || db22 != "testdb" || err != nil {
  		t.Error("test parseAddr antdb failed.")
  	}
  ```

- [ ] **Step 3: 运行测试确认失败**

  ```bash
  go test ./orm/dialector/ -run "AntDB" -count=1
  ```

  预期：FAIL，`undefined: AntDBDb` / `undefined: NewAntDBDialector`。

- [ ] **Step 4: 实现 — `orm/dialector/types.go`（5 处修改）**

  1. 常量块 `GreatDBDb` 行之后追加：

  ```go
  	AntDBDb          DatabaseType = "antdb"
  ```

  2. `Family()` 中 PG case `case PostgresDb, KingbaseDb, OpenGaussDb, GaussDBDb, HighGoDb, VastbaseDb:` 追加成员，改为：

  ```go
  	case PostgresDb, KingbaseDb, OpenGaussDb, GaussDBDb, HighGoDb, VastbaseDb, AntDBDb:
  ```

  3. `ParseDatabaseType` 中 `case "greatdb", "great-db":` 块之后追加：

  ```go
  	case "antdb", "ant-db":
  		return AntDBDb, nil
  ```

  4. `GetDriverName` 中 `case GreatDBDb:` 块之后追加：

  ```go
  	case AntDBDb:
  		return "antdb"
  ```

  5. `NewForType` 中 `case GreatDBDb:` 块之后追加：

  ```go
  	case AntDBDb:
  		return NewAntDBDialector(cfg), nil
  ```

- [ ] **Step 5: 实现 — 新建 `orm/dialector/antdb.go`**

  ```go
  package dialector

  import (
  	"database/sql"

  	"github.com/lib/pq"
  )

  func init() {
  	for _, name := range []string{"antdb"} {
  		if isDriverRegistered(name) {
  			continue
  		}
  		sql.Register(name, &pq.Driver{})
  	}
  }

  type AntDBDialector struct {
  	PostgresDialector
  }

  func NewAntDBDialector(cfg ConfigProvider) *AntDBDialector {
  	params := cfg.GetConnectParams()
  	dsn := cfg.GetDSN()
  	if dsn == "" {
  		dsn = generatePostgresDSN(params)
  	}
  	return &AntDBDialector{
  		PostgresDialector: PostgresDialector{
  			BaseDialector: BaseDialector{
  				params:           params,
  				name:             "antdb",
  				driverName:       GetDriverName(params.Type),
  				dsn:              dsn,
  				conn:             cfg.GetConnPool(),
  				family:           FamilyPostgres,
  				placeholderStyle: PlaceholderDollar,
  				defaultMaxIdle:   DefaultMaxIdle,
  				maxTimeout:       cfg.GetMaxTimeout(),
  				maxOpen:          cfg.GetMaxOpen(),
  			},
  		},
  	}
  }
  ```

- [ ] **Step 6: 实现 — `orm/interfaces.go`**

  `GreatDBDb` 行之后追加：

  ```go
  	AntDBDb          = dialector.AntDBDb
  ```

- [ ] **Step 7: 运行测试确认通过**

  ```bash
  go test ./orm/dialector/ -run "AntDB|ParseDatabaseType|DatabaseTypeFamily|GetDriverName|EffectiveSchema|GenerateDSN|NewForType" -count=1
  go test ./orm/ -run "Test_parseDatabaseType|Test_parseAddr" -count=1
  ```

  预期：全部 PASS。

- [ ] **Step 8: 创建 demo — `cmd/antdbdemo/`（3 个文件）**

  `cmd/antdbdemo/main.go`：与 Task 1 Step 8 的 `cmd/goldendbdemo/main.go` 同构，差异——
  - blank import 改为 `_ "github.com/lib/pq"`
  - `log.ConfigLocalFileSystemLogger("logs", "antdbdemo")`
  - `orm.Initialize("cmd/antdbdemo/application-antdb.properties")`

  `cmd/antdbdemo/application-antdb.properties`：

  ```properties
  # AntDB（亚信，PostgreSQL 兼容，默认端口 6600）
  spring.datasource.url= jdbc:antdb://localhost:6600/testdb
  spring.datasource.username= root
  spring.datasource.password= 123456
  # 连接池设置（可选，省略则使用框架默认值）
  spring.datasource.max-idle= 100
  spring.datasource.max-open= 100
  spring.datasource.max-timeout= 100
  mybatis.mapper-locations= resources/mapper
  ```

  `cmd/antdbdemo/README.md`：

  ```markdown
  # AntDB Demo（亚信）

  mybatis-go 连接 AntDB（亚信）的使用示例。PostgreSQL 兼容，默认端口 6600。

  ## 配置

  本目录的 [`application-antdb.properties`](./application-antdb.properties) 为连接配置模板，请将其中的地址 / 账号 / 密码改为你的环境。

  ## 驱动

  无需引入——框架自动以 `antdb` 名称注册 `lib/pq` 驱动。

  ## 运行

  在**仓库根目录**执行（示例依赖根目录的 `resources/mapper` 与本目录配置文件）：

  ```bash
  go run ./cmd/antdbdemo
  ```
  ```

- [ ] **Step 9: 更新文档（5 个文件）**

  1. `cmd/schema2code/main.go:23` usage 字符串 `/greatdb` 后追加 `/antdb`；
  2. `docs/code-generation.md:54` `/greatdb` 后追加 `/antdb`；
  3. `README.md`：
     - 「多数据库」特性行：`GreatDB（万里）、` 之后插入 `AntDB（亚信）、`；
     - P0/P1/P2 清单：`- [x] Vastbase（海量数据）` 条目（「P1 — PostgreSQL 兼容族」段）之后追加 `- [x] AntDB（亚信）— PostgreSQL 兼容，内嵌 PostgresDialector（PG 兼容族），cmd/antdbdemo 示例`（注意归入 P1 段，勿放其他段落）；
     - 驱动引入表：GreatDB 行之后追加 `| AntDB（亚信） | 无需引入（框架自动以 antdb 名称注册 lib/pq 驱动） |`；
     - 「完整示例见」行末尾句号前追加 ``、`cmd/antdbdemo/main.go` ``；
     - 运行示例命令区：greatdb 行之后追加 `go run ./cmd/antdbdemo      # AntDB（亚信，PG 兼容族）`；
     - 项目结构：`│   ├── greatdbdemo/` 之后追加 `│   ├── antdbdemo/      # AntDB（亚信）使用示例`；`│   │   ├── greatdb.go` 之后追加 `│   │   ├── antdb.go     # AntDB（PG 兼容族）`；
  4. `docs/configuration.md`：GreatDB 小节之后新增小节：

  ````markdown
  ### AntDB（亚信）

  ```properties
  # AntDB（亚信，PostgreSQL 兼容，默认端口 6600）
  spring.datasource.url= jdbc:antdb://localhost:6600/testdb
  spring.datasource.username= root
  spring.datasource.password= 123456
  mybatis.mapper-locations= resources/mapper
  ```

  AntDB（亚信）为 PostgreSQL 兼容族：占位符 `?` → `$n`（`PlaceholderDollar`）；`NeedsReturning` 返回 true（自增主键回填走 `RETURNING`）；`EffectiveSchema` 默认 `public`。框架自动以 `antdb` 名称注册 `lib/pq` 驱动，无需额外引入。
  ````

  「编程式配置」中 greatdb 行之后追加：

  ```go
  // 或 orm.InitializeDatabase("antdb", "localhost", 6600, "root", "123456", "testdb")
  ```

  5. `CHANGELOG.md` 顶部插入 v0.3.16 条目：

  ```markdown
  - **v0.3.16（信创数据库 AntDB 适配，2026-10-12）**：亚信 AntDB — PostgreSQL 协议兼容，内嵌 `PostgresDialector`（PG 兼容族） — 新增 `DatabaseType` 常量 `AntDBDb`（`Family()` 映射 `FamilyPostgres`）；`ParseDatabaseType` 支持 `antdb`/`ant-db` 别名；`GetDriverName` 返回 `antdb`，框架 init 自动以 `antdb` 名称注册 `lib/pq` 驱动别名（无需新增依赖）；`NewForType` 支持 `AntDBDb`；JDBC URL 支持 `jdbc:antdb://` 格式；新增 `cmd/antdbdemo` 示例 + `application-antdb.properties` 配置模板；schema2code `-type antdb`；dialector + orm 单测覆盖
  ```

- [ ] **Step 10: 编译 + 全量测试**

  ```bash
  go build ./...
  go vet ./...
  go test -count=1 ./...
  ```

  预期：全部 PASS，零回归。

- [ ] **Step 11: Commit**

  ```bash
  git add orm/dialector/types.go orm/dialector/antdb.go orm/dialector/kingbase_test.go \
          orm/interfaces.go orm/database_config_test.go \
          cmd/antdbdemo/ cmd/schema2code/main.go README.md docs/configuration.md docs/code-generation.md CHANGELOG.md
  git commit -m "feat: 亚信 AntDB 数据库适配 — antdb (PostgreSQL 兼容族)

  - 新增 DatabaseType 常量: AntDBDb，Family() 映射 FamilyPostgres
  - ParseDatabaseType 支持 antdb/ant-db
  - GetDriverName 返回 antdb，框架 init 自动注册 lib/pq 别名
  - 新增 orm/dialector/antdb.go（内嵌 PostgresDialector）
  - NewForType 支持 AntDBDb
  - JDBC URL 解析支持 jdbc:antdb:// 格式
  - 新增 cmd/antdbdemo 示例 + application-antdb.properties 配置模板 + demo README
  - schema2code -type 支持 antdb
  - dialector + orm 单测覆盖
  - README/docs/CHANGELOG 更新"
  ```

---

### Task 4: 全量回归验证

**Files:** 无新增改动（只读验证）。

- [ ] **Step 1: 全量编译与测试**

  ```bash
  go build ./...
  go vet ./...
  go test -v -count=1 ./...
  ```

  预期：全部 PASS。重点确认：
  - `orm/dialector` 包：新增 GoldenDB/GreatDB/AntDB 测试全绿；
  - `orm` 包：`Test_parseAddr` / `Test_parseDatabaseType` 含 3 组新断言通过；
  - SQLite 端到端测试（`Test_Sqlite*`）零回归。

- [ ] **Step 2: demo 编译冒烟**

  ```bash
  go build -o D:\Develop\bin\goldendbdemo.exe ./cmd/goldendbdemo
  go build -o D:\Develop\bin\greatdbdemo.exe ./cmd/greatdbdemo
  go build -o D:\Develop\bin\antdbdemo.exe ./cmd/antdbdemo
  ```

  预期：三个 demo 编译成功（无数据库环境，不要求运行连通）。

- [ ] **Step 3: 汇报**

  向用户汇报：3 款适配完成情况、测试结果、Oscar 调研结论（Task 0 产出），并询问是否 push。

---

## Self-Review 结论

- **覆盖检查**：4 款数据库各有对应任务（Task 1-3 实现 + Task 0 Oscar 调研）；add-dialector.md 步骤 1-8 全部映射到各 Task 的 Step 1-11。
- **占位符扫描**：Oscar 的实现细节依赖 Task 0 调研结论，故本计划将 Oscar 明确限定为调研任务（产出决策文档），实现计划在调研完成后另立——非占位符，是明确的范围边界。
- **类型一致性**：`GoldenDBDb`/`GreatDBDb`/`AntDBDb`、`NewGoldenDBDialector`/`NewGreatDBDialector`/`NewAntDBDialector`、测试变量编号（r24-26 / tp20-22 / d19-21）在定义与引用处一致；`isDriverRegistered`（base.go）、`generateMySQLDSN`/`generatePostgresDSN`（types.go）均已存在。
- **风险提示**：① AntDB 个别部署默认端口为 5432，6600 仅为常见默认值，properties 模板中已注明可改；② 三款数据库均无法本地端到端验证（无实例），验证止步于编译 + 单测 + 驱动注册，真机连通由用户环境确认。
