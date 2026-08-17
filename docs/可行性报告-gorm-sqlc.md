# mybatis-go 兼容 GORM 风格 / sqlc 风格 可行性报告

> 撰写日期：2026-08-15
> 基线：`go build ./... && go vet ./... && go test ./orm/... ./types/... ./utils/...` 全绿
> 结论：**两条路线均可行**。现有架构提供了足够的底层复用面（连接池 / 事务 / 方言 / 结果转换 / 表结构内省 / 代码生成），GORM 风格与 sqlc 风格可作为**与 XML Mapper 并存的两种新开发范式**引入，互不冲突；关键前提是把「全局单事务槽」调整为「每实例事务」。

---

## 1. 现状架构盘点（可复用的基础设施）

先厘清现有实现已经具备、可以被两种新风格直接复用的能力：

| 能力 | 现有实现 | 复用面 |
|------|----------|--------|
| 连接池封装 | `orm.DB`（`database_connection.go`），持有 `ConnPool`、`Statement`、事务槽，实现 `ExecContext/QueryContext/QueryRowContext` | 两种风格共用的「执行后端」（对应 gorm 的 `ConnPool`、sqlc 生成的 `db 接口` 目标） |
| 方言抽象 | `Dialector` 接口（`Name/Initialize/FormatPrepareSQL`），MySQL/PG/Kingbase/SQLite 四实现 | gorm 的 `?`→`$n` 占位符转换、sqlc 的方言差异处理可直接复用 |
| 预编译缓存 | `PreparedStmtDB`（LRU 上限 100，会话级缓存） | 链式 API 与生成代码直接透传即可获得 |
| 事务 | `orm.Begin/BeginTx/Commit/Rollback`，Mapper 与 `Execute/Query` 自动参与 | gorm 回调式 `Transaction(fn)` 需在此之上新增；事务槽需重构（见 §4） |
| 多数据源 | `InitializeDataSources / UseDataSource / AddDataSource` | gorm 链式 API 按 session 选源；sqlc 生成的实现按需要绑定数据源 |
| 结果行读取 | `fetchRows` → `[]map[string]interface{}`，含扫描目标复用 + `convertFn` 预编译（`common.go`） | gorm/sqlc 需要的「rows → struct/slice」扫描在此之上新增一个 scan 到强类型指针的通道，但 `newInstance / resolveConverter / 时间转换` 等全部复用 |
| 模型注册与字段缓存 | `modelCache`（`orm/model_cache.go`），已缓存 `reflect.Type` | gorm schema 解析（tag → 列名/主键）以此为基础扩展 |
| 表结构内省 | `newTableStruct / newDatabaseStructure`，从 information_schema / pragma_table_info 读列、类型、注释、主键 | sqlc 风格的类型推断、gorm AutoMigrate 的必要输入，**已四方言齐备** |
| 代码生成 | `cmd/generator`（XML→Go）、`cmd/schema2code`（DB→Go），产物含 model + Mapper 函数字段骨架 | sqlc 风格是「SQL/模式 → 类型安全 Querier 代码」的自然延伸，同一套骨架可复用 |

**结论**：本项目名为 ORM，实则已经是一套「ORM 运行时 + 代码生成 + 表结构元数据」的平台。GORM / sqlc 真正值钱的部分——链式 SQL 构造器、struct tag 驱动 schema、SQL 静态分析、类型安全代码生成——才是需要新增的，底层能力基本齐备。

---

## 2. GORM 风格可行性

### 2.1 可行性结论

**高可行性**。链式 API 本质上是「对 `database/sql` 的一个友好封装层」——本项目已经有这个封装层（`orm.DB`），缺的是：

1. 链式 SQL 构造器（`Where/Select/Order/Limit/Joins/...`）
2. struct tag 驱动的 schema 元数据（`gorm:"column:xx;primaryKey"`）
3. 强类型扫描通道（`Find(&users)` 直接填充 struct 切片）
4. 回调式事务（`Transaction(func(tx) error)`，含 savepoint 嵌套）
5. AutoMigrate（结构体 → DDL，基于已有的表结构内省）

### 2.2 与现有架构的契合点（逐项映射）

