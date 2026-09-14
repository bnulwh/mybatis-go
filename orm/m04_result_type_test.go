package orm

import (
	"os"
	"path/filepath"
	"testing"
)

type M04CustomModel struct {
	Id   int
	Name string
}

type M04CustomMapper struct {
	BaseMapper
	SelectById   func(id int) ([]M04CustomModel, error)
	SelectAll    func() ([]M04CustomModel, error)
	SelectByMap  func(params map[string]interface{}) ([]map[string]interface{}, error)
}

func initM04Sqlite(t *testing.T) string {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="M04CustomMapper">
  <select id="selectById" resultType="M04CustomModel" parameterType="int">
    select id, name from m04_table where id = #{id}
  </select>
  <select id="selectAll" resultType="M04CustomModel">
    select id, name from m04_table
  </select>
  <select id="selectByMap" resultType="map" parameterType="map">
    select id, name from m04_table where name = #{name}
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "M04CustomMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	dbPath := filepath.Join(dir, "m04.db")
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

func Test_M04_CustomResultType_Registration(t *testing.T) {
	dir := initM04Sqlite(t)
	if dir == "" {
		return
	}
	defer Close()
	if _, err := Execute(`CREATE TABLE m04_table (id INTEGER, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	if _, err := Execute(`INSERT INTO m04_table (id, name) VALUES (1, 'Alice'), (2, 'Bob')`); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	RegisterModel(new(M04CustomModel))
	if err := RegisterMapper(new(M04CustomMapper)); err != nil {
		t.Errorf("M-04: register mapper with custom resultType should succeed, got: %v", err)
		return
	}
	mp := NewMapper("M04CustomMapper").(M04CustomMapper)

	rs, err := mp.SelectById(1)
	if err != nil {
		t.Errorf("selectById failed: %v", err)
		return
	}
	if len(rs) != 1 {
		t.Errorf("selectById expect 1 row, got %d", len(rs))
	}
	if rs[0].Name != "Alice" {
		t.Errorf("selectById name = %q, want Alice", rs[0].Name)
	}

	rs2, err := mp.SelectAll()
	if err != nil {
		t.Errorf("selectAll failed: %v", err)
		return
	}
	if len(rs2) != 2 {
		t.Errorf("selectAll expect 2 rows, got %d", len(rs2))
	}

	rs3, err := mp.SelectByMap(map[string]interface{}{"name": "Bob"})
	if err != nil {
		t.Errorf("selectByMap failed: %v", err)
		return
	}
	if len(rs3) != 1 {
		t.Errorf("selectByMap expect 1 row, got %d", len(rs3))
	}
}

func Test_M04_CustomResultType_FullyQualifiedName(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="M04FqMapper">
  <select id="selectById" resultType="com.example.domain.M04CustomModel" parameterType="int">
    select id, name from m04_table where id = #{id}
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "M04FqMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return
	}
	dbPath := filepath.Join(dir, "m04fq.db")
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return
	}
	defer Close()
	RegisterModel(new(M04CustomModel))

	type M04FqMapper struct {
		BaseMapper
		SelectById func(id int) ([]M04CustomModel, error)
	}
	if err := RegisterMapper(new(M04FqMapper)); err != nil {
		t.Errorf("M-04: register with fully-qualified resultType should succeed, got: %v", err)
	}
}

func Test_M04_ParseResultTypeName(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="M04NameTest">
  <select id="selectA" resultType="SysUser" parameterType="int">
    select id from t
  </select>
  <select id="selectB" resultType="map">
    select id from t
  </select>
  <select id="selectC" resultType="com.foo.BarModel">
    select id from t
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "M04NameTest.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return
	}
	dbPath := filepath.Join(dir, "namedb")
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return
	}
	defer Close()
	smp := gCache.sqls.NamedMappers["m04nametest"]
	if smp == nil {
		t.Error("M04NameTest mapper not found")
		return
	}
	sa, ok := smp.NamedFunctions["selecta"]
	if !ok {
		t.Error("selectA not found")
		return
	}
	if sa.Result.ResultTypeName != "SysUser" {
		t.Errorf("selectA ResultTypeName = %q, want %q", sa.Result.ResultTypeName, "SysUser")
	}
	sb, ok := smp.NamedFunctions["selectb"]
	if !ok {
		t.Error("selectB not found")
		return
	}
	if sb.Result.ResultTypeName != "map" {
		t.Errorf("selectB ResultTypeName = %q, want %q", sb.Result.ResultTypeName, "map")
	}
	sc, ok := smp.NamedFunctions["selectc"]
	if !ok {
		t.Error("selectC not found")
		return
	}
	if sc.Result.ResultTypeName != "com.foo.BarModel" {
		t.Errorf("selectC ResultTypeName = %q, want %q", sc.Result.ResultTypeName, "com.foo.BarModel")
	}
}
