# MyBatis-Plus 内置表（BaseMapper 内置 CRUD）使用说明

> 适用范围：RuoYi + MyBatis-Plus 项目。MP 内置操作（insert/deleteById/updateById/selectById/selectList…）在 Java 端无 XML，此前需手写 `GoExtraMapper`（TODO P16）；本框架支持从数据库表结构直接生成 MP 标准方法名的 XML，mybatis-go 原生加载、生成的 Go Mapper 直接可调用。

## 1. 功能定位与方法对照

XML 方法 ID = Java `BaseMapper<T>` 标准方法名；codegen 自动识别 `<foreach>` 把批量方法参数生成为切片签名。

| MP 内置方法（Java） | 生成的 XML id | 生成的 Go 方法 |
|---|---|---|
| `insert(T)` | `insert` | `Insert(model T) (int64, error)` |
| `deleteById(Serializable)` | `deleteById` | `DeleteById(id int64) (int64, error)` |
| `updateById(T)` | `updateById` | `UpdateById(model T) (int64, error)` |
| `selectById(Serializable)` | `selectById` | `SelectById(id int64) ([]T, error)` |
| `selectOne(Wrapper)` | `selectOne` | `SelectOne() ([]T, error)` |
| `selectList(Wrapper)` | `selectList` | `SelectList() ([]T, error)` |
| `selectPage(Page, Wrapper)` | `selectPage` | `SelectPage() ([]T, error)` |
| `selectCount(Wrapper)` | `selectCount` | `SelectCount() ([]int64, error)` |
| `selectBatchIds(Collection)` | `selectBatchIds` | `SelectBatchIds(ids []int64) ([]T, error)` |
| `deleteBatchIds(Collection)` | `deleteBatchIds` | `DeleteBatchIds(ids []int64) (int64, error)` |

> 批量方法签名：codegen 检测到函数体含 `<foreach>` 且参数为标量类型时，自动生成 `[]T`（如 `parameterType="Long"` → `[]int64`），运行时 `effectiveParamType` 按实际切片参数分派（S-05）。此能力对所有 Mapper 生效（samples `deleteConfigByIds` 亦生成 `[]int64`）。

## 2. 接入方式（三选一）

### 方式 A：命令行 schema2code（推荐）

```bash
go build -o schema2code cmd/schema2code/main.go
./schema2code -type postgres -host localhost -port 5432 -username xxx -password xxx \
  -db mydb -tables sys_user,sys_role -prefix sys_ -output ./out -mp
```

`-mp` 开关：生成 MP 风格 XML（方法名见 §1）；不带 `-mp` 则维持原 `deleteByPrimaryKey`/`selectAll` 风格。产物：
- `out/resources/mapper/SysUserMapper.xml`（MP 内置 CRUD）
- `out/src/mapper/SysUserMapper.go`（可调用 Mapper + `GetSysUserMapper()` 单例）
- `out/src/models/SysUserModel.go`（模型，来自 resultMap）

### 方式 B：代码 API

```go
orm.InitializeDatabase("postgres", "localhost", 5432, "user", "pwd", "mydb")
defer orm.Close()
orm.SchemaToCodeMP("./out", "sys_", "sys_user") // 等价于 SchemaToCode + -mp
```

### 方式 C：手写 XML（模板见 §3），直接放 `resources/mapper` 下即可被 `NewSqlMappers` 加载。

### 方式 D：内存自动生成（无需落盘 CRUD XML）

**如果 XML 里有 resultMap（含基本类型列 `<id>`/`<result>`）、但没有 MP 内置 CRUD**，框架在 **加载时（`NewSqlMappers`/`orm.Initialize`）于内存中按缺失 ID 补生成** 10 个内置方法，不落盘、不改动 XML。已有方法（手写或 XML 自带）不覆盖。

自动生成规则：
- **表名**：resultMap `type` 短名去 `Model` 后缀 + camelCase→snake_case（`SysUser` / `SysUserModel` → `sys_user`）；`type` 为 JDK 基础类型（map/int/string…）或 resultMap 无基本列/无 `<id>` 主键时不生成
- **主键**：`<id>` 项的 column 即为主键列
- **列类型**：`jdbcType` 缺失时主键默认 `BIGINT`（签名 int64）、普通列默认 `VARCHAR`
- **逻辑删除**：支持两种约定——`deleted`/`delete_time`（`deleted=true` + 过滤 `deleted = false`）与 RuoYi `del_flag`（`del_flag='2'` 软删 + 过滤 `del_flag = '0'`）
- **resultMap 引用**：生成的 select 直接引用原 XML 的 resultMap（含 `<association>`/`<collection>` 时模型类型随之生成）

