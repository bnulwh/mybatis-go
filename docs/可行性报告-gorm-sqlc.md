# mybatis-go 兼容 GORM 风格 / sqlc 风格 可行性报告

> 撰写日期：2026-08-15　｜　修订日期：2026-09-16
> 基线：`go build ./... && go vet ./... && go test ./orm/... ./types/... ./utils/...` 全绿（2026-09-16 实测通过）
> 结论：**两条路线均可行，且可行性较初版进一步增强**。初版撰写后项目已落地「P0 框架扩展五件套」（QueryWrapper / db tag 元数据 / sqlfragment 独立包 / TypeHandler / Hook）与「P1~P4 数据库适配」（方言 4 → 19 种）；**G0 前置改造（事务上下文 + 强类型扫描直通道）与 G1（gormish 链式 API 最小可用）已于 2026-09-16 实施完毕**——`orm.WithTx` 回调事务（不占 TCC 全局槽、嵌套复用、ctx 优先级）、Mapper 方法 `context.Context` 参数自动提取、`orm.QueryTo/QueryToContext` 强类型扫描直通道、`orm/gormish` 链式执行器（链式子句 + Find/First/Last/Count/Scan + Create/Update/Updates/Delete/Exec + Transaction，全 `?` 占位参数绑定）。**S1（sqlc A 方案：XML 存量静态 select 抽取 → 类型安全 Querier 代码生成）亦已于 2026-09-17 实施完毕**——`types/sqlfragment.ExtractStaticSQL` 静态性判定、`types.GenerateQuerierFiles` 生成器（model + Querier 接口 + 实现，免连库，resultMap/resultType 双轨定型）、`cmd/sqlc` 命令（samples 22 mappers → 121 个静态函数），生成代码经 `QueryToContext` 执行（占位符转换/表前缀/ctx 事务由 orm 层承担）。GORM 路线的最小可用形态与 sqlc 路线的 A 方案均已就绪，剩余为 G2+（AutoMigrate/软删除/hooks/savepoint/关联）、S2（`.sql` 输入直通）与 G3（关联）的路线级工作。

---

## 0. 本次修订摘要（2026-08-15 → 2026-09-16 的关键变化）

| 变化 | 与本报告的关系 |
|------|----------------|
| **P0-1 QueryWrapper 条件构造器**（`orm/query_wrapper.go`，MyBatis-Plus 风格）：链式 `Eq/Like/Between/In/Or/Nested/OrderBy/...`，作为 mapper 方法参数由 `wrapper_rewrite.go` 自动注入 SQL | 链式条件构造语义**已在 XML Mapper 路线上落地**，但它不是 GORM 式独立执行器（无 `Table()/Find(&users)`），且条件值**内联渲染**（`FormatValue` 转义）而非 `?` 占位 |
| **P0-2 struct db tag 元数据**（`orm/model_info.go`）：`db:"col,-,pk,logic,version,fill:insert\|update\|insert_update"` + `TableName()` 方法 | 初版 §2.2 中「struct tag 驱动 schema 元数据」这一差距**已基本消除**（fill 目前仅为元数据标记，无内置自动填充实现） |
| **P0-3 types/sqlfragment 独立包**：`Node` 接口 + `RegisterNodeParser` 开放注册，支持 `if/where/set/foreach/choose/include/sql/trim` | 动态 SQL 引擎与 XML 解析解耦，`FormatValue`/`SQLSegmentProvider` 已被 orm 侧复用——GORM 链式 Builder 的条件渲染可直接对接 |
| **P0-4 TypeHandler**（`orm/type_handler.go`）：`RegisterTypeHandlerFor[T]`，`convertFieldValue` 统一入口、优先于内置 `ChangeType` | 初版 §4.2 强类型扫描通道所需的「列值 → 目标类型」转换扩展点**已就位** |
| **P0-5 Hook 拦截链**（`orm/hook.go`）：`HookBeforeExecute`（可改写 SQL）/ `HookAfterExecute`，三条执行路径统一挂载 | 不是 GORM 生命周期 hooks（BeforeCreate 等），但「执行前改写 SQL」的机制**已可支撑**时间戳自动填充等横切需求的实现 |
| **方言 4 → 19 种**（`orm/dialector/` 独立子包，8 个方言族；`TableStructureSQL` 全覆盖） | sqlc 类型推断（§3）的内省输入从 4 方言**免费扩大到 19 方言**；dialector 子包化验证了「核心包拆子包」的组织先例 |
| **MP 内置 CRUD**（`types/mp_builtin.go` 加载期内存补生成 + `schema2code -mp` 落盘）：BaseMapper 10 方法、逻辑删除 `deleted/del_flag` 双约定 + `db:",logic"` tag 优先 | 「struct → INSERT/UPDATE/DELETE 语句生成」初版列为 GORM 路线差距，现已有 **MyBatis-Plus 形态的运行时实现**（仍是反射代理，非编译期生成） |
| **RowStream 流式扫描**（`orm/row_stream.go`，P4-2）：`Scan(dest)` 支持 `*map` 与 struct 指针，列名匹配 原名→首字母大写→蛇形转驼峰→大小写不敏感 | 强类型扫描的**列名匹配与反射填充逻辑已验证**，但仍走「先扫 map 再反射」中转，直通道（§4.2）仍缺 |
| 事务模型、强类型直通道、`gormish`/`clause` 包、`cmd/sqlc` | **G0（事务上下文改造 + 强类型扫描直通道）与 G1（`orm/gormish` 链式执行器）已于 2026-09-16 实施完毕（见 §4/§5.1）**；`clause` 包未单设（SQL 组装内聚于 gormish/builder.go）；**S1（`cmd/sqlc`：XML 存量静态 select 抽取 → Querier 生成）已于 2026-09-17 实施完毕（见 §3.4/§5.1）**，S2（`.sql` 输入直通）未动工 |

