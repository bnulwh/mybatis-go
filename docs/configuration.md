# 配置指南

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
| `mybatis.configuration.safe-update` | UPDATE/DELETE 无 WHERE 子句时阻止执行 | false |
| `mybatis.configuration.pretty-sql` | 日志中 SQL 格式化输出（参数绑定替换 + 关键字换行缩进，仅用于调试） | false |
| `mybatis.configuration.cache-enabled` | 启用二级缓存（namespace 级 LRU+TTL） | false |
| `mybatis.configuration.local-cache-size` | 二级缓存每 namespace 最大条目数 | 1024 |
| `mybatis.configuration.local-cache-ttl` | 二级缓存 TTL（秒） | 3600 |
| `mybatis.configuration.id-type` | 主键生成策略：`snowflake` / `uuid` / `assign_id` | - |
| `mybatis.configuration.snowflake-worker-id` | Snowflake 工作节点 ID | 1 |

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
