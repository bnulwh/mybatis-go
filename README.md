# mybatis-go

Go 语言实现的 MyBatis 风格 ORM 框架。通过 XML Mapper 文件定义 SQL，利用反射为 struct 的函数字段注入代理实现，支持 **PostgreSQL、MySQL、SQLite 和人大金仓 KingbaseES**。

## 特性

- **MyBatis 风格**：XML Mapper 定义 SQL，`#{}` / `${}` 参数绑定，完整动态 SQL（`<if>` / `<where>` / `<set>` / `<trim>` / `<foreach>` / `<choose>` / `<include>`），动态 SQL 引擎独立为 `types/sqlfragment` 包（`Node` 接口 + 节点注册表，开放封闭扩展新节点）
- **反射代理**：Mapper struct 的函数字段在运行时自动注入代理，无需手动实现
- **结果自动映射**：查询结果自动映射到 Go struct，支持 `resultMap`（含 `<association>` / `<collection>` 嵌套关联类型生成）
- **自增主键回填**：`useGeneratedKeys` / `keyProperty` 支持，Insert 后自动回填自增主键到入参 struct 指针
- **MyBatis-Plus 内置 CRUD**：`schema2code -mp` 从表结构直接生成 BaseMapper 标准方法名（insert/deleteById/updateById/selectById/selectList/selectOne/selectPage/selectCount/selectBatchIds/deleteBatchIds）的 XML，原生加载、无需手写 GoExtraMapper；亦可在 **XML 含 resultMap 时加载期内存自动补生成**（无需落盘 CRUD XML），使用说明见 **docs/agents/mybatis-plus.md**
- **struct `db` tag 元数据**：`db:"user_name"` 显式列名、`db:"id,pk"` 主键、`db:"deleted,logic"` 逻辑删除列、`db:"version,version"` 乐观锁版本列、`db:"create_time,fill:insert"` 填充标记（解析入 `ModelInfo.FillColumns`，供 Hook 自动填充等横切逻辑取用）、`db:"-"` 排除映射；`TableName() string` 方法指定表名；`orm.RegisterModel` 注册后 MP 内置 CRUD 按元数据精确生成（表名/主键/逻辑列以 tag 为准，无需依赖表结构查询）
- **P1 框架增强七件套**：① 乐观锁（`version,version` tag 驱动 MP `updateById` 自动 `version+1` + CAS 检测，失败返回 `orm.ErrOptimisticLock`）② 自动填充（`orm.RegisterFillHandler` 按 `fill:insert|insert_update` 标记在 insert/update 前注入审计字段）③ 二级缓存（`mybatis.configuration.cache-enabled`，namespace 级 LRU+TTL，DML 自动 flush）④ 批量操作（MP 内置 `insertBatch` 多行 VALUES + `orm.UpdateBatch` 事务分批）⑤ `<discriminator>` 鉴别器（按列值切换子 resultMap）⑥ ID 生成策略（`id-type=snowflake|uuid|assign_id` insert 前零值主键自动填充，`RegisterIdGenerator` 可扩展）⑦ SQL 安全防护（`safe-update=true` 后无 WHERE 的 UPDATE/DELETE 返回 `ErrSafeUpdateBlocked`）
- **QueryWrapper 条件构造器**：`orm.NewQueryWrapper().Eq("col", v).LikeRight("name", "ad").Between("age", 18, 30).OrderByDesc("id")` 链式构造动态条件；mapper 方法声明 `*orm.QueryWrapper` 参数即可自动注入 SQL（原 SQL 有 WHERE 追加 `AND`、无 WHERE 补 `WHERE`，尾部 `ORDER BY`/`LIMIT` 之前插入；与 `*orm.PageParam` 组合时分页 COUNT 先派生，Total 与 Records 天然一致），XML 亦支持 `${ew}` 显式占位
- **Hook 拦截链**：`orm.RegisterHook(HookBeforeExecute|HookAfterExecute, hook)` 在 SQL 生成后/执行前后双挂载点拦截全部 Mapper 执行路径（普通方法/流式查询/分页）；Before 可改写 SQL，After 可见 Error 与 Duration，适配 SQL 审计/多租户/乐观锁等横切场景；Hook panic 自动 recover 不中断执行链
- **自定义 TypeHandler**：`orm.RegisterTypeHandlerFor[T](fn)` 为任意 Go 类型注册参数写入/结果扫描转换器（自定义时间、枚举、JSON 字段等），经 `convertFieldValue` 统一入口覆盖参数绑定、resultMap、无 resultMap 与流式查询全路径
- **多数据库**：PostgreSQL、MySQL、SQLite、人大金仓 KingbaseES、TiDB、TDSQL、PolarDB-MySQL、openGauss、GaussDB、HighGo DB、Vastbase、OceanBase（MySQL/Oracle 模式）、达梦 DM8、GBase 8s（南大通用）、MS SQL Server、Oracle、IBM DB2、ClickHouse，国产数据库适配已完成
- **Schema 缓存列类型推断**：查询结果列类型自动从 `information_schema` 推断（无需手写 resultMap 即可正确映射 time/bool/数字类型），可配置 TTL，DDL 后自动失效
- **代码生成**：内置 `generator`（XML → Go）和 `schema2code`（数据库表 → Go）工具
- **预编译缓存**：Prepared Statement 自动缓存和复用
- **事务支持**：`orm.Begin()` / `Commit()` / `Rollback()` TCC 风格全局事务（开启后 Mapper 方法与 SQL 自动参与）；`orm.WithTx(ctx, fn)` 回调式事务（fn 返回 nil 提交 / 返回 error 或 panic 回滚，嵌套调用复用外层事务，不占用 TCC 全局槽，多 goroutine 可并发各自开启）；context 携带事务优先于全局槽
- **强类型扫描直通道**：`orm.QueryTo(&users, sql, args...)` 查询结果直接扫描到 struct / []struct / []*struct / 标量切片 / []map / map / 标量指针（不经 map 中转，扫描缓冲与字段索引一次预编译）；列名匹配优先 `db` tag 显式列名，回退原名 → 首字母大写 → 蛇形转驼峰 → 大小写不敏感四策略；类型转换优先自定义 TypeHandler；`QueryToContext` 支持 WithTx 事务与超时控制；Mapper 方法声明 `context.Context` 参数即可自动参与 ctx 事务
- **gormish 链式 API（G1）**：`gormish.Open()` 获取链式句柄，GORM 风格链式查询/CRUD——`Table/Model/Select/Where/Or/Order/Limit/Offset/Group/Having/Distinct/Raw` 链式子句 + `Find/First/Last/Count/Scan/Create/Update/Updates/Delete/Exec` finisher；链式方法克隆语句状态（多克隆互不污染），finisher 返回 `(value, error)` 双返回值；条件一律 `?` 占位 + 参数绑定（经方言自动转换为 `$n`/`:n`/`@pn`，表名前缀自动改写）；结果扫描复用 QueryTo 直通道（db tag / TypeHandler / DefaultRowLimit 自动生效）；`Create` 零值主键视为自增列并回填（PG 族经 RETURNING），批量插入多行 VALUES；`First/Last` 未显式排序时按主键排序 LIMIT 1；`Update/Updates/Delete` 无 WHERE 条件报错防全表；`Transaction(fn)` 回调事务（基于 `orm.WithTx`，嵌套复用、不占 TCC 槽）
- **sqlc 风格 Querier 代码生成（S1）**：`go run ./cmd/sqlc` 扫描 XML Mapper，把静态 `<select>`（无动态标签、无 `${}`）抽取为类型安全的 Querier 接口 + 实现（model / `XxxQuerier` 接口 / `NewXxxQuerier(db)` 构造，一个 mapper 一个文件）；免连库——resultMap 按 jdbcType、resultType 按声明类型定型，标量参数按占位符名具名定型；动态语句自动跳过（`-v` 查看原因）；生成代码走 `QueryToContext`（方言占位符转换 / 表名前缀 / ctx 事务全部由 orm 层承担），select 恒返回切片；resultMap model 带 `db:"column,pk"` tag
- **xml2go 逆向代码生成**：`go run ./cmd/xml2go -m samples -d gen -p github.com/xxx/app` 从既有 MyBatis/MyBatis-Plus XML 全量语句（含动态 SQL、MP 内置 CRUD）生成 `<output>/models/`（模型 struct + `db:"column,pk/logic/version"` 元数据 + TableName）与 `<output>/mapper/`（orm.BaseMapper 代理 struct + init 注册 + sync.Once 单例 getter），与手写 Mapper 语义一致、产物保证可编译（跨 Mapper 模型去重、引用未生成模型的语句跳过并记录原因）；亦可用 `-c <配置文件>` 直接给 Java 侧 MyBatis/MyBatis-Plus 配置（.properties/.yml/.yaml/.xml：mapper-locations / `<mappers>` / mapperLocations，支持 classpath* 前缀、`**` 通配、config-location 链式解析），先定位 XML 再生成；与 sqlc 互补——动态语句/MP CRUD 选 xml2go，纯静态查询选 sqlc（见 docs/agents/xml2go.md）
- **大结果集流式读取**：`orm.QueryStream` / Mapper 流式 select 方法返回 `*orm.RowStream`，`Next()` 逐行消费、内存 O(1)，百万行结果集也不会 OOM（配合全局行数上限 `orm.SetDefaultRowLimit` 兜底）
- **内嵌 Mapper（go:embed）**：`orm.RegisterMapperFS` 直接读取 `embed.FS`，XML 无需解出到临时目录即可用于单文件二进制部署；亦支持「内嵌为基础 + 磁盘覆盖」合并加载（见 docs/features.md「内嵌 Mapper」）
- **零 CGO 依赖**：SQLite 走纯 Go 驱动（`modernc.org/sqlite`），交叉编译与静态链接无额外工具链要求