| GORM 概念 | 本项目现有对应 | 差距 |
|-----------|----------------|------|
| `*gorm.DB`（可克隆的连接句柄） | `*orm.DB`（全局单例） | 需新增「session 克隆」概念：链式调用多次 `Where` 不能污染共享实例，需拆 `Statement`/`Clause` 状态 |
| `db.Table("t").Where(...)` | XML 的 `SqlFunction` 在注册期解析，运行时 `GenerateSQL` | 链式构造器把「SQL 生成」从 XML 解析迁移为运行时逐条追加，输出仍是「SQL + 参数 []interface{}」，与 `execute/queryRows` 天然对接 |
| `Find(&users)` / `Scan` | `convert2Results`（map → struct），`fetchRows`（rows → map） | **只需要新增一个中间通道**：rows → 目标 struct 切片（直接 `Scan` 到强类型，或复用现有 map 通道再定型）。强类型扫描方案见下 |
| `Create/Update/Delete` | XML 的 Insert/Update/Delete 函数 | 需新增「struct → INSERT/UPDATE/DELETE 语句生成」（利用 §1 表结构内省或 struct tag） |
| `Transaction(fn)` | `Begin/Commit/Rollback`（TCC 风格） | 需加回调式 API 与嵌套 savepoint |
| AutoMigrate | `schema2code`（DB→struct）已存在**反方向**能力 | struct→DB DDL 生成是新增项，但列类型↔Go 类型映射在 `ChangeType`/`toGolangType` 已有残缺对应 |
| Hooks（BeforeCreate 等） | 无 | 全新，但 Go 接口回调（非反射）实现成本低 |
| 软删除 / CreatedAt 自动填充 | 无 | struct tag + 编译器在 Create/Update 时注入，成本低 |
| 关联（Preload/Joins） | 无 | **最重**，建议 P2 之后再评估 |

### 2.3 推荐引入方式（保兼容）

```go
// 既有写法完全不动
mp := orm.NewMapper("UserInfoModelMapper").(UserInfoModelMapper)
rs, _ := mp.SelectAll()

// 新增链式写法（独立 API surface）
db := gormish.Use("default")                    // 复用 orm 的连接池/多数据源
db.Table("user_info").Where("group_id = ?", 1).Order("id DESC").Find(&users)
db.Create(&user)
gormish.Transaction(func(tx *gormish.DB) error { ... })
```

架构上分三层，底层与 XML Mapper 完全共享：

```
database/sql  +  orm.DB（连接池/方言/预编译/事务/统计）
        │
   ┌────┴──────────────┬──────────────────┐
   │ 现有：XML 解析      │ 新增：链式 Builder │ 新增：SQL渲染引擎(供代码生成回填)
   │ (types/sql_fragment)│ (orm/clause 包)  │ (orm/sqlbuilder)
   └────┬──────────────┴──────────────────┘
        │
   ┌────┴──────────────┬──────────────────┐
   │ Mapper 反射代理     │ gormish.Session   │ sqlc 生成的 Querier 实现
   └───────────────────┴──────────────────┘
```

### 2.4 关键技术点与现有代码的衔接

**强类型扫描是核心差距**。现有 `fetchRows` 走 `[]map[string]interface{}`，列名→目标字段可以用与 `buildFieldIndexMap` 相同的「预编译字段索引」思路，直接新增：

```go
// 新增（复用 common.go 的 newInstance/resolveConverter 与时间处理）
func scanInto(dst interface{}) error   // rownum 循环：逐列 Scan 到目标字段指针，NULL→零值
```

顺序：`ColumnTypes()` → 目标 struct 字段（tag or 蛇形名匹配，可走现有 `buildFieldIndexMap` 逻辑）→ 每列一个 scan 目标 → 复用 `convertFn` 预编译。

**事务槽重构是唯一硬性前提**（详见 §4）。现有 `db.curTx` 是全局单槽（README 明确「请勿多 goroutine 交错开启事务」），gorm 风格的 `db.Transaction(fn)` 需要**每事务独立**、且支持并发与 savepoint 嵌套。改造面小：事务上下文从 `DB` 移到 session/goroutine 局部，或提供 `Transaction` 的独立连接获取路径；现有 TCC API（`Begin/Commit/Rollback`）保持全局槽语义以兼容旧代码。

**并发语义**：gorm 约定「一个 `*gorm.DB` 实例（从其 `Session()`/`Where()` 克隆）同一时刻只能在一个 goroutine 使用」。本项目 Mapper 代理 + 全局 `gDbConn` 已是如此约定（响应式单会话风格），迁移文档成本低。

---

## 3. sqlc 风格可行性

### 3.1 可行性结论

**高可行性，且与项目气质最匹配**（本项目已有 generator / schema2code 两条代码生成线）。差异在于思路反转：

