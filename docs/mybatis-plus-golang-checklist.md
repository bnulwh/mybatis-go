# MyBatis / MyBatis-Plus → Golang 可实现内容清单

> 基于 MyBatis 3.5.x + MyBatis-Plus 3.5.x 全量功能，结合 Go 语言特性（无泛型擦除、无注解、无 OGNL、无运行时字节码增强）逐项评估可行性。
> 标记：✅ 已实现 | 🔶 可实现需适配 | 🟡 Go 下意义有限/需大幅改造 | ❌ 不适用于 Go | 🆕 Go 原生增强

---

## 一、MyBatis 核心功能

### 1. XML Mapper 配置

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 1.1 | `<select>/<insert>/<update>/<delete>` | XML 声明式 SQL | ✅ 完全可行 | ✅ 已实现 | |
| 1.2 | `#{param}` 预编译参数绑定 | PreparedStatement 占位符 | ✅ 完全可行 | ✅ 已实现 | 支持 jdbcType |
| 1.3 | `${param}` 原始字符串替换 | 拼接 SQL | ✅ 完全可行 | ✅ 已实现 | 需注意注入风险 |
| 1.4 | `parameterType` 参数类型声明 | Java 全限定类名 | ✅ 可行 | ✅ 已实现 | Go 用 struct/短类名 |
| 1.5 | `resultType` 自动映射 | 反射 | ✅ 完全可行 | ✅ 已实现 | snake→PascalCase |
| 1.6 | `resultMap` 显式映射 | XML `<resultMap>` | ✅ 完全可行 | ✅ 已实现 | id/result/association/collection |
| 1.7 | `useGeneratedKeys` 自增主键回填 | JDBC getGeneratedKeys | ✅ 完全可行 | ✅ 已实现 | PG:RETURNING; MySQL/SQLite:LastInsertId |
| 1.8 | 多参数绑定 | `@Param` 注解 | ✅ 可行 | ✅ 已实现 | Go 用 `args:"a,b"` tag + slot |
| 1.9 | Dot 表达式 `#{a.b}` | OGNL | ✅ 可行 | ✅ 已实现 | 反射链式取值 |
| 1.10 | 命名空间 namespace | Java 接口全限定名 | ✅ 完全可行 | ✅ 已实现 | Go 自定义字符串 |
| 1.11 | `flushCache`/`useCache` | 语句级缓存控制 | 🔶 可实现 | — | 见缓存章节 |
| 1.12 | `timeout` 语句级超时 | JDBC queryTimeout | ✅ 完全可行 | ✅ 已实现 | context.WithTimeout |
| 1.13 | `statementType` STATEMENT/PREPARED/CALLABLE | JDBC 语句类型 | 🔶 可实现 | — | PREPARED 已实现；CALLABLE 见存储过程 |
| 1.14 | `keyColumn` 自增列名 | 配合 useGeneratedKeys | ✅ 完全可行 | ✅ 已实现 | 自动从 keyProperty 推导 |
| 1.15 | `databaseId` 多数据库 SQL 选择 | 多数据库支持 | 🔶 可实现 | — | 见多数据库方言 |
| 1.16 | `lang` 自定义脚本语言 | 实现 LanguageDriver | 🟡 意义有限 | — | Go 无脚本引擎生态 |
| 1.17 | `<sql>` + `<include refid>` | SQL 片段复用 | ✅ 完全可行 | ✅ 已实现 | 含参数替换 |

### 2. 动态 SQL

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 2.1 | `<if test="...">` 条件判断 | OGNL 表达式 | ✅ 可行 | ✅ 已实现 | null/empty/bool/numeric，不支持 OGNL 全集 |
| 2.2 | `<choose>/<when>/<otherwise>` | switch-case | ✅ 完全可行 | ✅ 已实现 | |
| 2.3 | `<where>` 前缀 AND/OR 清理 | 动态 SQL | ✅ 完全可行 | ✅ 已实现 | |
| 2.4 | `<set>` UPDATE SET 子句 | 动态 SQL | ✅ 完全可行 | ✅ 已实现 | |
| 2.5 | `<foreach>` 集合遍历 | OGNL + Iterable | ✅ 完全可行 | ✅ 已实现 | collection/item/open/close/separator |
| 2.6 | `<trim>` 自定义前后缀 | 动态 SQL | 🔶 可实现 | — | where/set 是 trim 的特化；Go 可直接实现 |
| 2.7 | `<bind>` 变量绑定 | OGNL 赋值 | 🔶 可实现 | — | Go 需要简单表达式引擎，或改用 Go 模板函数 |
| 2.8 | `<if test>` OGNL 完整表达式 | 方法调用/三元/正则 | 🟡 需大幅改造 | — | 见「表达式引擎」章节 |
| 2.9 | OGNL 静态方法调用 | `@java.lang.Math@abs(x)` | ❌ 不适用 | — | Go 无静态方法；需替代方案 |

