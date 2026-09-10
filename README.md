# mybatis-go

Go 语言实现的 MyBatis 风格 ORM 框架。通过 XML Mapper 文件定义 SQL，利用反射为 struct 的函数字段注入代理实现，支持 **PostgreSQL、MySQL、SQLite 和人大金仓 KingbaseES**。

## 特性

- **MyBatis 风格**：XML Mapper 定义 SQL，`#{}` / `${}` 参数绑定，完整动态 SQL（`<if>` / `<where>` / `<set>` / `<foreach>` / `<choose>` / `<include>`）
- **反射代理**：Mapper struct 的函数字段在运行时自动注入代理，无需手动实现
- **结果自动映射**：查询结果自动映射到 Go struct，支持 `resultMap`（含 `<association>` / `<collection>` 嵌套关联类型生成）
- **自增主键回填**：`useGeneratedKeys` / `keyProperty` 支持，Insert 后自动回填自增主键到入参 struct 指针
- **MyBatis-Plus 内置 CRUD**：`schema2code -mp` 从表结构直接生成 BaseMapper 标准方法名（insert/deleteById/updateById/selectById/selectList/selectOne/selectPage/selectCount/selectBatchIds/deleteBatchIds）的 XML，原生加载、无需手写 GoExtraMapper；亦可在 **XML 含 resultMap 时加载期内存自动补生成**（无需落盘 CRUD XML），使用说明见 **docs/agents/mybatis-plus.md**
- **多数据库**：PostgreSQL、MySQL、SQLite、人大金仓 KingbaseES
- **代码生成**：内置 `generator`（XML → Go）和 `schema2code`（数据库表 → Go）工具
- **预编译缓存**：Prepared Statement 自动缓存和复用
- **事务支持**：`orm.Begin()` / `Commit()` / `Rollback()`，事务开启后 Mapper 方法与 SQL 自动参与
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

完整示例见 `cmd/postgresdemo/main.go`、`cmd/mysqldemo/main.go`、`cmd/sqlitedemo/main.go` 和 `cmd/kingbasedemo/main.go`。

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

支持 Spring Boot 风格的 `.properties` 文件，仓库根目录附有各数据库的示例配置。

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
- `-type` 数据库类型：`mysql` / `postgres` / `kingbase` / `sqlite`
- `-host` 数据库地址
- `-port` 端口
- `-username` / `-password` 认证信息（SQLite 无需填写）
- `-db` 数据库名（SQLite 为 `.db` 文件路径）
- `-output` 输出目录
- `-prefix` 可选，表名前缀
- `-tables` 可选，指定表名（逗号分隔），为空则生成全部表
- `-mp` 可选，生成 MyBatis-Plus 内置 CRUD XML（BaseMapper 标准方法名；批量方法自动生成切片签名，如 `DeleteBatchIds func([]int64)`；`SelectCount` 返回 `[]int64`）

## 运行示例

```bash
go run ./cmd/sqlitedemo      # SQLite（自动建表 + Mapper 全流程，生成 test.db）
go run ./cmd/postgresdemo    # PostgreSQL（需先准备 application-pg.properties 指向的库）
go run ./cmd/mysqldemo       # MySQL
go run ./cmd/kingbasedemo    # KingbaseES
```

## 重要说明

- `orm.NewMapper("MapperName")` 创建的对象必须先通过 `orm.RegisterMapper` 注册
- `orm.RegisterModel` 用于注册模型类，注册后的类在调用 Mapper 函数时可以自动创建并填充值
- 函数字段的 tag（如 `` `args:id` ``）可用于指定输入参数名称映射
- `useGeneratedKeys` 回填需向 Insert 方法传 **struct 指针**（值传递无法写回调用方）；入参为 map 时同样支持
- SELECT 方法的返回值类型为 `([]Model, error)`，INSERT/UPDATE/DELETE 为 `(int64, error)`；流式 select 可返回 `(*orm.RowStream, error)`（逐行消费，调用方必须 `Close()` 释放连接）
- KingbaseES 驱动由框架自动注册（`sql.Register("kingbase", &pq.Driver{})`），无需也不应重复引入驱动
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
│   └── demo/            # 通用使用示例
├── orm/                 # 核心 ORM 框架
│   ├── transaction.go   # 事务支持（Begin/Commit/Rollback）
│   ├── multi_datasource.go  # 多数据源注册表（InitializeDataSources / UseDataSource / AddDataSource）
│   ├── row_stream.go    # 大结果集流式读取（QueryStream / RowStream，Mapper 流式 select）
│   ├── embed_fs.go      # go:embed 内嵌 Mapper 加载（RegisterMapperFS / RegisterMapperSources / ReloadMappers）
│   ├── mysql_dialector.go / postgres_dialector.go
│   ├── sqlite_dialector.go / kingbase_dialector.go   # 数据库方言
│   └── ...              # 初始化、代理、SQL 执行、结果转换、缓存等
├── types/               # XML 解析引擎和数据类型
│   └── sql_mappers.go   # Mapper 加载（NewSqlMappers 磁盘 / NewSqlMappersFrom embed.FS 等任意 io/fs.FS）
├── utils/               # 工具函数
├── log/                 # 日志接口
├── mapper/              # 生成的 Mapper 示例
├── resources/mapper/    # XML Mapper 文件
└── samples/             # RuoYi Mapper 兼容性回归样本（KingbaseES 方言，S-01~S-11 已全部修复）
```

## 测试

```bash
go test ./types/... ./utils/...        # 不依赖数据库的单元测试
go test ./orm/... -v                   # 含 SQLite 端到端测试（Mapper / 事务 / 表结构，无需外部数据库）
go test -v -count=1 ./... -coverprofile=cover.out
```

## 更新日志

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