> ⚠️ 编号约定：本文原用 P0~P3 标注 GORM 路线阶段，与仓库现行编号（P0 框架扩展五件套、P0~P8 数据库适配路线图）冲突。本次修订起改用 **G0~G3**（GORM 路线）与 **S1~S2**（sqlc 路线）。

---

## 1. 现状架构盘点（可复用的基础设施）

先厘清现有实现已经具备、可以被两种新风格直接复用的能力：

| 能力 | 现有实现 | 复用面 |
|------|----------|--------|
| 连接池封装 | `orm.DB`（`database_connection.go`），持有 `ConnPool`、`Statement`、事务槽，实现 `ExecContext/QueryContext/QueryRowContext` | 两种风格共用的「执行后端」（对应 gorm 的 `ConnPool`、sqlc 生成的 `db 接口` 目标） |
| 方言抽象 | `Dialector` 接口，`orm/dialector/` 独立子包，**19 种数据库类型 / 8 个方言族**（Postgres/MySQL/SQLite/Oracle/Informix/MSSQL/DB2/ClickHouse 族，族内继承） | gorm 的 `?`→`$n` 占位符转换、sqlc 的方言差异处理可直接复用；新增方言经族继承自动获得全部能力 |
| 预编译缓存 | `PreparedStmtDB`（LRU 上限 100，会话级缓存） | 链式 API 与生成代码直接透传即可获得 |
| 事务 | `orm.Begin/BeginTx/Commit/Rollback`，Mapper 与 `Execute/Query` 自动参与 | gorm 回调式 `Transaction(fn)` 需在此之上新增；**事务槽仍需重构（见 §4.1，初版提出后尚未实施）** |
| 多数据源 | `InitializeDataSources / UseDataSource / AddDataSource` | gorm 链式 API 按 session 选源；sqlc 生成的实现按需要绑定数据源 |
| 结果行读取 | `fetchRows` → `[]map[string]interface{}`，含扫描目标复用 + `convertFn` 预编译（`common.go`）；**`RowStream.Scan(dest)` 已支持 struct 指针**（列名匹配四策略，P4-2） | 「rows → struct/slice」扫描所需的列名匹配、反射填充、NULL→零值逻辑**已在 RowStream 验证**，仍缺不经 map 中转的直通道（§4.2） |
| 类型转换扩展点 | **`TypeHandler`（P0-4）**：`RegisterTypeHandlerFor[T]`，`convertFieldValue` 统一入口，覆盖参数绑定/resultMap/RowStream 全路径 | gorm/sqlc 扫描通道的「列值 → 目标类型」定制点**已就位** |
| 模型注册与字段缓存 | `modelCache`（`orm/model_cache.go`）+ **db tag 元数据（P0-2，`orm/model_info.go`）**：列名/主键/逻辑删除/乐观锁版本/填充标记/`TableName()` | 初版「gorm schema 解析需从零扩展 tag → 列名/主键」的差距**已基本消除**，gorm 风格可直接复用该元数据体系 |
| 表结构内省 | `newTableStruct / newDatabaseStructure`，经 `Dialector.TableStructureSQL`，**19 种数据库全覆盖**（10 直接实现 + 9 族继承） | sqlc 风格的类型推断、gorm AutoMigrate 的必要输入 |
| 动态 SQL 引擎 | **`types/sqlfragment` 独立包（P0-3）**：`Node` 接口 + `RegisterNodeParser` 开放注册，`if/where/set/foreach/choose/include/sql/trim` 齐备 | 链式 Builder 的 WHERE/HAVING 片段渲染可复用 `FormatValue`/`SQLSegmentProvider` 对接协议 |
| 条件构造器 | **`QueryWrapper`（P0-1）**：链式条件/排序/分组/子查询，mapper 方法参数自动注入（`wrapper_rewrite.go`） | GORM 链式语义的项目内先例；但其「参数内联」策略与 gormish 执行器的「`?` 占位」策略需明确分界（见 §2.4） |
| SQL 改写与拦截 | **Hook 拦截链（P0-5）** + `sql_rewrite.go`（M-07 分页） | gorm 生命周期 hooks（BeforeCreate 等）、时间戳/逻辑删除自动填充可基于 Hook 机制派生 |
| MP 内置 CRUD | `ensureMPBuiltinCRUD`（加载期内存补生成）+ `schema2code -mp`（落盘）：BaseMapper 10 方法、逻辑删除 | 「struct 元数据 → 增删改查 SQL」已有运行时实现；gorm 路线的 Create/Update/Delete 可视为同能力的编程式重包装 |
| 代码生成 | `cmd/generator`（XML→Go）、`cmd/schema2code`（DB→Go，含 `-mp` 模式），产物含 model + Mapper 函数字段骨架 | sqlc 风格是「SQL/模式 → 类型安全 Querier 代码」的自然延伸，同一套骨架可复用 |

