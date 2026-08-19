package types

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testLongParamMapperXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="TestLongParamMapper">
	<select id="selectConfigById" parameterType="Long" resultType="map">
		select * from sys_con where config_id = #{configId}
	</select>

	<delete id="deleteConfigByIds" parameterType="Long">
		delete from sys_con where config_id in
		<foreach item="configId" collection="array" open="(" separator="," close=")">
			#{configId}
		</foreach>
	</delete>

	<select id="selectUserByUserName" parameterType="String" resultType="map">
		select * from sys_user where user_name = #{userName}
	</select>
</mapper>
`

func writeLongParamTestMapper(t *testing.T) string {
	dir := t.TempDir()
	file := filepath.Join(dir, "TestLongParamMapper.xml")
	if err := os.WriteFile(file, []byte(testLongParamMapperXML), 0644); err != nil {
		t.Error("write test mapper failed:", err)
	}
	return dir
}

func Test_LongParam_Scalar(t *testing.T) {
	mps := NewSqlMappers(writeLongParamTestMapper(t))
	fn := getFunction(t, mps, "selectConfigById")
	if fn == nil {
		return
	}
	sql, _, err := fn.GenerateSQL(int64(1))
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if strings.Contains(sql, "#{") {
		t.Error("Long scalar param literal remains, sql:", sql)
	}
	if !strings.Contains(sql, "where config_id = 1") {
		t.Error("Long scalar param not replaced, sql:", sql)
	}
	t.Log("selectConfigById sql:", sql)
}

func Test_LongParam_Slice(t *testing.T) {
	mps := NewSqlMappers(writeLongParamTestMapper(t))
	fn := getFunction(t, mps, "deleteConfigByIds")
	if fn == nil {
		return
	}
	sql, _, err := fn.GenerateSQL([]int64{1, 2, 3})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if strings.Contains(sql, "#{") {
		t.Error("Long slice param literal remains, sql:", sql)
	}
	if strings.Contains(sql, "in ()") {
		t.Error("empty in () clause, sql:", sql)
	}
	if !strings.Contains(sql, "where config_id in ( 1, 2, 3)") {
		t.Error("Long slice param not expanded, sql:", sql)
	}
	t.Log("deleteConfigByIds sql:", sql)
}

func Test_LongParam_SliceEmpty(t *testing.T) {
	mps := NewSqlMappers(writeLongParamTestMapper(t))
	fn := getFunction(t, mps, "deleteConfigByIds")
	if fn == nil {
		return
	}
	// 空切片：foreach 渲染为空，不产生非法 in () 子句
	sql, _, err := fn.GenerateSQL([]int64{})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if strings.Contains(sql, "in ()") || strings.Contains(sql, "#{") {
		t.Error("empty slice should render nothing, sql:", sql)
	}
	t.Log("deleteConfigByIds(empty) sql:", sql)
}

func Test_LongParam_String(t *testing.T) {
	mps := NewSqlMappers(writeLongParamTestMapper(t))
	fn := getFunction(t, mps, "selectUserByUserName")
	if fn == nil {
		return
	}
	sql, _, err := fn.GenerateSQL("admin")
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if !strings.Contains(sql, "user_name = 'admin'") {
		t.Error("String param not replaced, sql:", sql)
	}
	t.Log("selectUserByUserName sql:", sql)
}

func Test_LongParam_PrepareSQL_Slice(t *testing.T) {
	mps := NewSqlMappers(writeLongParamTestMapper(t))
	fn := getFunction(t, mps, "deleteConfigByIds")
	if fn == nil {
		return
	}
	sql, args, err := fn.PrepareSQL([]int64{1, 2})
	if err != nil {
		t.Error("PrepareSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if !strings.Contains(sql, "in ( ?, ?)") {
		t.Error("prepare slice placeholders wrong, sql:", sql)
	}
	if len(args) != 2 {
		t.Error("prepare slice args count wrong:", args)
	}
	t.Log("deleteConfigByIds prepare sql:", sql, "args:", args)
}

func Test_parseSqlParamTypeFrom_Long(t *testing.T) {
	mp := map[string]SqlParamType{
		"Long":           BaseSqlParam,
		"BIGINT":         BaseSqlParam,
		"Integer":        BaseSqlParam,
		"String":         BaseSqlParam,
		"INT":            BaseSqlParam,
		"java.util.List": SliceSqlParam,
		"Map":            MapSqlParam,
		"SysConfig":      StructSqlParam,
	}
	for k, v := range mp {
		r := parseSqlParamTypeFrom(k)
		if r != v {
			t.Errorf("parseSqlParamTypeFrom(%v) = %v, want %v", k, r, v)
		}
	}
}

func Test_toGolangType_Long(t *testing.T) {
	if r := toGolangType("Long"); r != "int64" {
		t.Error("toGolangType(Long) =", r, "want int64")
	}
	if r := toGolangType("BIGINT"); r != "int64" {
		t.Error("toGolangType(BIGINT) =", r, "want int64")
	}
}

// Test_LongParam_Samples 使用 samples/（RuoYi Mapper）真实文件回归：
// deleteConfigByIds（Long + collection="array" 批量删除）与 deleteConfigById（Long 标量）。
func Test_LongParam_Samples(t *testing.T) {
	mps := NewSqlMappers("../samples")
	if mps == nil {
		t.Error("load samples failed")
		return
	}
	if len(mps.Mappers) == 0 {
		t.Error("no samples mapper loaded")
		return
	}
	// 1) deleteConfigByIds: 批量删除不再生成空 in () 子句
	for _, m := range mps.Mappers {
		fn := m.NamedFunctions["deleteConfigByIds"]
		if fn == nil {
			continue
		}
		sql, _, err := fn.GenerateSQL([]int64{1, 2, 3})
		if err != nil {
			t.Error("deleteConfigByIds GenerateSQL failed:", err)
			return
		}
		sql = collapseSpace(sql)
		t.Log("deleteConfigByIds sql:", sql)
		if strings.Contains(sql, "#{") || strings.Contains(sql, "in ()") {
			t.Error("deleteConfigByIds broken, sql:", sql)
		}
		if !strings.Contains(sql, "where config_id in ( 1, 2, 3)") {
			t.Error("deleteConfigByIds not expanded, sql:", sql)
		}
	}
	// 2) deleteConfigById: Long 标量
	for _, m := range mps.Mappers {
		fn := m.NamedFunctions["deleteConfigById"]
		if fn == nil {
			continue
		}
		sql, _, err := fn.GenerateSQL(int64(1))
		if err != nil {
			t.Error("deleteConfigById GenerateSQL failed:", err)
			return
		}
		sql = collapseSpace(sql)
		t.Log("deleteConfigById sql:", sql)
		if strings.Contains(sql, "#{") {
			t.Error("deleteConfigById literal remains, sql:", sql)
		}
		if !strings.Contains(sql, "where config_id = 1") {
			t.Error("deleteConfigById not replaced, sql:", sql)
		}
	}
}