示例：RuoYi `SysUserMapper.xml`（有 `SysUserResult`、无 CRUD）加载后自动具备 `SelectById` 等 10 个方法，`SelectById(1)` 生成 `select ... from sys_user where user_id=1 and del_flag = '0'`。

> 适用场景：RuoYi 等「XML 只有自定义查询、无内置 CRUD」的项目——无需 schema2code 预生成，也无需 GoExtraMapper。

## 3. 生成的 XML（真实产物，表 `sys_user`：id bigint PK / user_name / deleted / delete_time）

```xml
<mapper namespace="SysUserMapper">
  <resultMap id="BaseResultMap" type="SysUserModel">
    <result column="id" jdbcType="BIGINT" property="id"/>
    <result column="user_name" jdbcType="VARCHAR" property="userName"/>
  </resultMap>
  <sql id="base_column_list">id, user_name</sql>

  <insert id="insert" parameterType="SysUserModel">
    insert into sys_user (id, user_name) values (#{id,jdbcType=BIGINT}, #{userName,jdbcType=VARCHAR})
  </insert>
  <delete id="deleteById" parameterType="java.lang.Long">
    update sys_user set deleted=true,delete_time=now() where id=#{id,jdbcType=BIGINT}
  </delete>
  <update id="updateById" parameterType="SysUserModel">
    update sys_user set user_name=#{userName,jdbcType=VARCHAR} where id=#{id,jdbcType=BIGINT}
  </update>
  <select id="selectById" parameterType="java.lang.Long" resultMap="BaseResultMap">
    select <include refid="base_column_list"/> from sys_user where id=#{id,jdbcType=BIGINT} and deleted = false
  </select>
  <select id="selectOne" resultMap="BaseResultMap">
    select <include refid="base_column_list"/> from sys_user where deleted = false
  </select>
  <select id="selectList" resultMap="BaseResultMap">...同 selectOne...</select>
  <select id="selectPage" resultMap="BaseResultMap">...同 selectOne...</select>
  <select id="selectCount" resultType="long">
    select count(*) from sys_user where deleted = false
  </select>
  <select id="selectBatchIds" parameterType="java.lang.Long" resultMap="BaseResultMap">
    select <include refid="base_column_list"/> from sys_user where id in
    <foreach item="id" collection="collection" open="(" separator="," close=")">#{id}</foreach> and deleted = false
  </select>
  <delete id="deleteBatchIds" parameterType="java.lang.Long">
    update sys_user set deleted=true,delete_time=now() where id in
    <foreach item="id" collection="collection" open="(" separator="," close=")">#{id}</foreach>
  </delete>
</mapper>
```

生成 SQL 示例（`GenerateSQL` 实际输出）：
```
DeleteById(1)            → update sys_user set deleted=true,delete_time=now() where id=1
SelectById(1)            → select id, user_name from sys_user where id=1 and deleted = false
SelectList()             → select id, user_name from sys_user where deleted = false
SelectCount()            → select count(*) from sys_user where deleted = false
SelectBatchIds([1,2])    → select id, user_name from sys_user where id in ( 1, 2) and deleted = false
DeleteBatchIds([1,2])    → update sys_user set deleted=true,delete_time=now() where id in ( 1, 2)
```

## 4. 生成的 Go Mapper（真实 codegen 产物）

```go
type SysUserMapper struct {
	orm.BaseMapper
	Insert          func(models.SysUserModel) (int64, error)
	DeleteById      func(int64) (int64, error)
	UpdateById      func(models.SysUserModel) (int64, error)
	SelectById      func(int64) ([]models.SysUserModel, error)
	SelectOne       func() ([]models.SysUserModel, error)
	SelectList      func() ([]models.SysUserModel, error)
	SelectPage      func() ([]models.SysUserModel, error)
	SelectCount     func() ([]int64, error)
	SelectBatchIds  func([]int64) ([]models.SysUserModel, error)
	DeleteBatchIds  func([]int64) (int64, error)
}

func init() {
	orm.RegisterModel(new(models.SysUserModel))
	orm.RegisterMapper(new(SysUserMapper))
}
func GetSysUserMapper() *SysUserMapper { ... } // 单例
```