**结论**：相比初版撰写时，本项目的「ORM 运行时 + 代码生成 + 表结构元数据」平台属性更加完整：GORM 路线的 **tag schema、类型转换扩展点、SQL 改写挂载点** 三个原计划新增项已经落地，sqlc 路线的**内省输入面扩大到 19 方言**。两条路线真正剩下的核心新增——gormish 独立链式执行器、SQL 静态分析生成链路、事务模型改造、强类型扫描直通道——边界比初版更清晰。

---

## 2. GORM 风格可行性

### 2.1 可行性结论

**高可行性，且较初版差距收敛**。链式 API 本质上是「对 `database/sql` 的一个友好封装层」——本项目已经有这个封装层（`orm.DB`）。初版列出的 5 项缺口，现状如下：

| 初版缺口 | 现状 |
|----------|------|
| 1. 链式 SQL 构造器 | **已落地**（G1：`orm/gormish` 独立链式执行器，`Table()/Model()/Where()/Find()` 等全链子句 + finisher，`?` 占位参数绑定，statement 克隆互不污染；QueryWrapper 继续服务 mapper 参数场景） |
| 2. struct tag 驱动的 schema 元数据 | **已落地**（P0-2 `model_info.go`：column/pk/logic/version/fill/TableName） |
| 3. 强类型扫描通道（`Find(&users)`） | **已落地**（G0：`orm.QueryTo/QueryToContext` 直通道，struct/slice/map/标量，TypeHandler + db tag 列名匹配） |
| 4. 回调式事务（含 savepoint 嵌套） | **回调式已落地**（G0：`orm.WithTx`，嵌套复用外层事务、不占 TCC 槽）；savepoint 嵌套留给 G2 |
| 5. AutoMigrate | **仍缺**（struct→DDL）；但列类型↔Go 类型映射、表结构内省已 19 方言齐备 |

### 2.2 与现有架构的契合点（逐项映射）

