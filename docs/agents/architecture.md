# mybatis-go 整体架构图

> 生成时间：2026-09-16 | 基于 types/sqlfragment 拆分 + P0 扩展点（Hook/TypeHandler/db tag/QueryWrapper）后的最新代码

---

## 1. 顶层包依赖关系

```
┌──────────────────────────────────────────────────────────────────┐
│                        cmd/ (入口)                                │
│  demo / mysqldemo / postgresdemo / sqlitedemo / kingbasedemo /   │
│  lkdemo / schema2code / generator                                │
└────────────┬───────────────────────────────────┬────────────────┘
             │                                   │
             ▼                                   ▼
┌────────────────────────┐          ┌─────────────────────────┐
│       orm (主框架)      │          │   types (XML解析引擎)    │
│                        │◄─────────┤                         │
│  ┌──────────────────┐  │  引用     │  SqlMapper / SqlFunction│
│  │ orm/dialector    │  │          │  SqlResult / xml_parse  │
│  │ (数据库方言)      │  │          │  mp_builtin / bridge    │
│  └──────────────────┘  │          │        │                │
└───────┬───────┬────────┘          └────────┼────────────────┘
        │       │                   提取     │
        ▼       ▼                            ▼
┌──────────┐ ┌──────────┐          ┌─────────────────────────┐
│   log    │ │  utils   │          │  types/sqlfragment      │
│ Logger   │ │ ChangeType│         │  (动态 SQL 引擎)         │
│ 接口     │ │ EnvUtils  │          │  Node/注册表/7+1 节点    │
└──────────┘ │ FileUtils │          │  ◄── orm 亦直接引用      │
             │ ListUtils │          └─────────────────────────┘
             └──────────┘
```

**依赖方向**：`cmd` → `orm` → `types` → `types/sqlfragment`（orm 亦直接引用 sqlfragment），`orm` → `utils`/`log`，`types` → `mapper/`（生成产物）；`orm/dialector` 不导入 `orm`，`types` 不导入 `orm`（P0-2 经 `SetModelStructureProvider` 钩子反向注入，无循环依赖）

---

## 2. orm 包内部架构