## 路线图

- [x] PostgreSQL 支持 — 已实现并测试通过
- [x] MySQL 支持 — 已实现（`cmd/mysqldemo/main.go`）
- [x] SQLite 支持 — 已实现（纯 Go 驱动 modernc.org/sqlite，无需 CGO，`cmd/sqlitedemo` 示例）
- [x] KingbaseES 支持 — 已实现（人大金仓，兼容 PostgreSQL 线协议，复用 `lib/pq` 驱动，`cmd/kingbasedemo` 示例）
- [x] 事务支持 — 已实现（`orm.Begin()` / `Commit()` / `Rollback()`）
- [x] 多数据源支持 — 已实现（`orm.InitializeDataSources` / `orm.UseDataSource`）

### 国产数据库适配计划

**P0 — MySQL 兼容族**（零成本，复用 MySQL dialector）：

- [x] TiDB（PingCAP）— MySQL 协议高度兼容，复用 `go-sql-driver/mysql`，默认端口 4000，`cmd/tidbdemo` 示例
- [x] TDSQL（腾讯云）— MySQL 兼容，复用 `go-sql-driver/mysql`，`cmd/tdsqldemo` 示例
- [x] PolarDB-MySQL（阿里云）— MySQL 兼容，复用 `go-sql-driver/mysql`，`cmd/polardbdemo` 示例

