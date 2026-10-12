# 安可测评数据库零成本别名适配实施方案（7 款）

> 计划日期：2026-10-12
> 依据：`docs/agents/anke-database-matrix.md`（commit `3e58d79`）——安可测评 19 款在册产品中 7 款可零成本别名适配
> 模式：与 `docs/agents/plans/2026-10-12-xinchuang-databases.md`（GoldenDB/GreatDB/AntDB）完全相同的零成本别名适配——新 DatabaseType 常量 + 内嵌既有 Dialector + `lib/pq` / `go-sql-driver/mysql` 驱动别名注册，**go.mod 零改动、零新增依赖**
> 执行模式：Subagent-Driven——每 Task 派发实施子代理 → 规格审查 → 质量审查

## 0. 决策摘要

| 决策 | 结论 |
|------|------|
| 适配范围 | 安可矩阵「零成本」类 7 款：MySQL 兼容族 3 款 + PG 兼容族 4 款 |
| 版本拆分 | Task 1 MySQL 族 3 款 → **v0.3.17**（单 commit）；Task 2 PG 族 4 款 → **v0.3.18**（单 commit）；Task 3 回归 + 矩阵更新（docs commit） |
| 命名 | 常量 `TaurusDBDb`/`UPDRDBDb`/`KingwowDb`/`TeleDBDb`/`DWSDb`/`ADBPGDb`/`SeaboxSQLDb`；构造函数 `New<X>Dialector` |
| 测试文件策略 | 每款专属测试函数入**新文件** `orm/dialector/mysql_family_test.go`（Task 1）/ `pg_family_test.go`（Task 2），不再塞进 kingbase_test.go（回应质量审查遗留建议）；表驱动条目（ParseDatabaseType/Family/GetDriverName/GenerateDSN/NewForType）仍追加在 kingbase_test.go 既有测试函数内 |
| parseAddr | `orm/database_config.go` **零改动**——通用正则 `jdbc:([\w-]+)://` 已覆盖全部 7 个新 scheme |
| README 完整示例句 | README.md:208 已列举 21 个 demo，再加 7 个不可持续——**改写为通用表述**（见 §2.4，有意偏离惯例） |
| 测试变量编号 | MySQL 族 r27-29 / tp23-25 / d22-24；PG 族 r30-33 / tp26-29 / d25-28（接续已用的 r24-26 / tp20-22 / d19-21） |

## 1. 七款产品参数总表

### Task 1 — MySQL 兼容族（内嵌 `MySqlDialector`，驱动 `go-sql-driver/mysql`，默认端口 3306）

| 产品 | 厂商 | 常量（值） | ParseDatabaseType 别名 | demo 目录 | 置信度 |
|------|------|-----------|------------------------|-----------|--------|
| TaurusDB | 华为云 | `TaurusDBDb`（`taurusdb`） | `taurusdb`, `taurus-db` | `cmd/taurusdbdemo` | 高（官方 100% MySQL 兼容） |
| UPDRDB | 中国银联 | `UPDRDBDb`（`updrdb`） | `updrdb`, `updr-db` | `cmd/updrdbdemo` | 高（官方 MySQL 协议确认） |
| Kingwow 金乌 | 北京自然原数 | `KingwowDb`（`kingwow`） | `kingwow` | `cmd/kingwowdemo` | **低**（社区来源，待真机验证，文档须标注） |

### Task 2 — PG 兼容族（内嵌 `PostgresDialector`，驱动 `lib/pq`）