```
 orm/
 │
 ├─── ┌─────────────────────────────────────────────────────────────┐
 │    │                    初始化层 (Init Layer)                     │
 │    │                                                             │
 │    │  orm_init.go      Initialize() / Reconnect() / Close()     │
 │    │       │                                                     │
 │    │       ├──► database_config.go   Config / MyBatisSetting     │
 │    │       │        └──► ConfigProvider 接口方法                  │
 │    │       │              (GetDSN / GetConnPool / EffectiveSchema│
 │    │       │               GetMaxTimeout / GetMaxOpen / GetSchema)│
 │    │       ├──► orm_cache.go        gCache 全局缓存入口          │
 │    │       └──► multi_datasource.go  gDataSources 多数据源       │
 │    └─────────────────────────────────────────────────────────────┘
 │
 ├─── ┌─────────────────────────────────────────────────────────────┐
 │    │                  连接与方言层 (Connection Layer)             │
 │    │                                                             │
 │    │  database_connection.go   DB 结构体 / Open() / formatSQL() │
 │    │       │                                                     │
 │    │       ├──► orm/dialector/  (独立子包，无反向依赖)            │
 │    │       │     ├── types.go      Dialector / ConfigProvider    │
 │    │       │     │                  ConnPool / PlaceholderStyle  │
 │    │       │     │                  DatabaseFamily / GetDBConnector│
 │    │       │     ├── base.go       BaseDialector (共享默认实现)   │
 │    │       │     │                  FormatPrepareSQLByStyle()     │
 │    │       │     │                  ApplyPagination()             │
 │    │       │     │                  formatDollarPlaceholders()    │
 │    │       │     │                  formatColonPlaceholders()     │
 │    │       │     ├── postgres.go    PostgresDialector             │
 │    │       │     ├── mysql.go       MySqlDialector                │
 │    │       │     ├── sqlite.go      SqliteDialector               │
 │    │       │     └── kingbase.go    KingbaseDialector (嵌PG)      │
 │    │       │                      + pq.Driver 注册 init()        │
 │    │       │                                                     │
 │    │       ├──► interfaces.go    类型别名(= dialector.X)         │
 │    │       ├──► prepared_stmt.go PreparedStmtDB 预编译缓存       │
 │    │       └──► transaction.go   Transaction 事务               │
 │    └─────────────────────────────────────────────────────────────┘
 │
 ├─── ┌─────────────────────────────────────────────────────────────┐
 │    │                  SQL执行层 (Execution Layer)                 │
 │    │                                                             │
 │    │  sql_execute.go     executeWithResult() / queryRows()      │
 │    │       │                                                     │
 │    │       ├──► base_mapper.go    BaseMapper.executeMethod()     │
 │    │       │      └──► sql_rewrite.go  applyPagination()        │
 │    │       │             buildCountSQL()                         │
 │    │       │             └──► dialector.ApplyPagination()        │
 │    │       │                                                     │
 │    │       ├──► row_stream.go     RowStream 流式读取             │
 │    │       └──► table_prefix.go   applyTablePrefix() 改写表名   │
 │    │              isDDLStatement() / invalidateTableNames()      │
 │    └─────────────────────────────────────────────────────────────┘
 │
 ├─── ┌─────────────────────────────────────────────────────────────┐
 │    │                 结果映射层 (Result Layer)                    │
 │    │                                                             │
 │    │  result_convert.go  convert2Results() / prepareColumns()   │
 │    │       │                                                     │
 │    │       ├──► common.go        newInstance() / buildConverters│
 │    │       │                     convertXxx2Yyy 系列转换函数     │
 │    │       └──► return_type.go   ReturnType 类型推断            │
 │    │              return_value.go buildReturnValues()            │
 │    └─────────────────────────────────────────────────────────────┘
 │
  ├─── ┌─────────────────────────────────────────────────────────────┐
  │    │                  缓存与代理层 (Cache & Proxy Layer)         │
  │    │                                                             │
  │    │  orm_cache.go      ormCache / gCache 全局缓存编排           │
  │    │       │                                                     │
  │    │       ├──► mapper_cache.go  mapperCache Mapper注册/查找     │
  │    │       ├──► model_cache.go   modelCache Model注册/查找       │
  │    │       │            + Infos map[string]*ModelInfo (P0-2)    │
  │    │       ├──► proxy_value.go   proxy() / proxyValue() 代理注入│
  │    │       ├──► proxy_arg.go     ProxyArg 参数封装              │
  │    │       │            (PageParam + QueryWrapper 双提取, P0-1) │
  │    │       ├──► param_type.go    ParamType 参数类型解析          │
  │    │       └──► embed_fs.go      RegisterMapperFS 内嵌加载      │
  │    └─────────────────────────────────────────────────────────────┘
  │
  ├─── ┌─────────────────────────────────────────────────────────────┐
  │    │                  扩展点层 (Extension Layer, P0)              │
  │    │                                                             │
  │    │  hook.go          RegisterHook/ClearHooks 拦截链            │
  │    │       │             HookBeforeExecute/HookAfterExecute     │
  │    │       │             (base_mapper 三执行方法统一挂载)        │
  │    │  type_handler.go  RegisterTypeHandlerFor[T] 类型转换器      │
  │    │       │             convertFieldValue 统一入口             │
  │    │  model_info.go    struct db tag 元数据 (ModelInfo/FieldInfo)│
  │    │       │             buildTableStructure / FillColumns      │
  │    │       │             init() 注册 provider → types 包        │
  │    │  query_wrapper.go QueryWrapper 条件构造器（链式 API）       │
  │    │  wrapper_rewrite.go applyQueryWrapper SQL 自动注入          │
  │    │                    splitWrapperTail / injectWrapperArg     │
  │    └─────────────────────────────────────────────────────────────┘
  │
 ├─── ┌─────────────────────────────────────────────────────────────┐
 │    │                 元数据层 (Metadata Layer)                    │
 │    │                                                             │
 │    │  schema_cache.go   tableStructureCache / columnSchemaHint  │
 │    │  table_structure.go    newTableStruct() 表结构查询          │
 │    │  database_structure.go newDatabaseStructure() 全库结构      │
 │    │  schema_utils.go       SchemaToCode() 代码生成入口          │
 │    └─────────────────────────────────────────────────────────────┘
 │
 └─── ┌─────────────────────────────────────────────────────────────┐
      │                    辅助模块 (Auxiliary)                      │
      │                                                             │
      │  errors.go         ErrInvalidDB                             │
      │  statement.go      Statement 执行统计                       │
      │  page.go           PageParam / Page 分页                   │
      │  tag_arg.go        TagArg 参数标签解析                      │
      │  bean_utils.go     beanCheck / methodFieldCheck             │
      │  sql_cache.go      sqlCache (包装 types.SqlMappers)         │
      └─────────────────────────────────────────────────────────────┘
```

