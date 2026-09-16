# MyBatis / MyBatis-Plus 功能与 XML Mapper 完整参考

## 一、MyBatis XML Mapper 顶层元素

Mapper XML 文件的顶层元素（按定义顺序）：

| 元素 | 说明 |
|------|------|
| `cache` | 给定命名空间的缓存配置 |
| `cache-ref` | 引用其他命名空间的缓存配置 |
| `resultMap` | 最复杂最强大的元素，描述如何从结果集加载对象 |
| `parameterMap` | **已废弃**，旧式参数映射 |
| `sql` | 可被其他语句引用的可重用 SQL 片段 |
| `insert` | 映射 INSERT 语句 |
| `update` | 映射 UPDATE 语句 |
| `delete` | 映射 DELETE 语句 |
| `select` | 映射 SELECT 语句 |

---

## 二、select 元素

```xml
<select
  id="selectPerson"
  parameterType="int"
  resultType="hashmap"
  resultMap="personResultMap"
  flushCache="false"
  useCache="true"
  timeout="10"
  fetchSize="256"
  statementType="PREPARED"
  resultSetType="FORWARD_ONLY"
  databaseId="mysql"
  resultOrdered="false"
  resultSets="blogs,authors"
  affectData="false">
```

| 属性 | 描述 | 默认值 |
|------|------|--------|
| `id` | 命名空间内唯一标识符 | — |
| `parameterType` | 传入参数的全限定类名或别名 | unset |
| `parameterMap` | **已废弃** | — |
| `resultType` | 返回类型的全限定类名或别名（集合时为元素类型） | unset |
| `resultMap` | 外部 resultMap 的命名引用。与 resultType 二选一 | unset |
| `flushCache` | 调用时是否刷新本地和二级缓存 | `false` |
| `useCache` | 结果是否缓存到二级缓存 | `true` |
| `timeout` | 驱动等待数据库返回的最长秒数 | unset |
| `fetchSize` | 驱动每次批量返回的行数提示 | unset |
| `statementType` | `STATEMENT`/`PREPARED`/`CALLABLE` | `PREPARED` |
| `resultSetType` | `FORWARD_ONLY`/`SCROLL_SENSITIVE`/`SCROLL_INSENSITIVE`/`DEFAULT` | unset |
| `databaseId` | 匹配配置的 databaseIdProvider | unset |
| `resultOrdered` | 仅适用于嵌套结果 select：若为 true 假设嵌套结果分组在一起 | `false` |
| `resultSets` | 仅适用于多结果集，列出并命名返回的结果集 | unset |
| `affectData` | INSERT/UPDATE/DELETE 返回数据时设为 true（3.5.12+） | `false` |

---

## 三、insert / update / delete 元素

```xml
<insert
  id="insertAuthor"
  parameterType="domain.blog.Author"
  flushCache="true"
  statementType="PREPARED"
  keyProperty=""
  keyColumn=""
  useGeneratedKeys=""
  timeout="20">

<update id="updateAuthor" ...>
<delete id="deleteAuthor" ...>
```

| 属性 | 描述 | 默认值 |
|------|------|--------|
| `id` | 命名空间内唯一标识符 | — |
| `parameterType` | 传入参数类型 | unset |
| `flushCache` | 调用时是否刷新缓存 | `true` |
| `timeout` | 等待数据库返回最长秒数 | unset |
| `statementType` | `STATEMENT`/`PREPARED`/`CALLABLE` | `PREPARED` |
| `useGeneratedKeys` | （insert/update）是否使用 JDBC getGeneratedKeys 获取自增主键 | `false` |
| `keyProperty` | （insert/update）getGeneratedKeys 或 selectKey 结果写入的属性 | unset |
| `keyColumn` | （insert/update）表中生成键的列名 | unset |
| `databaseId` | 匹配 databaseIdProvider | unset |

### selectKey 子元素（主键生成）

```xml
<selectKey
  keyProperty="id"
  resultType="int"
  order="BEFORE"
  statementType="PREPARED">
  select CAST(RANDOM()*1000000 as INTEGER) a from SYSIBM.SYSDUMMY1
</selectKey>
```