**P1 — PostgreSQL 兼容族**（低难度，复用 KingbaseES 模式）：

- [x] openGauss（华为开源）— PG 兼容，`openGauss-connector-go-pq` 驱动（lib/pq fork），框架自动以 `opengauss` 名称注册 `lib/pq`，`cmd/opengaussdemo` 示例
- [x] GaussDB（华为云）— PG 兼容，官方 `gaussdb-go` 驱动（基于 pgx），框架自动以 `gaussdb` 名称注册 `lib/pq`，`cmd/gaussdbdemo` 示例
- [x] HighGo DB（瀚高）— PG 兼容，`highgo-lib` 驱动（lib/pq fork），框架自动以 `highgo` 名称注册 `lib/pq`，`cmd/highgodemo` 示例
- [x] Vastbase（海量数据）— PG 兼容，直接使用 `lib/pq`，框架自动以 `vastbase` 名称注册 `lib/pq`，`cmd/vastbasedemo` 示例

**P2 — Oracle 方言族**（中高难度，需独立 dialector）：

- [x] OceanBase-MySQL（蚂蚁集团）— MySQL 模式零成本适配，复用 `go-sql-driver/mysql`，默认端口 2881，`cmd/oceanbasedemo` 示例
- [x] OceanBase-Oracle（蚂蚁集团）— Oracle 模式，独立 `OceanBaseOracleDialector`（Oracle 方言族），占位符 `?`→`:n`，`cmd/oboracledemo` 示例
- [x] DM8 / 达梦 — 自有协议（Oracle 风格），独立 `DamengDialector`（Oracle 方言族），占位符 `?`→`:n`，`cmd/damengdemo` 示例

