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
- **内嵌 Mapper（go:embed）**：`orm.RegisterMapperFS` 直接读取 `embed.FS`，XML 无需解出到临时目录即可用于单文件二进制部署；亦支持「内嵌为基础 + 磁盘覆盖」合并加载（见「内嵌 Mapper」章节）
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

## 大结果集流式查询

大结果集（如 10 万行以上）用 `Query` 会一次性全部读进内存。改用流式读取：游标保持打开、任意时刻内存只保留当前一行，逐行处理后由调用方 `Close()` 释放连接。

### 顶层 API

```go
st, err := orm.QueryStream(context.Background(), `SELECT * FROM t_user`)
if err != nil {
    return err
}
defer st.Close() // 无论是否读完都必须 Close，否则连接被占住

for st.Next() {
    row := st.Row() // 当前行 map（与 Query 的行结构一致）
    // 或 st.Scan(&user) 填充到结构体（列名自动匹配字段）
    process(row)
}
if err := st.Err(); err != nil {
    return err // 扫描/游标错误（流式场景不静默丢行）
}
```

### Mapper 流式 select

Mapper 方法返回 `(*orm.RowStream, error)` 即自动走流式路径（XML 仍是普通 `<select>`，无需特殊声明）：

```go
type UserInfoModelMapper struct {
    orm.BaseMapper
    StreamAll func() (*orm.RowStream, error)
    // ...
}

st, err := mp.StreamAll()
if err != nil {
    return err
}
defer st.Close()
for st.Next() {
    var u UserInfoModel
    if err := st.Scan(&u); err != nil {
        return err
    }
    log.Infof("row: %+v", u)
}
```

> 说明：流式方法的结果类型与 XML `resultType` / `resultMap` 解耦，逐行 `Row()` / `Scan()` 由调用方决定；行数上限仍遵循全局 `orm.SetDefaultRowLimit`（默认 10000，负数不限制返回全部）。

## 事务

```go
tx, err := orm.Begin()
if err != nil {
    log.Errorf("begin failed: %v", err)
    return
}
defer tx.Rollback() // 未 Commit 时自动回滚（重复调用是安全 no-op）

// 事务开启后，Mapper 方法 / orm.Execute / orm.Query 自动在事务内执行
if _, err := mp.Insert(UserInfoModel{Username: "tx_user"}); err != nil {
    return // defer 回滚
}
if _, err := mp.UpdateByPrimaryKey(...); err != nil {
    return
}

if err := tx.Commit(); err != nil {
    log.Errorf("commit failed: %v", err)
}
```

也可以直接使用 `tx.Exec` / `tx.Query` / `tx.QueryRow` 执行 SQL，或通过 `orm.BeginTx(ctx, opts)` 指定事务选项（如隔离级别）。

> 注意：事务绑定全局连接（单事务槽），多个 goroutine 之间不要交错开启事务。

## 多数据源

支持在单个配置文件中声明多个数据源，并通过 `orm.UseDataSource(name)` 切换活跃数据源（影响后续 Mapper 方法 / `orm.Execute` / `orm.Query` / `Begin`）。

```properties
# 默认数据源（无名称前缀）
spring.datasource.url= jdbc:mysql://localhost:3306/db1
spring.datasource.username= root
spring.datasource.password= 123456

# 附加数据源：在 mybatis.datasources 中列出名称，配置键带 <name> 前缀
mybatis.datasources= secondary
spring.datasource.secondary.url= jdbc:postgresql://localhost:5432/db2
spring.datasource.secondary.username= root
spring.datasource.secondary.password= 123456
```

```go
orm.InitializeDataSources("application.properties") // 或 orm.InitializeDataSourcesFromSettings(cm)

orm.UseDataSource("secondary") // 切换，后续操作走 db2
mp.SelectAll()

orm.UseDataSource("default")   // 切回默认
```

编程方式注册命名数据源：

> **多数据源 × schema × 前缀 配置矩阵**（v0.2.0）：默认源用无名称前缀键；附加源键带 `<name>` 前缀，
> 逐源可覆盖 schema 与前缀（未配置时继承默认源前缀/前缀映射）。

```properties
# 默认源：public schema + threedb_ 前缀（开发环境）
spring.datasource.url= jdbc:kingbase://localhost:54321/db?currentSchema=public
mybatis.table-prefix= threedb_

# 附加源 report：subsp schema + 移除 threedb_ 前缀（生产环境，物理表无前缀）
mybatis.datasources= report
spring.datasource.report.url= jdbc:kingbase://localhost:54321/db
spring.datasource.report.schema= subsp
spring.datasource.report.table-prefix-map= threedb_:
```

> 约定：SQL 一律写无 schema 名（`sys_user`），由 `search_path=<schema>` 路由到目标 schema；
> 显式 `schema.table` 限定名中，仅 `public/main` 参与前缀正向改写，其余 schema 整段跳过
> （前缀映射为显式配置意图，对限定名的表名部分同样生效）——两种写法不要混用。

```go
orm.AddDataSource("report", "postgres", "10.1.2.3", 5432, "root", "123456", "reportdb")
orm.UseDataSource("report")
```

其他 API：`orm.GetDataSource(name)`、`orm.GetDataSourceNames()`、`orm.ReConnectDataSource(name)`。

> 注意：切换发生在全局层，请避免在并发 goroutine 中交错切换数据源；Mapper XML 定义对所有数据源共享。

## 配置文件

支持 Spring Boot 风格的 `.properties` 文件，各数据库的示例配置位于对应的 demo 目录（`cmd/<数据库>demo/application-*.properties`，详见各 demo 目录下的 README）。

### PostgreSQL

```properties
spring.datasource.url= jdbc:postgresql://localhost:5432/testdb
spring.datasource.username= root
spring.datasource.password= 123456
mybatis.mapper-locations= resources/mapper
```

### MySQL

```properties
spring.datasource.url= jdbc:mysql://localhost:3306/kubecloud?useUnicode=true&characterEncoding=utf-8&useSSL=false
spring.datasource.username= root
spring.datasource.password= 123456
spring.datasource.max-idle= 100
spring.datasource.max-open= 100
spring.datasource.max-timeout= 100
mybatis.mapper-locations= resources/mapper
```

### SQLite

```properties
# 纯文件数据库，无需用户名/密码，Name 为 .db 文件路径
spring.datasource.url= jdbc:sqlite:test.db
mybatis.mapper-locations= resources/mapper
```

### KingbaseES（人大金仓）

```properties
spring.datasource.url= jdbc:kingbase8://localhost:54321/testdb
spring.datasource.username= system
spring.datasource.password= 123456
mybatis.mapper-locations= resources/mapper
```

URL 类型支持 `jdbc:kingbase8://`、`jdbc:kingbase://` 等（parseDatabaseType 兼容 kingbase5~8 各版本号）。

### openGauss（华为开源）

```properties
# openGauss — PostgreSQL 兼容
spring.datasource.url= jdbc:opengauss://localhost:5432/testdb
spring.datasource.username= root
spring.datasource.password= 123456
mybatis.mapper-locations= resources/mapper
```

框架自动以 `opengauss` 名称注册 `lib/pq` 驱动，无需额外引入。

### GaussDB（华为云）

```properties
# GaussDB — PostgreSQL 兼容
spring.datasource.url= jdbc:gaussdb://localhost:5432/testdb
spring.datasource.username= root
spring.datasource.password= 123456
mybatis.mapper-locations= resources/mapper
```

框架自动以 `gaussdb` 名称注册 `lib/pq` 驱动，无需额外引入。

### HighGo DB（瀚高）

```properties
# HighGo DB — PostgreSQL 兼容
spring.datasource.url= jdbc:highgo://localhost:5432/testdb
spring.datasource.username= root
spring.datasource.password= 123456
mybatis.mapper-locations= resources/mapper
```

框架自动以 `highgo` 名称注册 `lib/pq` 驱动，无需额外引入。

### Vastbase（海量数据）

```properties
# Vastbase — PostgreSQL 兼容
spring.datasource.url= jdbc:vastbase://localhost:5432/testdb
spring.datasource.username= root
spring.datasource.password= 123456
mybatis.mapper-locations= resources/mapper
```

框架自动以 `vastbase` 名称注册 `lib/pq` 驱动，无需额外引入。

### OceanBase MySQL 模式（蚂蚁集团）

```properties
# OceanBase MySQL 模式 — MySQL 兼容，默认端口 2881
spring.datasource.url= jdbc:oceanbase://localhost:2881/testdb?useUnicode=true&characterEncoding=utf-8&useSSL=false
spring.datasource.username= root
spring.datasource.password= 123456
spring.datasource.max-idle= 100
spring.datasource.max-open= 100
spring.datasource.max-timeout= 100
mybatis.mapper-locations= resources/mapper
```

框架自动以 `oceanbase` 名称注册 `go-sql-driver/mysql` 驱动，无需额外引入。

### OceanBase Oracle 模式（蚂蚁集团）

```properties
# OceanBase Oracle 模式 — Oracle 方言族，默认端口 2881
spring.datasource.url= jdbc:oceanbase-oracle://localhost:2881/testdb
spring.datasource.username= SYS
spring.datasource.password= 123456
mybatis.mapper-locations= resources/mapper
```

Oracle 方言族占位符自动 `?`→`:1`/`:2`…；`NeedsReturning` 返回 false（Oracle 风格使用自增序列）。用户需引入 OceanBase Oracle 驱动（`_ "github.com/oceanbase/oceanbase-driver-go"`）。

### 达梦 DM8