---

## 3. orm/dialector 子包内部结构

```
 orm/dialector/
 │
 ├── types.go ─────────── 核心类型定义（零依赖）
 │   ├── PlaceholderStyle     (Question / Dollar / Colon)
 │   ├── DatabaseFamily       (Postgres / MySQL / SQLite / Oracle / Informix)
 │   ├── ConnPool             (PrepareContext / ExecContext / QueryContext / ...)
 │   ├── Dialector            (10 方法接口)
 │   ├── ConfigProvider       (8 方法接口，打破 orm↔dialector 循环依赖)
 │   └── GetDBConnector       (GetDBConn)
 │
 ├── base.go ──────────── BaseDialector 共享实现
 │   ├── BaseDialector struct
 │   │     cfg / name / driverName / dsn / conn / family
 │   │     placeholderStyle / defaultMaxIdle / maxTimeout / maxOpen
 │   ├── Initialize() (ConnPool, error)    ← 新签名，返回连接池
 │   ├── FormatPrepareSQL()                ← 用字段 placeholderStyle，非虚方法
 │   ├── DSN() string                      ← 新增 getter
 │   ├── PlaceholderStyle() / NeedsReturning() / Family()
 │   ├── ApplyPagination() / SystemTablePrefixes()
 │   ├── TableStructureSQL() / TableListSQL()
 │   └── DefaultMaxIdle()
 │   ├── FormatPrepareSQLByStyle()         ← 独立函数，供外部直接调用
 │   ├── ApplyPagination()                 ← 独立函数，orm/sql_rewrite.go 委托
 │   └── formatPlaceholders / formatDollar / formatColon
 │
 ├── postgres.go ──────── PostgresDialector
 │   └── 覆写: NeedsReturning=true, SystemTablePrefixes,
 │            TableStructureSQL(PG专有), TableListSQL(PG专有)
 │
 ├── mysql.go ─────────── MySqlDialector
 │   └── 覆写: SystemTablePrefixes, TableStructureSQL(MySQL),
 │            TableListSQL(MySQL)
 │
 ├── sqlite.go ────────── SqliteDialector
 │   └── 覆写: defaultMaxIdle=1, SystemTablePrefixes,
 │            TableStructureSQL(pragma), TableListSQL(sqlite_master)
 │
 ├── kingbase.go ──────── KingbaseDialector (嵌入 PostgresDialector)
 │   └── init() 注册 kingbase*/pq.Driver
 │
 └── kingbase_test.go ─── 方言单元测试
     └── testConfig (ConfigProvider 桩实现)
```

---

## 4. types/sqlfragment 子包内部结构（P0-3）

