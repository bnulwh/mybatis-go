# github.com/bnulwh/mybatis-go 使用问题与解决方案全记录

> 日期：2026-08-19
> 项目：中国化学分包安全监管数据平台（Go 后端改造适配 Java 接口）
> 场景：使用 mybatis-go 在 Go 中直接执行 Java 后端 Mapper XML（RuoYi + MDM，金仓 KingbaseES 数据库）
> 版本：**v0.1.7**（0.1.6 存在 parseTime 缺陷，见问题 P2）

---

## 一、背景与用法概述

### 1.1 为什么用它

Java 后端有 24 个 Mapper XML（166 个操作，含动态 SQL `<if>/<foreach>/<include>`、resultMap、数据权限 `${params.dataScope}`）。为让 Go 后端复用同一套 SQL 逻辑（避免双份维护），选择 mybatis-go 在 Go 中加载并执行这些 XML，数据源为金仓（KingbaseES，与 Java 后端同库）。

### 1.2 基本接入方式

```go
// 1. 配置（金仓：kingbase8 协议，内部用 lib/pq 驱动）
cm := map[string]string{
    "spring.datasource.url":      "jdbc:kingbase8://127.0.0.1:5432/gzwsk",
    "spring.datasource.username": "postgres",
    "spring.datasource.password": "root123",
    "mybatis.mapper-locations":   "mappers", // XML 目录（递归扫描）
}
orm.InitializeFromSettings(cm)

// 2. 注册 model（resultMap 实例化用）与 mapper
orm.RegisterModel(new(SysUser))
orm.RegisterMapper(new(SysUserMapper)) // struct{ orm.BaseMapper; 各操作 func 字段 }

// 3. 调用（func 字段名 = XML 操作 id，首字母大写）
mp := orm.NewMapper("SysUserMapper").(SysUserMapper)
rows, err := mp.SelectUserByUserName("admin") // -> []SysUser, error
```

### 1.3 Mapper 定义规则（关键）

```go
type SysUserMapper struct {
    orm.BaseMapper
    // 单参数：func(参数名 类型) ([]Model, error)
    SelectUserByUserName func(userName string) ([]SysUser, error)
    // 多参数：args tag 声明参数名（对应 Java @Param）
    UpdateLoginInfo func(userId int64, loginIp string, loginDate time.Time) (int64, error) `args:"userId,loginIp,loginDate"`
    // 自定义 model 参数：用 map（见 P6/P10）
    SelectUserList func(sysUser map[string]interface{}) ([]SysUser, error)
}
```

---

## 二、问题清单（按类别）

### A. 数据源 / 驱动类

---

#### P1. 连接初始化失败：`parse datbase type failed` / `get database username failed`

- **现象**：`orm.InitializeFromSettings` panic。
- **原因**：mybatis-go 通过解析 `spring.datasource.url` 的 JDBC 前缀判断库类型，并从配置 map 中读取 username/password。url 格式不符或缺少 username/password key 都会失败。
- **解决**：
  ```go
  cm := map[string]string{
      "spring.datasource.url":      "jdbc:kingbase8://127.0.0.1:5432/gzwsk",
      "spring.datasource.username": "postgres",
      "spring.datasource.password": "root123",
      "mybatis.mapper-locations":   "mappers",
  }
  ```
  注意 `jdbc:postgresql://` 前缀同样可用（kingbase/postgres 都走 lib/pq）。
- **要点**：url 中**不支持带 query 参数**（如 `?parseTime=true`），解析正则 `([\w._-]+)` 不含 `?`；需要自定义 DSN 时用 `Config.DSN` 字段（0.1.7 新增，非空时优先于 GenerateDSN）。

---

#### P2. 0.1.6 的 DATETIME 列全部丢行：`unsupported Scan, storing driver.Value type []uint8 into type *time.Time`（升级到 0.1.7 修复）

- **现象**：mapper 查询返回 0 行，日志大量 `scan error ... type []uint8 into type *time.Time`；但用 `database/sql` 直接执行同一 SQL 有数据。
- **原因**（0.1.6）：`Config.GenerateDSN()` 生成的 MySQL DSN 是 `user:pass@tcp(host)/db`，**缺 `parseTime=true`**。go-sql-driver 默认把 DATETIME 返回 `[]byte`，而 mysql 驱动的 `ColumnType.ScanType()` 声明为 `*time.Time`，Scan 时类型不匹配被跳过 → 行全丢。
- **解决**：**升级到 v0.1.7**。0.1.7 的 `generateConn()` 对 MySQL 自动追加 `?parseTime=true&loc=Local`：
  ```go
  // 0.1.7 database_config.go
  return fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&loc=Local", ...)
  ```