### 3. ResultMap 高级映射

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 3.1 | `<id>` 主键映射 | 标记唯一性字段 | ✅ 完全可行 | ✅ 已实现 | |
| 3.2 | `<result>` 普通字段映射 | 列→字段 | ✅ 完全可行 | ✅ 已实现 | |
| 3.3 | `<association>` 一对一 | 嵌套对象 | ✅ 完全可行 | ✅ 已实现 | |
| 3.4 | `<collection>` 一对多 | 嵌套集合 | ✅ 完全可行 | ✅ 已实现 | |
| 3.5 | `<discriminator>` 鉴别器 | 根据列值选择不同 resultMap | 🔶 可实现 | — | 可在 XML 解析层实现，Go 端返回 interface/any |
| 3.6 | 嵌套 Select（N+1） | 延迟加载 | 🟡 需改造 | — | Go 无懒加载代理；需显式调用或改用嵌套 Join |
| 3.7 | 嵌套 ResultMap（Join 映射） | 前缀列映射 | ✅ 完全可行 | ✅ 已实现 | association/collection 内嵌 resultMap |
| 3.8 | `constructor` 构造器注入 | Java 构造函数 | 🟡 需改造 | — | Go 无构造函数注入；用 struct tag 或工厂函数 |
| 3.9 | 多结果集映射 | JDBC multiple ResultSet | ❌ 不适用 | — | database/sql 不支持多结果集 |
| 3.10 | `columnPrefix` 列前缀 | Join 结果去歧义 | 🔶 可实现 | — | 可在 resultMap 解析层加前缀过滤 |
| 3.11 | `notNullColumn` 非空列检测 | 跳过全 null 行 | 🔶 可实现 | — | 对 collection 去空行有用 |

### 4. 缓存

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 4.1 | 一级缓存（SqlSession 级） | 默认开启 | 🔶 可实现 | — | Go 无 Session 概念；可在事务级别实现 |
| 4.2 | 二级缓存（namespace 级） | CachingExecutor 装饰器 | 🔶 可实现 | — | 需要 LRU + TTL 实现；需考虑并发安全 |
| 4.3 | 自定义缓存实现 | 实现 Cache 接口 | ✅ 完全可行 | — | Go interface 天然适配（如 Redis adapter） |
| 4.4 | `cache-ref` 跨命名空间引用 | 共享缓存实例 | 🔶 可实现 | — | 缓存层实现后自然支持 |
| 4.5 | `flushCache="true"` 语句级 | 写操作刷新缓存 | 🔶 可实现 | — | 依赖缓存层 |
| 4.6 | `readOnly` 只读缓存 | 序列化 vs 引用 | 🟡 意义有限 | — | Go 无序列化要求差异；缓存就是引用 |
| 4.7 | `eviction` 淘汰策略 | LRU/FIFO/SOFT/WEAK | 🔶 可实现 | — | LRU/FIFO 可实现；SOFT/WEAK 依赖 GC |
| 4.8 | `blocking` 防缓存击穿 | 同一 key 只加载一次 | ✅ 完全可行 | — | Go singleflight 天然解决 |

### 5. 插件 / 拦截器

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 5.1 | Interceptor 拦截链 | JDK 动态代理 | 🔶 可实现 | — | Go 无动态代理；用中间件/拦截器模式 |
| 5.2 | 拦截 Executor | 拦截 query/update | 🔶 可实现 | — | 在 BaseMapper 层加 Hook 函数链 |
| 5.3 | 拦截 StatementHandler | 拦截 prepare/parameterize | 🔶 可实现 | — | 在 SQL 执行层加 Hook |
| 5.4 | 拦截 ResultSetHandler | 拦截结果映射 | 🔶 可实现 | — | 在 resultConvert 层加 Hook |
| 5.5 | 拦截 ParameterHandler | 拦截参数设置 | 🔶 可实现 | — | 在 buildArgs 层加 Hook |
| 5.6 | `@Intercepts` + `@Signature` | 注解声明拦截点 | 🟡 需改造 | — | Go 用注册式 API：`orm.RegisterHook(point, fn)` |
| 5.7 | Plugin.wrap 代理生成 | 动态代理 | ❌ 不适用 | — | Go 用显式中间件链代替 |