```properties
# 达梦 DM8 — Oracle 方言族，默认端口 5236
spring.datasource.url= jdbc:dameng://localhost:5236/testdb
spring.datasource.username= SYSDBA
spring.datasource.password= 123456
mybatis.mapper-locations= resources/mapper
```

Oracle 方言族占位符自动 `?`→`:1`/`:2`…；`NeedsReturning` 返回 false。用户需引入达梦驱动（`_ "dmdb.com/dm"`）。

### GBase 8s（南大通用）

```properties
# GBase 8s — Informix 兼容族，默认端口 9088
spring.datasource.url= jdbc:gbase8s://localhost:9088/testdb
spring.datasource.username= informix
spring.datasource.password= 123456
mybatis.mapper-locations= resources/mapper
```

Informix 方言族占位符保持 `?`；`NeedsReturning` 返回 false；`EffectiveSchema` 返回 `UPPER(Username)`。用户需引入 Informix 驱动（`_ "github.com/alexgarrec/go-informix"`）。

### MS SQL Server

```properties
# MS SQL Server — 独立方言族，默认端口 1433
spring.datasource.url= jdbc:sqlserver://localhost:1433/testdb
spring.datasource.username= sa
spring.datasource.password= 123456
spring.datasource.max-idle= 100
spring.datasource.max-open= 100
spring.datasource.max-timeout=100
mybatis.mapper-locations= resources/mapper
```

MSSQL 方言族占位符自动 `?`→`@p1/@p2`…；`NeedsReturning` 返回 false（使用 `SCOPE_IDENTITY()` + `LastInsertId()`）；分页使用 `OFFSET N ROWS FETCH NEXT M ROWS ONLY`（需 SQL 含 `ORDER BY`）；默认 Schema 为 `dbo`。用户需引入 MSSQL 驱动（`_ "github.com/microsoft/go-mssqldb"`）。

### Oracle

```properties
# Oracle — Oracle 方言族，默认端口 1521
spring.datasource.url= jdbc:oracle://localhost:1521/testdb
spring.datasource.username= system
spring.datasource.password= 123456
mybatis.mapper-locations= resources/mapper
```

Oracle 方言族占位符自动 `?`→`:1/:2`…；`NeedsReturning` 返回 false（Oracle 使用自增序列）；分页使用 `OFFSET N ROWS FETCH NEXT M ROWS ONLY`（需 Oracle 12c+ 且 SQL 含 `ORDER BY`）；`EffectiveSchema` 返回 `UPPER(Username)`。框架自动以 `oracle` 名称注册 go-ora 驱动，无需额外引入。

### DB2

```properties
# IBM DB2 — DB2 方言族，默认端口 50000
spring.datasource.url= jdbc:db2://localhost:50000/testdb
spring.datasource.username= db2inst1
spring.datasource.password= 123456
mybatis.mapper-locations= resources/mapper
```

DB2 方言族占位符自动 `?`→`:1/:2`…；`NeedsReturning` 返回 false（DB2 使用 IDENTITY 列 + `IDENTITY_VAL_LOCAL()`）；分页使用 `OFFSET N ROWS FETCH NEXT M ROWS ONLY`（需 DB2 10.1+ 且 SQL 含 `ORDER BY`）；`EffectiveSchema` 返回 `UPPER(Username)`；DSN 格式 `HOSTNAME=host;PORT=port;DATABASE=dbname;UID=username;PWD=password`。用户需引入 DB2 驱动（`_ "github.com/ibmdb/go_ibm_db"`），且需预先安装 DB2 ODBC/CLI 客户端库。

### ClickHouse

```properties
# ClickHouse — ClickHouse 方言族，默认端口 9000（原生 TCP）/ 8123（HTTP）
spring.datasource.url= jdbc:clickhouse://localhost:9000/testdb
spring.datasource.username= default
spring.datasource.password=
mybatis.mapper-locations= resources/mapper
```

ClickHouse 方言族占位符保持 `?`（`PlaceholderQuestion`）；`NeedsReturning` 返回 false（ClickHouse 无 RETURNING 支持）；分页使用 `LIMIT n OFFSET m`；`EffectiveSchema` 返回数据库名；DSN 格式 `clickhouse://user:password@host:port/dbname`（clickhouse-go/v2 URL 格式）。框架自动以 `clickhouse` 名称注册 clickhouse-go/v2 驱动，无需额外引入。注意：ClickHouse 是列式 OLAP 引擎，不支持事务，UPDATE/DELETE 为异步 mutation，ORM 适用于批量插入 + 查询分析场景。

### TiDB（PingCAP）

```properties
# TiDB — MySQL 协议高度兼容，默认端口 4000
spring.datasource.url= jdbc:tidb://localhost:4000/testdb?useUnicode=true&characterEncoding=utf-8&useSSL=false
spring.datasource.username= root
spring.datasource.password=
mybatis.mapper-locations= resources/mapper
```

框架自动以 `tidb` 名称注册 `go-sql-driver/mysql` 驱动，无需额外引入。

### TDSQL（腾讯云）

```properties
# TDSQL — MySQL 兼容
spring.datasource.url= jdbc:tdsql://localhost:3306/testdb?useUnicode=true&characterEncoding=utf-8&useSSL=false
spring.datasource.username= root
spring.datasource.password= 123456
mybatis.mapper-locations= resources/mapper
```

框架自动以 `tdsql` 名称注册 `go-sql-driver/mysql` 驱动，无需额外引入。

### PolarDB-MySQL（阿里云）

```properties
# PolarDB-MySQL — MySQL 兼容
spring.datasource.url= jdbc:polardb://localhost:3306/testdb?useUnicode=true&characterEncoding=utf-8&useSSL=false
spring.datasource.username= root
spring.datasource.password= 123456
mybatis.mapper-locations= resources/mapper
```

框架自动以 `polardb` 名称注册 `go-sql-driver/mysql` 驱动，无需额外引入。

### 配置项说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `spring.datasource.url` | JDBC 连接 URL（必填） | - |
| `spring.datasource.username` | 用户名（SQLite 不需要） | - |
| `spring.datasource.password` | 密码（SQLite 不需要） | - |
| `spring.datasource.max-idle` | 连接池最大空闲连接数 | 100 |
| `spring.datasource.max-open` | 连接池最大打开连接数 | 100 |
| `spring.datasource.max-timeout` | 连接最大存活时长（秒） | 300 |
| `spring.datasource.prepared-stmt` | 是否启用预编译语句缓存（`false` 关闭，适合 PgBouncer 等不支持服务端预编译的代理场景） | true |
| `mybatis.mapper-locations` | XML Mapper 文件目录（也支持 `go:embed` 内嵌，见下文） | - |
| `mybatis.table-prefix` | 数据表名前缀（如 `test_`），SQL 执行时自动拼接到表名前，XML Mapper 语句无需改动 | - |

> **MySQL DATETIME 列**：框架自动在 MySQL DSN 追加 `?parseTime=true&loc=Local`（与 SQLite 的 `_loc=auto` 同理），DATETIME/TIMESTAMP 列直接扫描为 `time.Time`；否则 go-sql-driver 返回原始 `[]byte`，时间字段无法赋值。

> **预编译语句缓存**：默认开启。参数化 SQL（Mapper 的 `#{}` 或 `Execute/Query` 带参调用）会按 SQL 文本缓存预编译语句，避免数据库重复解析与生成执行计划；无参数 SQL（DDL、静态查询）直接执行，不进入缓存。PostgreSQL/KingbaseES 的占位符会自动从 `?` 转为 `$1、$2…`（lib/pq 不支持 `?`）。

支持环境变量覆盖：配置值形如 `${ENV_NAME}` 或 `${ENV_NAME:default}` 时会自动替换。

JDBC URL 支持 IPv6 地址，如 `jdbc:postgresql://[2001:db8::1]:5432/testdb`（MySQL DSN 会自动补方括号）。

### 编程式配置（不使用 properties 文件）

```go
err := orm.InitializeDatabase("postgres", "localhost", 5432, "root", "123456", "testdb")
// 或 orm.InitializeDatabase("kingbase", "localhost", 54321, "system", "123456", "testdb")
// 或 orm.InitializeDatabase("tidb", "localhost", 4000, "root", "", "testdb")
// 或 orm.InitializeDatabase("tdsql", "localhost", 3306, "root", "123456", "testdb")
// 或 orm.InitializeDatabase("polardb", "localhost", 3306, "root", "123456", "testdb")
// 或 orm.InitializeDatabase("opengauss", "localhost", 5432, "root", "123456", "testdb")
// 或 orm.InitializeDatabase("gaussdb", "localhost", 5432, "root", "123456", "testdb")
// 或 orm.InitializeDatabase("highgo", "localhost", 5432, "root", "123456", "testdb")
// 或 orm.InitializeDatabase("vastbase", "localhost", 5432, "root", "123456", "testdb")
// 或 orm.InitializeDatabase("oceanbase", "localhost", 2881, "root", "123456", "testdb")
// 或 orm.InitializeDatabase("oceanbase-oracle", "localhost", 2881, "SYS", "123456", "testdb")
// 或 orm.InitializeDatabase("dameng", "localhost", 5236, "SYSDBA", "123456", "testdb")
// 或 orm.InitializeDatabase("gbase8s", "localhost", 9088, "informix", "123456", "testdb")
// 或 orm.InitializeDatabase("mssql", "localhost", 1433, "sa", "123456", "testdb")
// 或 orm.InitializeDatabase("oracle", "localhost", 1521, "system", "123456", "testdb")
// 或 orm.InitializeDatabase("db2", "localhost", 50000, "db2inst1", "123456", "testdb")
// 或 orm.InitializeDatabase("clickhouse", "localhost", 9000, "default", "", "testdb")
```