| GORM 概念 | 本项目现有对应 | 差距 |
|-----------|----------------|------|
| `*gorm.DB`（可克隆的连接句柄） | `*orm.DB`（全局单例）+ **`gormish.DB`（G1：链式方法克隆 statement，写时复制切片，多克隆互不污染）** | 已消除（G1） |
| `db.Table("t").Where(...)` | **`QueryWrapper` 已提供链式条件构造 + SQL 自动注入**（P0-1）；XML `SqlFunction` 注册期解析、运行时 `GenerateSQL` | **已消除（G1）**：gormish builder.go 独立组装「SQL + 参数 []interface{}」，经 `(db *DB) QueryToContext/ExecContext` 对接 orm 执行层 |
| `Find(&users)` / `Scan` | **`QueryTo/QueryToContext` 直通道已落地**（G0，`scan_rows.go`）；**gormish `Find/First/Last/Count/Scan` 已构建其上（G1）**；`RowStream.Scan` 支持 struct 指针（P4-2） | 已消除（G1） |
| `Create/Update/Delete` | **MP 内置 CRUD 已覆盖** BaseMapper 10 方法；**gormish `Create/Update/Updates/Delete` 已落地（G1）**：零值主键跳过并回填（LastInsertId / PG 族 RETURNING）、Updates struct 跳零值、无 WHERE 条件拒绝执行 | 剩余：fill 自动填充、软删除改写（G2） |
| `Transaction(fn)` | **`orm.WithTx` 已落地**（G0：回调式、嵌套复用、不占 TCC 槽、ctx 优先级）；`Begin/Commit/Rollback`（TCC 风格）语义保留 | 嵌套 savepoint 留给 G2 |
| AutoMigrate | `schema2code`（DB→struct）已存在**反方向**能力；`TableStructure` 内省 19 方言齐备 | struct→DB DDL 生成是新增项，但列类型↔Go 类型映射在 `ChangeType`/`toGolangType` 已有残缺对应 |
| Hooks（BeforeCreate 等） | **Hook 拦截链已有**（P0-5：`HookBeforeExecute` 可改写 SQL / `HookAfterExecute`，三条执行路径统一挂载） | 生命周期语义（按 Create/Update 细分、按模型注册）需在现有全局拦截链之上派生，实现成本低 |
| 软删除 | **MP 逻辑删除已支持**（`deleted/del_flag` 双约定 + `db:",logic"` tag 优先，select 自动过滤、delete 改写 UPDATE） | gormish 侧仅需复用；CreatedAt/UpdatedAt 自动填充仍缺——`db:",fill:insert_update"` 标记已定义，可经 Hook 实现（见 §2.4） |
| 关联（Preload/Joins） | 无 | **最重**，建议 G3 之后再评估 |

### 2.3 推荐引入方式（保兼容）

```go
// 既有写法完全不动（含 P0-1 QueryWrapper 参数注入）
mp := orm.NewMapper("UserInfoModelMapper").(UserInfoModelMapper)
rs, _ := mp.SelectList(orm.NewQueryWrapper().Eq("group_id", 1).OrderByDesc("id"))

// 新增链式写法（G1 已按此形态落地，独立 API surface）
db, _ := gormish.Open()                          // 或 gormish.Use("default") 命名数据源
db.Table("user_info").Where("group_id = ?", 1).Order("id DESC").Find(&users)
db.Model(&user).Create(&user)                    // 零值主键跳过并自增回填
db.Transaction(func(tx *gormish.DB) error {      // 基于 orm.WithTx，嵌套复用
    return tx.Create(&user)
})
```

QueryWrapper 与 gormish 定位互补、不冲突：前者服务 **XML Mapper 的参数化动态条件**（MyBatis-Plus 兼容语义，值内联），后者服务 **编程式完整查询构造**（`?` 占位、独立 session）。二者在 `sqlfragment.SQLSegmentProvider`/`FormatValue` 协议层可共享渲染基建。

架构上分三层，底层与 XML Mapper 完全共享：

```
database/sql  +  orm.DB（连接池/方言/预编译/事务/统计/TypeHandler/Hook）
        │
   ┌────┴──────────────┬──────────────────┐
   │ 现有：XML 解析      │ 新增：链式 Builder │ 新增：SQL渲染引擎(供代码生成回填)
   │ (types/sqlfragment │ (orm/gormish/    │ (orm/sqlbuilder)
   │  + QueryWrapper 注入)│  clause 包)     │
   └────┬──────────────┴──────────────────┘
        │
   ┌────┴──────────────┬──────────────────┐
   │ Mapper 反射代理     │ gormish.Session   │ sqlc 生成的 Querier 实现
   │ (+MP 内置 CRUD)    │                  │
   └───────────────────┴──────────────────┘
```

### 2.4 关键技术点与现有代码的衔接

**强类型扫描直通道已落地（G0）**。`orm/scan_rows.go` 的 `QueryTo/QueryToContext`：`buildScanBindings` 一次预编译「扫描缓冲 + 转换函数 + 字段索引」，每行 `Scan` + 反射 `Set`，无 map 中转；列名匹配 db tag 显式列名优先、回退 `RowStream.Scan` 四策略；转换经 `convertFieldValue`（TypeHandler 优先）。gormish 的 `Find(&users)` 可直接构建其上。

**事务槽重构已完成（G0）**。`orm.WithTx` 经 context 携带事务（`TxFromContext` 提取），不占 TCC 全局槽、支持并发与嵌套复用；执行层 ctx 事务优先于全局槽；Mapper 方法声明 `context.Context` 参数即自动加入（超出原设计的增强）。savepoint 嵌套留给 G2。

**自动填充的实现载体已就位**：`db:",fill:insert|update|insert_update"` 标记（P0-2）已定义 `FillColumns(when)` 查询接口，`HookBeforeExecute`（P0-5）已具备执行前改写 SQL/参数的能力——gormish 的 CreatedAt/UpdatedAt 自动填充可在二者之上实现，无需引入 gorm 式编译器注入。

