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
| `mybatis.mapper-locations` | XML Mapper 文件目录 | - |

> **MySQL DATETIME 列**：框架自动在 MySQL DSN 追加 `?parseTime=true&loc=Local`（与 SQLite 的 `_loc=auto` 同理），DATETIME/TIMESTAMP 列直接扫描为 `time.Time`；否则 go-sql-driver 返回原始 `[]byte`，时间字段无法赋值。

> **预编译语句缓存**：默认开启。参数化 SQL（Mapper 的 `#{}` 或 `Execute/Query` 带参调用）会按 SQL 文本缓存预编译语句，避免数据库重复解析与生成执行计划；无参数 SQL（DDL、静态查询）直接执行，不进入缓存。PostgreSQL/KingbaseES 的占位符会自动从 `?` 转为 `$1、$2…`（lib/pq 不支持 `?`）。

支持环境变量覆盖：配置值形如 `${ENV_NAME}` 或 `${ENV_NAME:default}` 时会自动替换。

JDBC URL 支持 IPv6 地址，如 `jdbc:postgresql://[2001:db8::1]:5432/testdb`（MySQL DSN 会自动补方括号）。

### 编程式配置（不使用 properties 文件）

```go
err := orm.InitializeDatabase("postgres", "localhost", 5432, "root", "123456", "testdb")
// 或 orm.InitializeDatabase("kingbase", "localhost", 54321, "system", "123456", "testdb")
```

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
│   ├── mysql_dialector.go / postgres_dialector.go
│   ├── sqlite_dialector.go / kingbase_dialector.go   # 数据库方言
│   └── ...              # 初始化、代理、SQL 执行、结果转换、缓存等
├── types/               # XML 解析引擎和数据类型
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