| | sqlc | 本项目现状 |
|---|------|-----------|
| 输入 | 纯 `.sql` 查询文件 + DDL | XML Mapper 声明式 SQL |
| 产物 | **编译期**生成的类型安全 Querier 接口 + 实现 | **运行时**反射注入的函数字段代理 |
| 类型安全 | 强（参数/结果均生成 struct） | 弱（`func(X) ([]Y, error)`，X/Y 需手动对齐 XML） |
| 执行开销 | 无反射 | 反射代理 + 动态 SQL 生成 |

### 3.2 与现有架构的契合点

1. **执行后端现成**：sqlc 生成的 Querier 实现本质是「SQL 文本 + 参数 → `database/sql`」。本项目 `orm.DB` 的 `ExecContext/QueryContext/QueryRowContext` + `Dialector.FormatPrepareSQL`（`?`→`$n`）**可以直接作为生成代码的 target**，连预编译缓存都免费获得：
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
2. **类型推断的基础已有**：`newTableStruct` 已能从四方言读取列类型/可空/主键/注释（§1）。sqlc 的「DDL 推断」可以降级为「**information_schema 实时推断**」：生成时连一次库拿列类型即可，无需引入重型 SQL DDL 解析器。
3. **SQL 文本来源可多路**：
   - 普通路径：`.sql` 查询文件（注解驱动，sqlc 向）
   - 兼容路径：**从现有 XML Mapper 抽取静态 `<select>`**（无动态标签的），生成等价 Querier——让 XML 存量用户无痛获得类型安全产物。动态标签段（`<if>/<where>`）跳过或用哨兵占位。
4. **与现有 Mapper 的关系**：完全独立，生成的是普通 Go 函数而非反射字段，可与 XML Mapper 并存。甚至同一张表两种 API 同时可用。

### 3.3 需要新增的组件

| 组件 | 说明 | 依赖评估 |
|------|------|----------|
| SQL 参数/结果轻量解析 | 从 `.sql` 中提取 `:name` / `?` 占位符、SELECT 列名、表名 | **不需要完整 SQL parser**——限定输入为「参数化 SELECT/INSERT/UPDATE/DELETE 模板」（类 MyBatis `#{}` 语义），词法级提取即可；四方言 SELECT 列名提取用轻量 tokenizer |
| 类型推断 | 列名 + 表名 → Go 类型 | 复用 `newTableStruct`（information_schema / pragma），已有四方言支持 |
| 代码生成器 | `cmd/sqlc`（新）或扩展现有 `cmd/generator` | 输出 model + Querier 接口 + 实现；复用 `GetShortName/toGolangType/UpperFirst` 等 |
| 运行时支持 | 极小——只需在 `orm` 暴露一个「生成代码执行接口」（`QueryContext` 已达） | 甚至可以零新增 |

### 3.4 两种子方案对比

| 子方案 | 实现量 | 特点 |
|--------|--------|------|
| **A. XML 存量抽取**：跑一遍现有 XML，对每个静态 `<select>` 生成 Querier | 小（复用 `SqlFunction.Items` 的静态文本拼接，过滤动态标签） | 对当前用户零迁移成本，立即获得类型安全产物 |
| **B. sqlc 直通**：新写 `.sql` 查询文件 → 生成 | 中（新增 `.sql` 读取 + 轻量解析 + 类型推断链路） | 面向新用户/新模块，脱离 XML；与 A 共用生成器骨架 |

▸ **建议 A、B 同做**：先做 A（2~3 天量级）验证端到端，再开放 B 的 `.sql` 输入。

> ⚠️ 与真 sqlc 的边界：真 sqlc 手写完整 SQL（含书写顺序、`FROM` 推导 join），本项目建议**限定为参数化模板**（`SELECT ... WHERE id = @id`，`@` 参数显式声明），参数/结果 struct 由生成器推断——这实际是「MyBatis XML 的 SQL 优先简化版 + 编译期生成」，规避了引入 `pingcap/parser` 级别重型解析器的风险。

---

## 4. 两条路线的共同改造点（前置条件）

### 4.1 事务模型调整（硬性）

现状：`DB.curTx` 全局单事务槽。gorm 风格回调事务与并发、sqlc 生成代码的多实例并发都需要**每事务独立**。

改造建议（向后兼容）：
- `orm.DB.BeginTx` 增加「不占用全局槽」的轻量事务句柄，或新增 `WithTx(ctx, fn)` 回调式 API；
- 现有 TCC API（`Begin/Commit/Rollback` + Mapper 自动参与）语义保留；
- 事务感知从「DB 结构体字段」改为「执行上下文」（`context` 携带 `*sql.Tx`，`ExecContext` 优先取上下文事务）。

### 4.2 结果扫描通道（gorm 必需，sqlc 可选）

