package orm

import (
	"os"
	"path/filepath"
	"testing"
)

// E1：xml2go 签名推导不再丢动态体（<if>）内占位符——
// 无 parameterType 语句存在动态体专属占位符时退化为单 map 参数按名取数。
// 本测试按新签名手写 Mapper（等价于 xml2go 产物），验证注册期校验 +
// 运行期 if 条件求值 + map 按名取数全链路。
type E1DynMapper struct {
	BaseMapper
	SelectByIf  func(param map[string]interface{}) ([]map[string]interface{}, error)
	SelectMixed func(param map[string]interface{}) ([]map[string]interface{}, error)
}

func initE1DynSqlite(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="E1DynMapper">
  <select id="selectByIf" resultType="map">
    select id, status from e1_dyn_table
    <where>
      <if test="status != null">and status = #{status}</if>
    </where>
  </select>
  <select id="selectMixed" resultType="map">
    select id, status from e1_dyn_table where id &gt; #{minId}
    <if test="status != null">and status = #{status}</if>
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "E1DynMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	dbPath := filepath.Join(dir, "e1dyn.db")
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

func Test_E1DynamicSlotMapParam(t *testing.T) {
	dir := initE1DynSqlite(t)
	if dir == "" {
		return
	}
	defer Close()
	if _, err := Execute(`CREATE TABLE e1_dyn_table (id INTEGER, status INTEGER)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	for i := 1; i <= 4; i++ {
		st := i % 2
		if _, err := Execute(`INSERT INTO e1_dyn_table (id, status) VALUES (?, ?)`, i, st); err != nil {
			t.Errorf("insert row %v failed: %v", i, err)
			return
		}
	}
	if err := RegisterMapper(new(E1DynMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("E1DynMapper").(E1DynMapper)
	// 占位符全在 <if> 体内：map 带 status → 条件生效；空 map → 全量
	rs, err := mp.SelectByIf(map[string]interface{}{"status": 1})
	if err != nil {
		t.Errorf("SelectByIf(status=1) failed: %v", err)
	} else if len(rs) != 2 {
		t.Errorf("SelectByIf(status=1) expect 2 rows, got %v", len(rs))
	}
	rs, err = mp.SelectByIf(map[string]interface{}{})
	if err != nil {
		t.Errorf("SelectByIf(empty) failed: %v", err)
	} else if len(rs) != 4 {
		t.Errorf("SelectByIf(empty) expect 4 rows, got %v", len(rs))
	}
	// 无条件（minId）+ 条件（status）混合：map 按名取数
	rs, err = mp.SelectMixed(map[string]interface{}{"minId": 2, "status": 1})
	if err != nil {
		t.Errorf("SelectMixed(minId=2,status=1) failed: %v", err)
	} else if len(rs) != 1 {
		t.Errorf("SelectMixed(minId=2,status=1) expect 1 row, got %v", len(rs))
	}
	rs, err = mp.SelectMixed(map[string]interface{}{"minId": 0})
	if err != nil {
		t.Errorf("SelectMixed(minId=0) failed: %v", err)
	} else if len(rs) != 4 {
		t.Errorf("SelectMixed(minId=0) expect 4 rows, got %v", len(rs))
	}
}