## 5. 运行时调用示例

```go
import "github.com/bnulwh/mybatis-go/orm"

orm.InitializeDatabase("kingbase", "host", 54321, "user", "pwd", "mydb")
defer orm.Close()

mp := orm.NewMapper("SysUserMapper").(SysUserMapper)

// 单条查询：select 统一返回切片，取 rs[0]（项目约定 P13）
rows, _ := mp.SelectById(1001)
user := rows[0]

// 列表 / 计数
list, _ := mp.SelectList()
count, _ := mp.SelectCount() // []int64，取 count[0]

// 新增 / 更新（自增主键回填见 §6.6）
n, _ := mp.Insert(models.SysUserModel{UserName: "admin"})
_, _ = mp.UpdateById(models.SysUserModel{Id: 1001, UserName: "admin2"})

// 删除（有 deleted 列 → 逻辑删除）
_, _ = mp.DeleteById(1001)
_, _ = mp.DeleteBatchIds([]int64{1, 2})

// 批量查询
rows2, _ := mp.SelectBatchIds([]int64{1, 2})
```

## 6. 关键语义与注意事项

1. **逻辑删除自动适配**：表含 `deleted`/`delete_time` 列 → `deleteById`/`deleteBatchIds` 变 `update set deleted=true,delete_time=now()`，所有 select 自动加 `and deleted = false`；无这些列 → 物理删除、不过滤。`deleted`/`delete_time` 列不进入 resultMap 和 insert/update 列清单。
2. **主键类型**：bigint/uint64 主键 → XML `parameterType="java.lang.Long"` → Go 签名 `int64`；int 主键 → `java.lang.Integer` → `int32`。
3. **批量方法签名**：codegen 检测 `<foreach>` 自动生成 `[]T`（`parameterType="Long"` → `[]int64`），直接传切片即可；框架运行时对任意切片均可分派（S-05）。
4. **`SelectCount` 返回 `[]int64`**（resultType="long"，沿用框架 select 统一返回切片约定，取 `rs[0]`）。
5. **`selectPage` 无 limit**：MP 端分页由 `IPage` 追加；本框架 `SelectPage()` 与 `SelectList()` 等价（全表），分页需业务层处理（TODO P4-2/P22）。
6. **自增主键回填**：`Insert` 走 `useGeneratedKeys` 需要 XML 加 `useGeneratedKeys="true" keyProperty="id"`（框架 S-11 已支持；PG/金仓 `LastInsertId` 不可用，见 TODO M-03）。
7. **`-mp` 只影响 schema2code 产出**：手写 XML 不受影响；`TableStructure.SaveToFile`（旧 CRUD 风格）行为不变。
8. **文件组织**：与普通 mapper 一样支持子目录（如 `mybatis/system/`），`mybatis-config.xml` 会被跳过（S-10）。

## 7. 限制与后续（已知边界）

- 仅生成「全表 / 按主键」的内置 CRUD；**Wrapper（`ew.customSqlSegment`）动态条件暂不支持**——条件查询仍用 RuoYi 传统 XML `<if>` 写法。
- 内存自动生成要求 XML 含可推导的 resultMap：type 为业务模型 + 至少一个 `<id>`/`<result>` 基本列 + 一个 `<id>` 主键；不满足则不生成（编译器/加载器不报错）。
- 若 Java 端存在**无 XML 的自定义方法**（如 `selectUserRoleGroup` 关联查询），仍需 GoExtraMapper 或手写 XML 补充。
- codegen 的 foreach→切片仅对**标量参数**生效；`map`/`struct`/`Slice` 参数不包切片（避免生成 `[]map` 等错误签名）。

## 8. 验证命令

```bash
go test -count=1 -run 'Test_TableStruct|Test_MPGeneratedSQL|Test_ContainsForEach|Test_GenerateDefine_ForEachSlice' ./types/
go test -count=1 -run 'Test_MPBuiltin' ./types/   # 内存自动生成回归（含 samples RuoYi 真实回归）
go run ./cmd/sqlitedemo   # 端到端冒烟
```