### 6. TypeHandler 类型处理器

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 6.1 | 内置类型映射 | JDBC Type ↔ Java Type | ✅ 完全可行 | ✅ 已实现 | 自动 scan 到基本类型 |
| 6.2 | 自定义 TypeHandler | 实现 TypeHandler 接口 | ✅ 完全可行 | — | `orm.RegisterTypeHandler(t, scanFn, valueFn)` |
| 6.3 | Enum → varchar/int 映射 | 枚举序列化 | 🔶 可实现 | — | Go enum 是 iota int；需自定义映射 |
| 6.4 | JSON/JSONB 列映射 | 存为 JSON 字符串 | ✅ 完全可行 | — | TypeHandler 扫描为 []byte → json.Unmarshal |
| 6.5 | `jdbcType` 声明 | 强制类型 | 🔶 可实现 | ✅ 部分实现 | 当前用于参数；可扩展到结果 |
| 6.6 | `javaType` 声明 | 目标类型 | ✅ 完全可行 | — | Go 已有 resultType，等效 |

### 7. 事务管理

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 7.1 | 编程式事务 | sqlSession.commit/rollback | ✅ 完全可行 | ✅ 已实现 | Begin/Commit/Rollback |
| 7.2 | 声明式事务 | Spring @Transactional | 🟡 需改造 | — | Go 无 AOP；可用 context 传递 tx |
| 7.3 | 事务传播行为 | Spring 7 种传播 | 🔶 可实现 | — | Go 实现Required/New 就够用 |
| 7.4 | 事务隔离级别 | JDBC isolation | ✅ 完全可行 | ✅ 已实现 | BeginTx(ctx, &sql.TxOptions{Isolation:...}) |
| 7.5 | 嵌套事务（Savepoint） | JDBC savepoint | 🔶 可实现 | — | database/sql 的 Tx 不直接支持；需底层驱动支持 |
| 7.6 | 事务超时 | Spring timeout | ✅ 完全可行 | ✅ 已实现 | context.WithTimeout |

### 8. 数据源 & 连接池

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 8.1 | 多数据源切换 | Spring AbstractRoutingDS | ✅ 完全可行 | ✅ 已实现 | UseDataSource(name) |
| 8.2 | 连接池配置 | HikariCP/Druid | ✅ 完全可行 | ✅ 已实现 | Go 用 database/sql 内置池 |
| 8.3 | 连接池监控 | 活跃/空闲/等待 | 🔶 可实现 | — | database/sql.DB.Stats() |
| 8.4 | 数据源动态增删 | 运行时注册/注销 | ✅ 完全可行 | ✅ 已实现 | AddDataSource/GetDataSource |
| 8.5 | 读写分离 | 主从路由 | 🔶 可实现 | — | 多数据源 + 路由策略 |
| 8.6 | 分库分表路由 | ShardingSphere | 🟡 需大幅改造 | — | 超出框架范畴；表前缀 map 已部分覆盖 |

### 9. 其他 MyBatis 功能

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 9.1 | 存储过程调用 | `{call proc(?)}` | 🔶 可实现 | — | database/sql 支持；需包装 API |
| 9.2 | mybatis-config.xml | 全局配置 | ✅ 完全可行 | ✅ 已实现 | .properties 文件 |
| 9.3 | Settings 全局参数 | cacheEnabled/lazyLoadingEnabled 等 | 🔶 可实现 | ✅ 部分实现 | 可逐步扩展配置项 |
| 9.4 | TypeAliases 类型别名 | 短名→全限定 | ✅ 完全可行 | ✅ 已实现 | 短类名自动注册 |
| 9.5 | ObjectFactory 对象工厂 | 自定义实例化 | 🔶 可实现 | — | Go 用 reflect.New；可注册自定义构造函数 |
| 9.6 | DatabaseIdProvider | 数据库标识 | 🔶 可实现 | — | dialector 已识别 dbType；可扩展 SQL 选择 |
| 9.7 | Lazy loading 延迟加载 | CGLIB/Javassist 代理 | ❌ 不适用 | — | Go 无运行时代理生成 |
| 9.8 | 日志集成 | slf4j/log4j/logback | ✅ 完全可行 | ✅ 已实现 | 可插拔 Logger 接口 |
| 9.9 | SQL 打印 | 日志输出完整 SQL | 🔶 可实现 | ✅ 部分实现 | Debug 级别可输出；可增强参数替换后的完整 SQL |
| 9.10 | Mapper 注解模式 | `@Select/@Insert` 注解 | 🟡 需改造 | — | Go 无注解；可用代码生成或 Builder API |