| 产品 | 厂商 | 常量（值） | ParseDatabaseType 别名 | demo properties 端口 | 置信度 / 注记 |
|------|------|-----------|------------------------|----------------------|---------------|
| TeleDB | 天翼云 | `TeleDBDb`（`teledb`） | `teledb`, `tele-db` | 15432（官方未公布默认端口，示例值） | 高（官方 PG 协议确认） |
| DWS | 华为云 | `DWSDb`（`dws`） | `dws`, `gaussdb-dws` | 8000 | 高；**须 MD5 认证注记**（lib/pq 不支持 DWS SHA256 认证，需配置 MD5 账户） |
| ADB-PG | 阿里云 | `ADBPGDb`（`adbpg`） | `adbpg`, `adb-pg` | 5432 | 高；OLAP 场景注记 |
| SeaboxSQL | 东方金信 | `SeaboxSQLDb`（`seaboxsql`） | `seaboxsql`, `seabox` | 5432（推断） | 中（PG 血统，**待真机验证**，文档须标注） |

## 2. Task 1 — MySQL 兼容族 3 款（v0.3.17）

### 2.1 框架代码

**`orm/interfaces.go`** — 常量别名块（`AntDBDb` 行后）追加：

```go
	TaurusDBDb      = dialector.TaurusDBDb
	UPDRDBDb        = dialector.UPDRDBDb
	KingwowDb       = dialector.KingwowDb
```

**`orm/dialector/types.go`** — 5 处：

1. `DatabaseType` 常量块（`AntDBDb` 行 58 后）：
   ```go
   TaurusDBDb       DatabaseType = "taurusdb"
   UPDRDBDb         DatabaseType = "updrdb"
   KingwowDb        DatabaseType = "kingwow"
   ```
2. `Family()`：`case MySqlDb, TiDBDb, TDSQLDb, PolarDBMyDb, OceanBaseDb, GoldenDBDb, GreatDBDb:` 追加 `, TaurusDBDb, UPDRDBDb, KingwowDb`
3. `ParseDatabaseType()`（`antdb` case 后）：
   ```go
   case "taurusdb", "taurus-db":
       return TaurusDBDb, nil
   case "updrdb", "updr-db":
       return UPDRDBDb, nil
   case "kingwow":
       return KingwowDb, nil
   ```
4. `GetDriverName()`（`AntDBDb` case 后）：
   ```go
   case TaurusDBDb:
       return "taurusdb"
   case UPDRDBDb:
       return "updrdb"
   case KingwowDb:
       return "kingwow"
   ```
5. `NewForType()`（`AntDBDb` case 后）：
   ```go
   case TaurusDBDb:
       return NewTaurusDBDialector(cfg), nil
   case UPDRDBDb:
       return NewUPDRDBDialector(cfg), nil
   case KingwowDb:
       return NewKingwowDialector(cfg), nil
   ```

**新文件 `orm/dialector/taurusdb.go`**（模板 = `goldendb.go`，全文如下；`updrdb.go`/`kingwow.go` 同构替换标识符）：

```go
package dialector

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
)

func init() {
	for _, name := range []string{"taurusdb"} {
		if isDriverRegistered(name) {
			continue
		}
		sql.Register(name, &mysql.MySQLDriver{})
	}
}

type TaurusDBDialector struct {
	MySqlDialector
}

func NewTaurusDBDialector(cfg ConfigProvider) *TaurusDBDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generateMySQLDSN(params)
	}
	return &TaurusDBDialector{
		MySqlDialector: MySqlDialector{
			BaseDialector: BaseDialector{
				params:           params,
				name:             "taurusdb",
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

替换对照：`taurusdb` → `updrdb`（`UPDRDBDialector`/`NewUPDRDBDialector`）、`kingwow`（`KingwowDialector`/`NewKingwowDialector`）。

### 2.2 测试

**新文件 `orm/dialector/mysql_family_test.go`** — 每款 4 个函数（以 TaurusDB 为例，UPDRDB/Kingwow 同构替换）：

```go
func Test_TaurusDBDriverRegistered(t *testing.T) {
	if !isDriverRegistered("taurusdb") {
		t.Error("taurusdb driver should be registered by init")
	}
}

func Test_TaurusDBFormatPrepareSQL(t *testing.T) {
	d := NewTaurusDBDialector(&testConfig{dbType: TaurusDBDb})
	src := "select * from t where a = ? and b = ?"
	if got := d.FormatPrepareSQL(src); got != src {
		t.Errorf("taurusdb FormatPrepareSQL should keep ?, got %q", got)
	}
}

