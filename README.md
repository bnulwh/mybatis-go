# mybatis-go

Go 语言实现的 MyBatis 风格 ORM 框架。通过 XML Mapper 文件定义 SQL，利用反射为 struct 的函数字段注入代理实现，支持 **PostgreSQL、MySQL、SQLite 和人大金仓 KingbaseES**。

## 特性

- **MyBatis 风格**：XML Mapper 定义 SQL，`#{}` / `${}` 参数绑定，`<if>` / `<where>` 等动态 SQL
- **反射代理**：Mapper struct 的函数字段在运行时自动注入代理，无需手动实现
- **结果自动映射**：查询结果自动映射到 Go struct，支持 `resultMap` 配置
- **多数据库**：PostgreSQL、MySQL、SQLite、人大金仓 KingbaseES
- **代码生成**：内置 `generator`（XML → Go）和 `schema2code`（数据库表 → Go）工具
- **预编译缓存**：Prepared Statement 自动缓存和复用

## 路线图

- [x] PostgreSQL 支持 — 已实现并测试通过
- [x] MySQL 支持 — 已实现（`cmd/mysqldemo/main.go`）
- [x] SQLite 支持 — 已实现（纯 Go 驱动 modernc.org/sqlite，无需 CGO，`cmd/sqlitedemo` 示例）
- [x] KingbaseES 支持 — 已实现（人大金仓，兼容 PostgreSQL 线协议，复用 `lib/pq` 驱动，`cmd/kingbasedemo` 示例）
- [x] 事务支持 — 已实现（`orm.Begin()` / `Commit()` / `Rollback()`）
- [ ] 多数据源支持

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
| `mybatis.mapper-locations` | XML Mapper 文件目录 | - |

支持环境变量覆盖：配置值形如 `${ENV_NAME}` 或 `${ENV_NAME:default}` 时会自动替换。

### 编程式配置（不使用 properties 文件）

```go
err := orm.InitializeDatabase("postgres", "localhost", 5432, "root", "123456", "testdb")
// 或 orm.InitializeDatabase("kingbase", "localhost", 54321, "system", "123456", "testdb")
```

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
- SELECT 方法的返回值类型为 `([]Model, error)`，INSERT/UPDATE/DELETE 为 `(int64, error)`
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
│   ├── mysql_dialector.go / postgres_dialector.go
│   ├── sqlite_dialector.go / kingbase_dialector.go   # 数据库方言
│   └── ...              # 初始化、代理、SQL 执行、结果转换、缓存等
├── types/               # XML 解析引擎和数据类型
├── utils/               # 工具函数
├── log/                 # 日志接口
├── mapper/              # 生成的 Mapper 示例
└── resources/mapper/    # XML Mapper 文件
```

## 测试

```bash
go test ./types/... ./utils/...        # 不依赖数据库的单元测试
go test ./orm/... -v                   # 含 SQLite 端到端测试（无需外部数据库）
go test -v -count=1 ./... -coverprofile=cover.out
```