---

## 二、MyBatis-Plus 功能

### 10. BaseMapper 内置 CRUD

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 10.1 | `insert(T)` | 插入一条 | ✅ 完全可行 | ✅ 已实现 | |
| 10.2 | `deleteById(Serializable)` | 按 ID 删除 | ✅ 完全可行 | ✅ 已实现 | 含逻辑删除 |
| 10.3 | `updateById(T)` | 按 ID 更新 | ✅ 完全可行 | ✅ 已实现 | |
| 10.4 | `selectById(Serializable)` | 按 ID 查询 | ✅ 完全可行 | ✅ 已实现 | |
| 10.5 | `selectBatchIds(Collection)` | 批量 ID 查询 | ✅ 完全可行 | ✅ 已实现 | |
| 10.6 | `deleteBatchIds(Collection)` | 批量 ID 删除 | ✅ 完全可行 | ✅ 已实现 | |
| 10.7 | `selectList(Wrapper)` | 条件查询列表 | 🔶 可实现 | ✅ 部分实现 | 当前无 Wrapper，返回全表；见 Wrapper 章节 |
| 10.8 | `selectOne(Wrapper)` | 条件查单条 | 🔶 可实现 | ✅ 部分实现 | 同上 |
| 10.9 | `selectCount(Wrapper)` | 条件计数 | 🔶 可实现 | ✅ 已实现 | |
| 10.10 | `selectPage(Page, Wrapper)` | 分页查询 | 🔶 可实现 | ✅ 已实现 | 有 PageParam/Page；无 Wrapper |
| 10.11 | `selectByMap(Map)` | 按列值查询 | 🔶 可实现 | — | Go 用 map[string]any 或 struct tag |
| 10.12 | `deleteByMap(Map)` | 按列值删除 | 🔶 可实现 | — | 同上 |
| 10.13 | `update(Wrapper)` | 按条件更新 | 🔶 可实现 | — | 依赖 Wrapper |
| 10.14 | `insertOrUpdate(T)` | 存在则更新 | 🔶 可实现 | — | 需主键判断 + dialect UPSERT |
| 10.15 | `insertBatch(Collection)` | 批量插入 | 🔶 可实现 | — | 需 foreach 批量 SQL 或 COPY 协议 |
| 10.16 | `insertBatchSomeColumn(Collection)` | 指定列批量插入 | 🔶 可实现 | — | |

### 11. Wrapper 条件构造器 ⭐ 重点

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 11.1 | QueryWrapper 链式条件 | Lambda/字符串 API | ✅ 完全可行 | — | **Go Builder 模式天然适配** |
| 11.2 | UpdateWrapper 更新条件 | SET + WHERE | ✅ 完全可行 | — | 同上 |
| 11.3 | LambdaQueryWrapper 列引用 | Java 方法引用 | 🟡 需改造 | — | Go 无方法引用；用字段名 string 或 struct tag |
| 11.4 | `eq/ne/gt/ge/lt/le` | 比较条件 | ✅ 完全可行 | — | |
| 11.5 | `like/notLike/likeLeft/likeRight` | LIKE 条件 | ✅ 完全可行 | — | |
| 11.6 | `between/notBetween` | BETWEEN 条件 | ✅ 完全可行 | — | |
| 11.7 | `isNull/isNotNull` | NULL 条件 | ✅ 完全可行 | — | |
| 11.8 | `in/notIn` | IN 条件 | ✅ 完全可行 | — | |
| 11.9 | `groupBy/having` | 分组 | ✅ 完全可行 | — | |
| 11.10 | `orderBy/orderByAsc/orderByDesc` | 排序 | ✅ 完全可行 | — | |
| 11.12 | `or/and` | 逻辑连接 | ✅ 完全可行 | — | |
| 11.13 | `apply` | 自定义 SQL 片段 | ✅ 完全可行 | — | |
| 11.14 | `last` | 拼接到 SQL 末尾 | ✅ 完全可行 | — | LIMIT 等 |
| 11.15 | `exists/notExists` | EXISTS 子查询 | ✅ 完全可行 | — | |
| 11.16 | `nested` | 嵌套条件 | ✅ 完全可行 | — | `(a AND b) OR (c AND d)` |
| 11.17 | `select` 指定查询列 | SELECT 子句定制 | ✅ 完全可行 | — | |
| 11.18 | `set` 更新字段 | UPDATE SET 子句 | ✅ 完全可行 | — | UpdateWrapper |
| 11.19 | `setSql` 原生 SET | SQL 片段 | ✅ 完全可行 | — | |
| 11.20 | Wrapper → XML 互操作 | `ew.customSqlSegment` | 🔶 可实现 | — | Wrapper 生成的 SQL 片段注入到 XML 的 `${ew}` |