### 数据表名前缀

同一套 XML Mapper 可在不同环境指向不同的物理表：测试环境配置 `mybatis.table-prefix=test_`，
SQL 执行时所有表引用自动改为 `test_sys_user`；生产环境不配置（或配置 `prod_`）即可，
**XML Mapper 里的 SQL 语句保持 `sys_user` 不变**。

```properties
# 测试环境
mybatis.table-prefix= test_
```

```properties
# 生产环境（不配置或用 prod_）
# mybatis.table-prefix= prod_
```

特性说明：

- 前缀改写发生在执行入口（`orm.Execute`/`orm.Query`、Mapper 代理、事务、流式查询全覆盖），
  对 `FROM/JOIN/INTO/UPDATE/TABLE` 等表位置的标识符生效，不留改列名、字符串字面量、别名与 `#{}` 占位符；
- 系统表/目录不参与改写：`information_schema`、`pg_%`、`sqlite_%`、`pragma_%`；
- 已带前缀的表名不再叠加（`test_sys_user` 不会被改成 `test_test_sys_user`）；
- **真实表集合校验（提高准确率）**：改写前先从配置的 schema 查询真实表名集合并缓存（DDL 后自动失效），
  若 SQL 引用的表名在库中真实存在（物理表未带前缀，如改了前缀但表未改名）则保持原样，
  带前缀的表名真实存在才改写，两者均不存在（如 `CREATE TABLE` 新建表）才按前缀匹配兜底；
  该特性不依赖连库即可用的纯前缀逻辑保留为降级路径；
- CTE（`WITH x AS (...)`）名、`CREATE TABLE IF NOT EXISTS` 等 DDL 修饰词不会被误改为表名；
- 多数据源：附加数据源未单独配置前缀时继承默认源前缀；
- 兼容 MyBatis-Plus 风格配置键 `mybatis-plus.global-config.db-config.table-prefix`。

编程方式设置全局前缀 / 前缀映射（作用于未在配置中单独指定前缀的数据源）：

```go
orm.SetTablePrefix("test_")
prefix := orm.GetTablePrefix()
orm.SetTablePrefixMap(map[string]string{"threedb_": ""}) // 移除 threedb_ 前缀
```

**前缀映射（移除/替换，v0.2.0）**：存量 XML 硬编码了 `threedb_` 之类的旧前缀、而生产物理表已改名为无前缀
（或 `app_` 前缀）时，无需批量改写 XML，配置 `mybatis.table-prefix-map` 即可在 SQL 执行入口统一翻译：

```properties
# 移除旧前缀（threedb_sys_user → sys_user）
mybatis.table-prefix-map= threedb_:
# 或替换为另一前缀（threedb_sys_user → app_sys_user）
mybatis.table-prefix-map= threedb_:app_
```

映射优先级高于真实表集合判定；映射值为空（移除）时按表集合判定：库中该无前缀表存在则保持、
带前缀表存在则保留原样（防指向错误表）、其余按配置剥离。逐源可覆盖
（`spring.datasource.<name>.table-prefix-map`），未配置的附加源继承默认源。

实现细节（表位置改写算法、词法扫描/状态机、边界与已知限制）见 **docs/agents/table-prefix.md**。

## 内嵌 Mapper（go:embed）

单文件二进制部署时无需把 XML 解出到临时目录：`orm.RegisterMapperFS` 直接读取 `embed.FS`，
在内存中解析 Mapper（不落盘、无临时文件清理负担）。

```go
//go:embed resources/mapper/*.xml
var mapperXML embed.FS

func init() {
    // 目录递归加载：与 mybatis.mapper-locations 语义一致
    if err := orm.RegisterMapperFS(mapperXML, "resources/mapper"); err != nil {
        log.Fatalf("load mapper failed: %v", err)
    }
    orm.RegisterModel(new(model.SysUser))
    orm.RegisterMapper(new(mapper.SysUserMapper)) // XML 未就绪时先注册也不报错
}
```

要点：

- **pattern 可为目录或单个 `.xml` 文件**，可一次传入多个（`RegisterMapperFS(fsys, "m/a", "m/b.xml")`）；
  目录递归收集全部 `.xml`，`mybatis-config.xml` 等无 `namespace` 的非 Mapper XML 自动跳过。
- **`embed.FS` 之外的只读文件系统同样可用**（任何 `io/fs.FS`，含 `fstest.MapFS`），便于测试注入。
- **加载顺序无关**：`RegisterMapper` 在 XML 已就绪时会自动重新绑定；若先注册结构体、后加载 XML，
  调用一次 `orm.ReloadMappers()` 即可补绑。
- **替换语义**：`RegisterMapperFS` 会替换当前全部 SQL 定义（与 `Initialize` 一致）。

需要「内嵌为基础 + 磁盘覆盖」时用 `RegisterMapperSources`，后列来源覆盖同 `namespace` 的 Mapper
（便于线上热修 SQL 而不重新发版）：

```go
err := orm.RegisterMapperSources(
    orm.MapperSource{FS: mapperXML, Patterns: []string{"resources/mapper"}}, // 内嵌
    orm.MapperSource{Patterns: []string{"override/mapper"}},                 // 磁盘覆盖（可选）
)
```

底层解析入口在 `types` 包，可脱离 orm 单独使用：`types.NewSqlMappersFrom(fsys, patterns...)`
与 `types.NewSqlMappersFromSources(sources...)`。

## 注入自定义 DB / DSN

框架支持注入自定义连接池（`*sql.DB` 或任意实现 `orm.ConnPool` 接口的对象）与自定义 DSN，适用于连接代理、测试桩、已有连接池复用等场景。

### 注入自定义连接池

通过 `orm.Open(cfg)` 拿到 `*orm.DB` 后自行接入（配合 `orm.InitializeDataSources` 或多数据源管理）：

```go
cfg := orm.NewConfigFromSettings(cm) // 或 orm.NewConfig("application.properties")
cfg.ConnPool = myDB   // 注入自定义连接池（*sql.DB 或实现 orm.ConnPool 的对象），跳过 DSN 建连
cfg.DSN = "root:pwd@tcp(10.0.0.1:3307)/db?parseTime=true&charset=utf8mb4&loc=Local" // 自定义 DSN（可选）
db, err := orm.Open(cfg)
```

- `cfg.ConnPool` 非空时优先使用注入的连接池，不再按 DSN 新建连接
- `cfg.DSN` 非空时优先于自动生成的 DSN（各方言 dialector 均支持）
- 预编译缓存包装（`PreparedStmtDB`）对注入的连接池同样生效

## 性能优化

框架在执行热路径上做了以下优化（详见 `TODO.md`）：

- **预编译语句缓存**：参数化 SQL 按文本缓存 Prepared Statement，数据库不再重复解析 + 生成执行计划（见上方配置项说明）
- **占位符方言转换**：PostgreSQL/KingbaseES 的 `?` 自动转为 `$n`，MySQL/SQLite 原样保留；仅在存在参数时转换，避免误伤无参 SQL 中的字面量 `?`
- **扫描目标复用**：结果集扫描目标（`sql.NullXxx` 指针）每次查询只分配一次、跨行复用，替代逐行分配
- **反射预编译**：列值转换函数表（`convertFn`）与 resultMap 的 property→字段索引映射在查询开始时预编译一次，行循环内直接调用，避免每行每列 `ScanType` switch 分派与 `FieldByName` O(N) 名称匹配
- **无参 SQL 生成缓存**：无参 SQL 的拼接结果静态不变，首次生成后缓存复用
- **大结果集流式读取**：`QueryStream` / Mapper 流式 select 逐行消费（内存 O(1)），配合全局行数上限（P4-3，默认 10000 行）兜底截断，避免大结果集 OOM / 拖垮连接

## 代码生成

### generator — 从 XML Mapper 生成 Go 代码

```bash
go build -o generator cmd/generator/main.go
./generator -p mypackage -d temp -m resources/mapper
```

参数说明：
- `-p` 包名（默认 `temp`）
- `-d` 输出目录（默认 `temp`）
- `-m` XML Mapper 文件目录（默认 `resources/mapper`）

### schema2code — 从数据库表结构生成代码