func Test_TaurusDBNeedsReturning(t *testing.T) {
	d := NewTaurusDBDialector(&testConfig{dbType: TaurusDBDb})
	if d.NeedsReturning() {
		t.Error("taurusdb should not need RETURNING")
	}
}

func Test_TaurusDBDialectorName(t *testing.T) {
	d := NewTaurusDBDialector(&testConfig{dbType: TaurusDBDb})
	if d.Name() != "taurusdb" {
		t.Errorf("taurusdb dialector name = %q", d.Name())
	}
}
```

**`orm/dialector/kingbase_test.go`** — 既有测试函数内追加（参照 goldendb/greatdb 条目格式）：

- `Test_ParseDatabaseType` 表：`{"taurusdb", TaurusDBDb}`、`{"taurus-db", TaurusDBDb}`、`{"updrdb", UPDRDBDb}`、`{"updr-db", UPDRDBDb}`、`{"kingwow", KingwowDb}`
- `Test_DatabaseTypeFamily` 表：`{TaurusDBDb, FamilyMySQL}`、`{UPDRDBDb, FamilyMySQL}`、`{KingwowDb, FamilyMySQL}`
- `Test_GetDriverName` 表：`{TaurusDBDb, "taurusdb"}`、`{UPDRDBDb, "updrdb"}`、`{KingwowDb, "kingwow"}`
- `Test_GenerateDSN`：每款一块（端口 3306，期望 `"root:123456@tcp(10.0.0.1:3306)/testdb?parseTime=true&loc=Local"`，同 goldendb 块）
- `Test_NewForType`：`d22`（`taurusdb`）、`d23`（`updrdb`）、`d24`（`kingwow`）

**`orm/database_config_test.go`** — 追加：

- `Test_parseDatabaseType`：`r27`（`taurusdb`→`TaurusDBDb`）、`r28`（`updrdb`）、`r29`（`kingwow`）
- `Test_parseAddr`：`tp23`（`jdbc:taurusdb://10.1.2.3:3306/testdb`）、`tp24`（`jdbc:updrdb://…`）、`tp25`（`jdbc:kingwow://…`）——验证通用正则零改动即生效

### 2.3 demo 三件套 ×3

每款复制 `cmd/goldendbdemo`（main.go + `application-<db>.properties` + README.md），全局替换 `goldendb`→`<db>`、`GoldenDB`→`<产品名>`：

- `cmd/taurusdbdemo/`：properties `type=taurusdb`、port 3306；README 标注「华为云 TaurusDB，MySQL 100% 兼容」
- `cmd/updrdbdemo/`：properties `type=updrdb`、port 3306；README 标注「中国银联 UPDRDB，MySQL 协议兼容」
- `cmd/kingwowdemo/`：properties `type=kingwow`、port 3306；README 标注「北京自然原数 Kingwow 金乌，MySQL 线协议兼容（低置信度，待真机验证）」

main.go 保留 `_ "github.com/go-sql-driver/mysql"` blank import（与 goldendb/greatdb demo 一致）。

### 2.4 文档

**`README.md`** 6 处：

1. 「多数据库」特性行（line 17）：`AntDB（亚信）、` 之后插入 `TaurusDB（华为云）、UPDRDB（中国银联）、Kingwow 金乌（自然原数）、`
2. 驱动表（lines 101-124）：AntDB 行后追加 3 行（产品 / 驱动 / 协议 / 备注，格式同既有行）
3. **完整示例句（line 208）改写**（有意偏离——原句已列 21 个 demo，不可持续增长）：
   > 完整示例见各数据库对应的 `cmd/<数据库>demo/` 目录（完整清单见下方「项目结构」与「运行示例」）。