**P3 — Informix 方言族**：

- [x] GBase 8s（南大通用）— Informix 兼容，独立 `GBase8sDialector`（Informix 方言族），占位符 `?`，`cmd/gbase8sdemo` 示例

**P4 — 国际主流数据库**：

- [x] MS SQL Server — 独立 `MssqlDialector`，占位符 `?`→`@p1/@p2`，`go-mssqldb` 官方纯 Go 驱动，分页 `OFFSET…ROWS FETCH NEXT…ROWS ONLY`
- [x] Oracle — 独立 `OracleDialector`，占位符 `?`→`:1/:2`，`go-ora` 纯 Go 驱动（与 OceanBase-Oracle/达梦同族），Oracle 12c+ 分页 `OFFSET…ROWS FETCH NEXT…ROWS ONLY`
- [x] DB2 — 独立 `Db2Dialector`，占位符 `?`→`:1/:2`，`go_ibm_db` 官方驱动（需 CGo + 客户端库），DB2 10.1+ 分页 `OFFSET…ROWS FETCH NEXT…ROWS ONLY`
- [x] ClickHouse — 独立 `ClickHouseDialector`，`clickhouse-go/v2` 官方纯 Go 驱动（OLAP 场景，无事务）

**P5 — 云原生 / NewSQL 数据库**：

- [ ] CockroachDB — PG 协议兼容，复用 PG dialector 零成本适配
- [ ] Google Cloud Spanner — 独立 `SpannerDialector`，`go-sql-spanner` 官方驱动（gRPC 协议）
- [ ] YDB — 独立 `YdbDialector`，`ydb-go-sdk` 官方纯 Go 驱动

**P6 — 嵌入式 / 分析型数据库**：

- [ ] DuckDB — 独立 `DuckdbDialector`，`go-duckdb` CGo 驱动（嵌入式列式 OLAP）
- [ ] SAP HANA — 独立 `HanaDialector`，`go-hdb` 官方纯 Go 驱动
- [ ] Snowflake — 独立 `SnowflakeDialector`，`gosnowflake` 官方纯 Go 驱动（云端 OLAP）

**P7 — 其他国产数据库**：

- [ ] MogDB（云和恩墨）— PG 兼容，复用 PG dialector 零成本适配
- [ ] IvorySQL（瀚高）— PG 兼容，复用 PG dialector 零成本适配
- [ ] GoldenDB（中兴）— MySQL 兼容，复用 MySQL dialector 零成本适配
- [ ] SequoiaDB/巨杉 — MySQL 兼容，复用 MySQL dialector 零成本适配
- [ ] 神通/Oscar（神舟通用）— ODBC 桥接（需 CGo），待厂商提供纯 Go 驱动

## 安装

```bash
go get github.com/bnulwh/mybatis-go
```

引入方式：使用哪个数据库就 blank import 对应的驱动（KingbaseES 无需额外引入，框架自动注册）：