| 属性 | 描述 |
|------|------|
| `keyProperty` | 目标属性 |
| `keyColumn` | 结果集中匹配的列名 |
| `resultType` | 结果类型 |
| `order` | `BEFORE`（先取键再 insert）/ `AFTER`（先 insert 再取键，适合 Oracle 内嵌序列） |
| `statementType` | `STATEMENT`/`PREPARED`/`CALLABLE` |

---

## 四、sql 片段

```xml
<sql id="userColumns"> ${alias}.id,${alias}.username,${alias}.password </sql>

<select id="selectUsers" resultType="map">
  select <include refid="userColumns"><property name="alias" value="t1"/></include>
  from some_table t1
</select>
```

- `<include>` 的 `refid` 和 `<property>` 值均可使用 `${var}` 变量

---

## 五、参数映射

```xml
#{property}
#{property,javaType=int,jdbcType=NUMERIC}
#{property,javaType=int,jdbcType=NUMERIC,typeHandler=MyTypeHandler}
#{property,javaType=double,jdbcType=NUMERIC,numericScale=2}
#{property,mode=OUT,jdbcType=CURSOR,javaType=ResultSet,resultMap=departmentResultMap}
```

| 写法 | 说明 |
|------|------|
| `#{...}` | PreparedStatement 参数（安全，预编译 `?`） |
| `${...}` | 字符串直接替换（有 SQL 注入风险，用于表名/列名等元数据） |
| `javaType` | Java 类型 |
| `jdbcType` | JDBC 类型（nullable 列必须指定） |
| `typeHandler` | 自定义类型处理器 |
| `numericScale` | 小数位数 |
| `mode` | `IN`/`OUT`/`INOUT` |

---

## 六、resultMap 结果映射

```xml
<resultMap id="detailedBlogResultMap" type="Blog">
  <constructor>
    <idArg column="blog_id" javaType="int"/>
    <arg column="blog_title" javaType="String"/>
  </constructor>
  <id property="id" column="blog_id"/>
  <result property="title" column="blog_title"/>
  <association property="author" javaType="Author">
    <id property="id" column="author_id"/>
    <result property="username" column="author_username"/>
  </association>
  <collection property="posts" ofType="Post">
    <id property="id" column="post_id"/>
    <result property="subject" column="post_subject"/>
    <collection property="comments" ofType="Comment">
      <id property="id" column="comment_id"/>
    </collection>
  </collection>
  <discriminator javaType="int" column="draft">
    <case value="1" resultType="DraftPost"/>
  </discriminator>
</resultMap>
```

### resultMap 子元素结构

| 元素 | 说明 |
|------|------|
| `constructor` | 构造器注入结果 |
| `idArg` | ID 参数（标记为 ID 提升缓存/嵌套映射性能） |
| `arg` | 普通构造器参数 |
| `id` | ID 结果映射 |
| `result` | 普通字段映射 |
| `association` | 一对一复杂类型关联 |
| `collection` | 一对多复杂类型集合 |
| `discriminator` | 根据结果值决定使用哪个 resultMap |
| `case` | discriminator 的分支 |

### association 的两种加载方式

1. **嵌套 Select**（有 N+1 问题）：
```xml
<association property="author" column="author_id" javaType="Author" select="selectAuthor"/>
```

2. **嵌套结果**（JOIN 查询，推荐）：
```xml
<association property="author" resultMap="authorResult" />
<!-- 或 -->
<association property="author" javaType="Author">
  <id property="id" column="author_id"/>
</association>
```

### association 属性

| 属性 | 描述 |
|------|------|
| `property` | 映射到的字段/属性 |
| `javaType` | Java 全限定类名或别名 |
| `jdbcType` | JDBC 类型 |
| `typeHandler` | 类型处理器 |
| `column` | 传递给嵌套 select 的列名 |
| `select` | 嵌套 select 语句 ID |
| `fetchType` | `lazy`/`eager`，覆盖全局 lazyLoadingEnabled |
| `resultMap` | 嵌套结果 resultMap ID |
| `columnPrefix` | 列名前缀（复用 resultMap） |
| `notNullColumn` | 非空列（任一非空才创建子对象） |
| `autoMapping` | 覆盖全局 autoMappingBehavior |