4. 运行示例命令区（antdbdemo 行 298 后）：
   ```
   go run ./cmd/taurusdbdemo   # TaurusDB（华为云，MySQL 兼容族）
   go run ./cmd/updrdbdemo     # UPDRDB（中国银联，MySQL 兼容族）
   go run ./cmd/kingwowdemo    # Kingwow 金乌（自然原数，MySQL 兼容族）
   ```
5. 项目结构树 cmd/（lines 333-355）：`demo/` 行前插入 3 个 demo 目录行（保持 ├──/└── 对齐）
6. dialector 树（lines 362-374）：追加 `taurusdb.go`/`updrdb.go`/`kingwow.go`

**`docs/configuration.md`**：AntDB 小节（line 277-288）后、TiDB 小节前插入 3 个小节（properties 块 + 1-2 句说明；Kingwow 小节含低置信度标注）。

**`cmd/schema2code/main.go`** line 23 与 **`docs/code-generation.md`**：`-type` 列表追加 `/taurusdb/updrdb/kingwow`。

**`CHANGELOG.md`**：顶部插入 v0.3.17 条目（一段式，格式同 v0.3.15/0.3.16）。

### 2.5 验证与提交

```
go build ./...
go vet ./...
go test -count=1 ./orm/... ./types/... ./utils/...
gofmt -l orm cmd（输出须为空）
go build -o D:\Develop\bin\taurusdbdemo.exe ./cmd/taurusdbdemo（×3 编译冒烟）
```

commit message：

```
feat: 安可测评数据库适配（MySQL 兼容族 3 款）— TaurusDB/UPDRDB/Kingwow (v0.3.17)

- TaurusDB（华为云）：MySQL 100% 兼容，内嵌 MySqlDialector，零新增依赖
- UPDRDB（中国银联）：MySQL 协议兼容，内嵌 MySqlDialector
- Kingwow 金乌（自然原数）：MySQL 线协议兼容，内嵌 MySqlDialector（低置信度，待真机验证）
- 每款：DatabaseType 常量 + 别名解析 + Family/GetDriverName/NewForType + 驱动别名注册 + demo 三件套 + 单测
- 新增 orm/dialector/mysql_family_test.go；README 完整示例句改通用表述；schema2code -type 扩充
```

## 3. Task 2 — PG 兼容族 4 款（v0.3.18）

结构与 Task 1 完全同构，差异如下：

**框架代码**：

- `orm/interfaces.go` 追加 `TeleDBDb`/`DWSDb`/`ADBPGDb`/`SeaboxSQLDb`
- `orm/dialector/types.go`：常量 4 行；`Family()` PG case 追加 4 值；`ParseDatabaseType` 4 case（teledb/tele-db、dws/gaussdb-dws、adbpg/adb-pg、seaboxsql/seabox）；`GetDriverName` 4 case；`NewForType` 4 case
- 新文件 `teledb.go`/`dws.go`/`adbpg.go`/`seaboxsql.go`，模板 = `antdb.go`（`sql.Register(name, &pq.Driver{})`，内嵌 `PostgresDialector`，`PlaceholderDollar`，`generatePostgresDSN`）

**测试**：

- 新文件 `orm/dialector/pg_family_test.go`：每款 4 函数（DriverRegistered / FormatPrepareSQL——`?`→`$1` 转换断言 / NeedsReturning 为 true / DialectorName）
- `kingbase_test.go`：ParseDatabaseType 表 7 条（含别名）、Family 表 4 条、GetDriverName 表 4 条、GenerateDSN 4 块（期望 `host=10.0.0.1 port=<端口> user=root password=123456 dbname=testdb sslmode=disable`，端口分别 15432/8000/5432/5432）、NewForType `d25`-`d28`
- `orm/database_config_test.go`：`r30`-`r33`、`tp26`-`tp29`（`jdbc:teledb://…`、`jdbc:dws://…`、`jdbc:adbpg://…`、`jdbc:seaboxsql://…`）

**demo 三件套 ×4**（模板 = `cmd/antdbdemo`，blank import `_ "github.com/lib/pq"`）：