| 数据库 | 驱动引入 |
|--------|---------|
| PostgreSQL | `_ "github.com/lib/pq"` |
| MySQL | `_ "github.com/go-sql-driver/mysql"` |
| SQLite | `_ "modernc.org/sqlite"` |
| KingbaseES | 无需引入（框架自动以 `kingbase` 名称注册 `lib/pq`） |
| TiDB | 无需引入（框架自动以 `tidb` 名称注册 `go-sql-driver/mysql`） |
| TDSQL | 无需引入（框架自动以 `tdsql` 名称注册 `go-sql-driver/mysql`） |
| PolarDB-MySQL | 无需引入（框架自动以 `polardb` 名称注册 `go-sql-driver/mysql`） |
| openGauss | 无需引入（框架自动以 `opengauss` 名称注册 `lib/pq`） |
| GaussDB | 无需引入（框架自动以 `gaussdb` 名称注册 `lib/pq`） |
| HighGo DB | 无需引入（框架自动以 `highgo` 名称注册 `lib/pq`） |
| Vastbase | 无需引入（框架自动以 `vastbase` 名称注册 `lib/pq`） |
| OceanBase | 无需引入（框架自动以 `oceanbase` 名称注册 `go-sql-driver/mysql`） |
| OceanBase-Oracle | `_ "github.com/oceanbase/oceanbase-driver-go"`（需用户引入驱动） |
| 达梦 DM8 | `_ "dmdb.com/dm"`（需用户引入驱动） |
| GBase 8s | `_ "github.com/alexgarrec/go-informix"`（需用户引入驱动） |
| MS SQL Server | `_ "github.com/microsoft/go-mssqldb"`（需用户引入驱动） |
| Oracle | 无需引入（框架自动以 `oracle` 名称注册 `go-ora` 驱动） |
| DB2 | `_ "github.com/ibmdb/go_ibm_db"`（需用户引入驱动，且需安装 DB2 ODBC/CLI 客户端库） |
| ClickHouse | 无需引入（框架自动以 `clickhouse` 名称注册 `clickhouse-go/v2` 驱动） |

## 快速开始

### 1. 定义模型

```go
package main

import "time"

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
```

### 2. 定义 Mapper

注意：必须内嵌 `orm.BaseMapper`，`func` 类型字段由框架在初始化时注入代理，字段名与 XML Mapper 中的操作 id 对应。

```go
package main

import "github.com/bnulwh/mybatis-go/orm"

type UserInfoModelMapper struct {
    orm.BaseMapper
    DeleteByPrimaryKey func(int) (int64, error)
    Insert             func(UserInfoModel) (int64, error)
    UpdateByPrimaryKey func(UserInfoModel) (int64, error)
    SelectByPrimaryKey func(int) ([]UserInfoModel, error)
    SelectAll          func() ([]UserInfoModel, error)
}
```

### 3. 初始化 ORM

```go
import (
    log "github.com/bnulwh/logrus"
    "github.com/bnulwh/mybatis-go/orm"
    _ "github.com/lib/pq"               // PostgreSQL 驱动
    // _ "github.com/go-sql-driver/mysql" // MySQL 驱动
    // _ "modernc.org/sqlite"             // SQLite 驱动
    // KingbaseES 无需驱动引入
)

func init() {
    orm.SetLogger(log.StandardLogger())
    if err := orm.Initialize("application.properties"); err != nil {
        panic(err)
    }
    orm.RegisterModel(new(UserInfoModel))
    orm.RegisterMapper(new(UserInfoModelMapper))
}
```

### 4. 使用 ORM

```go
func main() {
    defer orm.Close()
    mp := orm.NewMapper("UserInfoModelMapper").(UserInfoModelMapper)
    rs, err := mp.SelectAll()
    if err != nil {
        log.Errorf("select failed: %v", err)
    } else {
        for _, row := range rs {
            log.Infof("row: %v", row)
        }
    }
}
```