```
  types/sqlfragment/
  │
  ├── node.go ─────────── 核心抽象
  │   ├── Node 接口         (Generate(param) (sql, args, err))
  │   ├── containerBase     (子节点容器共享实现)
  │   ├── 节点注册表         (registerNode + init() 注册，开放封闭)
  │   └── ParseFragments    (遍历器入口)
  │
  ├── xml.go / stack.go / element.go ── XML 解析与元素栈
  │
  ├── helpers.go ──────── FormatValue 值内联（字符串加引号/单引号转义）
  │                      SQLSegmentProvider 接口（P0-1 ${ew} 支持）
  │
  ├── simple.go ───────── 文本节点（#{} 参数化 / ${} 内联）
  ├── if.go ───────────── <if> 条件（test 表达式求值）
  ├── foreach.go ──────── <foreach> 集合展开（collection/item/separator）
  ├── choose.go ───────── <choose>/<when>/<otherwise>
  ├── where.go ────────── <where>（前缀 WHERE + 剥离头部 AND/OR）
  ├── set.go ──────────── <set>（前缀 SET + 剥离尾部逗号）
  ├── include.go ──────── <include> + <sql> 片段引用
  └── trim.go ─────────── <trim>（P0-3b，prefix/suffix/prefixOverrides/
                           suffixOverrides，<where>/<set> 的泛化形式）
```

**types 侧适配**：`types/bridge.go` 以类型别名重导出（`type Node = sqlfragment.Node` 等），旧 API 零改动；`sql_function.go` 的 `GenerateSQL` 委托 sqlfragment 遍历器。

---

## 5. 核心调用链

```
用户代码
  │
  ▼
orm.Initialize("application.properties")
  ├── LoadSettings() ──► 解析 properties
  ├── NewConfigFromSettings() ──► Config (含 ConfigProvider 方法)
  ├── Open(cfg) ──►
  │     ├── cfg.Setting.Type.Family() ──► 选择方言族
  │     ├── dialector.NewXxxDialector(cfg) ──► 构造方言
  │     ├── dialector.Initialize() ──► (ConnPool, error) ──► db.ConnPool = pool
  │     └── PreparedStmtDB 包装
  └── gDataSources.add(db)
  │
  ▼
orm.RegisterMapper(new(UserMapper))
  ├── gCache.bindSqls() ──► 关联 XML SqlFunction → struct 函数字段
  └── proxy() ──► 反射注入代理函数
  │
  ▼
userMapper.SelectById(1)
  └── 代理函数 ──► BaseMapper.executeMethod()
        ├── SqlFunction.GenerateSQL(args) ──► 生成参数化 SQL + 参数列表
        ├── normalizeSQL()
        ├── [P0-1] QueryWrapper 注入 ──► applyQueryWrapper（携带 *QueryWrapper 参数时）
        │     └── executePage 场景先注入后派生 COUNT
        ├── [P0-5] runBeforeHooks ──► HookBeforeExecute（可改写 SQL）
        ├── db.applyTablePrefix(sql) ──► 词法改写表名
        ├── db.formatSQL(sql, args) ──► Dialector.FormatPrepareSQL()
        │     └── formatPlaceholders(?→$1/$2 或 :1/:2 或原样)
        ├── needsReturning() ──► Dialector.NeedsReturning()
        │     └── PG: INSERT ... RETURNING col ──► QueryContext
        │     └── MySQL/SQLite: LastInsertId()
        ├── db.ExecContext / db.QueryContext ──► ConnPool
        ├── convert2Results(rows, result) ──► 反射映射到目标类型
        │     └── [P0-4] convertFieldValue ──► 自定义 TypeHandler 优先转换
        └── [P0-5] runAfterHooks ──► HookAfterExecute（Error/Duration/Cost）
```

---

## 6. 关键设计决策

