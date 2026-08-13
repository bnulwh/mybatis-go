# mybatis-go

Go 语言实现的 MyBatis 风格 ORM 框架。通过 XML Mapper 文件定义 SQL，利用反射为 struct 的函数字段注入代理实现，支持 PostgreSQL 和 MySQL。

## 特性

- **MyBatis 风格**：XML Mapper 定义 SQL，`#{}` / `${}` 参数绑定，`<if>` / `<where>` 等动态 SQL
- **反射代理**：Mapper struct 的函数字段在运行时自动注入代理，无需手动实现
- **结果自动映射**：查询结果自动映射到 Go struct，支持 `resultMap` 配置
- **多数据库**：支持 PostgreSQL、MySQL 和 SQLite
- **代码生成**：内置 `generator` 和 `schema2code` 工具，从 XML 或数据库表结构生成 Go 代码
- **预编译缓存**：Prepared Statement 自动缓存和复用

## 路线图

- [x] PostgreSQL 支持 — 已实现并测试通过
- [x] MySQL 支持 — 已实现（`cmd/mysqldemo/main.go`）
- [x] SQLite 支持 — 已实现（纯 Go 驱动 modernc.org/sqlite，无需 CGO，`cmd/sqlitedemo` 示例）
- [ ] 多数据源支持
- [ ] 其他改进和优化

## 快速开始

### 1. 定义模型

```go
package main

import (
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
```

### 2. 定义 Mapper

注意：必须内嵌 `orm.BaseMapper`

```go
package main

import (
    "github.com/bnulwh/mybatis-go/orm"
)

type UserInfoModelMapper struct {
    orm.BaseMapper
    DeleteByPrimaryKey func(int) (int64, error)
    Insert             func(UserInfoModel) (int64, error)
    UpdateByPrimaryKey func(UserInfoModel) (int64, error)
    SelectByPrimaryKey func(int) ([]UserInfoModel, error)
    SelectAll          func() ([]UserInfoModel, error)
}
```

Mapper struct 的 `func` 类型字段由 ORM 框架在初始化时注入代理实现，每个字段对应 XML Mapper 中同名的 SQL 操作。

### 3. 初始化 ORM

```go
import (
    log "github.com/bnulwh/logrus"
    "github.com/bnulwh/mybatis-go/orm"
    _ "github.com/lib/pq"         // PostgreSQL 驱动
    // _ "github.com/go-sql-driver/mysql" // MySQL 驱动
)

func init() {
    orm.SetLogger(log.StandardLogger())
    orm.Initialize("application.properties")
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

完整示例见 `cmd/postgresdemo/main.go` 和 `cmd/mysqldemo/main.go`。

## 配置文件

支持 Spring Boot 风格的 `.properties` 文件：

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
- `-type` 数据库类型：`mysql` / `postgres`
- `-host` 数据库地址
- `-port` 端口
- `-username` / `-password` 认证信息
- `-db` 数据库名
- `-output` 输出目录
- `-prefix` 可选，表名前缀
- `-tables` 可选，指定表名（逗号分隔），为空则生成全部表

## 重要说明

- `orm.NewMapper("MapperName")` 创建的对象必须先通过 `orm.RegisterMapper` 注册
- `orm.RegisterModel` 用于注册模型类，注册后的类在调用 Mapper 函数时可以自动创建并填充值
- 函数字段的 tag（如 `` `args:id` ``）可用于指定输入参数名称映射
- SELECT 方法的返回值类型为 `([]Model, error)`，INSERT/UPDATE/DELETE 为 `(int64, error)`

## 项目结构

```
├── cmd/
│   ├── generator/       # 从 XML Mapper 生成代码
│   ├── schema2code/     # 从数据库表结构生成代码
│   ├── postgresdemo/    # PostgreSQL 使用示例
│   └── mysqldemo/       # MySQL 使用示例
├── orm/                 # 核心 ORM 框架
├── types/               # XML 解析引擎和数据类型
├── utils/               # 工具函数
├── log/                 # 日志接口
├── mapper/              # 生成的 Mapper 示例
└── resources/mapper/    # XML Mapper 文件
```