**占位符策略需明确分界**：QueryWrapper 遵循 MyBatis-Plus 语义采用**值内联**（`FormatValue` 转义，注入面已在文档明示）；gormish 独立执行器必须走 **`?` 占位 + 参数绑定**（现有 `formatSQL`/`FormatPrepareSQL` 兜底，`?`→`$n`/`@p1` 按方言转换）。两套策略并存但互不混用。

**并发语义**：gorm 约定「一个 `*gorm.DB` 实例（从其 `Session()`/`Where()` 克隆）同一时刻只能在一个 goroutine 使用」。本项目 Mapper 代理 + 全局 `gDbConn` 已是如此约定（响应式单会话风格），迁移文档成本低。

---

## 3. sqlc 风格可行性

### 3.1 可行性结论

**高可行性，与项目气质最匹配**（本项目已有 generator / schema2code 两条代码生成线，且 `schema2code -mp` 已验证「schema → CRUD 声明 → 标准方法名」的端到端生成链路）。**A 方案（XML 存量抽取）已于 2026-09-17 实施完毕**（`types/sqlfragment/static_extract.go` 静态性判定 + `types/querier_gen.go` 生成器 + `cmd/sqlc` 命令，见 §3.4）；B 方案（`.sql` 直通，S2）未动工。差异在于思路反转：

| | sqlc | 本项目现状 |
|---|------|-----------|
| 输入 | 纯 `.sql` 查询文件 + DDL | XML Mapper 声明式 SQL |
| 产物 | **编译期**生成的类型安全 Querier 接口 + 实现 | **运行时**反射注入的函数字段代理（MP 内置 CRUD 亦为加载期内存生成） |
| 类型安全 | 强（参数/结果均生成 struct） | 弱（`func(X) ([]Y, error)`，X/Y 需手动对齐 XML；MP 内置方法签名固定但仍是反射执行） |
| 执行开销 | 无反射 | 反射代理 + 动态 SQL 生成 |

### 3.2 与现有架构的契合点

1. **执行后端现成**：sqlc 生成的 Querier 实现本质是「SQL 文本 + 参数 → `database/sql`」。本项目 `orm.DB` 的 `ExecContext/QueryContext/QueryRowContext` + `Dialector.FormatPrepareSQL`（`?`→`$n`，19 方言/8 族）**可以直接作为生成代码的 target**，连预编译缓存都免费获得：
   ```go
   // 生成的代码（示意）
   type UserInfoQuerier interface {
       GetUser(ctx context.Context, id int32) (UserInfo, error)
       ListUsers(ctx context.Context) ([]UserInfo, error)
   }
   func New(db *orm.DB) *Queries { return &Queries{db: db} }
   func (q *Queries) GetUser(ctx context.Context, id int32) (UserInfo, error) {
       row := q.db.QueryRowContext(ctx, `SELECT id, username FROM user_info WHERE id = ?`, id)
       // ... 列类型由生成器静态定死，直接 Scan，无 Null 类型转换开销
   }
   ```
2. **类型推断的基础已扩大到 19 方言**：`newTableStruct` 经 `Dialector.TableStructureSQL` 已覆盖全部 19 种数据库（10 直接实现 + 9 族继承，初版仅 4 方言）。sqlc 的「DDL 推断」可以降级为「**information_schema 实时推断**」：生成时连一次库拿列类型即可，无需引入重型 SQL DDL 解析器。
3. **SQL 文本来源可多路**：
   - 普通路径：`.sql` 查询文件（注解驱动，sqlc 向）
   - 兼容路径：**从现有 XML Mapper 抽取静态 `<select>`**（无动态标签的），生成等价 Querier——让 XML 存量用户无痛获得类型安全产物。动态标签段（`<if>/<where>/<foreach>`）跳过或用哨兵占位；`types/sqlfragment` 的 `Node` 遍历接口（`CollectSlots`/`ContainsForEach`）可精确判定「静态性」。
4. **与现有 Mapper 的关系**：完全独立，生成的是普通 Go 函数而非反射字段，可与 XML Mapper 并存。甚至同一张表三种 API（XML 代理 / MP 内置 / Querier）同时可用。MP 内置 CRUD 与 sqlc 生成是同一诉求（免去手写样板 CRUD）的两种形态——前者运行时反射、零工具链，后者编译期生成、零反射开销，面向不同场景互补。

### 3.3 需要新增的组件

