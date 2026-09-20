# 功能专题

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

## 性能优化

框架在执行热路径上做了以下优化（详见 `TODO.md`）：

- **预编译语句缓存**：参数化 SQL 按文本缓存 Prepared Statement，数据库不再重复解析 + 生成执行计划（见 docs/configuration.md「配置项说明」）
- **占位符方言转换**：PostgreSQL/KingbaseES 的 `?` 自动转为 `$n`，MySQL/SQLite 原样保留；仅在存在参数时转换，避免误伤无参 SQL 中的字面量 `?`
- **扫描目标复用**：结果集扫描目标（`sql.NullXxx` 指针）每次查询只分配一次、跨行复用，替代逐行分配
- **反射预编译**：列值转换函数表（`convertFn`）与 resultMap 的 property→字段索引映射在查询开始时预编译一次，行循环内直接调用，避免每行每列 `ScanType` switch 分派与 `FieldByName` O(N) 名称匹配
- **无参 SQL 生成缓存**：无参 SQL 的拼接结果静态不变，首次生成后缓存复用
- **大结果集流式读取**：`QueryStream` / Mapper 流式 select 逐行消费（内存 O(1)），配合全局行数上限（P4-3，默认 10000 行）兜底截断，避免大结果集 OOM / 拖垮连接

## SQL 格式化输出（Pretty SQL）

开启后日志中的 SQL 将自动格式化：参数值替换占位符 + 关键字换行缩进，便于开发调试。**仅用于日志展示，不影响实际执行。**

### 配置

```properties
mybatis.configuration.pretty-sql=true
```

或运行时开关：

```go
orm.SetPrettySQL(true)   // 开启
orm.SetPrettySQL(false)  // 关闭
```

### 效果示例

原始日志（默认）：

```
sql: SELECT id, name FROM user WHERE id = ? AND name = ? ORDER BY id
```

开启 `pretty-sql` 后：

```
sql: SELECT id, name 
FROM user 
WHERE id = 1 
  AND name = 'alice' 
ORDER BY id
```

- 参数值直接替换：字符串加引号、nil→NULL、时间→`'2006-01-02 15:04:05'`、`[]byte`→`x'hex'`
- 支持 4 种占位符：`?`（MySQL/SQLite）、`$n`（PostgreSQL/KingbaseES）、`:n`（Oracle/DB2）、`@pN`（SQL Server）
- 关键字换行：SELECT / FROM / WHERE / AND / OR / JOIN / ON / SET / VALUES / ORDER BY / GROUP BY / HAVING / LIMIT / INSERT INTO / UPDATE / DELETE FROM
- 子查询自动缩进

## 执行计划自动分析（Slow SQL EXPLAIN）

SELECT 执行耗时超过阈值时，自动执行 `EXPLAIN` 并输出执行计划到日志，辅助性能调优。

### 配置

```properties
mybatis.configuration.explain-slow-sql=true
mybatis.configuration.slow-sql-threshold=3000
```

或运行时开关：

```go
orm.SetExplainSlowSQL(true)                    // 开启
orm.SetSlowSQLThreshold(5 * time.Second)       // 设置阈值 5s
orm.SlowSQLThreshold()                         // 查询当前阈值
```

### 各数据库 EXPLAIN 语法

| 数据库 | EXPLAIN 语法 |
|--------|-------------|
| MySQL / SQLite / TiDB / OceanBase | `EXPLAIN SELECT ...` |
| PostgreSQL / KingbaseES / OpenGauss | `EXPLAIN ANALYZE SELECT ...` |
| Oracle / 达梦 | `EXPLAIN PLAN FOR SELECT ...` |
| SQL Server | `SET SHOWPLAN_TEXT ON; SELECT ...` |
| DB2 | `EXPLAIN ALL FOR SELECT ...` |
| GBase 8s (Informix) | `SET EXPLAIN ON; SELECT ...` |

### 效果示例

```log
[WARN] Slow SQL EXPLAIN (cost=3.2s, namespace=UserMapper.selectAll):
  2 | 0 | 0 | SCAN TABLE t_user
```

- 仅对 `SELECT` 语句触发，INSERT/UPDATE/DELETE 不触发
- 基于内置 After 钩子实现，不影响正常执行流程

## 连接池监控

基于 `database/sql.DBStats` 提供连接池运行时状态查询与定期日志输出。

### API

```go
// 当前活跃数据源连接池状态
st := orm.PoolStats()
fmt.Printf("InUse=%d Idle=%d WaitCount=%d\n", st.InUse, st.Idle, st.WaitCount)

// 指定数据源
st, err := orm.PoolStatsFor("secondary")

// 全部数据源
all := orm.PoolStatsAll() // map[string]sql.DBStats

// 格式化输出（日志友好）
log.Info(orm.PoolStatsString())
```

### 定期日志

```properties
mybatis.configuration.pool-stats-interval=60
```

每 60 秒自动输出连接池状态到 INFO 日志。设为 0（默认）关闭。

运行时控制：

```go
orm.StartPoolStatsLogger(30 * time.Second) // 启动，每 30s 输出
orm.StopPoolStatsLogger()                  // 停止
orm.SetPoolStatsInterval(60 * time.Second) // 动态调整间隔
```

### 输出示例

```log
[INFO] Pool Stats:
[default] MaxOpenConnections=100 OpenConnections=2 InUse=1 Idle=1 WaitCount=0 WaitDuration=0s MaxIdleClosed=0 MaxIdleTimeClosed=0 MaxLifetimeClosed=0
```

- 多数据源时每个源一行
- `Close()` 时自动停止定期输出

## 读写分离路由

SELECT 自动路由到副本数据源，INSERT/UPDATE/DELETE 走主库；事务内所有操作强制走主库。

### 配置

```properties
# 1. 配置多数据源（主库 + 副本）
mybatis.datasources= replica
spring.datasource.replica.url= jdbc:mysql://replica-host:3306/db
spring.datasource.replica.username= reader
spring.datasource.replica.password= reader_pwd

# 2. 启用读写分离
mybatis.configuration.read-write-splitting=true

# 3. 指定副本数据源名称
mybatis.replicas= replica
```

多副本时逗号分隔（`replica1,replica2`），框架轮询选择。

### 编程方式

```go
orm.SetReadWriteSplitting(true)
orm.SetReplicaNames([]string{"replica1", "replica2"})

// 或注册新副本
orm.RegisterReplica("replica2", "mysql", "replica2-host", 3306, "reader", "pwd", "db")

// 选取副本（调试用）
db, err := orm.PickReplica()
```

### 路由规则

| 场景 | 路由目标 |
|------|---------|
| SELECT（无事务） | 副本（轮询） |
| INSERT / UPDATE / DELETE | 主库 |
| 事务内（Begin / WithTx） | 主库 |
| 读写分离关闭 | 主库 |
| 无可用副本 | 降级到主库 |

- 基于 context 传递路由 DB，不修改全局 `gDbConn`，并发安全
- `orm.Execute` / `orm.Query` 等顶层 API 不走读写分离（无 SQL 类型信息）；Mapper 方法自动路由
