package types

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeXml2GoTestMapper 写入测试用 Mapper XML（业务模型 + 关联 + MP 内置补生成）。
func writeXml2GoTestMapper(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("create mapper dir failed: %v", err)
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="com.demo.mapper.SysUserMapper">
  <resultMap id="sysUserResult" type="com.demo.domain.SysUser">
    <id column="user_id" jdbcType="BIGINT" property="userId" />
    <result column="user_name" jdbcType="VARCHAR" property="userName" />
    <result column="version" jdbcType="INTEGER" property="version" />
    <result column="del_flag" jdbcType="CHAR" property="delFlag" />
    <result column="create_time" jdbcType="TIMESTAMP" property="createTime" />
    <association property="dept" resultMap="deptResult" />
    <collection property="roles" ofType="SysRole" />
  </resultMap>
  <resultMap id="deptResult" type="com.demo.domain.SysDept">
    <id column="dept_id" jdbcType="BIGINT" property="deptId" />
    <result column="dept_name" jdbcType="VARCHAR" property="deptName" />
  </resultMap>
  <insert id="insertUser" parameterType="com.demo.domain.SysUser" useGeneratedKeys="true" keyProperty="userId">
    insert into sys_user (user_name, del_flag) values (#{userName}, '0')
  </insert>
  <delete id="deleteUserById" parameterType="java.lang.Long">
    delete from sys_user where user_id = #{userId}
  </delete>
  <select id="selectUserById" parameterType="java.lang.Long" resultMap="sysUserResult">
    select * from sys_user where user_id = #{userId}
  </select>
  <select id="selectByStatus" resultMap="sysUserResult">
    select * from sys_user where status = #{status} and dept_id = #{deptId}
  </select>
  <select id="selectUserCount" resultType="long">
    select count(*) from sys_user
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(dir, "SysUserMapper.xml"), []byte(xml), 0644); err != nil {
		t.Fatalf("write mapper xml failed: %v", err)
	}
}

// Test_GenerateXml2GoFiles_Basic 基础生成：模型 db/json tag、TableName、
// 指针参数、按位绑定参数、MP 内置方法、init 注册、单例 getter、语法合法。
func Test_GenerateXml2GoFiles_Basic(t *testing.T) {
	src := t.TempDir()
	writeXml2GoTestMapper(t, src)
	mps := NewSqlMappers(src)
	if len(mps.Mappers) == 0 {
		t.Error("load test mapper failed")
		return
	}
	out := t.TempDir()
	result, err := mps.GenerateXml2GoFiles(&Xml2GoOptions{OutputDir: out, ModulePrefix: "example.com/demo/app"})
	if err != nil {
		t.Error("GenerateXml2GoFiles failed:", err)
		return
	}
	if len(result.ModelFiles) == 0 {
		t.Error("no model files generated")
		return
	}
	// 模型文件语法合法
	for _, f := range result.ModelFiles {
		if err := ParseQuerierFileSyntax(f); err != nil {
			t.Errorf("model file %v syntax invalid: %v", f, err)
		}
	}
	// SysUser 模型：pk/logic/version tag、关联类型、json tag、TableName、time 导入
	userModel, err := os.ReadFile(filepath.Join(out, "models", "SysUser.go"))
	if err != nil {
		t.Error("read SysUser.go failed:", err)
		return
	}
	um := collapseSpace(string(userModel))
	for _, want := range []string{
		`package models`,
		`import "time"`,
		`UserId int64 ` + "`" + `db:"user_id,pk" json:"userId"` + "`",
		`DelFlag string ` + "`" + `db:"del_flag,logic" json:"delFlag"` + "`",
		`Version int ` + "`" + `db:"version,version" json:"version"` + "`",
		`Dept *SysDept ` + "`" + `db:"-" json:"dept"` + "`",
		`Roles []SysRole ` + "`" + `db:"-" json:"roles"` + "`",
		`func (SysUser) TableName() string { return "sys_user" }`,
	} {
		if !strings.Contains(um, want) {
			t.Errorf("SysUser model missing %q, got:\n%v", want, um)
		}
	}
	// Mapper 文件：签名 + 注册 + getter
	var mapperFile string
	var st *Xml2GoStats
	for _, s := range result.Stats {
		if GetShortName(s.Mapper) == "SysUserMapper" {
			st = s
			mapperFile = s.MapperFile
		}
	}
	if st == nil || mapperFile == "" {
		t.Error("SysUserMapper stats/file not found:", result.Stats)
		return
	}
	if err := ParseQuerierFileSyntax(mapperFile); err != nil {
		t.Error("mapper file syntax invalid:", err)
		return
	}
	mc, err := os.ReadFile(mapperFile)
	if err != nil {
		t.Error("read mapper file failed:", err)
		return
	}
	src2 := collapseSpace(string(mc))
	for _, want := range []string{
		"package mapper",
		`models "example.com/demo/app/models"`,
		"type SysUserMapper struct { orm.BaseMapper",
		"InsertUser func(model *models.SysUser) (int64, error)",
		"DeleteUserById func(userId int64) (int64, error)",
		"SelectUserById func(userId int64) ([]models.SysUser, error)",
		"SelectByStatus func(status interface{}, deptId interface{}) ([]models.SysUser, error)",
		"SelectUserCount func() ([]int64, error)",
		// MP 内置（resultMap 含主键 → 内存补生成）
		"Insert func(model *models.SysUser) (int64, error)",
		"SelectById func(userId int64) ([]models.SysUser, error)",
		"SelectPage func(page *orm.PageParam) (*orm.Page, error)",
		"InsertBatch func(models []*models.SysUser) (int64, error)",
		"orm.RegisterModel(new(models.SysUser))",
		"orm.RegisterModel(new(models.SysDept))",
		"orm.RegisterMapper(new(SysUserMapper))",
		"func GetSysUserMapper() *SysUserMapper",
		`orm.NewMapperPtr("SysUserMapper").(*SysUserMapper)`,
	} {
		if !strings.Contains(src2, want) {
			t.Errorf("mapper file missing %q, got:\n%v", want, src2)
		}
	}
}