### collection 属性

与 association 类似，额外增加：
| 属性 | 描述 |
|------|------|
| `ofType` | 集合中元素的 Java 类型 |

### 多结果集关联（3.2.3+）

```xml
<select id="selectBlog" resultSets="blogs,authors" resultMap="blogResult" statementType="CALLABLE">
  {call getBlogsAndAuthors(#{id,jdbcType=INTEGER,mode=IN})}
</select>

<association property="author" javaType="Author" resultSet="authors" column="author_id" foreignColumn="id">
  <id property="id" column="id"/>
</association>
```

---

## 七、动态 SQL

| 元素 | 说明 | 示例 |
|------|------|------|
| `if` | 条件包含 | `<if test="title != null">AND title like #{title}</if>` |
| `choose`/`when`/`otherwise` | 多条件分支（类似 switch） | 见下 |
| `trim` | 自定义前缀/后缀裁剪 | `<trim prefix="WHERE" prefixOverrides="AND \|OR ">` |
| `where` | 智能插入 WHERE 并去除前导 AND/OR | `<where><if test="...">...</if></where>` |
| `set` | 动态 UPDATE SET，自动去除末尾逗号 | `<set><if test="...">username=#{username},</if></set>` |
| `foreach` | 遍历集合构建 IN 等 | `<foreach item="item" collection="list" open="(" separator="," close=")">#{item}</foreach>` |
| `bind` | 创建 OGNL 变量绑定到上下文 | `<bind name="pattern" value="'%' + _parameter.getTitle() + '%'" />` |
| `script` | 注解 Mapper 中使用动态 SQL | `@Update({"<script>...", "</script>"})` |

### if

```xml
<select id="findActiveBlogLike" resultType="Blog">
  SELECT * FROM BLOG WHERE state = 'ACTIVE'
  <if test="title != null">AND title like #{title}</if>
  <if test="author != null and author.name != null">AND author_name like #{author.name}</if>
</select>
```

### choose / when / otherwise

```xml
<select id="findActiveBlogLike" resultType="Blog">
  SELECT * FROM BLOG WHERE state = 'ACTIVE'
  <choose>
    <when test="title != null">AND title like #{title}</when>
    <when test="author != null and author.name != null">AND author_name like #{author.name}</when>
    <otherwise>AND featured = 1</otherwise>
  </choose>
</select>
```

### where

```xml
<select id="findActiveBlogLike" resultType="Blog">
  SELECT * FROM BLOG
  <where>
    <if test="state != null">state = #{state}</if>
    <if test="title != null">AND title like #{title}</if>
  </where>
</select>
```

### set

```xml
<update id="updateAuthorIfNecessary">
  update Author
    <set>
      <if test="username != null">username=#{username},</if>
      <if test="password != null">password=#{password},</if>
    </set>
  where id=#{id}
</update>
```

### foreach

```xml
<select id="selectPostIn" resultType="domain.blog.Post">
  SELECT * FROM POST P
  <where>
    <foreach item="item" index="index" collection="list"
        open="ID in (" separator="," close=")" nullable="true">
      #{item}
    </foreach>
  </where>
</select>
```

| 属性 | 描述 |
|------|------|
| `collection` | 集合参数名（List→list, 数组→array, 或 @Param 名） |
| `item` | 迭代变量名 |
| `index` | 索引变量名 |
| `open` | 开始字符串 |
| `close` | 结束字符串 |
| `separator` | 分隔符 |
| `nullable` | 是否允许 null（3.5.7+） |

### bind

```xml
<select id="selectBlogsLike" resultType="Blog">
  <bind name="pattern" value="'%' + _parameter.getTitle() + '%'" />
  SELECT * FROM BLOG WHERE title LIKE #{pattern}
</select>
```

### 多数据库厂商支持

```xml
<insert id="insert">
  <selectKey keyProperty="id" resultType="int" order="BEFORE">
    <if test="_databaseId == 'oracle'">select seq_users.nextval from dual</if>
    <if test="_databaseId == 'db2'">select nextval for seq_users from sysibm.sysdummy1</if>
  </selectKey>
  insert into users values (#{id}, #{name})
</insert>
```

