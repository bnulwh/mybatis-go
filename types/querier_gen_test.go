package types

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeQuerierTestMapper 将测试 Mapper XML 写入临时目录，返回目录路径。
func writeQuerierTestMapper(t *testing.T, xml string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "TestQuerierMapper.xml"), []byte(xml), 0640); err != nil {
		t.Error("write test mapper failed:", err)
	}
	return dir
}

const testQuerierMapperXML = `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="TqUserMapper">
  <resultMap id="BaseResultMap" type="TqUser">
    <id column="id" jdbcType="BIGINT" property="id" />
    <result column="user_name" jdbcType="VARCHAR" property="userName" />
    <result column="age" jdbcType="INTEGER" property="age" />
  </resultMap>
  <sql id="base_columns">id, user_name, age</sql>
  <select id="selectAll" resultMap="BaseResultMap">
    select <include refid="base_columns"/>
    from tq_user
    order by id
  </select>
  <select id="selectById" parameterType="Long" resultMap="BaseResultMap">
    select <include refid="base_columns"/>
    from tq_user
    where id = #{id,jdbcType=BIGINT}
  </select>
  <select id="selectByAgeRange">
    select id from tq_user
    where age &gt;= #{minAge,jdbcType=INTEGER} and age &lt;= #{maxAge,jdbcType=INTEGER}
  </select>
  <select id="countAll" resultType="int">
    select count(*) from tq_user
  </select>
  <select id="selectDynamic" parameterType="TqUser" resultMap="BaseResultMap">
    select <include refid="base_columns"/>
    from tq_user
    <where>
      <if test="userName != null and userName != ''">
        and user_name like concat('%', #{userName}, '%')
      </if>
    </where>
  </select>
  <select id="selectOrderBy" parameterType="String" resultMap="BaseResultMap">
    select id from tq_user order by ${orderCol}
  </select>
</mapper>
`

// Test_GenerateQuerierFile_Static 静态 select 生成接口 + 实现 + 模型；动态语句按原因跳过。
func Test_GenerateQuerierFile_Static(t *testing.T) {
	dir := writeQuerierTestMapper(t, testQuerierMapperXML)
	mps := NewSqlMappers(dir)
	if len(mps.Mappers) != 1 {
		t.Error("want 1 mapper, got", len(mps.Mappers))
		return
	}
	out := t.TempDir()
	stats, err := mps.Mappers[0].GenerateQuerierFile(out, "querier", nil)
	if err != nil {
		t.Error("GenerateQuerierFile failed:", err)
		return
	}
	// 手写静态 4 个（selectAll/selectById/selectByAgeRange/countAll）
	// + MP 内置补生成静态 3 个（selectOne/selectList/selectCount）；
	// selectPage 显式跳过、selectBatchIds 含 foreach 非静态。
	if stats.Functions != 7 {
		t.Error("want 7 functions, got", stats.Functions, "skipped:", stats.Skipped)
	}
	if len(stats.Skipped) != 4 {
		t.Error("want 4 skipped (selectDynamic/selectOrderBy/selectPage/selectBatchIds), got", len(stats.Skipped), stats.Skipped)
	}
	skipIds := map[string]string{}
	for _, sk := range stats.Skipped {
		skipIds[sk.Id] = sk.Reason
	}
	if _, ok := skipIds["selectDynamic"]; !ok {
		t.Error("selectDynamic should be skipped:", stats.Skipped)
	}
	if _, ok := skipIds["selectOrderBy"]; !ok {
		t.Error("selectOrderBy (${} 注入) should be skipped:", stats.Skipped)
	}
	if _, ok := skipIds[MPSelectPageID]; !ok {
		t.Error("selectPage should be skipped:", stats.Skipped)
	}
	if stats.Models != 1 {
		t.Error("want 1 model, got", stats.Models)
	}
	// 文件内容断言
	content, err := os.ReadFile(stats.File)
	if err != nil {
		t.Error("read generated file failed:", err)
		return
	}
	src := string(content)
	for _, want := range []string{
		"type TqUserQuerier interface {",
		"func NewTqUserQuerier(db *orm.DB) TqUserQuerier",
		"SelectAll(ctx context.Context) ([]TqUser, error)",
		"SelectById(ctx context.Context, id int64) ([]TqUser, error)",
		"SelectByAgeRange(ctx context.Context, minAge int, maxAge int) ([]int64, error)",
		"CountAll(ctx context.Context) ([]int32, error)",
		"type TqUser struct {",
		`db:"id,pk"`,
		`db:"user_name"`,
		"where id = ?",
		"where age >= ? and age <= ?",
		"order by id",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated file missing %q\n", want)
		}
	}
	if strings.Contains(src, "#{") {
		t.Error("generated file should not contain #{} placeholders")
	}
	if strings.Contains(src, "${") {
		t.Error("generated file should not contain ${} raw injection")
	}
	// 语法校验（go/parser，不依赖编译）
	if err := ParseQuerierFileSyntax(stats.File); err != nil {
		t.Error("generated file syntax invalid:", err)
	}
}