// Test_GenerateXml2GoFiles_SkipMP skip-mp：MP 内置字段全部剔除，手写语句保留。
func Test_GenerateXml2GoFiles_SkipMP(t *testing.T) {
	src := t.TempDir()
	writeXml2GoTestMapper(t, src)
	mps := NewSqlMappers(src)
	out := t.TempDir()
	result, err := mps.GenerateXml2GoFiles(&Xml2GoOptions{OutputDir: out, SkipMP: true})
	if err != nil {
		t.Error("GenerateXml2GoFiles failed:", err)
		return
	}
	var mapperFile string
	for _, s := range result.Stats {
		if GetShortName(s.Mapper) == "SysUserMapper" {
			mapperFile = s.MapperFile
		}
	}
	if mapperFile == "" {
		t.Error("SysUserMapper file not found")
		return
	}
	mc, err := os.ReadFile(mapperFile)
	if err != nil {
		t.Error("read mapper file failed:", err)
		return
	}
	src2 := collapseSpace(string(mc))
	if !strings.Contains(src2, "InsertUser func(") {
		t.Error("handwritten InsertUser should be kept, got:\n", src2)
	}
	for _, mp := range mpBuiltinIDs {
		field := UpperFirst(sanitizeQuerierArgName(mp)) + " func("
		if strings.Contains(src2, "\t"+field) || strings.Contains(src2, " "+field) {
			t.Errorf("mp builtin %v should be skipped, got:\n%v", mp, src2)
		}
	}
}