### 12. 注解映射（Go 用 struct tag 替代）

| # | Java 注解 | 功能 | Go 替代方案 | 可行性 | 状态 | 备注 |
|---|-----------|------|-------------|--------|------|------|
| 12.1 | `@TableName` | 表名映射 | `db:"table_name"` tag | ✅ 完全可行 | — | |
| 12.2 | `@TableId` | 主键标记 | `db:",pk"` tag | ✅ 完全可行 | — | |
| 12.3 | `@TableField` | 字段映射 | `db:"column_name"` tag | ✅ 完全可行 | — | |
| 12.4 | `@TableLogic` | 逻辑删除 | `db:",logic"` tag | ✅ 完全可行 | ✅ 部分实现 | 当前 XML 配置；可增加 tag 方式 |
| 12.5 | `@Version` | 乐观锁 | `db:",version"` tag | ✅ 完全可行 | — | |
| 12.6 | `@TableField(exist=false)` | 非数据库字段 | `db:"-"` tag | ✅ 完全可行 | — | |
| 12.7 | `@EnumValue` | 枚举值映射 | 自定义 TypeHandler | 🔶 可实现 | — | |
| 12.8 | `@TableField(fill=...)` | 自动填充 | `db:",fill:insert"` tag | 🔶 可实现 | — | 见自动填充章节 |
| 12.9 | `@OrderBy` | 默认排序 | `db:",orderby:asc"` tag | ✅ 完全可行 | — | |

### 13. ActiveRecord 模式

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 13.1 | Entity CRUD 方法 | 继承 Model | 🔶 可实现 | — | Go 用组合 + 代码生成；或嵌入 BaseMapper |
| 13.2 | `entity.insertOrUpdate()` | 自判断 | 🔶 可实现 | — | |
| 13.3 | `entity.selectById()` | 查询 | 🔶 可实现 | — | |

### 14. IService / Service 层

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 14.1 | IService 接口 | 批量操作/链式/闭包 | 🔶 可实现 | — | Go 用 struct 组合；非必需（Mapper 层已足够） |
| 14.2 | `saveBatch(Collection)` | 批量保存 | 🔶 可实现 | — | |
| 14.3 | `saveOrUpdateBatch` | 批量保存或更新 | 🔶 可实现 | — | |
| 14.4 | `removeBatchByIds` | 批量删除 | 🔶 可实现 | — | |
| 14.5 | `updateBatchById` | 批量更新 | 🔶 可实现 | — | |
| 14.5 | `lambdaQuery()` | 链式查询 | 🔶 可实现 | — | 见 Wrapper |
| 14.6 | `lambdaUpdate()` | 链式更新 | 🔶 可实现 | — | 见 Wrapper |

### 15. 分页插件

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 15.1 | 自动分页拦截 | 拦截 Executor 改写 SQL | ✅ 完全可行 | ✅ 已实现 | executePage + buildCountSQL |
| 15.2 | COUNT 优化 | 智能去 ORDER BY | ✅ 完全可行 | ✅ 已实现 | buildCountSQL 去排序 |
| 15.3 | 多方言分页 | 不同数据库 LIMIT 语法 | ✅ 完全可行 | ✅ 已实现 | dialector.ApplyPagination |
| 15.4 | Page 对象 | Total/Records/PageNum/PageSize | ✅ 完全可行 | ✅ 已实现 | |
| 15.5 | 游标分页 | Keyset pagination | 🔶 可实现 | — | WHERE id > lastId ORDER BY id LIMIT n |
| 15.6 | 无限滚动分页 | 前端无尽滚动 | 🔶 可实现 | — | 基于 cursor 分页 |