- **要点**：升级前必须用 `orm.Query` 做原生对比实验定位（`orm.Query("SELECT ...")` 与 mapper 方法走同一 `queryRows`，区别只在 SQL/参数，可快速区分是连接问题还是 mapper 问题）。
- **排查思路**：先 `orm.Query("SELECT DATABASE()")` 确认连接；再用 `orm.Query` 执行与 mapper 完全相同的 SQL；都不行再看 scan error 日志。

---

#### P3. 金仓（KingbaseES）连接方式

- **现象**：无金仓官方 Go 驱动。
- **原因/解决**：金仓与 PostgreSQL 使用相同 wire 协议。mybatis-go 0.1.7 的 `kingbase_dialector.go` 用 **lib/pq 驱动**并以 `kingbase/kingbase8/kingbase7/...` 名称注册：
  ```go
  sql.Register(name, &pq.Driver{})
  ```
  因此 `jdbc:kingbase8://host:port/db` 可直接使用，无需额外驱动依赖。
- **本地验证**：无金仓环境时用 Docker PostgreSQL 模拟（`postgres:16-alpine`），连接串改 `jdbc:kingbase8://127.0.0.1:5432/gzwsk` 即可，SQL 兼容。

---

#### P4. PG/金仓不支持 `LastInsertId()`：自增主键不回填

- **现象**：`useGeneratedKeys="true" keyProperty="userId"` 的 insert 执行后，入参 map 里 `userId` 仍为 0，导致后续关联表（如 sys_user_role）插入 `user_id=0`，且触发唯一约束冲突。
- **原因**：`backfillGeneratedKey()` 依赖 `sql.Result.LastInsertId()`，**lib/pq 不支持**（返回 error）→ 回填被跳过。
- **解决**：插入后从序列回读主键：
  ```go
  // 金仓/PG 专有
  rows, _ := mybatis.RawQuery("SELECT currval('seq_sys_user') AS id")
  userId := rows[0]["id"].(int64)
  ```
  封装为 `lastInsertID(seq string) int64` 工具（见 backend-go/domain/system/common.go）。
- **要点**：`currval` 要求**当前会话内已 nextval**（即刚执行过插入）；插入失败时返回 error，工具函数返回 0。各表序列名：`seq_sys_user / seq_sys_role / seq_sys_dept / seq_sys_post / seq_sys_dict_type / seq_sys_dict_data / seq_sys_config / seq_sys_notice / seq_sys_menu / seq_sys_job / seq_sys_job_log / seq_sys_logininfor / seq_sys_oper_log / seq_sys_notice_read / seq_gen_table / seq_gen_table_column`。
- **注意**：`backfillGeneratedKey` 本身支持 map 参数回填（SetMapIndex 大写+原 key），**仅当 LastInsertId 可用时**（MySQL/SQLite）才生效。

---

### B. 参数绑定类

---

#### P5. 多参数必须用 `args` tag（对应 Java @Param）

- **现象**：Java 接口 `updateUserStatus(@Param("userId") Long userId, @Param("status") String status)` 在 Go 端直接 `func(userId int64, status string)` 报错 `not need func args` 或参数值错乱。
- **原因**：mybatis-go 通过反射拿不到 Go 参数名；无 tag 时按位置取参数，无法与 XML `#{userId}` 对应。
- **解决**：func 字段加 tag：
  ```go
  UpdateUserStatus func(userId int64, status string) (int64, error) `args:"userId,status"`
  ```
  `buildArgs()` 会把前 N 个参数打包成 `map{userId:..., status:...}`（**key 会被小写化**），XML 的 `#{userId}` 经 `buildKey`（小写）匹配。
- **要点**：tag 长度必须等于 func 参数个数，否则 `makeParamType` panic（`the tag "args" length != args length`）。

---

#### P6. XML 无 `parameterType` 但方法有参数：`not need func args`

