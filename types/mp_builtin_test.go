package types

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testMPBuiltinMapperXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="SysUserMapper">
	<resultMap id="SysUserResult" type="SysUser">
		<id     property="userId"   column="user_id"/>
		<result property="userName" column="user_name"/>
		<result property="delFlag"  column="del_flag"/>
	</resultMap>
	<select id="selectUserList" parameterType="SysUser" resultMap="SysUserResult">
		select user_id, user_name from sys_user where del_flag = '0'
	</select>
</mapper>
`

func writeMPBuiltinTestMapper(t *testing.T, content string) string {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SysUserMapper.xml"), []byte(content), 0644); err != nil {
		t.Error("write test mapper failed:", err)
	}
	return dir
}

// Test_MPBuiltin_AutoGenerate XML 有 resultMap（基本类型列、无 jdbcType）+ 无 MP CRUD → 内存补生成全部 10 个内置方法。
func Test_MPBuiltin_AutoGenerate(t *testing.T) {
	mps := NewSqlMappers(writeMPBuiltinTestMapper(t, testMPBuiltinMapperXML))
	if mps == nil || len(mps.Mappers) == 0 {
		t.Error("load mapper failed")
		return
	}
	m := mps.Mappers[0]
	// 原有方法保留
	if m.NamedFunctions["selectUserList"] == nil {
		t.Error("existing selectUserList should be kept")
	}
	// 10 个 MP 内置方法齐全
	var missing []string
	for _, id := range mpBuiltinIDs {
		if m.NamedFunctions[id] == nil {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		t.Error("missing mp builtin methods:", missing)
	}
	// 表名推导：SysUser → sys_user（日志可见于 TS.Table 间接验证 SQL）
	fn := m.NamedFunctions["selectById"]
	if fn == nil {
		return
	}
	// selectById：含逻辑删除过滤（del_flag 约定）
	sql, _, err := fn.GenerateSQL(int64(1))
	if err != nil {
		t.Error("selectById GenerateSQL failed:", err)
	} else {
		sql = collapseSpace(sql)
		t.Log("selectById sql:", sql)
		if !strings.Contains(sql, "from sys_user where user_id=1") || !strings.Contains(sql, "del_flag = '0'") {
			t.Error("selectById sql unexpected:", sql)
		}
	}
	// deleteById：RuoYi del_flag 逻辑删除
	sql, _, err = m.NamedFunctions["deleteById"].GenerateSQL(int64(1))
	if err != nil {
		t.Error("deleteById GenerateSQL failed:", err)
	} else if sql = collapseSpace(sql); !strings.Contains(sql, "update sys_user set del_flag='2'") {
		t.Error("deleteById should soft-delete via del_flag, sql:", sql)
	}
	// deleteBatchIds
	sql, _, err = m.NamedFunctions["deleteBatchIds"].GenerateSQL([]int64{1, 2})
	if err != nil {
		t.Error("deleteBatchIds GenerateSQL failed:", err)
	} else if sql = collapseSpace(sql); !strings.Contains(sql, "update sys_user set del_flag='2' where user_id in (") {
		t.Error("deleteBatchIds should soft-delete via del_flag, sql:", sql)
	}
	// insert：parameterType = resultMap 原 type（SysUser）
	if m.NamedFunctions["insert"].Param.TypeName != "SysUser" {
		t.Error("insert parameterType =", m.NamedFunctions["insert"].Param.TypeName, "want SysUser")
	}
	// codegen：DeleteById func(int64)、select 返回 models.SysUser
	content := string(m.generateContent("src"))
	if !strings.Contains(content, "DeleteById \tfunc (int64)") {
		t.Error("codegen DeleteById should be func(int64), content:\n", content)
	}
	if !strings.Contains(content, "SelectById \tfunc (int64) ([]models.SysUser,error)") {
		t.Error("codegen SelectById should return []models.SysUser, content:\n", content)
	}
	if !strings.Contains(content, "DeleteBatchIds \tfunc ([]int64)") {
		t.Error("codegen DeleteBatchIds should be func([]int64), content:\n", content)
	}
}

// Test_MPBuiltin_SkipIfExists 已有部分 MP 方法 → 按 ID 补缺、不覆盖手写实现。
func Test_MPBuiltin_SkipIfExists(t *testing.T) {
	custom := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="SysUserMapper">
	<resultMap id="SysUserResult" type="SysUser">
		<id     property="userId"   column="user_id"/>
		<result property="userName" column="user_name"/>
	</resultMap>
	<select id="selectList" resultMap="SysUserResult">
		select user_id, user_name from sys_user order by user_id
	</select>
</mapper>
`
	mps := NewSqlMappers(writeMPBuiltinTestMapper(t, custom))
	if mps == nil || len(mps.Mappers) == 0 {
		t.Error("load mapper failed")
		return
	}
	m := mps.Mappers[0]
	// 手写 selectList 不被覆盖
	if sql, _, err := m.NamedFunctions["selectList"].GenerateSQL(); err != nil {
		t.Error("hand-written selectList failed:", err)
	} else if !strings.Contains(sql, "order by user_id") {
		t.Error("hand-written selectList should be kept, sql:", sql)
	}
	// 缺失的补上
	for _, id := range []string{"insert", "deleteById", "updateById", "selectById", "selectOne", "selectPage", "selectCount", "selectBatchIds", "deleteBatchIds"} {
		if m.NamedFunctions[id] == nil {
			t.Errorf("missing auto-generated method %q", id)
		}
	}
}

