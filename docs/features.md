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

推荐使用回调式事务 `orm.WithTx`（goroutine 安全、支持嵌套、不占全局槽）：

```go
err := orm.WithTx(ctx, func(ctx context.Context) error {
    // 声明了 context.Context 参数的 Mapper 方法传入 ctx 后自动加入事务
    if _, err := mp.Insert(ctx, UserInfoModel{Username: "tx_user"}); err != nil {
        return err // 返回 error → 自动回滚并原样返回
    }
    if _, err := mp.UpdateByPrimaryKey(ctx, ...); err != nil {
        return err
    }
    return nil // 返回 nil → 自动提交
})
```

要点：

- **fn 返回 nil → Commit，返回 error → Rollback 并原样返回，panic → Rollback 后重新抛出**；
- **嵌套复用**：`WithTx` 内再 `WithTx` 直接复用外层事务，事务边界由最外层决定；
- **不占全局事务槽**，多个 goroutine 可各自 `WithTx` 互不冲突；
- `orm.ExecuteContext` / `QueryContext` / `QueryToContext` 传入 ctx 同样加入事务；
- **C2 诊断**：`WithTx` 内调用未声明 `context.Context` 参数的 Mapper 方法不会加入事务
  （走独立连接、不随回滚），首次执行时框架 `Warn` 提示；可用
  `orm.SetTxScopeWarn(false)` 或 `mybatis.configuration.tx-scope-warn=false` 关闭。

旧版 TCC 三件套（`Begin`/`Commit`/`Rollback`）仍可用但**已标记 Deprecated**（占用全局
单事务槽，多 goroutine 交错开启会互相劫持提交/回滚），首次使用时框架输出迁移告警：

```go
// 已废弃，仅存量代码兼容 —— 请迁移到 orm.WithTx
tx, err := orm.Begin()
defer tx.Rollback()
if _, err := mp.Insert(UserInfoModel{Username: "tx_user"}); err != nil { // 自动在此事务内
    return
}
err = tx.Commit()
```

也可以在 `WithTx` 内通过 `tx.Exec` / `tx.Query` / `tx.QueryRow` 直接执行 SQL
（`orm.TxFromContext(ctx)` 取出事务对象）。

> 注意：`Begin`/`BeginTx` 绑定全局连接（单事务槽），多个 goroutine 之间不要交错开启事务；
> 新代码一律使用 `WithTx`。

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

### namespace 冲突检测与前缀/别名（E3）

- **同 namespace 多文件检测**：同一 `namespace` 被多个 XML 文件加载时（后者覆盖前者），
  加载期 `log.Warnf` 列出全部文件，并记入 `types.SqlMappers.NamespaceConflicts`
  （orm 侧 `orm.NamespaceConflicts()` 查询）。覆盖行为不变，仅提升可观测性——
  「共享 Java XML 与本地 Mapper 同名互踩」「来源顺序写反」等问题加载期即暴露。
- **`MapperSource.NamespacePrefix`**：该来源下所有 namespace 加 `prefix.` 前缀，
  绑定键改为「前缀 + 短名」且不注册裸短名键，可与无前缀同名 Mapper 并存互不覆盖；
  Go 结构体命名为 前缀+短名 即可绑定（prefix 应为合法 Go 标识符段，不含点）：

```go
// Go 端已有 PlatformDisplayMapper 时，共享 Java XML 无需改名即可并存：
err := orm.RegisterMapperSources(
    orm.MapperSource{Patterns: []string{"my/mapper"}},                                       // 本地
    orm.MapperSource{FS: sharedXML, Patterns: []string{"java/mapper"}, NamespacePrefix: "Quality"}, // 共享，前缀隔离
)
orm.RegisterMapper(new(mapper.QualityPlatformDisplayMapper)) // 结构体名 = Quality + 短名
```

- **`orm.RegisterNamespaceAlias(alias, name)`**：为已加载 Mapper 注册别名
  （alias 各查找键解析到与 name 相同的 SqlMapper），并自动重绑全部已注册结构体；
  namespace 改名 / 加前缀后旧引用（旧名结构体、`NewMapper("OldName")`）不失效：

```go
// XML namespace 已由 PlatformDisplayMapper 改名为 QualityPlatformDisplayMapper：
err := orm.RegisterNamespaceAlias("PlatformDisplayMapper", "QualityPlatformDisplayMapper")
```

alias 键覆盖既有不同 Mapper 时仅告警不报错（与加载期「后者覆盖前者」语义一致）。

## 性能优化

框架在执行热路径上做了以下优化（详见 `TODO.md`）：