```bash
go build -o schema2code cmd/schema2code/main.go
./schema2code -type mysql -host localhost -port 3306 -username root -password 123456 -db mydb -output temp
# MyBatis-Plus 内置 CRUD：加 -mp 生成 BaseMapper 标准方法名（insert/deleteById/updateById/selectById/selectList/selectOne/selectPage/selectCount/selectBatchIds/deleteBatchIds）
./schema2code -type postgres -host localhost -port 5432 -username root -password 123456 -db mydb -prefix sys_ -tables sys_user -output temp -mp
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

### xml2go — 从 XML Mapper 逆向生成 Go 代码

```bash
go build -o xml2go cmd/xml2go/main.go
# 入口一：已知 Mapper XML 目录
./xml2go -m resources/mapper -d gen -p github.com/xxx/app
# 入口二：直接给 MyBatis/MyBatis-Plus 配置文件（先解析 mapper 位置再生成）
./xml2go -c src/main/resources/application.yml -d gen -p github.com/xxx/app
```

参数说明：
- `-c` mybatis/mybatis-plus 配置文件（.properties/.yml/.yaml/.xml），先从配置解析 Mapper XML 位置（`mybatis[-plus].mapper-locations`、`<mappers><mapper resource/>`、`<property name="mapperLocations">`；支持 `classpath*:` 前缀、`*`/`**` 通配、逗号多值、`config-location` 链式解析）再生成；设置时 `-m` 被忽略
- `-m` XML Mapper 目录（默认 `resources/mapper`）
- `-d` 输出目录（默认 `gen`，内含 `models/`、`mapper/` 两个子目录）
- `-p` 输出目录自身的模块导入路径前缀（生成 import 为 `<prefix>/models` 等；留空退化为裸导入，GOPATH 兼容）
- `-skip-mp` 可选，跳过 MyBatis-Plus 内置 CRUD 字段
- `-v` 可选，打印跳过语句及原因

生成代码用法与手写 Mapper 一致（`mapper.GetSysUserMapper().SelectUserByUserName("admin")`）；完整签名推导规则与已知边界见 **docs/agents/xml2go.md**。

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
- 事务：`orm.Begin()` 开启后 Mapper 方法自动在事务内执行，`Commit()` / `Rollback()` 结束事务（见「事务」章节）

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

- **v0.3.10（xml2go 配置文件直入，2026-09-18）**：`cmd/xml2go -c` 直接给 MyBatis / MyBatis-Plus 配置文件生成 Go 代码 —
  - **`types.LoadMappersFromMyBatisConfig`（`types/mybatis_config.go`）**：从配置文件解析 Mapper XML 位置并加载，返回解析明细（`MyBatisConfigInfo{ConfigFile, BaseDir, Locations, XmlFiles}`）；支持三种配置形态——① Spring Boot `.properties`：`mybatis[-plus].mapper-locations`（兼容驼峰 `mapperLocations`）；② `.yml`/`.yaml`：`mybatis`/`mybatis-plus` 节点下 `mapper-locations`（标量/行内列表/块列表）；③ `.xml`：mybatis-config.xml（`<mappers><mapper resource|url/>`）与 Spring/MP Spring XML（`<property name="mapperLocations" value|<list><value>`）；仅 `config-location` 时链式解析（最深 3 层，resource 仍按最外层配置的 classpath 根）
  - **位置展开 `ExpandMapperLocations`**：`classpath*:`/`classpath:`/`file:` 前缀剥离；相对路径以配置文件所在目录为 classpath 根，未命中回退父/祖父目录（兼容「配置在 mybatis/ 子目录、XML 在资源根」布局）；`*`/`?`/`**` 通配（`**` 跨目录层级，自实现 rune 级 glob→正则 + 静态根 WalkDir，Windows 盘符卷名正确处理）；逗号/分号多值；结果仅保留 .xml、去重排序
  - **命令 `cmd/xml2go -c`**：配置文件入口优先于 `-m`；打印「config → N 位置模式 → M 个 XML」解析摘要
  - **依赖**：`go.yaml.in/yaml/v3 v3.0.4` 由 indirect 提升为直接依赖（已在依赖图中，无新增下载）
  - **端到端测试 `types/mybatis_config_test.go`**：8 用例（properties 驼峰键 / yaml 块列表+`**` 递归 / mybatis-config.xml resource+file URL / Spring XML `<list>` / config-location 链式 / 不支持扩展名与空解析错误分支 / 父目录回退）+ CLI 冒烟（模拟 Java 工程 `src/main/resources/application.yml` + samples XML → 22 mappers / 331 functions，与 `-m samples` 一致）；全量 build/vet/test 通过
- **v0.3.9（xml2go 逆向代码生成，2026-09-18）**：从既有 MyBatis / MyBatis-Plus XML 全量语句生成可直接编译运行的 Go 代码 —
  - **引擎 `types.GenerateXml2GoFiles`（`types/xml2go_gen.go`）**：输入 `SqlMappers`（免连库），输出 `<output>/models/<短名>.go`（模型 struct：`db:"<列名>,pk/logic/version"` 元数据 tag——id/result 列 `,pk`、逻辑删除列 deleted/del_flag `,logic`、版本列 `,version`，association/collection 记 `db:"-"`；json tag；条件 `import "time"`；`TableName()` 方法）+ `<output>/mapper/<Mapper短名>.go`（`orm.BaseMapper` 内嵌 + 函数字段 + `init()` 注册模型与 Mapper + `sync.Once` 单例 `GetXxxMapper()`）；跨 Mapper 模型去重（首个 resultMap 生效）、Mapper 短名/函数字段名冲突跳过、引用未生成模型的语句跳过并记录原因（产物恒可编译）；全部产物 gofmt + `go/parser` 语法校验
  - **签名推导与 orm 注册期校验逐条对齐**：struct 参数为指针 `*models.X`（P1 ID 回填/自动填充要求可寻址）；无 parameterType 语句按占位符逐个生成具名 `interface{}` 参数（与运行期按位绑定对齐），含 `<foreach>` 时退化为单 `map[string]interface{}`；`parameterType="map"` → map 参数；基础类型 + foreach → 切片参数；MP `insertBatch` → `[]*models.X`；`selectPage` 固定 `*orm.PageParam` / `*orm.Page`；返回类型——resultMap → `[]models.X`、resultType=map/未注册 pojo → `[]map[string]interface{}`、resultType=业务模型且已生成 → `[]models.X`（M-04 运行期转换）、resultType 标量 → `[]标量`、DML → `int64`
  - **命令 `cmd/xml2go`**：`-m` mapper 目录（默认 resources/mapper）`-d` 输出目录（默认 gen）`-p` 输出目录导入路径前缀 `-skip-mp` 跳过 MP 内置方法 `-v` 打印跳过原因；逐 Mapper 一行统计 + 总计
  - **端到端测试 `types/xml2go_gen_test.go`**：4 用例（基础生成内容断言：db tag/指针参数/逐槽 AutoDerive 参数/MP 内置/init 注册/getter；SkipMP；SkipUnknownModel；samples 真实回归：20 个模型文件、22 个 mappers、331 个函数、0 跳过）+ 临时模块端到端类型检查（生成产物 + replace 指向本仓库 `go build ./...` 通过）+ 注册链路冒烟（import 触发全部 init，`orm.GetModelInfo` 元数据校验）
  - **文档**：docs/agents/xml2go.md（选型对比 / 运行方式 / 签名推导规则 / 验证步骤 / 已知边界）+ cmd/xml2go/README.md；README / AGENTS.md / docs/agents/commands.md 同步
  - **已知边界**：跨 Mapper 嵌套 resultMap 引用（`resultMap="XxxMapper.YyyResult"`）静态不可解析，字段退化为 `interface{}`（运行期仍按全局 resultMap 正常转换）；`${}` 注入占位符生成具名 `interface{}` 参数；与 sqlc 互补——动态语句/MP CRUD 选 xml2go，纯静态查询选 sqlc
- **v0.3.8（P1 框架增强七件套，2026-09-17）**：MyBatis-Plus P1 级全部 7 项功能落地（见 docs/mybatis-plus-golang-checklist.md「P1」节）—
  - **P1-1 乐观锁**：`db:"version,version"` tag 解析入 `TableStructure.VersionColumn`；MP 内置 `updateById` 自动生成 `version=version+1` SET + `AND version=#{version}` WHERE（`ensureMPBuiltinCRUD` 检出 VersionColumn 时置 `SqlFunction.HasVersion`）；CAS 检测——UPDATE 受影响行数为 0 时返回 `orm.ErrOptimisticLock`；`fill:insert` 列（如 create_time）不参与 update SET
  - **P1-2 自动填充**：`orm.RegisterFillHandler(column, FillInsert|FillUpdate|FillInsertUpdate, fn(ctx, field))` 注册列级填充器，insert/update 在 GenerateSQL 前按 `db:"...,fill:insert|insert_update"` 标记注入（需指针接收者 entity）；手写 XML 由 `ColumnStructure.Fill` 驱动
  - **P1-3 二级缓存**：`mybatis.configuration.cache-enabled`（默认 false）+ `local-cache-size`（默认 1024）+ `local-cache-ttl`（默认 3600s）；`orm/namespace_cache.go` namespace 级 LRU+TTL 缓存，SHA256(namespace.sqlID+args) 复合键；SELECT 命中直返，同 namespace 任意 DML 后整仓 flush（含 RETURNING 分支）；语句级开关 `useCache="false"` 跳过；`orm.SetCacheEnabled(on)` 运行时开关；`golang-lru/v2` 转 direct 依赖
  - **P1-4 批量操作**：MP 内置 CRUD 新增 `insertBatch`（多行 VALUES `<foreach collection="list">`）；`orm.InsertBatch(ctx, insertFn, entities)` 反射透传切片；`orm.UpdateBatch(ctx, updateFn, entities, batchSize)` 事务内分批循环（默认批 50，失败整体回滚）
  - **P1-5 `<discriminator>` 鉴别器**：`<resultMap>` 内 `<discriminator column="...">` + `<case value="..." resultMap="..."/>` 解析（`DiscriminatorCase`，子 resultMap 懒解析 + 首次命中缓存）；运行时 `resolveDiscriminator` 按行取鉴别列值匹配 case，切换子 resultMap 完成列映射（Go 无多态切片，限定同一 Go struct 类型）
  - **P1-6 ID 生成策略**：`mybatis.configuration.id-type=snowflake|uuid|assign_id`（默认空=不自填）+ `snowflake-worker-id`；insert 前主键为零值时自动填充（需指针接收者 entity）；`orm.RegisterIdGenerator(name, fn)` 自定义策略 + `orm.SetDefaultIdType(name)`；Snowflake 自实现零新增依赖，UUID 复用 google/uuid
  - **P1-7 SQL 安全防护**：`mybatis.configuration.safe-update=true`（默认 false）后 UPDATE/DELETE 无 WHERE 子句直接返回 `orm.ErrSafeUpdateBlocked`；词法检测复用表前缀 `tokenizeSQL`（不误判注释/字符串字面量/列名含 where）；`orm.SetSafeUpdate(on)` 运行时开关
  - **端到端测试**：`orm/sqlite_p1_test.go` 12 用例（安全防护开/关 + 词法、Snowflake/UUID、乐观锁 CAS、insert/update 自动填充、insertBatch、discriminator video/audio 分流、二级缓存命中/失效、缓存默认关闭回归）；全量 `go build` / `go vet` / `go test` 通过