// Test_MPBuiltin_NoResultMap 无 resultMap 的 Mapper 不生成。
func Test_MPBuiltin_NoResultMap(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="PlainMapper">
	<select id="countAll" resultType="int">
		select count(*) from plain_table
	</select>
</mapper>
`
	mps := NewSqlMappers(writeMPBuiltinTestMapper(t, xml))
	if mps == nil || len(mps.Mappers) == 0 {
		t.Error("load mapper failed")
		return
	}
	for _, id := range mpBuiltinIDs {
		if mps.Mappers[0].NamedFunctions[id] != nil {
			t.Errorf("mapper without resultMap should not generate %q", id)
		}
	}
}

// Test_MPBuiltin_JDKResultMap resultMap type 为 JDK 基础类型 → 不生成。
func Test_MPBuiltin_JDKResultMap(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="MapMapper">
	<resultMap id="R" type="map">
		<id     property="id"   column="id"/>
		<result property="name" column="name"/>
	</resultMap>
	<select id="list" resultMap="R">
		select * from t
	</select>
</mapper>
`
	mps := NewSqlMappers(writeMPBuiltinTestMapper(t, xml))
	if mps == nil || len(mps.Mappers) == 0 {
		t.Error("load mapper failed")
		return
	}
	for _, id := range mpBuiltinIDs {
		if mps.Mappers[0].NamedFunctions[id] != nil {
			t.Errorf("jdk-type resultMap should not generate %q", id)
		}
	}
}

// Test_MPBuiltin_Samples samples（RuoYi）真实回归：SysUserMapper 自动补 MP CRUD（del_flag 语义）。
func Test_MPBuiltin_Samples(t *testing.T) {
	mps := NewSqlMappers("../samples")
	if mps == nil || len(mps.Mappers) == 0 {
		t.Error("load samples failed")
		return
	}
	var fp *SqlMapper
	for i := range mps.Mappers {
		if strings.HasSuffix(mps.Mappers[i].Namespace, ".SysUserMapper") {
			fp = &mps.Mappers[i]
			break
		}
	}
	if fp == nil {
		t.Error("SysUserMapper not found in samples")
		return
	}
	fn := fp.NamedFunctions["selectById"]
	if fn == nil {
		t.Error("samples SysUserMapper should auto-generate selectById")
		return
	}
	sql, _, err := fn.GenerateSQL(int64(1))
	if err != nil {
		t.Error("selectById GenerateSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	t.Log("samples SysUserMapper selectById sql:", sql)
	if !strings.Contains(sql, "from sys_user where user_id=1") || !strings.Contains(sql, "del_flag = '0'") {
		t.Error("samples auto selectById should filter del_flag = '0', sql:", sql)
	}
	if fp.NamedFunctions["selectUserList"] == nil {
		t.Error("existing selectUserList should be kept")
	}
}

// Test_buildTableStructureFromResultMap 推导单元测试。
func Test_buildTableStructureFromResultMap(t *testing.T) {
	// SysUser → sys_user，Model 后缀剥离
	mps := NewSqlMappers(writeMPBuiltinTestMapper(t, testMPBuiltinMapperXML))
	m := &mps.Mappers[0]
	ts, rm := buildTableStructureFromResultMap(m)
	if ts == nil || rm == nil {
		t.Error("buildTableStructureFromResultMap should succeed")
		return
	}
	if ts.Table != "sys_user" {
		t.Error("table =", ts.Table, "want sys_user")
	}
	if ts.ModelName != "SysUser" {
		t.Error("modelName =", ts.ModelName, "want SysUser")
	}
	if ts.PrimaryColumn == nil || ts.PrimaryColumn.Name != "user_id" {
		t.Error("primary column should be user_id, got:", ts.PrimaryColumn)
	}
	if ts.PrimaryColumn.Type.String() != "int64" {
		t.Error("pk without jdbcType should default int64, got:", ts.PrimaryColumn.Type)
	}
	if !ts.hasColumn("del_flag") || !ts.hasLogicalDelete() {
		t.Error("del_flag should be detected as logical delete column")
	}
	// Model 后缀剥离：SysUserModel → sys_user
	xml := strings.ReplaceAll(testMPBuiltinMapperXML, `type="SysUser"`, `type="SysUserModel"`)
	mps2 := NewSqlMappers(writeMPBuiltinTestMapper(t, xml))
	if ts2, _ := buildTableStructureFromResultMap(&mps2.Mappers[0]); ts2 == nil || ts2.Table != "sys_user" {
		t.Error("SysUserModel should derive table sys_user, got:", ts2)
	}
}

// Test_camelToSnake
func Test_camelToSnake(t *testing.T) {
	cases := map[string]string{
		"SysUser":  "sys_user",
		"sys_user": "sys_user",
		"User":     "user",
		"ABC":      "a_b_c",
	}
	for in, want := range cases {
		if got := camelToSnake(in); got != want {
			t.Errorf("camelToSnake(%q) = %q, want %q", in, got, want)
		}
	}
}