完整示例见 `cmd/postgresdemo/main.go`、`cmd/mysqldemo/main.go`、`cmd/sqlitedemo/main.go`、`cmd/kingbasedemo/main.go`、`cmd/tidbdemo/main.go`、`cmd/tdsqldemo/main.go`、`cmd/polardbdemo/main.go`、`cmd/opengaussdemo/main.go`、`cmd/gaussdbdemo/main.go`、`cmd/highgodemo/main.go`、`cmd/vastbasedemo/main.go`、`cmd/oceanbasedemo/main.go`、`cmd/oboracledemo/main.go`、`cmd/damengdemo/main.go`、`cmd/gbase8sdemo/main.go`、`cmd/mssqldemo/main.go`、`cmd/oracledemo/main.go` 和 `cmd/db2demo/main.go`。

## 功能专题与详细文档

| 主题 | 说明 | 文档 |
|------|------|------|
| 大结果集流式查询 | `QueryStream` / Mapper 流式 select，游标逐行消费、内存 O(1) | [docs/features.md](docs/features.md) |
| 事务 | `orm.Begin/Commit/Rollback` + `orm.WithTx` 回调式事务 | [docs/features.md](docs/features.md) |
| 多数据源 | 单配置文件声明多数据源，`orm.UseDataSource` 切换 | [docs/configuration.md](docs/configuration.md) |
| 配置文件 | Spring Boot 风格 `.properties`，各数据库连接示例 + 配置项说明 + 编程式配置 | [docs/configuration.md](docs/configuration.md) |
| 数据表名前缀 | 同一套 XML 多环境物理表名切换（前缀改写 / 前缀映射） | [docs/configuration.md](docs/configuration.md) |
| 内嵌 Mapper（go:embed） | XML 随二进制分发，单文件部署，支持磁盘覆盖热修 | [docs/features.md](docs/features.md) |
| 注入自定义 DB / DSN | 连接代理、测试桩、已有连接池复用 | [docs/configuration.md](docs/configuration.md) |
| 性能优化 | 预编译语句缓存、反射预编译、扫描目标复用、流式读取 | [docs/features.md](docs/features.md) |
| 代码生成 | generator / schema2code / xml2go / sqlc 四款工具 | [docs/code-generation.md](docs/code-generation.md) |

## 运行示例

```bash
go run ./cmd/sqlitedemo      # SQLite（自动建表 + Mapper 全流程，生成 test.db）
go run ./cmd/postgresdemo    # PostgreSQL（需先准备 cmd/postgresdemo/application-pg.properties 指向的库）
go run ./cmd/mysqldemo       # MySQL
go run ./cmd/kingbasedemo    # KingbaseES
go run ./cmd/tidbdemo        # TiDB（MySQL 协议兼容，默认端口 4000）
go run ./cmd/tdsqldemo       # TDSQL（腾讯云，MySQL 兼容）
go run ./cmd/polardbdemo     # PolarDB-MySQL（阿里云，MySQL 兼容）
go run ./cmd/opengaussdemo  # openGauss（华为开源，PostgreSQL 兼容）
go run ./cmd/gaussdbdemo    # GaussDB（华为云，PostgreSQL 兼容）
go run ./cmd/highgodemo     # HighGo DB（瀚高，PostgreSQL 兼容）
go run ./cmd/vastbasedemo   # Vastbase（海量数据，PostgreSQL 兼容）
go run ./cmd/oceanbasedemo  # OceanBase MySQL 模式（蚂蚁集团，MySQL 兼容）
go run ./cmd/oboracledemo   # OceanBase Oracle 模式（蚂蚁集团，Oracle 方言族）
go run ./cmd/damengdemo     # 达梦 DM8（Oracle 方言族）
go run ./cmd/gbase8sdemo    # GBase 8s（南大通用，Informix 方言族）
go run ./cmd/mssqldemo     # MS SQL Server（独立方言族）
go run ./cmd/oracledemo     # Oracle（Oracle 方言族）
go run ./cmd/db2demo        # IBM DB2（DB2 方言族，需安装 DB2 客户端库）
go run ./cmd/clickhousedemo # ClickHouse（OLAP 列式引擎，无事务）
```

## 重要说明