| 组件 | 说明 | 依赖评估 |
|------|------|----------|
| SQL 参数/结果轻量解析 | 从 `.sql` 中提取 `:name` / `?` 占位符、SELECT 列名、表名 | **不需要完整 SQL parser**——限定输入为「参数化 SELECT/INSERT/UPDATE/DELETE 模板」（类 MyBatis `#{}` 语义），词法级提取即可；四方言 SELECT 列名提取用轻量 tokenizer |
| 类型推断 | 列名 + 表名 → Go 类型 | 复用 `newTableStruct`（information_schema / pragma / system.columns 等），**19 方言齐备**；`TypeHandler` 注册表可供生成期识别自定义类型 |
| 代码生成器 | `cmd/sqlc`（新）或扩展现有 `cmd/generator` | 输出 model + Querier 接口 + 实现；复用 `GetShortName/toGolangType/UpperFirst` 等 |
| 运行时支持 | 极小——只需在 `orm` 暴露一个「生成代码执行接口」（`QueryContext` 已达） | 甚至可以零新增 |

### 3.4 两种子方案对比

| 子方案 | 实现量 | 特点 |
|--------|--------|------|
| **A. XML 存量抽取**：跑一遍现有 XML，对每个静态 `<select>` 生成 Querier ✅ 已完成（2026-09-17） | 小（复用 `SqlFunction.Items` 的静态文本拼接 + `sqlfragment` 遍历接口过滤动态标签） | 对当前用户零迁移成本，立即获得类型安全产物。已实施：`ExtractStaticSQL`（simpleSql/纯文本 include 判静态，`#{}`→`?` 按出现序，`${}`/动态标签跳过并记原因）+ `GenerateQuerierFiles`（免连库，resultMap jdbcType / resultType 双轨定型；标量参数具名定型、struct/map 参数跳过；select 恒返回切片）+ `cmd/sqlc`（-p/-d/-m/-v）；生成代码走 `QueryToContext`（占位符/表前缀/ctx 事务由 orm 层承担），标量切片/`[]map` 扫描形态随 S1 落入直通道 |
| **B. sqlc 直通**：新写 `.sql` 查询文件 → 生成 | 中（新增 `.sql` 读取 + 轻量解析 + 类型推断链路） | 面向新用户/新模块，脱离 XML；与 A 共用生成器骨架 |

▸ **建议 A、B 同做**：先做 A（2~3 天量级）验证端到端，再开放 B 的 `.sql` 输入。

> ⚠️ 与真 sqlc 的边界：真 sqlc 手写完整 SQL（含书写顺序、`FROM` 推导 join），本项目建议**限定为参数化模板**（`SELECT ... WHERE id = @id`，`@` 参数显式声明），参数/结果 struct 由生成器推断——这实际是「MyBatis XML 的 SQL 优先简化版 + 编译期生成」，规避了引入 `pingcap/parser` 级别重型解析器的风险。

---

## 4. 两条路线的共同改造点（前置条件）

> **✅ G0 状态：已实施（2026-09-16）**。以下两项均已落地并通过 SQLite 端到端测试（`orm/g0_test.go`，16 个用例）与全量回归；下文保留原始设计描述并附实施差异。

### 4.1 事务模型调整（✅ 已实施）

现状：`DB.curTx` 仍为全局/每 DB 实例单事务槽（`database_connection.go`，`setCurTx` 占槽失败即报错），无回调式事务、无 savepoint、无 context 携带事务。gorm 风格回调事务与并发、sqlc 生成代码的多实例并发都需要**每事务独立**。

改造建议（向后兼容）：
- `orm.DB.BeginTx` 增加「不占用全局槽」的轻量事务句柄，或新增 `WithTx(ctx, fn)` 回调式 API；
- 现有 TCC API（`Begin/Commit/Rollback` + Mapper 自动参与）语义保留；
- 事务感知从「DB 结构体字段」改为「执行上下文」（`context` 携带 `*sql.Tx`，`ExecContext` 优先取上下文事务）。

**实施结果**（`orm/transaction.go`、`orm/database_connection.go`、`orm/proxy_arg.go`、`orm/base_mapper.go`）：
- `orm.WithTx(ctx, fn)` / `(db *DB).WithTx`：fn 返回 nil 提交 / 返回 error 回滚透传 / panic 回滚后 re-panic；ctx 已携带事务时嵌套复用外层（边界由最外层决定）；
- `beginDetachedTx` 轻量事务不占 TCC 全局槽，多 goroutine 可并发各自 WithTx；`orm.TxFromContext(ctx)` 提取 ctx 事务；
- 执行层 `ExecContext/QueryContext/QueryRowContext` ctx 事务优先于全局槽；TCC API 零改动；
- **超出原设计的增强**：Mapper 方法声明 `context.Context` 参数即被 `NewProxyArg` 自动提取（模式同 PageParam/QueryWrapper），方法内 SQL 自动加入 ctx 事务——原设计未覆盖 Mapper 路径的 ctx 透传；
- savepoint 嵌套仍留给 G2（当前嵌套语义为「复用外层事务」）。