---

## 八、缓存

```xml
<cache
  eviction="LRU"
  flushInterval="60000"
  size="512"
  readOnly="true"/>

<cache-ref namespace="com.someone.data.StudentMapper"/>
```

| 属性 | 描述 | 默认值 |
|------|------|--------|
| `eviction` | 回收策略：`LRU`/`FIFO`/`SOFT`/`WEAK` | `LRU` |
| `flushInterval` | 刷新间隔（毫秒） | 不刷新 |
| `size` | 缓存对象数 | 1024 |
| `readOnly` | 只读缓存 | `false` |

---

## 九、MyBatis-Plus 功能

### 9.1 BaseMapper 内置 CRUD 方法

| 类别 | 方法 | 生成的 SQL |
|------|------|-----------|
| **Insert** | `insert(T entity)` | `INSERT INTO table (fields) VALUES (?)` |
| **Delete** | `deleteById(Serializable id)` | `DELETE FROM table WHERE id = ?` |
| | `deleteByMap(Map<String,Object> columnMap)` | `DELETE FROM table WHERE key = ? AND ...` |
| | `delete(Wrapper<T> wrapper)` | `DELETE FROM table WHERE condition` |
| | `deleteBatchIds(Collection<? extends Serializable> idList)` | `DELETE FROM table WHERE id IN (?)` |
| **Update** | `updateById(T entity)` | `UPDATE table SET field=? WHERE id = ?` |
| | `update(T entity, Wrapper<T> whereWrapper)` | `UPDATE table SET field=? WHERE condition` |
| **Select** | `selectById(Serializable id)` | `SELECT * FROM table WHERE id = ?` |
| | `selectOne(Wrapper<T> queryWrapper)` | `SELECT * FROM table WHERE condition` |
| | `selectBatchIds(Collection<? extends Serializable> idList)` | `SELECT * FROM table WHERE id IN (?)` |
| | `selectList(Wrapper<T> queryWrapper)` | `SELECT * FROM table WHERE condition` |
| | `selectByMap(Map<String,Object> columnMap)` | `SELECT * FROM table WHERE key = ?` |
| | `selectMaps(Wrapper<T> queryWrapper)` | 返回 `List<Map<String,Object>>` |
| | `selectObjs(Wrapper<T> queryWrapper)` | 只返回第一个字段值 |
| | `selectCount(Wrapper<T> queryWrapper)` | `SELECT COUNT(*) FROM table WHERE condition` |
| | `selectPage(IPage<T> page, Wrapper<T> queryWrapper)` | 分页查询 |
| | `selectMapsPage(IPage<T> page, Wrapper<T> queryWrapper)` | 分页查询返回 Map |

### 9.2 IService / IRepository 方法

| 类别 | 方法 |
|------|------|
| **Save** | `save(T)`, `saveBatch(Collection<T>)`, `saveBatch(Collection<T>, int batchSize)` |
| **SaveOrUpdate** | `saveOrUpdate(T)`, `saveOrUpdateBatch(Collection<T>)`, `saveOrUpdateBatch(Collection<T>, int)` |
| **Remove** | `remove(Wrapper<T>)`, `removeById(Serializable)`, `removeByMap(Map)`, `removeByIds(Collection<?>)` |
| **Update** | `update(Wrapper<T>)`, `update(T, Wrapper<T>)`, `updateById(T)`, `updateBatchById(Collection<T>)`, `updateBatchById(Collection<T>, int)` |
| **Get** | `getById(Serializable)`, `getOne(Wrapper<T>)`, `getOne(Wrapper<T>, boolean)`, `getMap(Wrapper<T>)`, `getObj(Wrapper<T>, Function)` |
| **List** | `list()`, `list(Wrapper<T>)`, `listByIds(Collection<?>)`, `listByMap(Map)`, `listMaps()`, `listMaps(Wrapper<T>)`, `listObjs(...)` |
| **Page** | `page(IPage<T>)`, `page(IPage<T>, Wrapper<T>)`, `pageMaps(IPage<T>)`, `pageMaps(IPage<T>, Wrapper<T>)` |
| **Count** | `count()`, `count(Wrapper<T>)` |