### 16. 逻辑删除

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 16.1 | `deleted` 字段 | 删除时间戳 | ✅ 完全可行 | ✅ 已实现 | |
| 16.2 | `del_flag` 字段 | 0/1 标记 | ✅ 完全可行 | ✅ 已实现 | |
| 16.3 | 自动追加 WHERE 条件 | 拦截器注入 | ✅ 完全可行 | ✅ 已实现 | MP 内置 CRUD 自动处理 |
| 16.4 | SELECT 结果过滤 | 自动排除已删除 | ✅ 完全可行 | ✅ 已实现 | |
| 16.5 | 全局逻辑删除配置 | application.yml | ✅ 完全可行 | ✅ 已实现 | |
| 16.6 | `@TableLogic` 注解 | 字段级标记 | ✅ 完全可行 | — | 可加 struct tag |

### 17. 自动填充

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 17.1 | MetaObjectHandler | insertFill/updateFill | ✅ 完全可行 | — | `orm.RegisterFillHandler(field, when, fn)` |
| 17.2 | `create_time` 插入填充 | 自动设置 | ✅ 完全可行 | — | |
| 17.3 | `update_time` 更新填充 | 自动设置 | ✅ 完全可行 | — | |
| 17.4 | `create_by` 插入填充 | 当前用户 | ✅ 完全可行 | — | 需 context 传递用户信息 |
| 17.5 | `update_by` 更新填充 | 当前用户 | ✅ 完全可行 | — | 同上 |
| 17.6 | 自定义填充字段 | 任意字段 | ✅ 完全可行 | — | |

### 18. 乐观锁

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 18.1 | `@Version` 版本字段 | UPDATE SET version=version+1 WHERE version=old | ✅ 完全可行 | — | 拦截器模式或 BaseMapper 自动处理 |
| 18.2 | 自动版本+1 | 拦截器注入 | ✅ 完全可行 | — | |
| 18.3 | CAS 失败检测 | 影响行数=0 判定 | ✅ 完全可行 | — | |

### 19. ID 生成策略

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 19.1 | AUTO 自增 | 数据库自增 | ✅ 完全可行 | ✅ 已实现 | useGeneratedKeys |
| 19.2 | NONE 无策略 | 手动赋值 | ✅ 完全可行 | ✅ 已实现 | |
| 19.3 | INPUT 手动输入 | 用户设置 | ✅ 完全可行 | ✅ 已实现 | |
| 19.4 | ASSIGN_ID 雪花算法 | Snowflake(1,1) | ✅ 完全可行 | — | Go 有成熟 Snowflake 库 |
| 19.5 | ASSIGN_UUID | UUID 无中划线 | ✅ 完全可行 | — | Go 标准 crypto/rand 或 google/uuid |
| 19.6 | 自定义 ID 生成 | 实现 IdentifierGenerator | ✅ 完全可行 | — | `orm.RegisterIdGenerator(fn)` |

### 20. 多租户

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 20.1 | 租户 ID 字段 | tenant_id 列 | ✅ 完全可行 | — | 拦截器自动追加 WHERE tenant_id=? |
| 20.2 | 租户条件注入 | 拦截器改写 SQL | ✅ 完全可行 | — | 复用 table_prefix 的 SQL 改写能力 |
| 20.3 | 忽略特定表 | 不追加租户条件 | ✅ 完全可行 | — | |
| 20.4 | 租户 ID 获取 | TenantLineHandler | ✅ 完全可行 | — | context 传递或闭包 |
| 20.5 | 动态租户切换 | 运行时切换 | ✅ 完全可行 | — | |

### 21. 动态表名

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 21.1 | 运行时表名替换 | 拦截器替换 | ✅ 完全可行 | ✅ 已实现 | table-prefix + prefix-map |
| 21.2 | 动态表名策略 | TableNameHandler | ✅ 完全可行 | ✅ 已实现 | SetTablePrefix/SetPrefixMap |
| 21.3 | 按请求动态路由 | Request 级别 | 🔶 可实现 | — | context 传递表名映射 |

### 22. SQL 注入防护 & 安全

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 22.1 | 攻击 SQL 阻断 | 拦截全表更新/删除 | ✅ 完全可行 | — | 检测无 WHERE 的 UPDATE/DELETE |
| 22.2 | SQL 注入过滤 | 拦截器检查 | ✅ 完全可行 | — | `#{} 已防护；${} 需额外检查` |
| 22.3 | 安全 SQL 白名单 | 只允许特定 SQL 模式 | 🟡 意义有限 | — | |

### 23. 性能分析

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 23.1 | SQL 执行耗时统计 | PerformanceInterceptor | ✅ 完全可行 | ✅ 已实现 | 语句统计 total/failed/avg/min/max |
| 23.2 | 慢 SQL 告警 | 超时打印 | ✅ 完全可行 | ✅ 已实现 | Warnf 输出 |
| 23.3 | SQL 格式化输出 | Pretty SQL | 🔶 可实现 | — | |
| 23.4 | 执行计划分析 | EXPLAIN 自动分析 | 🔶 可实现 | — | 慢查询自动 EXPLAIN |