- **v0.3.7（Go 版本固定 1.24.0，2026-09-17）**：工具链与依赖对齐 —
  - go.mod 由 `go 1.25.0` 降回 `go 1.24.0`（本机/CI 工具链即 go 1.24.0，此前 GOTOOLCHAIN=auto 自动切换到 1.25+ 属非预期）
  - 依赖降级对齐（更高版本要求 go ≥ 1.25）：go-mssqldb v1.11.0→v1.9.7、clickhouse-go v2.48.0→v2.42.0（连带 ch-go v0.69.0、otel 1.39.0、x/sys 0.39.0 等 indirect 重解析）；`go mod vendor` 重新生成
  - 项目约定固化：AGENTS.md 硬性约定 + docs/agents/conventions.md「Go 版本」节（依赖的 go 要求必须 ≤ 1.24.0，超出降级选版；建议 GOTOOLCHAIN=local）
  - 全量 build / vet / test 在 go 1.24.0（GOTOOLCHAIN=local）下通过
- **v0.3.6（G0 事务上下文改造 + G1 gormish 链式 API + S1 Querier 代码生成，2026-09-17）**：GORM/sqlc 两条路线首期落地（见 docs/可行性报告-gorm-sqlc.md）—
  - **S1 sqlc 风格 Querier 代码生成（2026-09-17）**：XML 存量静态 select 抽取 → 类型安全 Querier（sqlc A 方案，见 §3.4/§5.1）—
  - **静态性判定 `types/sqlfragment.ExtractStaticSQL`**：`ExtractStaticSQL(items) (sql, params, ok)`——仅 simpleSql 与纯文本 `<include>` 判静态（include 含动态标签则整句跳过）；`#{}`→`?` 按出现顺序（与 `PrepareSqlWithMap` 逐个 Replace 语义一致），`StaticParam{Name, JdbcType}` 保序返回；`${}`（raw 注入）/6 类动态标签（if/foreach/choose/where/set/trim）/空片段判非静态
  - **生成器 `types.GenerateQuerierFiles(dir, pkg)`**：一个 mapper 一个 `<namespace短名去Mapper后缀>_querier.go`，产出 model（resultMap 按 jdbcType 定型，`db:"column,pk"` tag，association/collection 记 `db:"-"`）+ `XxxQuerier` 接口 + `NewXxxQuerier(db *orm.DB)` 构造（db 为 nil 时取 `orm.GetActiveDataSource()`）+ 实现；**免连库**（XML 元数据双轨定型：resultMap jdbcType / resultType 声明类型）；select 恒返回切片（`[]短名` / `[]map[string]interface{}` / `[]标量`）；参数定型：显式 parameterType 标量→单参绑定全部占位符，无 parameterType→按占位符名生成具名参数（jdbcType 定型，未知类型回退 interface{}），parameterType 为 struct/map/slice→跳过；动态语句跳过并记录原因（`QuerierSkip`/`QuerierGenStats`，`-v` 查看）；MP 内置静态方法（selectById/selectOne/selectList/selectCount）纳入生成，selectPage/selectBatchIds 跳过；生成后经 gofmt + `go/parser` 语法校验
  - **命令 `cmd/sqlc`**：`-p` 输出包名（默认 querier）`-d` 输出目录（默认 querier）`-m` mapper 目录（默认 resources/mapper）`-v` 打印跳过原因；逐 mapper 一行统计 + 总计
  - **生成代码运行时**：走 `(db *DB) QueryToContext`（方言占位符转换 / 表名前缀改写 / ctx 携带 WithTx 事务全部由 orm 层承担）；直通道随 S1 扩展两类 dst 形态：`*[]T`（T 为标量，单列）与 `*[]map[string]interface{}`（全行转 map），均遵循 DefaultRowLimit
  - **端到端测试**：`types/sqlfragment/static_extract_test.go` 8 用例（含 include 展开/动态标签/raw 注入/多片段拼接）+ `types/querier_gen_test.go` 5 用例（生成内容断言/手写全动态回退/samples 真实回归：22 mappers → 121 个静态函数全部语法合法）+ `orm/s1_test.go` 3 用例（SQLite 真实执行：无参/单参/多占位符按位绑定/标量 count/生成文件签名/WithTx 事务内 ctx 路由提交前后可见性）
  - **G1 gormish 链式 API（2026-09-16）**：GORM 风格链式查询/CRUD 最小可用集（见 docs/可行性报告-gorm-sqlc.md §5.1）—
  - **新增 `orm/gormish` 包（单向依赖 orm）**：session.go（`Open`/`Use(name)`/`New(ormDB)` 构造、`WithContext`、`Transaction`、`Table`/`Model`）、query.go（`Select`/`Where`/`Or`/`Order`/`Limit`/`Offset`/`Group`/`Having`/`Distinct`/`Raw` 链式子句 + `Find`/`First`/`Last`/`Count`/`Scan` finisher）、crud.go（`Create`/`Update`/`Updates`/`Delete`/`Exec`）、builder.go（SQL 组装 + 无 tag 模型本地元数据回退）；链式方法克隆 statement（写时复制切片，多克隆互不污染），finisher 返回 `(value, error)` 双返回值（贴合项目约定，不采用 GORM 式 db.Error 链式错误）
  - **语义要点**：条件一律 `?` 占位 + 参数绑定（经 orm 执行层自动做方言占位符转换与表名前缀改写，与 QueryWrapper 值内联严格分界）；`Where` 支持 string / map 等值 / `"col IN ?"` + slice 自动展开；`First`/`Last` 未显式排序时按主键排序 LIMIT 1（无行返回 `sql.ErrNoRows`）；`Find` 未指定 Table/Model 时按 dst 元素类型推导表名；`Create` 零值主键视为自增列跳过并回填（`LastInsertId`，PG 族等 `NeedsReturning` 方言经 `RETURNING` 取回），批量插入多行 VALUES 不回填；`Updates` struct 跳零值字段（主键/逻辑删除/乐观锁列不参与 SET，GORM 语义）；`Update`/`Updates`/`Delete` 无 WHERE 条件返回 `ErrMissingWhere` 防全表；`Transaction` 基于 `orm.WithTx`（嵌套复用外层事务、不占 TCC 槽）；结果扫描复用 G0 `QueryToContext` 直通道（db tag 列名 / TypeHandler / DefaultRowLimit 自动生效）；G1 不做（归 G2）：软删除自动改写、生命周期 hooks、AutoMigrate、savepoint、Save
  - **orm 侧对接面**：`(db *DB) QueryToContext` 实例级扫描直通道（包级 `QueryToContext` 转发 gDbConn，行为不变）、`orm.GetActiveDataSource()` 活跃数据源访问器
  - **端到端测试 `orm/gormish/gormish_test.go`（SQLite，20 个用例）**：链式子句/分页 Limit+Offset/推导表名/First+Last/Count（含 DISTINCT）/Where 三变体+Or+IN 展开/Group+Having/主键回填+显式主键/批量插入（值切片+指针切片）/Update+Updates（map+struct 跳零值）/Delete（条件+模型主键）/事务提交+回滚+嵌套复用/WithContext 参与 WithTx/Raw+Scan+Exec/克隆隔离/无 tag 模型回退（TableName 方法 + 短名推导）/Model 绑定
  - **G0 前置改造（2026-09-16）**：事务上下文改造 + 强类型扫描直通道（gorm/sqlc 两条路线的共同地基，见 docs/可行性报告-gorm-sqlc.md §4）—
  - **回调式事务 `orm.WithTx(ctx, fn)`（G0 §4.1）**：在独立事务中执行 fn，返回 nil 提交 / 返回 error 回滚并透传 / panic 回滚后重新抛出；ctx 已携带事务时嵌套调用直接复用外层事务（边界由最外层决定）；**不占用 TCC 全局事务槽**（`Begin/Commit/Rollback` 旧语义零改动，多 goroutine 可各自 WithTx 互不冲突）；`orm.TxFromContext(ctx)` 提取 ctx 事务；`DB.WithTx` 支持多数据源实例；执行层 `ExecContext/QueryContext/QueryRowContext` **ctx 携带事务优先于全局槽**
  - **Mapper 方法 `context.Context` 参数（G0）**：Mapper 函数字段声明 `context.Context` 参数即被自动提取并透传（模式同 PageParam/QueryWrapper 参数提取，不影响 SQL 参数绑定），方法内 SQL 自动加入 ctx 携带的 WithTx 事务；Hook 的 `HookContext.Ctx` 同步接收真实 ctx；无 ctx 参数的既有 Mapper 行为完全不变
  - **强类型扫描直通道 `orm.QueryTo` / `QueryToContext`（G0 §4.2）**：rows → struct 直通道（不经 `[]map[string]interface{}` 中转）：扫描缓冲与「列 → 字段索引」一次预编译，每行 Scan 后直接写入目标字段；dst 支持 `*struct`（首行，无行返回 `sql.ErrNoRows`）/ `*[]struct` / `*[]*struct`（全行，遵循 `DefaultRowLimit`）/ `*map[string]interface{}` / 标量指针（单列）；列名匹配优先 `db` tag 显式列名（含非蛇形改名），回退 `RowStream.Scan` 同款四策略；类型转换经 `convertFieldValue`（自定义 TypeHandler 优先）；列级失败聚合为一条错误（M-05 精神）；是 gormish 链式 API（Find/Scan）与 sqlc 生成代码运行时的地基
