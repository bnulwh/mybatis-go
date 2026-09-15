# mybatis-go 整体架构图

> 生成时间：2026-09-14 | 基于 orm/dialector 子包拆分后的最新代码

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
│  │ orm/dialector    │  │          │  SqlFragment / SqlParam │
│  │ (数据库方言)      │  │          │  SqlResult / xml_parse  │
│  └──────────────────┘  │          └────────────┬────────────┘
└───────┬───────┬────────┘                       │
        │       │                                │
        ▼       ▼                                ▼
┌──────────┐ ┌──────────┐              ┌─────────────────┐
│   log    │ │  utils   │              │  mapper/ (生成)   │
│ Logger   │ │ ChangeType│              │  UserInfoModel   │
│ 接口     │ │ EnvUtils  │              │  Mapper 等       │
└──────────┘ │ FileUtils │              └─────────────────┘
             │ ListUtils │
             └──────────┘
```

**依赖方向**：`cmd` → `orm` → `types`/`utils`/`log`，`orm/dialector` 不导入 `orm`（无循环依赖）

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
 │    │       ├──► proxy_value.go   proxy() / proxyValue() 代理注入│
 │    │       ├──► proxy_arg.go     ProxyArg 参数封装              │
 │    │       ├──► param_type.go    ParamType 参数类型解析          │
 │    │       └──► embed_fs.go      RegisterMapperFS 内嵌加载      │
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

## 4. 核心调用链

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
        ├── db.applyTablePrefix(sql) ──► 词法改写表名
        ├── db.formatSQL(sql, args) ──► Dialector.FormatPrepareSQL()
        │     └── formatPlaceholders(?→$1/$2 或 :1/:2 或原样)
        ├── needsReturning() ──► Dialector.NeedsReturning()
        │     └── PG: INSERT ... RETURNING col ──► QueryContext
        │     └── MySQL/SQLite: LastInsertId()
        ├── db.ExecContext / db.QueryContext ──► ConnPool
        └── convert2Results(rows, result) ──► 反射映射到目标类型
```

---

## 5. 关键设计决策

| 决策 | 原因 |
|------|------|
| `orm/dialector` 独立子包 | 解耦方言实现与主框架，新增数据库方言只需改子包 |
| `ConfigProvider` 接口 | 打破 `orm ↔ dialector` 循环依赖：dialector 不导入 orm |
| `Initialize() (ConnPool, error)` | dialector 不依赖 `*DB`，返回连接池由调用方赋值 |
| `placeholderStyle` / `defaultMaxIdle` 字段化 | 绕开 Go 嵌入非虚分派问题：BaseDialector 方法读字段而非调方法 |
| 类型别名 `type Dialector = dialector.Dialector` | orm 包 API 不变，现有调用代码零改动 |
| 常量重导出 `FamilyPostgres = dialector.FamilyPostgres` | orm 包下游代码无需 import dialector |

---

## 6. 已知循环依赖（orm 包内部）

| 循环 | 涉及文件 | 性质 |
|------|----------|------|
| `orm_cache ↔ model_cache` | `getFullName()` 定义在 orm_cache，model_cache 调用 | 同包文件级循环，Go 允许 |
| `orm_init ↔ database_config` | `LoadSettings()` 在 orm_init，database_config 调用 | 同包文件级循环 |
| `common ↔ schema_cache` | `columnSchemaHint` 在 schema_cache，`buildConvertersBasic` 在 common | 同包文件级循环 |
| `sql_execute ↔ schema_cache` | `extractTableNamesFromSQL` ↔ `withExecTimeout` | 同包文件级循环 |
| `table_prefix ↔ database_connection` | `applyTablePrefix` ↔ `*DB` 方法 | 同包文件级循环 |

> 以上均为同包内文件级循环，Go 编译器允许。**跨包无循环依赖**。

---

## 7. 新增方言的扩展路径

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