- **现象**：注册 mapper 时 panic `check sql function selectDeptListByRoleId failed, not need func args`。
- **原因**：mybatis-go 的 `parseSqlParamFromXmlAttrs` **只看 XML 的 `parameterType` 属性**判断 `SqlParam.Need`。Java 端多参数/注解参数在 XML 中不写 parameterType（由 Java 接口 @Param 提供），mybatis-go 看不到 → `Need=false` → 校验失败。
- **解决**：**给 XML 副本补 parameterType**（生成器自动做）：
  - 多参数（@Param 2+）→ `parameterType="java.util.Map"`
  - 单参数 List/数组 → `parameterType="java.util.List"`
  - 单参数基础类型 → 原类型（`Long`/`String`）
  - 反向：XML 有 parameterType 但 Java 方法无参数（如 `selectGenTableAll`）→ **删除** parameterType，否则报 `need func args`。
- **要点**：补的参数类型只影响 `Need` 与类型检查；实际渲染类型由**运行时入参**决定（`effectiveParamType`）。

---

#### P7. `List<SysRoleDept>` 作为 parameterType 值破坏 XML

- **现象**：`parameterType="List<SysRoleDept>"` 补进 XML 后，整个 XML 解析失败 `there is tag no close`。
- **原因**：XML 属性值里的 `<` `>` 未转义，把标签截断。
- **解决**：统一规范化为 `parameterType="java.util.List"`（mybatis-go 的 `GetShortName("java.util.List")="List"` → `SliceSqlParam`）。

---

#### P8. 自定义 model 参数传 struct 的两个坑 → 统一用 `map[string]interface{}`

- **坑 1（nil 指针 panic）**：`convert2Map` 对 struct 字段 `reflect.Indirect(fval).Interface()`，**nil 指针字段（如 `*time.Time`）会 panic**（0.1.6 与 0.1.7 均未修复）。
- **坑 2（`!= 0` 语义）**：见 P10。
- **解决**：
  1. **model 字段全部用值类型**（`time.Time` 而非 `*time.Time`，`int64` 而非 `*int64`），规避 panic；
  2. **查询/写操作的 model 参数一律改 `map[string]interface{}`**（生成器 `_is_model_type` 判断自定义类型 → map），业务层用 `QueryMap(model)` 反射转 map 且**排除零值**。

---

#### P9. `Set<String>` / `Long[]` / 泛型类型映射

- **现象**：生成 Go struct 时 `Set<String>` 原样输出 → Go 语法错误；`Long[]` 误映射。
- **解决**（生成器 j2g）：
  - `Set<X>` / `List<X>` → `[]X`
  - `X[]` → `[]X`
  - `Map<String,Object>` → `map[string]interface{}`
  - `Long/Integer/String/Date/Boolean`（含短名）→ 基础类型
  - 自定义类型 → 同名 Go struct（须已注册 model）

---

### C. 结果映射类

---

#### P10. `<if test="userId != null and userId != 0">` 恒渲染：不支持 `!= 0`