- `orm.NewMapper("MapperName")` 创建的对象必须先通过 `orm.RegisterMapper` 注册
- `orm.RegisterModel` 用于注册模型类，注册后的类在调用 Mapper 函数时可以自动创建并填充值
- 函数字段的 tag（如 `` `args:id` ``）可用于指定输入参数名称映射
- `useGeneratedKeys` 回填需向 Insert 方法传 **struct 指针**（值传递无法写回调用方）；入参为 map 时同样支持
- SELECT 方法的返回值类型为 `([]Model, error)`，INSERT/UPDATE/DELETE 为 `(int64, error)`；流式 select 可返回 `(*orm.RowStream, error)`（逐行消费，调用方必须 `Close()` 释放连接）
- KingbaseES 驱动由框架自动注册（`sql.Register("kingbase", &pq.Driver{})`），无需也不应重复引入驱动
- TiDB/TDSQL/PolarDB 驱动由框架自动注册（`sql.Register("tidb"/"tdsql"/"polardb", &mysql.MySQLDriver{})`），无需也不应重复引入驱动
- openGauss/GaussDB/HighGo/Vastbase 驱动由框架自动注册（`sql.Register("opengauss"/"gaussdb"/"highgo"/"vastbase", &pq.Driver{})`），无需也不应重复引入驱动
- OceanBase MySQL 模式驱动由框架自动注册（`sql.Register("oceanbase", &mysql.MySQLDriver{})`），无需也不应重复引入驱动
- OceanBase Oracle 模式需用户引入驱动（`_ "github.com/oceanbase/oceanbase-driver-go"`），框架未内置该驱动
- 达梦 DM8 需用户引入驱动（`_ "dmdb.com/dm"`），框架未内置该驱动
- Oracle 方言族（OceanBase-Oracle / 达梦）占位符自动 `?`→`:1`/`:2`…，`NeedsReturning` 返回 false
- Informix 方言族（GBase 8s）占位符保持 `?`，`NeedsReturning` 返回 false，`EffectiveSchema` 返回 `UPPER(Username)`
- MSSQL 方言族占位符自动 `?`→`@p1/@p2`…，`NeedsReturning` 返回 false，分页使用 `OFFSET…ROWS FETCH NEXT…ROWS ONLY`（需 SQL 含 `ORDER BY`），默认 Schema 为 `dbo`
- MSSQL 需用户引入驱动（`_ "github.com/microsoft/go-mssqldb"`），框架未内置该驱动
- Oracle 驱动由框架自动注册（go-ora 自注册为 `"oracle"`），无需额外引入
- Oracle 方言族占位符自动 `?`→`:1/:2`…，`NeedsReturning` 返回 false，分页使用 `OFFSET…ROWS FETCH NEXT…ROWS ONLY`（需 Oracle 12c+），`EffectiveSchema` 返回 `UPPER(Username)`
- DB2 方言族占位符自动 `?`→`:1/:2`…，`NeedsReturning` 返回 false，分页使用 `OFFSET…ROWS FETCH NEXT…ROWS ONLY`（需 DB2 10.1+），`EffectiveSchema` 返回 `UPPER(Username)`
- DB2 需用户引入驱动（`_ "github.com/ibmdb/go_ibm_db"`），且需安装 DB2 ODBC/CLI 客户端库
- ClickHouse 驱动由框架自动注册（clickhouse-go/v2 自注册为 `"clickhouse"`），无需额外引入
- ClickHouse 方言族占位符保持 `?`，`NeedsReturning` 返回 false，分页使用 `LIMIT n OFFSET m`，`EffectiveSchema` 返回数据库名；ClickHouse 是列式 OLAP 引擎，不支持事务，UPDATE/DELETE 为异步 mutation
- 日志通过 `orm.SetLogger` 替换，实现 `log.Logger` 接口即可
- 事务：`orm.Begin()` 开启后 Mapper 方法自动在事务内执行，`Commit()` / `Rollback()` 结束事务（见 docs/features.md「事务」）

## 项目结构