- **v0.3.5（P0 框架扩展五件套，2026-09-16）**：动态 SQL 引擎独立 + 四大扩展点 —
  - **动态 SQL 引擎独立（P0-3）**：动态 SQL 片段引擎整体提取至 `types/sqlfragment` 独立子包（`Node` 接口 + `containerBase` + 节点注册表 + `ParseFragments` 遍历器入口），7 个节点（simple/if/foreach/choose/where/set/include）经 `init()` 注册、开放封闭可扩展，`types` 侧 bridge 别名保持旧 API 零改动兼容；新增 `<trim>` 通用前后缀节点（`prefix`/`prefixOverrides`/`suffix`/`suffixOverrides`，`<where>`/`<set>` 即其特例，与 MyBatis 行为等价性测试覆盖）
  - **自定义 TypeHandler（P0-4）**：`orm.RegisterTypeHandlerFor[T](fn)` / `orm.RegisterTypeHandler(reflect.Type, TypeHandler)`；扫描转换统一经 `convertFieldValue` 入口，覆盖参数绑定（写库）、resultMap 映射、无 resultMap schema 推断与流式查询（`RowStream.Scan`）三路径
  - **struct `db` tag 元数据驱动 MP 内置 CRUD（P0-2）**：`orm.RegisterModel` 解析 `db` tag（`col` 列名 / `pk` 主键 / `logic` 逻辑删除 / `version` 乐观锁 / `fill:insert|update|insert_update` 填充标记 / `-` 排除）与 `TableName() string` 方法入 `ModelInfo` 缓存，经 `SetModelStructureProvider` 注入 types 包；XML 含 resultMap 且缺 MP 内置方法时按元数据精确生成（表名/主键/逻辑列以 tag 为准优先于 resultMap 推导）；显式逻辑列按 Go 类型取值（bool→true/false、string→'1'/'0'、数值→1/0），`isLogicColumn` 统一排除出 resultMap/insert/update 列清单，select 自动过滤、delete 自动改写 UPDATE
  - **Hook 拦截链（P0-5）**：`orm.RegisterHook(HookBeforeExecute|HookAfterExecute, hook)` / `orm.ClearHooks()`；`HookContext` 含 Namespace/SqlId/SqlType/SQL/Args（Before 可改写 SQL，After 另有 Error/Duration/Cost）；覆盖 `executeMethod`/`executeStream`/`executePage` 全部执行路径（含 PG RETURNING 分支与分页 COUNT）；Hook panic 自动 recover + 日志，不中断执行链
  - **QueryWrapper 条件构造器（P0-1）**：`orm.NewQueryWrapper()` 链式条件（Eq/Ne/Gt/Ge/Lt/Le、Like/NotLike/LikeLeft/LikeRight、Between/NotBetween、IsNull/IsNotNull、In/NotIn、Or/Nested/Apply、GroupBy/Having/OrderByAsc/OrderByDesc/Last/Select）；mapper 方法带 `*orm.QueryWrapper` 参数自动注入（core/tail 切分算法，Wrapper 条件插在原尾部子句之前、GroupBy/Having/OrderBy 追加其后、Last 恒置末尾；分页场景先注入后派生 COUNT）；XML 显式 `${ew}` 占位（经 `sqlfragment.SQLSegmentProvider` 接口，`#{ew}`/`${ew}` 两路径均可）；空 Wrapper 原样执行；P0 限定 select 携带 Wrapper；使用说明见 docs/agents/mybatis-plus.md §8