### 4.2 结果扫描通道（✅ 已实施）

在 `fetchRows` 旁新增 `scanRowsTo(dst interface{})` 强类型直通道。相比初版，实现素材已齐备：`RowStream.Scan`（P4-2）的列名匹配四策略与反射填充、`convertFieldValue` + `TypeHandler`（P0-4）的统一转换入口、`prepareColumns` 的扫描目标策略均可直接复用。这是 gorm 路线最小的核心改动，工作量较初版下调。

**实施结果**（`orm/scan_rows.go`）：
- `orm.QueryTo(dst, sql, args...)` / `orm.QueryToContext(ctx, dst, ...)`：dst 支持 `*struct`（首行，无行 `sql.ErrNoRows`）/ `*[]struct` / `*[]*struct`（全行，遵循 `DefaultRowLimit`）/ `*map[string]interface{}` / 标量指针（单列）；
- `buildScanBindings` 预编译「扫描缓冲 + 转换函数 + 字段索引」，每行仅 `Scan` + 反射 `Set`，无 map 中转与逐行字符串查列；
- 列名匹配：db tag 显式列名索引（`buildTagColumnIndex`，含非蛇形改名）优先，回退 `findStreamField` 四策略；类型转换经 `convertFieldValue`（TypeHandler 优先）；列级失败聚合为一条错误（M-05 精神）。

### 4.3 新包组织（避免混淆）

```text
orm/            # 现有核心不动（dialector 已于 2026-09-14 拆出，验证了拆包先例）
orm/gormish/    # ✅ G1 已落地（2026-09-16）：链式 API（session/query/crud/builder）
cmd/sqlc/       # 新增：SQL → Querier 代码生成（或并入 cmd/generator）
```

> G1 实施差异：未单设 `clause` 包，SQL 组装内聚于 `orm/gormish/builder.go`（statement 克隆 + WHERE/SELECT/COUNT/INSERT 列组装），gormish 单向依赖 orm（`(db *DB) QueryToContext/ExecContext/WithTx`），无反向依赖。

---

## 5. 工作量与风险

### 5.1 工作量估算（相对值，较初版整体下调）

| 阶段 | 内容 | 工作量 |
|------|------|--------|
| G0（前置）✅ 已完成（2026-09-16） | 事务上下文改造（WithTx/TxFromContext/Mapper ctx 参数）+ 强类型扫描直通道（QueryTo/QueryToContext） | 已实施，16 个端到端用例 + 全量回归通过 |
| G1（GORM 最小可用）✅ 已完成（2026-09-16） | gormish session/克隆 + `Table/Where/Select/Order/Limit/Offset/Find/First/Scan/Create/Update/Delete/Count/Exec/Raw` + 回调事务；Create/Update/Delete 语句生成可参考 MP 内置 CRUD 实现 | 已实施（超出计划：另含 Or/Group/Having/Distinct/Last/Model/Use/New/WithContext/批量插入/主键回填），20 个端到端用例 + 全量回归通过 |
| G2（GORM 进阶） | AutoMigrate、时间戳/软删除自动填充（基于 `fill` 标记 + Hook）、生命周期 hooks（基于现有拦截链派生）、`Transaction` 嵌套 savepoint；~~struct tag schema~~（P0-2 已落地，移出） | 中（约 1.5~2.5 周，较初版下调） |
| G3（GORM 关联） | HasOne/HasMany/BelongsTo/Preload | 大（约 3~5 周，可延后） |
| S1（sqlc A 方案）✅ 已完成（2026-09-17） | XML 存量静态 select 抽取 + 生成器（`sqlfragment` 遍历接口可精确判定静态性） | 已实施：`types/sqlfragment/static_extract.go` + `types/querier_gen.go` + `cmd/sqlc`；types 单测 13 用例 + orm 端到端 3 用例（SQLite 真实执行/生成产物校验/WithTx 事务内路由）+ samples 真实回归（22 mappers → 121 函数）+ 全量回归通过 |
| S2（sqlc B 方案） | `.sql` 输入 + 轻量解析 + 类型推断 + 生成（内省 19 方言免费获得） | 中（约 1.5~2 周） |

两路线并行时资源共享（扫描通道/类型映射/表结构内省/TypeHandler/Hook），总工期接近「G 路线 + S 路线」而非简单相加。

### 5.2 风险清单

