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