### 24. 代码生成器

| # | 功能 | Java 实现 | Go 可行性 | 状态 | 备注 |
|---|------|-----------|-----------|------|------|
| 24.1 | 从数据库生成 Entity | AutoGenerator | ✅ 完全可行 | ✅ 已实现 | schema2code |
| 24.2 | 生成 Mapper XML | 模板生成 | ✅ 完全可行 | ✅ 已实现 | SaveMPToFile |
| 24.3 | 生成 Mapper 接口 | Java 接口 | ✅ 完全可行 | ✅ 已实现 | Go struct + function fields |
| 24.4 | 生成 Service 层 | Java 接口+实现 | 🔶 可实现 | — | Go 不强制 Service 层；可选择性生成 |
| 24.5 | 生成 Controller 层 | REST API | 🟡 意义有限 | — | 超出 ORM 范畴 |
| 24.6 | 自定义模板 | Freemarker/Velocity | 🔶 可实现 | — | Go 用 text/template |
| 24.7 | 策略配置 | 包名/表前缀/过滤 | ✅ 完全可行 | ✅ 已实现 | |

---

## 三、Go 原生增强（MyBatis/MP 无但 Go 生态需要）

### 25. Go 特有增强

| # | 功能 | 描述 | 优先级 | 备注 |
|---|------|------|--------|------|
| 25.1 | `context.Context` 全链路传递 | 超时/取消/值传递贯穿整个调用链 | 🆕 已实现 | Java 无对应概念 |
| 25.2 | Streaming 大结果集 | `QueryStream` 流式处理避免 OOM | 🆕 已实现 | `*RowStream` |
| 25.3 | 全局行数限制 | 防止无 LIMIT 的 SELECT 拉爆内存 | 🆕 已实现 | 默认 10000 行 |
| 25.4 | `go:embed` Mapper 嵌入 | 编译期打包 XML，单二进制部署 | 🆕 已实现 | Java 需 classpath |
| 25.5 | Struct Tag 元数据 | `db:"col,pk,logic,version,fill:insert"` | 🆕 推荐 | 替代 Java 注解 |
| 25.6 | Goroutine-safe 并发 | 全局缓存/连接池天然并发安全 | 🆕 已实现 | sync.RWMutex / sync.Once |
| 25.7 | `singleflight` 防击穿 | 缓存未命中时合并并发请求 | 🆕 推荐 | Java 需手写 |
| 25.8 | 错误聚合 | `ResultConvertReport` 收集所有字段错误 | 🆕 已实现 | Java 逐字段抛异常 |
| 25.9 | 零值安全 | nil 指针自动处理 | 🆕 已实现 | safeIndirectInterface |
| 25.10 | 编译期类型安全 | 泛型查询 `Query[T](sql)` | 🆕 推荐 | Go 1.18+ 泛型 |
| 25.11 | 环境变量注入 | `${ENV:default}` 配置 | 🆕 已实现 | 云原生友好 |
| 25.12 | Prepared Statement LRU 缓存 | 预编译语句复用 | 🆕 已实现 | Java 由连接池管理 |

---

## 四、优先级建议

### P0 — 核心 Gap（影响日常开发效率）

| 优先级 | 功能 | 工作量 | 影响 |
|--------|------|--------|------|
| P0-1 | **Wrapper 条件构造器** | 中 | 替代手写 WHERE，MP 最常用功能 |
| P0-2 | **Struct Tag 元数据**（TableName/PK/Logic/Version/Fill） | 小 | 消除 XML 中的重复映射声明 |
| P0-3 | **`<trim>` 动态 SQL** | 小 | 补全动态 SQL 缺失项（P0-3a：片段引擎已提取至 `types/sqlfragment` 独立包 + Node 接口/注册表，✅ 已完成；P0-3b：`<trim>` 节点待实现） |
| P0-4 | **自定义 TypeHandler** | 小 | JSON/Enum 等自定义类型映射 |
| P0-5 | **拦截器/Hook 链** | 中 | 乐观锁/多租户/自动填充的基础设施 |

### P1 — 重要增强（提升框架完整性）