| 风险 | 等级 | 缓释 |
|------|------|------|
| 全局单事务槽与并发新 API 冲突 | ~~高（硬性）~~ 已消除（G0） | WithTx 独立事务不占 TCC 槽、ctx 优先级明确，多 goroutine 并发已验证；TCC 旧语义保留 |
| 链式 builder 的 SQL 注入 | 中 | **已部分现实化**：QueryWrapper 走值内联路线（`FormatValue` 转义 + 文档明示注入边界）；gormish 独立执行器必须全部参数走 `?` 占位（`formatSQL`/`FormatPrepareSQL` 兜底），两套策略严格分界、禁止字符串拼接用户输入 |
| 方言占位符/类型差异面随 19 种数据库扩大 | 低 | 方言族继承机制（8 族）使新方言自动获得 `FormatPrepareSQL`/`TableStructureSQL`；gormish/sqlc 均只对接 `Dialector` 抽象，不感知具体方言 |
| SQLite `ScanType` 惰性填充（已踩过坑） | 低 | `RowStream`（P4-2）已沉淀「首行后构建」+ 兜底经验，直通道直接沿用 |
| sqlc 类型推断与列注释不一致（如 PG 无主键表、自增回填） | 中 | 类型推断以 information_schema 为准 + tag/注解显式覆盖；`LastInsertId` 各方言差异只支持「显式 `RETURNING`/`AUTOINCREMENT`」两种明确模式 |
| 关联加载带来的 N+1 与复杂度 | 高（若做） | 延后到 G3，暂不承诺 |
| 引入新包后 go.mod 依赖面扩大 | 低 | 两条路线均**不引入**重量级新依赖——P0 五件套（零新依赖落地）与 P1~P4 方言适配（仅 clickhouse-go 一款驱动）已验证该纪律可持续 |

### 5.3 明确不做（边界声明）

- **不引入上游 gorm 库**（会带来 api/风格/语义三套体系 + 依赖冲突），只复刻其「链式 API + tag schema」核心语义，跑在本项目自己的 `Dialector/Statement/预编译/TypeHandler/Hook` 之上；
- **不引入完整 SQL 解析器**（`pingcap/parser` 等）——sqlc 风格限定为「参数化模板 + information_schema 类型推断」；
- **不把 QueryWrapper 改造为 GORM 形态**——它是 MyBatis-Plus 兼容语义的既定 API（值内联、mapper 参数），与 gormish（占位符、独立执行器）并存且策略分界；
- XML Mapper / Mapper 反射代理 / TCC 事务 / 多数据源 / MP 内置 CRUD / QueryWrapper 现有 API **全部保持兼容**。

---

## 6. 结论

1. **GORM 风格——可行，差距较初版显著收敛**。底层（连接池/方言/预编译/统计/多数据源/TypeHandler/Hook）现成；初版计划的「struct tag schema」「类型转换扩展点」「SQL 改写挂载点」已随 P0 五件套落地；QueryWrapper 验证了链式构造语义。剩余核心新增收敛为「gormish 独立 session/Builder + 强类型扫描直通道」两层。唯一硬性改造仍是事务从「全局单槽」扩展出「每事务独立 + 回调式」（初版提出至今未动工，应最先排期）。
2. **sqlc 风格——可行，工程量最小，且基础较初版更厚**。项目已有两套代码生成器、`schema2code -mp` 端到端先例和 19 方言表结构内省；`sqlfragment` 遍历接口让「XML 静态性判定」变得精确。「XML 存量抽取」（S1）已于 2026-09-17 落地验证（免连库双轨定型 + `cmd/sqlc` 端到端），后续可开放 `.sql` 输入（S2）。
3. **两者可共存且与现有架构互补**：XML Mapper 继续服务声明式/动态 SQL 场景（QueryWrapper 增强），MP 内置 CRUD 服务零手写样板场景，链式 API 服务编程式 CRUD 场景，生成式 Querier 服务性能敏感/类型安全优先场景；四者在 `orm.DB` 执行层会合。
4. **推荐路径**：G0 前置改造已于 2026-09-16 完成落地（`orm.WithTx` 回调事务 + Mapper ctx 参数 + `orm.QueryTo` 强类型扫描直通道）；G1（gormish 链式 API 最小可用）亦于 2026-09-16 完成（`orm/gormish` 包：链式子句 + Find/First/Last/Count/Scan + Create/Update/Updates/Delete/Exec + Transaction，扫描直通道直接复用 QueryTo，事务复用 WithTx）；S1（sqlc A 方案）于 2026-09-17 完成（`cmd/sqlc`：XML 静态 select → 类型安全 Querier，samples 22 mappers 产出 121 个静态函数，动态语句跳过并记原因）。下一步按团队需求排序：S2（`.sql` 输入直通）可复用 S1 生成器骨架，GORM 路线按 G2（AutoMigrate/软删除/hooks/savepoint）推进，G3（关联）暂不承诺。