| demo | properties | README 关键标注 |
|------|-----------|----------------|
| `cmd/teledbdemo` | `type=teledb`，port 15432 | 天翼云 TeleDB，PG 协议兼容；端口为示例值（官方未公布默认端口） |
| `cmd/dwsdemo` | `type=dws`，port 8000 | 华为云 DWS，PG libpq 协议；**须使用 MD5 认证账户**（lib/pq 不支持 DWS SHA256） |
| `cmd/adbpgdemo` | `type=adbpg`，port 5432 | 阿里云 ADB-PG，PG 协议；OLAP 场景 |
| `cmd/seaboxsqldemo` | `type=seaboxsql`，port 5432 | 东方金信 SeaboxSQL，PG 血统（**待真机验证**） |

**文档**：README 同 6 处（特性行 Kingwow 后插 4 款；驱动表 4 行；line 208 已在 Task 1 改写无需再动；运行命令区 4 行；cmd 树 4 目录；dialector 树 4 文件）；configuration.md AntDB 后插 4 小节（Task 1 的 3 小节之后）；schema2code/code-generation.md 追加 `/teledb/dws/adbpg/seaboxsql`；CHANGELOG v0.3.18。

**验证与提交**：同 §2.5（4 个 demo 编译冒烟到 `D:\Develop\bin`）。commit message 同构（`feat: 安可测评数据库适配（PG 兼容族 4 款）— TeleDB/DWS/ADB-PG/SeaboxSQL (v0.3.18)`）。

## 4. Task 3 — 全量回归 + 安可矩阵更新

1. 全量回归：`go build ./... && go vet ./... && go test -v -count=1 ./...`（全绿）
2. 7 个 demo 全部编译冒烟（`-o D:\Develop\bin\<name>demo.exe`）
3. **`docs/agents/anke-database-matrix.md` 更新**：
   - 7 款由「零成本可支持」移入「已支持」（附 v0.3.17/v0.3.18 版本号与置信度注记）
   - **修正覆盖率统计错误**：原文「17/19（89%）」计算有误——本方案落地后为 **12/19 款（63%）**；若后续引入 openGauss 官方纯 Go 驱动（PanweiDB/GBase 8c/GaussDB sha256 场景）可达 14/19（74%）
   - 「后续行动」移除已完成项，保留 openGauss 驱动引入、真机验证清单（Kingwow/SeaboxSQL 置信度提升）
4. commit：`docs: 安可测评矩阵更新 — 7 款零成本别名落地（12/19 款已支持）`

## 5. 风险与边界

| 风险 | 应对 |
|------|------|
| Kingwow/SeaboxSQL 置信度低/中 | 代码路径与同族完全一致，风险仅在「产品是否真兼容 MySQL/PG 线协议」；文档与矩阵均标注待真机验证，不阻塞合入 |
| DWS SHA256 认证不兼容 | lib/pq 限制（与既有 opengauss/gaussdb 支持同边界）；demo/文档注明须 MD5 账户，不在本方案扩驱动 |
| TeleDB 默认端口未知 | demo properties 用示例端口 15432，用户按实际环境改 |
| 7 款全为内嵌别名，无真实连接验证 | 与 GoldenDB/GreatDB/AntDB 同模式——真机验证列入矩阵「后续行动」，不阻塞 |
| 一次性 7 款拆 2 个 commit | 每 commit 独立可编译可测试（族内 3+4 拆分），回归粒度可控 |

## 6. 有意偏离惯例记录

1. **README.md:208 完整示例句改通用表述**：原句穷举 21 个 demo main.go 路径，每新增一款都要追加，且已接近行长极限；改为指向「项目结构」与「运行示例」的通用句式。
2. **专属测试函数移入新文件**（`mysql_family_test.go`/`pg_family_test.go`）：kingbase_test.go 已是 1000+ 行多方言聚合文件且文件名与内容不符（质量审查遗留）；新方言测试不再追加至该文件，表驱动条目因既有表结构所在仍留在原处。