在 `fetchRows` 旁新增 `scanRowsTo(dst interface{})` 强类型通道，复用 `prepareColumns` 的扫描目标策略与 `common.go` 转换函数。这是 gorm 路线最小的核心改动。

### 4.3 新包组织（避免混淆）

```text
orm/            # 现有核心不动（少量内部重构）
orm/gormish/    # 新增：链式 API（session/builder/schema/hooks）
cmd/sqlc/       # 新增：SQL → Querier 代码生成（或并入 cmd/generator）
```

---

## 5. 工作量与风险

### 5.1 工作量估算（相对值）

| 阶段 | 内容 | 工作量 |
|------|------|--------|
| P0（前置） | 事务上下文改造 + 强类型扫描通道 | 中（约 1~2 周） |
| P1（GORM 最小可用） | `Table/Where/Select/Order/Limit/Offset/Find/First/Scan/Create/Update/Delete/Count/Exec/Raw` + 回调事务 | 中（约 2~3 周） |
| P2（GORM 进阶） | struct tag schema、AutoMigrate、时间戳/软删除、Hooks、`Transaction` 嵌套 savepoint | 中（约 2~3 周） |
| P3（GORM 关联） | HasOne/HasMany/BelongsTo/Preload | 大（约 3~5 周，可延后） |
| S1（sqlc A 方案） | XML 存量静态 select 抽取 + 生成器 | 小（约 3~5 天） |
| S2（sqlc B 方案） | `.sql` 输入 + 轻量解析 + 类型推断 + 生成 | 中（约 2 周） |

两路线并行时资源共享（扫描通道/类型映射/表结构内省），总工期接近「P 路线 + S 路线」而非简单相加。

### 5.2 风险清单

| 风险 | 等级 | 缓释 |
|------|------|------|
| 全局单事务槽与并发新 API 冲突 | 高（硬性） | 前置改造，TCC 旧语义保留；文档明确并发边界 |
| 链式 builder 的 SQL 注入 | 中 | 全部参数走 `?` 占位（现有 `formatSQL` 已兜底）；禁止字符串拼接用户输入 |
| SQLite `ScanType` 惰性填充（已踩过坑） | 低 | 复用现有「首行后构建」策略 + `NullString` 兜底 |
| sqlc 类型推断与列注释不一致（如 PG 无主键表、自增回填） | 中 | 类型推断以 information_schema 为准 + tag/注解显式覆盖；`LastInsertId` 各方言差异只支持「显式 `RETURNING`/`AUTOINCREMENT`」两种明确模式 |
| 关联加载带来的 N+1 与复杂度 | 高（若做） | 延后到 P3，暂不承诺 |
| 引入新包后 go.mod 依赖面扩大 | 低 | 两条路线均**不引入**重量级新依赖（无 gorm 依赖，无 SQL parser 依赖）——复刻核心语义而非依赖上游 |

### 5.3 明确不做（边界声明）

- **不引入上游 gorm 库**（会带来 api/风格/语义三套体系 + 依赖冲突），只复刻其「链式 API + tag schema」核心语义，跑在本项目自己的 `Dialector/Statemement/预编译` 之上；
- **不引入完整 SQL 解析器**（`pingcap/parser` 等）——sqlc 风格限定为「参数化模板 + information_schema 类型推断」；
- XML Mapper / Mapper 反射代理 / TCC 事务 / 多数据源现有 API **全部保持兼容**。

---

## 6. 结论

1. **GORM 风格——可行，代价可控**。底层（连接池/方言/预编译/统计/多数据源）全部现成；核心新增是「链式 Builder + struct tag schema + 强类型扫描」三层，且都能落在现有 `execute/queryRows` 与 `common.go` 转换体系上。唯一硬性改造是事务从「全局单槽」扩展出「每事务独立 + 回调式」。
2. **sqlc 风格——可行，且工程量最小**。项目已有两套代码生成器和四方言表结构内省，缺的只是「SQL 文本 → 参数/结果类型 → Querier 代码」的轻量生成链路。建议「XML 存量抽取」先行，验证后开放 `.sql` 输入。
3. **两者可共存且与现有架构互补**：XML Mapper 继续服务声明式/动态 SQL 场景，链式 API 服务编程式 CRUD 场景，生成式 Querier 服务性能敏感/类型安全优先场景；三者在 `orm.DB` 执行层会合。
4. **推荐路径**：先做 P0 前置改造（事务上下文 + 强类型扫描），它是两条路线共同地基；随后按团队需求排序，S1 可在 3~5 天内先出成绩。