- **v0.3.4（P4 国际主流数据库 MSSQL/Oracle/DB2/ClickHouse 适配）**：P4 四款数据库适配 — ① MS SQL Server：独立 `MssqlDialector`（MSSQL 方言族），占位符 `?`→`@p1/@p2`…，`NeedsReturning` 返回 false，`INFORMATION_SCHEMA` 表结构查询，分页 `OFFSET…ROWS FETCH NEXT…ROWS ONLY`；新增 `DatabaseFamily.MSSQL` 族、`PlaceholderStyle.PlaceholderAtP`；JDBC URL 支持 `jdbc:sqlserver://`；`ParseDatabaseType` 支持 `mssql`/`sqlserver`/`mssql-server` 别名；`GetDriverName` 返回 `sqlserver`；新增 `cmd/mssqldemo` 示例；`application-mssql.properties` 配置模板；用户需引入 MSSQL 驱动（`_ "github.com/microsoft/go-mssqldb"`）。② Oracle：独立 `OracleDialector`（Oracle 方言族），占位符 `?`→`:1/:2`…（`PlaceholderColon`），`NeedsReturning` 返回 false，`ALL_TAB_COLUMNS` + `ALL_CONSTRAINTS` + `ALL_COL_COMMENTS` 表结构查询，分页 `OFFSET…ROWS FETCH NEXT…ROWS ONLY`（需 Oracle 12c+），`EffectiveSchema` 返回 `UPPER(Username)`，DSN 格式 `user/password@host:port/dbname`（EZConnect）；JDBC URL 支持 `jdbc:oracle://`；`ParseDatabaseType` 支持 `oracle`/`oracledb`/`oracle-db` 别名；`GetDriverName` 返回 `oracle`；框架 blank import go-ora/v2，无需额外引入驱动；新增 `cmd/oracledemo` 示例；`application-oracle.properties` 配置模板。③ IBM DB2：独立 `Db2Dialector`（DB2 方言族），占位符 `?`→`:1/:2`…（`PlaceholderColon`），`NeedsReturning` 返回 false，`SYSCAT.COLUMNS` + `SYSCAT.KEYCOLUSE` + `SYSCAT.INDEXES` 表结构查询，分页 `OFFSET…ROWS FETCH NEXT…ROWS ONLY`（需 DB2 10.1+），`EffectiveSchema` 返回 `UPPER(Username)`，DSN 格式 `HOSTNAME=host;PORT=port;DATABASE=dbname;UID=username;PWD=password`；JDBC URL 支持 `jdbc:db2://`；`ParseDatabaseType` 支持 `db2`/`ibmdb2`/`db2-luw` 别名；`GetDriverName` 返回 `go_ibm_db`；新增 `cmd/db2demo` 示例；`application-db2.properties` 配置模板；用户需引入 DB2 驱动（`_ "github.com/ibmdb/go_ibm_db"`，需安装 DB2 ODBC/CLI 客户端库）。④ ClickHouse：独立 `ClickHouseDialector`（ClickHouse 方言族），占位符保持 `?`（`PlaceholderQuestion`），`NeedsReturning` 返回 false（ClickHouse 无 RETURNING 支持），`system.columns` + `system.tables` 表结构查询，分页 `LIMIT n OFFSET m`，`EffectiveSchema` 返回数据库名，DSN 格式 `clickhouse://user:password@host:port/dbname`；新增 `DatabaseFamily.ClickHouse` 族；JDBC URL 支持 `jdbc:clickhouse://`；`ParseDatabaseType` 支持 `clickhouse`/`click-house` 别名；`GetDriverName` 返回 `clickhouse`；框架 blank import clickhouse-go/v2，无需额外引入驱动；新增 `cmd/clickhousedemo` 示例；`application-clickhouse.properties` 配置模板；ClickHouse 是列式 OLAP 引擎，不支持事务，UPDATE/DELETE 为异步 mutation，ORM 适用于批量插入 + 查询分析场景。
- **v0.3.3（国产数据库 P3 适配，2026-09-15）**：P3 Informix 方言族 GBase 8s 适配 — GBase 8s（南大通用）：Informix 兼容，独立 `GBase8sDialector`（Informix 方言族），占位符保持 `?`，`NeedsReturning` 返回 false，`SYSCOLUMNS`+`SYSTABLES` 表结构查询，`EffectiveSchema` 返回 `UPPER(Username)`，DSN 格式 `user:password@host:port/dbname`；JDBC URL 支持 `jdbc:gbase8s://` 格式；新增 `DatabaseFamily.Informix` 族；`cmd/gbase8sdemo` 示例；`application-gbase8s.properties` 配置模板；用户需引入 Informix 驱动（`_ "github.com/alexgarrec/go-informix"`）；dialector 单元测试覆盖占位符格式、`NeedsReturning`、DSN 生成、`ParseDatabaseType`（含 `gbase`/`gbase-8s` 别名）、`Family` 映射、`GetDriverName`、`EffectiveSchema`
- **v0.3.2（国产数据库 P0+P1+P2 适配，2026-09-15）**：国产数据库三阶段适配 — ① P0 MySQL 兼容族三款零成本适配：TiDB（PingCAP，MySQL 协议高度兼容，复用 `go-sql-driver/mysql`，默认端口 4000，框架自动以 `tidb` 名称注册驱动）、TDSQL（腾讯云，MySQL 兼容）、PolarDB-MySQL（阿里云，MySQL 兼容）；三款 dialector 均内嵌 `MySqlDialector`，JDBC URL 支持 `jdbc:tidb://`/`jdbc:tdsql://`/`jdbc:polardb://` 格式；`cmd/tidbdemo`/`cmd/tdsqldemo`/`cmd/polardbdemo` 示例。② P1 PostgreSQL 兼容族四款低难度适配：openGauss（华为开源）、GaussDB（华为云）、HighGo DB（瀚高）、Vastbase（海量数据）；四款均复用 `lib/pq` 驱动（框架自动注册），内嵌 `PostgresDialector`，JDBC URL 支持 `jdbc:opengauss://`/`jdbc:gaussdb://`/`jdbc:highgo://`/`jdbc:vastbase://` 格式；`cmd/opengaussdemo`/`cmd/gaussdbdemo`/`cmd/highgodemo`/`cmd/vastbasedemo` 示例。③ P2 Oracle 方言族三款适配：OceanBase-MySQL（MySQL 模式零成本适配，框架自动以 `oceanbase` 名称注册驱动）、OceanBase-Oracle（Oracle 模式，独立 `OceanBaseOracleDialector`，占位符 `?`→`:n`，用户需引入 `go-oceanbase-driver`）、DM8/达梦（自有协议 Oracle 风格，独立 `DamengDialector`，占位符 `?`→`:n`，用户需引入 `dmdb.com/dm` 驱动）；新增 `DatabaseFamily.Oracle` 族：`EffectiveSchema` 返回 `UPPER(Username)`，DSN 格式 `user/password@host:port/dbname`；JDBC URL 支持 `jdbc:oceanbase://`/`jdbc:oceanbase-oracle://`/`jdbc:dameng://` 格式；`cmd/oceanbasedemo`/`cmd/oboracledemo`/`cmd/damengdemo` 示例；配置模板与 dialector 单元测试全覆盖
- **v0.3.1（dialector 重构，2026-09-14）**：将数据库方言从 `orm/` 根目录提取到 `orm/dialector/` 子包 — `mysql.go`/`postgres.go`/`sqlite.go`/`kingbase.go`/`types.go`/`base.go` 独立子包，`BaseDialector` 公共基类封装连接参数/DSN/驱动名/占位符风格等通用逻辑；`orm/interfaces.go` 导出常量（`PostgresDb`/`MySqlDb`/`SqliteDb`/`KingbaseDb` + `FamilyPostgres`/`FamilyMySQL`/`FamilySQLite`/`FamilyOracle`/`FamilyInformix`）；原有 `mysql_dialector.go`/`postgres_dialector.go`/`sqlite_dialector.go`/`kingbase_dialector.go` 删除；`NewForType` 工厂函数统一路由；回归测试全绿
- **v0.3.0（优化改造，2026-09-14）**：共享 Java Mapper XML 零改造接入 + 多环境前缀切换免改 XML
- **v0.2.2（内嵌 Mapper，2026-09-10）**：支持 `go:embed` 内嵌 Mapper 加载 — 新增 `orm.RegisterMapperFS(fsys, patterns...)` / `orm.RegisterMapperSources(sources...)` / `orm.ReloadMappers()`，XML 可随二进制一起分发，部署时无需携带 `resources/mapper` 目录；底层 `types` 包新增 `NewSqlMappersFrom(fsys, patterns...)` / `NewSqlMappersFromSources(...)` 与 `MapperSource{FS, Patterns}`（`FS == nil` 保持原有磁盘语义）；磁盘 pattern 保持「目录递归 / 单文件精确」语义，内嵌 pattern 用 `/` 分隔并兼容误写 `\`；`<configuration>` 根或 namespace 缺失的 XML 自动跳过；同一 namespace 后注册的源覆盖先注册的源（磁盘可覆盖内嵌）；注册与初始化顺序解耦（先 `RegisterMapperFS` 后 `Initialize` 亦可，反之用 `ReloadMappers()` 补绑定）；顺带修复空 `mybatis.mapper-locations` 会退化为扫描当前目录 XML 的问题；`mybatis.mapper-locations` 与 `NewSqlMappers(dir)` 行为完全不变（回归测试 `Test_NewSqlMappers_DiskUnchanged`）；端到端覆盖 `orm/embed_fs_test.go` + `types/embed_fs_test.go`（SQLite 全流程、MapFS、多源覆盖、非法 XML）
- **v0.2.1（补丁，2026-09-04）**：MySQL 文本列整列丢失修复 — go-sql-driver/mysql 在 `parseTime=true` 下对 VARCHAR/TEXT/CHAR 列报告 `ScanType() = sql.NullString`，`resolveConverter`（`orm/common.go`）缺少该分支导致 `createMapWithConverters` 静默跳过该列，查询结果中**所有字符串字段为空**（id/时间/数字正常）；补上 `sql.NullString → convertSqlString2String` 分支后全字段正常返回（该缺陷自 v0.1 系列即存在，SQLite/MySQL/PG 三 Demo 验证中发现）
- **v0.2.0（优化改造，2026-09-04 起）**：共享 Java Mapper XML 零改造接入 + 多环境前缀切换免改 XML
  - **参数需求由语句占位符自动推导（1.1/M-06）**：`<select|insert|update|delete>` 含 `#{}`/`${}` 或 `<foreach>` 即自动识别为需要参数（无需 `parameterType`），有参函数 + 无 `parameterType` 语句不再注册失败（samples/RuoYi 共享 XML 直接可用）；`<if>/<choose>` 条件体占位符不视为必参（保持静态无参调用契约）
  - **多参数按占位符按位绑定（1.2）**：`SelectTableColumns(#{schema},#{tableName})` 传入两个实参时按语句占位符名/位置分别绑定，不再出现「所有占位符都绑 args[0]」
  - **变参函数不再 panic（1.3）**：`func(args ...interface{})` + `args:"a,b"` 标签长度 > 参数槽数不再 panic（`makeParamType` 与代理安装期两处）；参数解析错误由 panic 改为 error 聚合上报；代理运行期自动展开变参切片，tag 按元素逐位绑定
  - **表前缀映射/替换（2.1）**：`mybatis.table-prefix-map=threedb_:`（空值=移除旧前缀、非空=替换），存量硬编码旧前缀的 XML 无需批量改写即可切生产；映射优先于真实表集合，逐源可覆盖并继承
  - **resultType="map" 合法化（1.4）**：不再刷 `unsupport type to parse: map` 告警
  - 宽松注册 `orm.SetStrictRegister(false)`（5.1）：失败函数跳过、其余正常注册，未绑定函数运行时返回明确错误
  - schema 逐源覆盖键 `spring.datasource.<name>.schema` 验证 + 多数据源×schema×前缀配置矩阵（3.1/3.2）
  - 表前缀细节：`MERGE INTO...USING`/`CREATE INDEX...ON`/`RENAME TABLE...TO` 表位置识别（2.2）；表集合缓存 TTL `mybatis.table-prefix-set-ttl`（2.4）；改写 debug 日志与 prepared 降级告警（2.5/5.3）；每文件解析汇总日志 + 生成 Go 文件 gofmt（5.2/4.2，含 resultType=map/无 parameterType 签名修复）
  - 端到端验收：`Test_P0_SharedJavaXml_NoParameterType`（无 parameterType 注册 + 变参运行）、`Test_TablePrefixMap_SqliteRemovePrefix`（三库 prod 前缀移除场景）