- **现象**：`selectUserList` 传 `SysUser{UserId:0}`（未指定用户）时，SQL 仍生成 `AND u.user_id = 0`，查询返回 0 行。
- **原因**：`parseIfConditionsFromText` 只识别三种条件：
  ```go
  reNC   := regexp.MustCompile(`[\w.]+[\s]*[!][=][\s]*null`) // X != null
  reEC   := regexp.MustCompile(`[\w.]+[\s]*[!][=][\s]*[']{2}`) // X != ''
  reBool := regexp.MustCompile(`^[\w.]+$`) // 裸布尔
  ```
  `userId != 0` 不匹配任何 → **被静默丢弃**，条件列表只剩 `userId != null`，而 `validValue(int64(0))` 返回 true → 恒渲染。
- **解决**：**查询参数用 map 且零值不入 map**。`QueryMap()` 反射 struct，跳过 `0/""/false/零时间/空map/空切片`：
  ```go
  func QueryMap(v interface{}) map[string]interface{} { ... } // mybatis/querymap.go
  ```
  这样 map 无 `userId` key → `lookupParam` 失败 → null 检查 false → 不渲染，语义与 Java `!= null and != 0` 一致。
- **要点**：这是 mybatis-go 相对 MyBatis（OGNL）最大的表达式能力缺口，**任何带 `!= 0` / `> 0` 比较的 `<if>` 都必须靠"零值不入参"来配合**。`> 0`、`== 0` 等数值比较同样不支持（会被丢弃或恒 true）。

---

#### P11. 嵌套 `association` / `collection` 不映射：联表列丢失

- **现象**：`selectUserByUserName` 联表返回 `d.dept_name`、`d.leader`、`r.role_name`，但 resultMap 里它们是 `<association property="dept" resultMap="deptResult">` 的嵌套列，Go struct 中拿不到。
- **原因**：mybatis-go 的 `makeColumnMap` 明确注释：
  ```go
  continue // association/collection 无 column 映射，不参与行列转换
  ```
  且 `setColumnValuesPrepared` 只按 `ColumnMap`（column→property 平铺映射）赋值，嵌套项被忽略。
- **解决**：**生成器自动补平铺映射**——分析每个 select（含 include 展开）的列与 resultMap 已有列的差集，向 XML 副本的 resultMap 追加 `<result property="deptName" column="dept_name"/>` 等平铺项，Go model 同步补字段（如 `SysUser.DeptName/Leader/RoleName`）。

---

#### P12. 自定义类型 `resultType="SysNotice"` 返回 `map[string]interface{}`

- **现象**：注册时报 `return type valid failed 'map[string]interface{}' != 'mybatis.SysNotice'`。
- **原因**：`parseResultTypeFrom` 只识别 JDBC 基础类型（STRING/LONG/INTEGER/TIMESTAMP/BOOLEAN/DOUBLE），未知类型返回 `map[string]interface{}`。Java 端 resultType 写短类名（如 `SysNotice`/`MdmOrgRaw`）能解析，mybatis-go 不行。
- **解决**：生成器对这类操作生成 `([]map[string]interface{}, error)`，业务层自行转换（受影响操作仅 4 个：`selectNoticeListWithReadStatus`、`selectByOrgCode`、`selectByStaffCode`、`selectById`）。

---

#### P13. select 返回值必须是切片：单对象也返回 `[]T`

- **现象**：Java 返回单对象 `SysMenu selectMenuById(...)`，Go 端写成 `func(...) (SysMenu, error)`。
- **原因**：`convert2Results` 永远构造 `[]itemTyp` 切片；`checkResults` 的严格类型校验被注释（直接 return nil），运行时 MakeFunc 返回值类型不匹配会 panic。
- **解决**：**所有 select 统一 `([]T, error)`**，单对象场景业务层取 `rs[0]`。
- **返回类型对应表**：
  | XML resultType/resultMap | Go 返回 |
  |---|---|
  | resultMap="XxxResult" | `([]Xxx, error)` |
  | java.lang.Long / Integer / int | `([]int64, error)` |
  | java.lang.String | `([]string, error)` |
  | java.util.Map | `([]map[string]interface{}, error)` |
  | 自定义短类型 | `([]map[string]interface{}, error)`（见 P12） |
  | insert/update/delete | `(int64, error)`（RowsAffected） |

---

#### P14. `sql: Scan error` 导致整行被丢弃（静默）

- **现象**：个别列 Scan 失败时 `convertMap2Result` 返回 err → 该行被 `continue` 跳过，且错误仅在 debug 日志可见。
- **原因**：`convert2Results` 对每行 `if err != nil { continue }`。
- **排查**：遇到"0 行但 SQL 能查到"时，必须看 **scan error 日志**（如 P2 的 []uint8→*time.Time、类型长度溢出等）。
- **解决**：保证 model 字段类型与列类型匹配（金仓/PG 数值列对应 int64、时间列 time.Time、布尔列 bool）。

---

### D. XML 加载 / 解析类

---

#### P15. 加载目录会递归扫描子目录；mybatis-config.xml 无 namespace 需排除

- **现象**：`mybatis.mapper-locations` 指向目录时 `filterMapperFiles` 用 `filepath.Walk` **递归**收集全部 `.xml`。
- **要点**：XML 可以按子目录组织（`mybatis/system/`、`mybatis/monitor/`、`mapper/mdm/`）。但 `mybatis-config.xml`（根标签 configuration、无 namespace）会被解析成空 mapper，需在业务层跳过 `namespace` 不含 "Mapper" 的文件。
- **配套**：Go 部署时用 `go:embed` 内嵌 XML 并在运行时解出到临时目录（mybatis-go 只支持文件路径加载，不支持 embed.FS 直接读取）。

---

#### P16. `MyBatis-Plus 内置操作`（insert/update 主键/关联）没有 XML

- **现象**：Java 的 `SysUserServiceImpl` 继承 MyBatis-Plus `ServiceImpl`，`insertUser/updateUser/selectUserById` 走 MP 内置（无 XML）；RuoYi 关联查询 `selectUserRoleGroup` 等也不在 XML。
- **解决**：Go 端补充 `GoExtraMapper`（`mybatis/mappers/extra/GoExtraMapper.xml` + `mybatis/extra_mapper.go`），手写这些 SQL（insertUser/updateUser/selectUserById/selectUserRoleGroup/selectUserPostGroup/selectRolesByUserName）。Java 端不加载此文件（namespace 不同），互不影响。

---

### E. 动态 SQL / 表达式类

---

#### P17. `${params.dataScope}` 未替换直接残留 → SQL 语法错误

- **现象**：`selectUserList/selectRoleList/selectDeptList` 报 `pq: syntax error at or near "$"`。
- **原因**：`${params.dataScope}` 需要入参 map 里存在 `params.dataScope`；缺 `params` key 时 raw 参数不替换（残留原样）。
- **解决**：**所有查询入参 map 必须带默认 `"params": map[string]interface{}{"dataScope": ""}`**（RuoYi 数据权限由 Java AOP 注入，Go 端由业务层提供；当前默认空 = 全量权限）。
- **要点**：`${...}` 是**字符串原样替换**（`rawFormatValue`），注意 SQL 注入面；仅用于数据权限等受控片段。

---

#### P18. `<foreach>` 批量参数：`collection="list"` 对应切片参数

- **现象**：`batchUserRole(List<SysUserRole>)`、`deleteUserByIds(Long[])` 等批量操作。
- **解决**：
  - 切片参数（`[]SysUserRole` / `[]int64`）→ `effectiveParamType` 走 `SliceSqlParam` → `generateSqlWithSlice`，`<foreach collection="list">` 或 `collection="array"` 均可。
  - 生成器签名：`BatchUserRole func(userRoleList []SysUserRole) (int64, error)`、`DeleteUserByIds func(userIds []int64) (int64, error)`。

---

#### P19. 表达式引擎不支持 `null` 字面量（govaluate）

- **现象**：直接测 `govaluate.Evaluate("userId != null")` 报 `Cannot transition token types from VARIABLE [null]`。
- **说明**：mybatis-go **不直接使用 govaluate 求值 if**，而是自己把条件拆分为 null/empty/bool 三类（见 P10）预处理。业务层无需关心，但**不要期望支持任意 OGNL 表达式**（如三元、方法调用、数值比较都不行）。

---

### F. 方言兼容类（金仓/PG vs MySQL）

---

#### P20. MySQL 方言函数/语法在金仓（PG）不兼容

| 方言 | 问题 | 生成器修正（仅 Go XML 副本） |
|---|---|---|
| `` `query` `` 反引号 | PG 语法错误 `syntax error at or near "`"` | 移除反引号（`query` 非 PG 保留字） |
| `ifnull(perms,'')` | PG 无此函数 | → `coalesce(perms,'')` |
| `WHERE find_in_set(x, y)` truthy | PG 要求布尔（find_in_set 返回 integer） | → `find_in_set(x, y) > 0` |
| `status = 0`（varchar=integer） | PG 严格类型 `operator does not exist: character = integer` | → `status = '0'` |

- **要点**：金仓本身有 MySQL 兼容模式（Java 端原样可用），但本地 PG 模拟与跨库通用性要求 Go 副本采用 PG/金仓通用语法；**Java 端原始 XML 不动**。

---

#### P21. `find_in_set` 参数类型重载

- **现象**：PG 报 `function find_in_set(integer, character varying) does not exist`。
- **原因**：金仓脚本定义的 `find_in_set(TEXT, TEXT)` 不匹配调用 `find_in_set(0, ancestors)`（int, varchar）。
- **解决**：本地 PG 补 int 重载函数（金仓原生兼容，无需处理）：
  ```sql
  CREATE OR REPLACE FUNCTION find_in_set(search_text INTEGER, text_list TEXT)
  RETURNS INTEGER AS $$ ... $$ LANGUAGE plpgsql;
  ```

---

### G. 其他

---

#### P22. 无分页支持（PageHelper 等价物缺失）

- **现象**：RuoYi 用 PageHelper 自动给 SQL 加 limit；mybatis-go 无此能力，XML 的 selectList 无 limit。
- **解决**：**内存分页**（数据量小可接受）：
  ```go
  func paginate[T any](rows []T, p pageParams) ([]T, int64) // 切片 + total
  ```
  前端契约 `pageNum/pageSize` → `(pageNum-1)*pageSize` 切片。数据量大时需改为 SQL 层分页（改写 XML 副本）。

---

#### P23. `ColumnType.ScanType()` 为空时的回退

- **现象**：部分驱动首行前 ScanType 为空。
- **解决**：mybatis-go 已内置 `scanTargetType` 回退 `sql.NullString`；无需处理。

---

#### P24. 日志与调试

- **要点**：
  - `orm.SetLogger(log.StandardLogger())` 接入 logrus，`log.IsDebugEnabled()` 时输出完整 SQL 与结果 JSON（`results: [...]`）。
  - 定位问题三步：① `orm.Query("SELECT DATABASE()")` 确认连接 → ② 用 `orm.Query` 执行与 mapper 相同 SQL 确认 SQL 本身 → ③ 看 scan error 日志（P14）。
  - 后端启用 debug：`orm.SetLogger` + 日志级别 DEBUG（项目用 `mybatis.RawQuery` 暴露原生查询便于对比）。

---

## 三、最佳实践总结（项目落地）

1. **版本**：必须 ≥ v0.1.7（parseTime 修复）。
2. **model 值类型化**：时间字段用 `time.Time`，ID 用 `int64`，杜绝 `*T`（P8）。
3. **查询参数统一 map**：自定义 model 参数生成器转 `map[string]interface{}`；业务层 `QueryMap()` 排除零值（P8/P10）。
4. **XML 副本修正由生成器维护**：跑 `python scripts/gen_mybatis_go.py` 重新生成，**不要手改** `models_gen.go / mappers_gen.go / mappers/*.xml`（Java 原始 XML 在 `backend-java/mybatis-xml/`）。
5. **主键回读**：金仓/PG 插入后 `lastInsertID('seq_xxx')`（P4）。
6. **params.dataScope 默认空**：所有 RuoYi 数据权限查询入参带 `params:{dataScope:""}`（P17）。
7. **返回切片**：select 一律 `([]T, error)`（P13）。
8. **补充 MP 内置操作**：Java 走 MyBatis-Plus 的用 `GoExtraMapper`（P16）。
9. **方言适配**：反引号/ifnull/find_in_set/status 比较统一 PG/金仓通用语法（P20/P21）。

---

## 四、附录：mybatis-go 0.1.7 公开 API 速查

| API | 用途 |
|---|---|
| `orm.InitializeFromSettings(map[string]string)` / `Initialize(filename)` / `InitializeDatabase(dbType, host, port, user, pwd, dbName)` | 初始化 |
| `orm.RegisterModel(ptr)` | 注册 model（resultMap 实例化） |
| `orm.RegisterMapper(ptr)` | 注册 mapper（校验 func 签名与 XML） |
| `orm.NewMapper(name)` / `NewMapperPtr(name)` | 获取 mapper 实例 |
| `orm.Query(sql, args...)` / `Execute(sql, args...)` | 原生查询/执行 |
| `orm.Begin()` / `BeginTx(ctx, opts)` | 事务 |
| `orm.SetLogger(logger)` | 接入日志 |
| `orm.Close()` | 关闭连接池 |
| `Config.DSN`（0.1.7 新增） | 自定义连接串（优先于 GenerateDSN） |
| mapper func `args:"a,b"` tag | 多参数命名（对应 Java @Param） |

## 五、相关文件索引（本项目）

| 文件 | 内容 |
|---|---|
| `backend-go/scripts/gen_mybatis_go.py` | 生成器（含全部适配逻辑：补列/补 parameterType/方言/类型映射） |
| `backend-go/mybatis/init.go` | 金仓连接 + embed 解出 XML + 注册 |
| `backend-go/mybatis/querymap.go` | QueryMap（零值排除） |
| `backend-go/mybatis/extra_mapper.go` | GoExtraMapper（MP 内置操作补充） |
| `backend-go/domain/system/common.go` | lastInsertID / paginate / idFromBody 等工具 |
| `TODO.md` | 项目改造进度总览 |