| 优先级 | 功能 | 工作量 | 影响 |
|--------|------|--------|------|
| P1-1 | **乐观锁** | 小 | 并发场景必备 |
| P1-2 | **自动填充** (create_time/update_time) | 小 | 审计字段自动化 |
| P1-3 | **二级缓存** (namespace 级 LRU + 可扩展) | 中 | 高频只读查询优化 |
| P1-4 | **批量操作** (insertBatch/updateBatch) | 中 | 批量场景性能 |
| P1-5 | **`<discriminator>` 鉴别器** | 小 | 多态映射 |
| P1-6 | **ID 生成策略** (Snowflake/UUID) | 小 | 分布式 ID |
| P1-7 | **SQL 安全防护** (无 WHERE 阻断) | 小 | 防误操作 |

### P2 — 实用扩展（特定场景需要）

| 优先级 | 功能 | 工作量 | 影响 |
|--------|------|--------|------|
| P2-1 | **多租户** | 中 | SaaS 场景 |
| P2-2 | **`<bind>` 变量绑定** | 小 | 复杂条件计算 |
| P2-3 | **泛型查询 API** `Query[T]` | 中 | 类型安全查询 |
| P2-4 | **游标分页** | 小 | 大数据量分页 |
| P2-5 | **`databaseId` 多数据库 SQL 选择** | 小 | 跨库 SQL 差异 |
| P2-6 | **存储过程调用** | 小 | 遗留数据库 |
| P2-7 | **Service 层代码生成** | 中 | 可选 |
| P2-8 | **`insertOrUpdate`** | 中 | Upsert 语义 |

### P3 — 锦上添花

| 优先级 | 功能 | 工作量 | 影响 |
|--------|------|--------|------|
| P3-1 | **ActiveRecord 模式** | 中 | 个人偏好 |
| P3-2 | **读写分离路由** | 中 | 高可用场景 |
| P3-3 | **连接池监控** | 小 | 运维可观测 |
| P3-4 | **SQL 格式化输出** | 小 | 调试体验 |
| P3-5 | **执行计划自动分析** | 小 | 性能调优 |
| P3-6 | **自定义模板代码生成** | 中 | 灵活定制 |

---

## 五、不适合 Go 的 Java 特性（排除清单）

| 功能 | 排除原因 |
|------|----------|
| OGNL 表达式完整支持 | Go 无运行时表达式引擎；可用简单条件判断替代 |
| JDK 动态代理 / CGLIB 字节码增强 | Go 无运行时代码生成 |
| 延迟加载（Lazy Loading） | 需代理对象拦截属性访问；Go 无此能力 |
| 注解驱动（`@Select/@Insert/@Transactional`） | Go 无注解；用 struct tag + 代码生成 |
| 多结果集映射 | database/sql 不支持 |
| Spring 集成（`@MapperScan` 等） | Go 无 Spring |
| AOP 切面 | Go 无 AOP；用中间件/拦截器 |
| Serializable 序列化接口 | Go 不需要统一序列化接口 |
| 枚举类型（Java Enum） | Go enum 是 int constant；需 TypeHandler 映射 |
| 泛型擦除 | Go 泛型是真泛型，反而更灵活 |

---

## 六、Go 实现策略差异总结

| 维度 | Java MyBatis/MP | Go mybatis-go |
|------|----------------|---------------|
| 代理模式 | JDK 动态代理 / CGLIB | **reflect 函数字段注入**（编译期确定，运行时填充闭包） |
| 注解 | `@TableName/@TableId/@TableField` | **struct tag**（`db:"col,pk,logic,version"`） |
| 条件构造 | Lambda 方法引用 | **Builder 链式 API**（`NewQueryWrapper().Eq("name", "x")`） |
| 事务传播 | Spring AOP 7 种传播 | **context 传递 + 显式 Begin/Commit** |
| 缓存 | 装饰器模式 + 序列化 | **sync.Map / LRU + singleflight** |
| 懒加载 | CGLIB 代理拦截 | **不支持；需显式查询或嵌套 Join** |
| 代码生成 | Freemarker/Velocity 模板 | **Go text/template + gofmt** |
| 并发安全 | synchronized / ConcurrentHashMap | **sync.RWMutex / sync.Once / singleflight** |
| 类型安全 | 运行时反射 | **泛型 + 编译期检查**（Go 1.18+） |
| 部署 | WAR/JAR + classpath | **单二进制 + go:embed** |

---

> **总计：MyBatis 核心 45 项 + MyBatis-Plus 53 项 + Go 增强 12 项 = 110 项**
> **已实现：~40 项 | 可实现需适配：~50 项 | 不适用 Go：~10 项 | Go 原生增强：~10 项**
