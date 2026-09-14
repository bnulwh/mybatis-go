package orm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bnulwh/mybatis-go/types"
)

type IfTestModel struct {
	Id     int
	Name   string
	Status int
}

type IfTestMapper struct {
	BaseMapper
	SelectWithIf    func(params map[string]interface{}) ([]map[string]interface{}, error)
	SelectWithBadIf func(params map[string]interface{}) ([]map[string]interface{}, error)
}

func initIfTestSqlite(t *testing.T) string {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="IfTestMapper">
  <select id="selectWithIf" resultType="map" parameterType="IfTestModel">
    select id, name from if_test_table
    <where>
      <if test="name != null and name != ''">and name = #{name}</if>
      <if test="status != null">and status = #{status}</if>
    </where>
  </select>
  <select id="selectWithBadIf" resultType="map" parameterType="IfTestModel">
    select id, name from if_test_table
    <where>
      <if test="nonExistField != null">and id = #{id}</if>
      <if test="name != null">and name = #{name}</if>
    </where>
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "IfTestMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	dbPath := filepath.Join(dir, "iftest.db")
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return ""
	}
	return dir
}

func Test_4_3_IfTestFieldValidation(t *testing.T) {
	dir := initIfTestSqlite(t)
	if dir == "" {
		return
	}
	defer Close()
	if _, err := Execute(`CREATE TABLE if_test_table (id INTEGER, name TEXT, status INTEGER)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	RegisterModel(new(IfTestModel))
	if err := RegisterMapper(new(IfTestMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("IfTestMapper").(IfTestMapper)
	rs, err := mp.SelectWithIf(map[string]interface{}{"name": "test"})
	if err != nil {
		t.Errorf("selectWithIf failed: %v", err)
	}
	if len(rs) != 0 {
		t.Errorf("selectWithIf expect 0 rows (empty table), got %d", len(rs))
	}
}

func Test_4_3_CollectIfTestFieldNames(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="IfTestCollect">
  <select id="testCollect" resultType="map" parameterType="map">
    select * from t
    <where>
      <if test="name != null">and name = #{name}</if>
      <if test="status != null">and status = #{status}</if>
      <if test="params.beginTime != null">and created &gt;= #{params.beginTime}</if>
      <if test="items.length > 0">and id in <foreach collection="items" item="id" open="(" close=")" separator=",">#{id}</foreach></if>
    </where>
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "IfTestCollect.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return
	}
	dbPath := filepath.Join(dir, "collect.db")
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return
	}
	defer Close()
	smp := gCache.sqls.NamedMappers["iftestcollect"]
	if smp == nil {
		t.Error("IfTestCollect mapper not found")
		return
	}
	sf, ok := smp.NamedFunctions["testcollect"]
	if !ok {
		t.Error("testCollect function not found")
		return
	}
	if len(sf.IfTestFields) != 3 {
		t.Errorf("IfTestFields count = %d, want 3 (name, status, items); got %v", len(sf.IfTestFields), sf.IfTestFields)
	}
	fieldMap := map[string]bool{}
	for _, f := range sf.IfTestFields {
		fieldMap[f] = true
	}
	if !fieldMap["name"] {
		t.Error("missing field 'name' in IfTestFields")
	}
	if !fieldMap["status"] {
		t.Error("missing field 'status' in IfTestFields")
	}
	if !fieldMap["items"] {
		t.Error("missing field 'items' (from items.length) in IfTestFields")
	}
	if fieldMap["params"] || fieldMap["params.beginTime"] {
		t.Error("params.beginTime should be excluded (params. prefix)")
	}
}

func Test_4_3_IsJdkType(t *testing.T) {
	for _, name := range []string{"string", "Long", "int", "map", "HashMap"} {
		if !types.IsJdkType(name) {
			t.Errorf("IsJdkType(%q) should be true", name)
		}
	}
	for _, name := range []string{"SysUser", "IfTestModel", "MdmOrgRaw"} {
		if types.IsJdkType(name) {
			t.Errorf("IsJdkType(%q) should be false", name)
		}
	}
}