// Test_GenerateQuerierFile_NoStatic 手写 select 全部不可静态化：仅 MP 内置补生成的
// 静态 select（selectById/selectOne/selectList/selectCount）产出，手写动态语句跳过。
func Test_GenerateQuerierFile_NoStatic(t *testing.T) {
	dir := writeQuerierTestMapper(t, `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="DynMapper">
  <resultMap id="R" type="DynUser">
    <id column="id" jdbcType="BIGINT" property="id" />
  </resultMap>
  <select id="selectDynamic" parameterType="DynUser" resultMap="R">
    select id from t
    <where>
      <if test="id != null">and id = #{id}</if>
    </where>
  </select>
</mapper>
`)
	mps := NewSqlMappers(dir)
	if len(mps.Mappers) != 1 {
		t.Error("want 1 mapper, got", len(mps.Mappers))
		return
	}
	stats, err := mps.Mappers[0].GenerateQuerierFile(t.TempDir(), "querier", nil)
	if err != nil {
		t.Error("GenerateQuerierFile failed:", err)
		return
	}
	if stats.Functions != 4 {
		t.Error("want 4 MP builtin static functions, got:", stats.Functions)
	}
	found := false
	for _, sk := range stats.Skipped {
		if sk.Id == "selectDynamic" {
			found = true
		}
	}
	if !found {
		t.Error("selectDynamic should be skipped:", stats.Skipped)
	}
}

// Test_GenerateQuerierFiles_Samples samples（RuoYi）真实回归：批量生成、模型去重、
// 已知静态/动态语句归类正确、全部产物语法合法。
func Test_GenerateQuerierFiles_Samples(t *testing.T) {
	mps := NewSqlMappers("../samples")
	if len(mps.Mappers) == 0 {
		t.Error("load samples failed")
		return
	}
	out := t.TempDir()
	stats, err := mps.GenerateQuerierFiles(out, "querier")
	if err != nil {
		t.Error("GenerateQuerierFiles failed:", err)
		return
	}
	if len(stats) == 0 {
		t.Error("no stats returned")
	}
	total := 0
	for _, st := range stats {
		total += st.Functions
		if st.File == "" {
			continue
		}
		if err := ParseQuerierFileSyntax(st.File); err != nil {
			t.Errorf("generated file %v syntax invalid: %v", st.File, err)
		}
	}
	if total == 0 {
		t.Error("samples should generate at least one querier function")
	}
	// SysPostMapper：selectPostListByUserId 静态（parameterType=Long + 单占位符）
	var postStats *QuerierGenStats
	for _, st := range stats {
		if GetShortName(st.Mapper) == "SysPostMapper" {
			postStats = st
			break
		}
	}
	if postStats == nil {
		t.Error("SysPostMapper stats not found")
		return
	}
	content, err := os.ReadFile(postStats.File)
	if err != nil {
		t.Error("read SysPostMapper querier failed:", err)
		return
	}
	src := string(content)
	if !strings.Contains(src, "SelectPostListByUserId(ctx context.Context, userId int64)") {
		t.Error("SelectPostListByUserId signature missing, got:\n", src)
	}
	// selectPostList 含 <if> 动态条件 → 跳过
	found := false
	for _, sk := range postStats.Skipped {
		if sk.Id == "selectPostList" {
			found = true
		}
	}
	if !found {
		t.Error("selectPostList (dynamic) should be skipped:", postStats.Skipped)
	}
	// SysDeptMapper：${params.dataScope} 注入 → 跳过
	for _, st := range stats {
		if GetShortName(st.Mapper) != "SysDeptMapper" {
			continue
		}
		found = false
		for _, sk := range st.Skipped {
			if sk.Id == "selectDeptList" {
				found = true
			}
		}
		if !found {
			t.Error("selectDeptList (${params.dataScope}) should be skipped:", st.Skipped)
		}
	}
	t.Logf("samples querier: %v mappers, %v functions total", len(stats), total)
}

// Test_QuerierBaseName namespace → Querier 基名（去 Mapper 后缀 / 保留原名）。
func Test_QuerierBaseName(t *testing.T) {
	cases := map[string]string{
		"SysPostMapper":              "SysPost",
		"UserInfoModelMapper":        "UserInfoModel",
		"com.a.b.UserMapper":         "User",
		"Plain":                      "Plain",
		"Mapper":                     "Mapper",
		"lowermapper":                "lower",
		"com.cncec.XxxModelMapper":   "XxxModel",
	}
	for ns, want := range cases {
		if got := querierBaseName(ns); got != want {
			t.Errorf("querierBaseName(%v) = %v, want %v", ns, got, want)
		}
	}
}

// Test_SanitizeQuerierArgName 占位符名 → 合法 Go 形参名。
func Test_SanitizeQuerierArgName(t *testing.T) {
	cases := map[string]string{
		"userId":     "userId",
		"id":         "id",
		"params.key": "params_key",
		"type":       "type_",
		"2abc":       "p2abc",
		"":           "arg",
		"a b":        "a_b",
	}
	for in, want := range cases {
		if got := sanitizeQuerierArgName(in); got != want {
			t.Errorf("sanitizeQuerierArgName(%v) = %v, want %v", in, got, want)
		}
	}
}