```
├── cmd/
│   ├── generator/       # 从 XML Mapper 生成代码
│   ├── schema2code/     # 从数据库表结构生成代码
│   ├── postgresdemo/    # PostgreSQL 使用示例
│   ├── mysqldemo/       # MySQL 使用示例
│   ├── kingbasedemo/    # KingbaseES（人大金仓）使用示例
│   ├── sqlitedemo/      # SQLite 使用示例
│   ├── tidbdemo/        # TiDB 使用示例
│   ├── tdsqldemo/       # TDSQL 使用示例
│   ├── polardbdemo/     # PolarDB-MySQL 使用示例
│   ├── opengaussdemo/   # openGauss 使用示例
│   ├── gaussdbdemo/     # GaussDB 使用示例
│   ├── highgodemo/      # HighGo DB 使用示例
│   ├── vastbasedemo/    # Vastbase 使用示例
│   ├── oceanbasedemo/   # OceanBase MySQL 模式使用示例
│   ├── oboracledemo/    # OceanBase Oracle 模式使用示例
│   ├── damengdemo/      # 达梦 DM8 使用示例
│   ├── gbase8sdemo/     # GBase 8s 使用示例
│   ├── mssqldemo/       # MS SQL Server 使用示例
│   ├── oracledemo/      # Oracle 使用示例
│   ├── db2demo/         # IBM DB2 使用示例
│   ├── clickhousedemo/  # ClickHouse 使用示例
│   └── demo/            # 通用使用示例
├── orm/                 # 核心 ORM 框架
│   ├── transaction.go   # 事务支持（Begin/Commit/Rollback）
│   ├── multi_datasource.go  # 多数据源注册表（InitializeDataSources / UseDataSource / AddDataSource）
│   ├── row_stream.go    # 大结果集流式读取（QueryStream / RowStream，Mapper 流式 select）
│   ├── embed_fs.go      # go:embed 内嵌 Mapper 加载（RegisterMapperFS / RegisterMapperSources / ReloadMappers）
│   ├── dialector/       # 数据库方言
│   │   ├── mysql.go / postgres.go / sqlite.go / kingbase.go
│   │   ├── tidb.go / tdsql.go / polardb.go   # P0 国产数据库（MySQL 兼容族）
│   │   ├── opengauss.go / gaussdb.go / highgo.go / vastbase.go   # P1 国产数据库（PostgreSQL 兼容族）
│   │   ├── oceanbase.go / oceanbase_oracle.go / dameng.go       # P2 国产数据库（Oracle 方言族）
│   │   ├── gbase8s.go                                          # P3 国产数据库（Informix 方言族）
│   │   ├── mssql.go                                            # P4 国际主流数据库（MSSQL 方言族）
│   │   ├── oracle.go                                           # P4 国际主流数据库（Oracle 方言族）
│   │   ├── db2.go                                              # P4 国际主流数据库（DB2 方言族）
│   │   ├── clickhouse.go                                       # P4 国际主流数据库（ClickHouse 方言族）
│   │   └── types.go / base.go                 # 类型定义 + 公共基类
│   └── ...              # 初始化、代理、SQL 执行、结果转换、缓存等
├── types/               # XML 解析引擎和数据类型
│   └── sql_mappers.go   # Mapper 加载（NewSqlMappers 磁盘 / NewSqlMappersFrom embed.FS 等任意 io/fs.FS）
├── utils/               # 工具函数
├── log/                 # 日志接口
├── mapper/              # 生成的 Mapper 示例
├── resources/mapper/    # XML Mapper 文件
└── samples/             # RuoYi Mapper 兼容性回归样本（KingbaseES 方言，S-01~S-11 已全部修复）
```

> 各数据库 demo 目录（`cmd/<数据库>demo/`）内含对应的 `application-*.properties` 连接配置模板与 README 说明，运行示例前请先修改其中的连接信息。

## 测试

```bash
go test ./types/... ./utils/...        # 不依赖数据库的单元测试
go test ./orm/... -v                   # 含 SQLite 端到端测试（Mapper / 事务 / 表结构，无需外部数据库）
go test -v -count=1 ./... -coverprofile=cover.out
```

## 更新日志

见 [CHANGELOG.md](CHANGELOG.md)。