| 决策 | 原因 |
|------|------|
| `orm/dialector` 独立子包 | 解耦方言实现与主框架，新增数据库方言只需改子包 |
| `ConfigProvider` 接口 | 打破 `orm ↔ dialector` 循环依赖：dialector 不导入 orm |
| `Initialize() (ConnPool, error)` | dialector 不依赖 `*DB`，返回连接池由调用方赋值 |
| `placeholderStyle` / `defaultMaxIdle` 字段化 | 绕开 Go 嵌入非虚分派问题：BaseDialector 方法读字段而非调方法 |
| 类型别名 `type Dialector = dialector.Dialector` | orm 包 API 不变，现有调用代码零改动 |
| 常量重导出 `FamilyPostgres = dialector.FamilyPostgres` | orm 包下游代码无需 import dialector |
| `types/sqlfragment` 独立子包 + 节点注册表（P0-3） | 动态 SQL 节点开放封闭扩展（新增节点只需新文件 + init 注册，不改引擎）；`<where>`/`<set>` 降级为 `<trim>` 特例 |
| `SetModelStructureProvider` 钩子（P0-2） | types 不能 import orm（依赖单向），struct db tag 元数据经回调注入 MP CRUD 生成期 |
| Hook panic recover（P0-5） | 拦截链是横切基础设施，单个钩子故障不应中断业务 SQL 执行 |
| QueryWrapper 值接收者 `GetSQLSegment`（P0-1） | 单 map 参数路径经 `convert2Map`/`safeIndirectInterface` 解引用为值类型，值/指针两形态均须满足 `SQLSegmentProvider` |

---

## 7. 已知循环依赖（orm 包内部）

| 循环 | 涉及文件 | 性质 |
|------|----------|------|
| `orm_cache ↔ model_cache` | `getFullName()` 定义在 orm_cache，model_cache 调用 | 同包文件级循环，Go 允许 |
| `orm_init ↔ database_config` | `LoadSettings()` 在 orm_init，database_config 调用 | 同包文件级循环 |
| `common ↔ schema_cache` | `columnSchemaHint` 在 schema_cache，`buildConvertersBasic` 在 common | 同包文件级循环 |
| `sql_execute ↔ schema_cache` | `extractTableNamesFromSQL` ↔ `withExecTimeout` | 同包文件级循环 |
| `table_prefix ↔ database_connection` | `applyTablePrefix` ↔ `*DB` 方法 | 同包文件级循环 |

> 以上均为同包内文件级循环，Go 编译器允许。**跨包无循环依赖**。

---

## 8. 新增方言的扩展路径

添加新数据库（如 openGauss）的完整操作步骤见 **docs/agents/add-dialector.md**。

核心改动点速览：

```
1. orm/dialector/types.go
   ├── const XxxDb DatabaseType = "xxx"
   ├── DatabaseType.Family() 添加 case → 所属族
   ├── ParseDatabaseType() 添加 case "xxx"
   ├── GetDriverName() 添加 case → 驱动名
   └── NewForType() 添加 case → NewXxxDialector()

2. orm/dialector/<name>.go
   ├── init() 注册驱动别名（sql.Register）
   ├── type XxxDialector struct { 嵌入父Dialector }
   └── func NewXxxDialector(cfg ConfigProvider) *XxxDialector

3. orm/interfaces.go
   └── XxxDb = dialector.XxxDb（常量重导出）

4. cmd/<name>demo/main.go + application-<name>.properties

5. orm/dialector/kingbase_test.go + orm/database_config_test.go
   └── 追加单元测试

6. README.md 更新
```

---

## 9. 新增动态 SQL 节点的扩展路径（P0-3 开放封闭）

在 `types/sqlfragment/` 新增一个节点文件即可，引擎与既有节点零改动（参照 `trim.go`）：

```
1. types/sqlfragment/<name>.go
   ├── type sqlXxx struct { 嵌入 containerBase 或独立字段 }
   ├── 实现 Node 接口（Generate(param) (sql, args, err)）
   └── func init() { registerNode("<name>", parseXxx) }

2. types/sqlfragment/<name>_test.go
   └── 解析 + 生成 + 与 MyBatis 语义等价性测试

3. 验证命令
   └── go test -count=1 ./types/sqlfragment/ ./types/ ./orm/
```

> 已注册节点名由 `registerNode` 统一管理；XML 解析期按元素名查注册表分派，未注册元素保持原有报错行为。