- **2026-09-04（v0.1.15）**：表名前缀按**真实表集合**精确改写（提高准确率）— 在纯前缀匹配之前，先从配置的 schema 获取数据库真实表名集合（`tableNameSet`，查询 `information_schema.COLUMNS` / `pg_class(+pg_namespace)` / `sqlite_master`，按数据源缓存、DDL 后自动失效），改写时与集合比对：带前缀表名在库中真实存在→按配置前缀改写；**无前缀表名在库中真实存在→保持原样**（修复「改了前缀但物理表未改名导致访问错误表」的核心场景，如配置 `test_` 但库中只有 `sys_user` 时不再改写为不存在的 `test_sys_user`）；两者均未收录（`CREATE TABLE` 新建表等）→沿用前缀匹配兜底（行为与旧版一致）；拉取失败自动降级纯前缀匹配并 Warn；多数据源按各自 schema 独立取集合；回归测试 `Test_rewriteSQLTables_tableSet` / `Test_TablePrefix_SqliteExistingUnprefixed` / `Test_TablePrefix_SqliteTableSetRefresh`；实现细节见 docs/agents/table-prefix.md
- **2026-09-03（v0.1.13）**：数据表名前缀（TablePrefix）— 配置 `mybatis.table-prefix=test_`（兼容 `mybatis-plus.global-config.db-config.table-prefix` 与逐源覆盖键 `spring.datasource.table-prefix`）后，SQL 执行入口自动把 `FROM/JOIN/INTO/UPDATE/TABLE` 表位置的表名改为 `test_sys_user`，XML Mapper 语句保持不变；词法扫描+状态机改写（tokenizeSQL + rewriteSQLTables），不碰列名/字符串字面量/注释/占位符/别名；跳过系统目录（information_schema / pg_% / sqlite_% / pragma_%）与已带前缀的表（不叠加）；CTE 名、`TABLE IF NOT EXISTS` 等 DDL 修饰词不误伤；多数据源未单独配置时继承默认源前缀；编程式 `orm.SetTablePrefix` / `orm.GetTablePrefix`；覆盖 Mapper 代理、`orm.Execute/Query`、事务、流式查询全部执行路径（含 `Transaction` 直调补齐 `formatSQL` 对齐）；实现细节见 docs/agents/table-prefix.md
- **2026-08-20（v0.1.12）**：PG/金仓 useGeneratedKeys RETURNING 支持（M-03）+ 依赖升级（P2-3）— ① PostgreSQL/KingbaseES 的 `sql.Result.LastInsertId()` 返回 error，自增主键回填失效。新增 RETURNING 路径：当数据库为 PostgreSQL/KingbaseES 且 `useGeneratedKeys` + `keyProperty` 已指定时，INSERT 自动追加 `RETURNING col`（keyColumn 显式指定或 keyProperty 驼峰转下划线），改用 `QueryContext` + `Scan` 读取生成的 ID 并回填；MySQL/SQLite 仍走 `LastInsertId()` 路径，行为不变。② `go-sql-driver/mysql` v1.6.0→v1.10.0、`beevik/etree` v1.1.0→v1.7.1、`lib/pq` v1.10.1→v1.12.3；`go.mod` go 版本升至 1.24.0；`lib/pq` → `pgx/v5` 迁移已评估，当前 lib/pq v1.12.3 仍可维护，迁移暂缓
- **2026-08-20（v0.1.11）**：大结果集流式读取（P4-2）+ MinDuration 并发修复（P2-5 跟进）— ① 新增 `orm.QueryStream(ctx, sql, args...)` 返回 `*orm.RowStream`：`Next()` / `Row()` 逐行消费（游标保持打开、内存 O(1)，10 万行以上不再整表进内存），`Scan(&dest)` 填充结构体或 map（列名→字段名：原名/首字母大写/下划线转驼峰/大小写不敏感），`Err()` / `Count()` / `Close()` 齐备（`Close` 幂等，未读完也必须 Close 释放连接）；② Mapper 代理支持 select 方法返回 `(*orm.RowStream, error)`（`BaseMapper.executeStream`），结果类型与 XML resultType 解耦；③ 语义对齐：行数上限遵循全局 `orm.SetDefaultRowLimit`（P4-3，打开时快照）、ctx 无 deadline 叠加全局默认超时（P4-1）、扫描失败不静默丢行（`Err()` 返回行号明细）；④ 修复 `updateMinDuration` 以 0 兼作「未初始化」哨兵与真实 0ms 测量值冲突（并发下最小值 0 被较大值覆盖）——新增 `minDurationInit` 原子标志区分，回归测试 `Test_updateMinDuration_ZeroCollision`
- **2026-08-19（v0.1.10）**：全局查询行数上限（P4-3）— `fetchRows` 默认最多返回 10000 行，防止大结果集 OOM/拖垮连接；新增 `orm.SetDefaultRowLimit(n)` / `orm.DefaultRowLimit()` 全局系统设置（负数不限制返回全部、0 不返回任何行），达到上限停止读取并 Warn 提示；`Query` / `QueryContext` / Mapper 代理所有查询路径共用 `fetchRows` 自动生效
- **2026-08-19（v0.1.9）**：超时/上下文执行 API（P4-1）+ scan 错误聚合（M-05）— ① 新增 `ExecuteContext(ctx, sql, args...)` / `QueryContext(ctx, sql, args...)`；新增 `orm.SetDefaultTimeout(d)` / `orm.DefaultTimeout()` 全局系统设置（默认 5 分钟，防止慢 SQL 无限挂起占住连接）；`executeWithResult` / `queryRows` 及 Mapper 代理执行路径通过 `withExecTimeout` 在 ctx 无 deadline 时自动叠加全局超时（有 deadline 时不叠加，避免误覆盖调用方显式控制）；`context.Background()` 的 `Execute` / `Query` 传统调用同样受全局超时保护。② `convert2Results` 转换失败的行不再静默丢弃：新增 `ResultConvertReport`/`ResultConvertError` 聚合错误明细（行号/列名），`executeMethod` 在 Skipped>0 或 Errors 非空时输出聚合错误日志，便于排查「0 行但 SQL 有数据」
- **2026-08-19（v0.1.8）**：MP 内置 CRUD 内存自动生成 — XML 含 resultMap（基本类型列 + `<id>` 主键）但缺 MP 内置方法时，加载期按缺失 ID 在内存补生成 10 个 CRUD（不落盘、不覆盖手写）；表名由 resultMap type 推导（SysUser→sys_user），jdbcType 缺失时主键默认 BIGINT/普通列默认 VARCHAR，逻辑删除支持 `deleted` 与 RuoYi `del_flag` 双约定；samples（RuoYi）真实回归验证（`SysUserMapper` 自动具备 SelectById 等，`del_flag='0'` 过滤生效）
- **2026-08-19（v0.1.8）**：MyBatis-Plus 内置 CRUD + codegen 增强 — `schema2code -mp` / `orm.SchemaToCodeMP` / `TableStructure.SaveMPToFile` 从表结构生成 BaseMapper 标准方法名 XML（insert/deleteById/updateById/selectById/selectOne/selectList/selectPage/selectCount/selectBatchIds/deleteBatchIds，逻辑删除自动适配）；codegen 检测 `<foreach>` 自动为批量方法生成切片签名（`deleteConfigByIds` → `func([]int64)`，对所有 Mapper 生效）；`SelectCount` 返回 `[]int64`；`<if>` 数值比较支持（M-02：`!= 0`/`> 0`/`== 0` 及 `x.length > 0`）；`convert2Map` nil 指针字段不再 panic（M-01）；使用说明见 docs/agents/mybatis-plus.md
- **2026-08-19（v0.1.7）**：MySQL DATETIME 修复 + 自定义 DB/DSN 注入 — MySQL DSN 自动追加 `?parseTime=true&loc=Local`（DATETIME/TIMESTAMP 列可直接扫描为 `time.Time`，不再因驱动返回 `[]byte` 导致时间字段丢失/行失败）；`resolveConverter`/`newInstance` 兼容 `[]uint8` 扫描类型兑底（未开 parseTime 时原始字节转字符串，`change2Time` 可再解析）；`Config` 新增 `ConnPool`/`DSN` 注入（`orm.Open(cfg)` 优先使用注入的连接池与自定义 DSN，各方言 dialector 均支持，预编译缓存包装共存）
- **2026-08-19（v0.1.6）**：RuoYi Mapper 兼容性修复 — `samples/` 目录 11 项兼容性缺陷（S-01~S-11）全部修复：`<where>` / `<set>` 标签支持、点号参数 `#{a.b}`、原始替换 `${...}`、`parameterType="Long"` 基础类型映射、resultMap `<association>` / `<collection>` 真实关联类型（`*SysDept` / `[]SysRole`）、裸标识符布尔 `<if>` 求值、`<include>` 内参数替换、nil 参数反射零值 panic 防御、排除非 Mapper XML（`mybatis-config.xml`）、`useGeneratedKeys` / `keyProperty` 自增主键回填（SQLite 端到端验证）
- **2026-08-14**：测试与 CI — 补齐核心代理机制单测（`proxyValue`/参数与返回类型校验），顺带修复 `args` tag 文档格式失效的 bug（兼容 `args:name` 与 `args:"name"`）；新增 utils 测试与性能基准；新增 GitHub Actions CI（build + vet + 测试 + 覆盖率）
- **2026-08-14**：工程化清理 — `ioutil` 弃用替换为 `os.ReadFile/WriteFile`；删除死代码（`convert2Interfaces`、`Rows` 接口）；`.gitignore` 去重
- **2026-08-14**：调试日志零开销 — 日志级别关闭时不再对整结果集做 `ToJson` 序列化（`log.IsDebugEnabled()` + 可选 `debugEnabler` 接口，未实现者保守返回 true 不丢日志）
- **2026-08-14**：健壮性增强 — 全局模型/Mapper 缓存 map 加读写锁（支持并发注册与访问）；预编译语句缓存加上限（`maxPreparedStmts=100`，满后降级直接执行，避免无界缓存钉死连接）；默认日志级别改为 Warn（不调用 `SetLogger` 时错误/警告也可见，新增 `ConsoleLogger.Level` 与 `log.DebugEnabled` 等级别查询）
- **2026-08-14**：性能优化 — 接通预编译语句缓存（`DB.ExecContext/QueryContext` 走 `PreparedStmtDB` 包装，新增 `spring.datasource.prepared-stmt` 配置默认开启）；PostgreSQL/KingbaseES 占位符自动 `?`→`$n`（修复 lib/pq 参数化查询跑不通的问题）；结果集转换优化（扫描目标复用 + 列转换函数/字段索引预编译，`fetchRows` 首次 `Next()` 后构建以兼容驱动惰性 `ScanType`）；无参 SQL 生成结果缓存（`sync.Once`）
- **2026-08-14**：新增多数据源支持（`InitializeDataSources` / `UseDataSource` / `AddDataSource`，配置 `mybatis.datasources` + `spring.datasource.<name>.*`）
- **2026-08-14**：新增人大金仓 KingbaseES 支持（兼容 PostgreSQL 线协议，自动以 `kingbase` 名称注册 `lib/pq` 驱动，含 `cmd/kingbasedemo` 示例与 `schema2code -type kingbase`）；新增事务支持（`Begin` / `Commit` / `Rollback`，Mapper 方法自动参与事务）；JDBC URL 支持 IPv6；`LoadProperties` 键值解析健壮性改进
- **2026-08-14**：可靠性修复 — `InitializeDatabase` 不再吞错、`PreparedStmt` 模式下连接健康检查恢复、`ReConnect` 重建预编译缓存、配置占位符非法输入不再 panic