// Test_GenerateXml2GoFiles_SkipUnknownModel parameterType 引用未生成模型 → 跳过并记录原因。
func Test_GenerateXml2GoFiles_SkipUnknownModel(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="com.demo.mapper.UnknownMapper">
  <resultMap id="baseResult" type="com.demo.domain.Known">
    <id column="id" jdbcType="BIGINT" property="id" />
  </resultMap>
  <insert id="insertMissing" parameterType="com.demo.domain.Missing">
    insert into t_known (id) values (#{id})
  </insert>
  <select id="selectKnown" resultMap="baseResult">
    select id from t_known
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(dir, "UnknownMapper.xml"), []byte(xml), 0644); err != nil {
		t.Fatal(err)
	}
	mps := NewSqlMappers(dir)
	out := t.TempDir()
	result, err := mps.GenerateXml2GoFiles(&Xml2GoOptions{OutputDir: out})
	if err != nil {
		t.Error("GenerateXml2GoFiles failed:", err)
		return
	}
	var st *Xml2GoStats
	for _, s := range result.Stats {
		if GetShortName(s.Mapper) == "UnknownMapper" {
			st = s
		}
	}
	if st == nil {
		t.Error("UnknownMapper stats not found")
		return
	}
	found := false
	for _, sk := range st.Skipped {
		if sk.Id == "insertMissing" && strings.Contains(sk.Reason, "Missing") {
			found = true
		}
	}
	if !found {
		t.Error("insertMissing should be skipped with reason, got:", st.Skipped)
	}
	mc, err := os.ReadFile(st.MapperFile)
	if err != nil {
		t.Error("read mapper file failed:", err)
		return
	}
	mc2 := collapseSpace(string(mc))
	if strings.Contains(mc2, "InsertMissing") {
		t.Error("InsertMissing should not be emitted, got:\n", mc2)
	}
	if !strings.Contains(mc2, "SelectKnown func() ([]models.Known, error)") {
		t.Error("SelectKnown should be emitted, got:\n", mc2)
	}
}

// Test_GenerateXml2GoFiles_Samples samples（RuoYi）真实回归：
// 全部模型与 Mapper 文件语法合法、模型跨 Mapper 去重、MP 内置自动补生成。
func Test_GenerateXml2GoFiles_Samples(t *testing.T) {
	mps := NewSqlMappers("../samples")
	if len(mps.Mappers) == 0 {
		t.Error("load samples failed")
		return
	}
	out := t.TempDir()
	result, err := mps.GenerateXml2GoFiles(&Xml2GoOptions{OutputDir: out, ModulePrefix: "example.com/demo"})
	if err != nil {
		t.Error("GenerateXml2GoFiles failed:", err)
		return
	}
	if len(result.ModelFiles) == 0 || len(result.Stats) == 0 {
		t.Error("no files generated for samples")
		return
	}
	for _, f := range result.ModelFiles {
		if err := ParseQuerierFileSyntax(f); err != nil {
			t.Errorf("model file %v syntax invalid: %v", f, err)
		}
	}
	userCount := 0
	for _, f := range result.ModelFiles {
		if filepath.Base(f) == "SysUser.go" {
			userCount++
		}
		if err := ParseQuerierFileSyntax(f); err == nil {
			continue
		}
	}
	if userCount != 1 {
		t.Errorf("SysUser model should be deduplicated to 1 file, got %v", userCount)
	}
	// SysUserMapper：MP 内置 + 手写方法并存，产物语法合法
	var st *Xml2GoStats
	for _, s := range result.Stats {
		if GetShortName(s.Mapper) == "SysUserMapper" {
			st = s
		}
		if s.MapperFile != "" {
			if err := ParseQuerierFileSyntax(s.MapperFile); err != nil {
				t.Errorf("mapper file %v syntax invalid: %v", s.MapperFile, err)
			}
		}
	}
	if st == nil || st.MapperFile == "" {
		t.Error("SysUserMapper stats/file not found")
		return
	}
	mc, err := os.ReadFile(st.MapperFile)
	if err != nil {
		t.Error("read SysUserMapper.go failed:", err)
		return
	}
	src := collapseSpace(string(mc))
	if !strings.Contains(src, "Insert func(model *models.SysUser) (int64, error)") {
		t.Error("samples SysUserMapper should contain MP Insert, got:\n", src)
	}
	// 本 samples 语料为裁剪版 RuoYi：无 selectUserById/deleteUserById，以实际存在的语句断言
	if !strings.Contains(src, "DeleteUserByIds func(ids []int64) (int64, error)") {
		t.Error("samples SysUserMapper should contain DeleteUserByIds, got:\n", src)
	}
	if !strings.Contains(src, "SelectUserByUserName func(userName string) ([]models.SysUser, error)") {
		t.Error("samples SysUserMapper should contain SelectUserByUserName, got:\n", src)
	}
	totalFn, skipped := 0, 0
	for _, s := range result.Stats {
		totalFn += s.Functions
		skipped += len(s.Skipped)
	}
	if totalFn == 0 {
		t.Error("samples should generate at least one function")
	}
	t.Logf("samples xml2go: %v model files, %v mappers, %v functions, %v skipped",
		len(result.ModelFiles), len(result.Stats), totalFn, skipped)
}