### 9.3 条件构造器 Wrapper

| 类 | 说明 |
|---|------|
| `QueryWrapper` | 查询条件构造（字符串字段名） |
| `UpdateWrapper` | 更新条件构造（字符串字段名） |
| `LambdaQueryWrapper` | Lambda 表达式查询（类型安全） |
| `LambdaUpdateWrapper` | Lambda 表达式更新（类型安全） |

#### 条件方法一览

| 方法 | 生成的 SQL 片段 |
|------|----------------|
| `allEq(Map)` | 多字段 = 条件 |
| `eq(column, val)` | `column = val` |
| `eqOrIsNull(column, val)` | `column = val`（val 为 null 时 → IS NULL） |
| `ne(column, val)` | `column <> val` |
| `gt(column, val)` | `column > val` |
| `ge(column, val)` | `column >= val` |
| `lt(column, val)` | `column < val` |
| `le(column, val)` | `column <= val` |
| `between(column, v1, v2)` | `column BETWEEN v1 AND v2` |
| `notBetween(column, v1, v2)` | `column NOT BETWEEN v1 AND v2` |
| `like(column, val)` | `column LIKE '%val%'` |
| `notLike(column, val)` | `column NOT LIKE '%val%'` |
| `likeLeft(column, val)` | `column LIKE '%val'` |
| `likeRight(column, val)` | `column LIKE 'val%'` |
| `notLikeLeft(column, val)` | `column NOT LIKE '%val'` |
| `notLikeRight(column, val)` | `column NOT LIKE 'val%'` |
| `isNull(column)` | `column IS NULL` |
| `isNotNull(column)` | `column IS NOT NULL` |
| `in(column, Collection)` | `column IN (v1, v2, ...)` |
| `notIn(column, Collection)` | `column NOT IN (v1, v2, ...)` |
| `inSql(column, sqlValue)` | `column IN (sqlValue)`（支持子查询） |
| `notInSql(column, sqlValue)` | `column NOT IN (sqlValue)` |
| `eqSql(column, sqlValue)` | `column = (sqlValue)`（3.5.6+） |
| `gtSql(column, sqlValue)` | `column > (sqlValue)` |
| `geSql(column, sqlValue)` | `column >= (sqlValue)` |
| `ltSql(column, sqlValue)` | `column < (sqlValue)` |
| `leSql(column, sqlValue)` | `column <= (sqlValue)` |
| `groupBy(column...)` | `GROUP BY col1, col2` |
| `orderByAsc(column...)` | `ORDER BY col1 ASC, col2 ASC` |
| `orderByDesc(column...)` | `ORDER BY col1 DESC` |
| `orderBy(column, isAsc)` | 指定排序 |
| `having(sqlHaving)` | `HAVING condition` |
| `or()` | `OR` 连接 |
| `and(Consumer)` | `AND` 嵌套 |
| `nested(Consumer)` | 嵌套条件（不加 AND/OR） |
| `apply(sql, params...)` | 拼接原生 SQL |
| `last(sql)` | 追加到 SQL 末尾（有 SQL 注入风险） |
| `exists(sql)` | `EXISTS (sql)` |
| `notExists(sql)` | `NOT EXISTS (sql)` |
| `select(column...)` | 指定查询字段 |
| `set(column, val)` | 设置更新字段值（UpdateWrapper） |
| `setSql(sql)` | 设置更新 SQL 片段 |
| `setIncrBy(column, val)` | 字段自增 |
| `setDecrBy(column, val)` | 字段自减 |

### 9.4 分页插件

```java
@Bean
public MybatisPlusInterceptor mybatisPlusInterceptor() {
    MybatisPlusInterceptor interceptor = new MybatisPlusInterceptor();
    interceptor.addInnerInterceptor(new PaginationInnerInterceptor(DbType.MYSQL));
    return interceptor;
}
```

Page 类属性：

