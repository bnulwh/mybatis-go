package types

import (
	"testing"
	"testing/fstest"
)

// A3：SqlFunction.UnsupportedIfTests 聚合（语句级未识别 <if test> 表达式汇总）。

func a3UnsupportedFS() fstest.MapFS {
	return fstest.MapFS{
		"m/A3Mapper.xml": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="A3Mapper">
  <select id="selectUsers" resultType="map">
    select * from t_user
    <where>
      <if test="name != null and name != ''">and name like #{name}</if>
      <if test="a.toUpperCase() == 'X'">and dept = #{dept}</if>
      <choose>
        <when test="flag != null and flag != 0">and flag = #{flag}</when>
        <otherwise>and flag is null</otherwise>
      </choose>
    </where>
  </select>
</mapper>`)},
	}
}

// Test_SqlFunction_UnsupportedIfTests 语句内未识别表达式去重聚合，
// 已识别条件（含 choose/when）不混入。
func Test_SqlFunction_UnsupportedIfTests(t *testing.T) {
	mps := NewSqlMappersFromSources(MapperSource{FS: a3UnsupportedFS(), Patterns: []string{"m"}})
	mp, ok := mps.NamedMappers["A3Mapper"]
	if !ok {
		t.Error("A3Mapper missing in NamedMappers")
		return
	}
	sf, ok := mp.NamedFunctions["selectusers"]
	if !ok {
		t.Error("selectUsers missing in NamedFunctions")
		return
	}
	got := sf.UnsupportedIfTests
	if len(got) != 1 || got[0] != "a.toUpperCase() == 'X'" {
		t.Errorf("UnsupportedIfTests = %v, want [a.toUpperCase() == 'X']", got)
	}
	// 已识别条件字段不受影响
	if len(sf.IfTestFields) == 0 {
		t.Error("IfTestFields should still collect recognized fields")
	}
}

// Test_SqlFunction_UnsupportedIfTests_None 全识别语句聚合结果为空。
func Test_SqlFunction_UnsupportedIfTests_None(t *testing.T) {
	mfs := fstest.MapFS{
		"m/OkMapper.xml": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="OkMapper">
  <select id="sel" resultType="map">
    select * from t
    <if test="name != null">where name = #{name}</if>
  </select>
</mapper>`)},
	}
	mps := NewSqlMappersFromSources(MapperSource{FS: mfs, Patterns: []string{"m"}})
	mp := mps.NamedMappers["OkMapper"]
	if mp == nil {
		t.Error("OkMapper missing")
		return
	}
	sf := mp.NamedFunctions["sel"]
	if sf == nil {
		t.Error("sel missing")
		return
	}
	if got := sf.UnsupportedIfTests; got != nil {
		t.Errorf("fully recognized statement should collect nothing, got %v", got)
	}
}