- **预编译语句缓存**：参数化 SQL 按文本缓存 Prepared Statement，数据库不再重复解析 + 生成执行计划（见 docs/configuration.md「配置项说明」）
- **占位符方言转换**：PostgreSQL/KingbaseES 的 `?` 自动转为 `$n`，MySQL/SQLite 原样保留；仅在存在参数时转换，避免误伤无参 SQL 中的字面量 `?`
- **扫描目标复用**：结果集扫描目标（`sql.NullXxx` 指针）每次查询只分配一次、跨行复用，替代逐行分配
- **反射预编译**：列值转换函数表（`convertFn`）与 resultMap 的 property→字段索引映射在查询开始时预编译一次，行循环内直接调用，避免每行每列 `ScanType` switch 分派与 `FieldByName` O(N) 名称匹配
- **无参 SQL 生成缓存**：无参 SQL 的拼接结果静态不变，首次生成后缓存复用
- **大结果集流式读取**：`QueryStream` / Mapper 流式 select 逐行消费（内存 O(1)），配合全局行数上限（P4-3，默认 10000 行）兜底截断，避免大结果集 OOM / 拖垮连接

## 类型安全取用与注册表自省

`NewMapper(name)` 返回 `interface{}` 且未注册即 panic，调用方只能裸类型断言。除保留该 API 外，
框架提供类型安全 / error / 泛型版本：

```go
// 类型安全 + error（T 为 Mapper 结构体值类型，也可为指针/接口形态）
mp, err := orm.Mapper[mapper.SysUserMapper]("SysUserMapper")

// 失败 panic（显式命名，适合 init 期快速失败）
mp := orm.MustMapper[mapper.SysUserMapper]("SysUserMapper")

// interface{} + error（不做类型断言，仅需 error 语义时）
v, err := orm.NewMapperE("SysUserMapper")
```

三者均把「未注册 / XML 绑定失败 / 类型不匹配」转为返回 error（bindMapper 内部的绑定 panic
同样转为 error，不再击穿调用栈）。

注册表自省（跨包 `init()` 注册的数量可自证，无需硬编码常量）：

```go
orm.RegisteredMapperNames()            // 已注册 Mapper 结构体名（去重、排序）
cnt, err := orm.MapperFuncCount("SysUserMapper") // 指定 Mapper 的函数字段数
```

## 严格结果模式（Strict Result）

默认情况下，结果集转换存在行级丢弃或列级类型失败时，框架输出聚合 ERROR 日志并**返回已
转换的部分数据**（`err == nil`）。开启严格模式后，转换不完整的查询返回
`ErrResultConvertIncomplete`（`errors.Is` 可判）而非部分数据：

```properties
mybatis.configuration.strict-result=true
```

```go
orm.SetStrictResult(true) // 运行时开关
_, err := mp.SelectUserList(ctx)
if errors.Is(err, orm.ErrResultConvertIncomplete) { ... }
```

无论开关状态，异常语句的最近一次转换报告均可查询：

```go
report := orm.LastConvertReport("SysUserMapper", "selectUserList") // 无记录返回 nil
// report.Total / report.Converted / report.Skipped / report.Errors（行号/列名/消息）
```


## MySQL 方言函数翻译（Function Rewrite）

共享 Java XML 中常见的 MySQL 方言函数/语法，在 PG 族数据源（PostgreSQL/KingbaseES PG 模式/
openGauss 等）上无法执行。开启翻译后，框架在 SQL 执行前自动改写为 PG 等价形式，
**共享 XML 无需手改副本**：

```properties
mybatis.configuration.function-rewrite=true
```

```go
orm.SetFunctionRewrite(true) // 运行时开关
```

| MySQL | PG / 金仓 PG 模式 |
|---|---|
| `ifnull(a, b)` | `coalesce(a, b)` |
| `find_in_set(x, csv)` | `position(','||x||',' in ','||csv||',')` |
| `date_format(d, 'fmt')` | `to_char(d, 'fmt')`（`%Y/%m/%d/%H/%i/%s/%T/%r` 等 25 个格式符映射） |
| `` `col` `` | `"col"`（反引号标识符 → PG 双引号标识符） |

特性与边界：

- 仅 PG 族数据源生效（MySQL 族原生支持上述语法，无需翻译）；开关默认关闭，存量行为不变。
- 引号（`'`/`"`，含转义）与注释（`--`、`/* */`）内不改写；函数名要求独立词边界，大小写不敏感。
- 改写可重入；嵌套参数先递归改写（如 `find_in_set(ifnull(a,b), csv)`）。
- `find_in_set` 返回值不等于 MySQL 的 1 基序号，适用于 `> 0` 存在性判断（共享 XML 的常见用法）。
- `date_format` 第二参数须为字符串字面量（绑定参数/列名跳过并告警）；`col = 0`（varchar 列）
  与 `col = '0'` 的类型差异不做推断改写。

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