| 属性 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `records` | `List<T>` | emptyList | 查询数据列表 |
| `total` | `Long` | 0 | 总记录数 |
| `size` | `Long` | 10 | 每页条数 |
| `current` | `Long` | 1 | 当前页 |
| `orders` | `List<OrderItem>` | emptyList | 排序 |
| `optimizeCountSql` | `boolean` | true | 自动优化 COUNT SQL |
| `searchCount` | `boolean` | true | 是否执行 count 查询 |

支持 30+ 种数据库（含 PostgreSQL、MySQL、SQLite、Oracle、SQL Server、达梦、人大金仓 KingbaseES 等）。

### 9.5 逻辑删除

```yaml
mybatis-plus:
  global-config:
    db-config:
      logic-delete-field: deleted
      logic-delete-value: 1
      logic-not-delete-value: 0
```

或实体类注解：

```java
@TableLogic
private Integer deleted;
```

| 操作 | 生成 SQL |
|------|---------|
| 删除 | `UPDATE table SET deleted=1 WHERE id=? AND deleted=0` |
| 查询 | `SELECT ... FROM table WHERE deleted=0` |
| 更新 | 自动加 `WHERE deleted=0` |

支持 `Integer`、`Boolean`、`LocalDateTime`、`bigint` 等类型。

### 9.6 其他核心插件/功能

| 功能 | 说明 |
|------|------|
| **乐观锁插件** | `OptimisticLockerInnerInterceptor`，自动处理 version 字段 |
| **多租户插件** | `TenantLineInnerInterceptor`，自动加租户条件 |
| **数据权限插件** | `DataPermissionInterceptor` |
| **动态表名插件** | `DynamicTableNameInnerInterceptor` |
| **防全表更新删除插件** | `BlockAttackInnerInterceptor` |
| **非法 SQL 拦截** | `IllegalSQLInnerInterceptor` |
| **自动填充字段** | `@TableField(fill = FieldFill.INSERT/UPDATE)` + `MetaObjectHandler` |
| **主键生成策略** | `@TableId(type = IdType.AUTO/ASSIGN_ID/ASSIGN_UUID/...)` |
| **SQL 注入器** | 自定义扩展 Mapper 方法（如 `alwaysUpdateSomeColumnById`、`insertBatchSomeColumn`） |
| **链式查询** | `query().eq("name", "John").list()` / `lambdaQuery().eq(User::getName, "John").one()` |
| **ActiveRecord** | 实体继承 `Model<T>`，可直接 `entity.insert()`/`entity.deleteById()` |
| **SimpleQuery** | `SimpleQuery.list(wrapper, User::getName)` 简化查询 |
| **Db Kit** | `Db.getById(User.class, 1)` 静态工具 |
| **流式查询** | 大数据量逐行处理 |
| **批量操作** | `saveBatch`/`updateBatchById`/`removeByIds` |
| **自动映射枚举** | `@EnumValue` 注解 |
| **自动维护 DDL** | 自动建表/更新表结构 |
| **多数据源** | `dynamic-datasource-spring-boot-starter` |
| **数据安全保护** | 字段加密脱敏 |

### 9.7 Mapper 层选装件（Sql 注入器扩展）

| 方法 | 说明 |
|------|------|
| `alwaysUpdateSomeColumnById(T)` | 强制更新所有字段（忽略 null 判断） |
| `insertBatchSomeColumn(List<T>)` | 批量插入指定字段 |
| `logicDeleteByIdWithFill(T)` | 逻辑删除并自动填充字段（3.5.0 废弃，推荐 deleteById） |

---

## 十、MyBatis 注解方式

| 注解 | 说明 |
|------|------|
| `@Select` / `@Insert` / `@Update` / `@Delete` | 注解定义 SQL |
| `@Param` | 命名参数 |
| `@Results` / `@Result` | 结果映射 |
| `@One` / `@Many` | 一对一 / 一对多关联 |
| `@ResultMap` | 引用 XML resultMap |
| `@CacheNamespace` | 命名空间缓存 |
| `@Options` | 语句选项（useGeneratedKeys 等） |
| `@Lang` | 指定 LanguageDriver |

---

*基于 MyBatis 3.5.19 官方文档和 MyBatis-Plus 最新官方文档整理*
