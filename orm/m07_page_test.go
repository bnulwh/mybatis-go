package orm

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type M07PageMapper struct {
	BaseMapper
	SelectPage     func(page *PageParam) (*Page, error)
	SelectPageWith func(page *PageParam, params map[string]interface{}) (*Page, error)
	SelectAll      func() ([]map[string]interface{}, error)
}

func initM07Sqlite(t *testing.T) string {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="M07PageMapper">
  <select id="selectPage" resultType="map">
    select id, name from m07_table order by id
  </select>
  <select id="selectPageWith" resultType="map" parameterType="map">
    select id, name from m07_table where name like #{name} order by id
  </select>
  <select id="selectAll" resultType="map">
    select id, name from m07_table order by id
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "M07PageMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	dbPath := filepath.Join(dir, "m07.db")
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

func Test_M07_SelectPage(t *testing.T) {
	dir := initM07Sqlite(t)
	if dir == "" {
		return
	}
	defer Close()
	if _, err := Execute(`CREATE TABLE m07_table (id INTEGER, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	for i := 1; i <= 25; i++ {
		if _, err := Execute(`INSERT INTO m07_table (id, name) VALUES (?, ?)`, i, fmt.Sprintf("user_%d", i)); err != nil {
			t.Errorf("insert failed: %v", err)
			return
		}
	}
	if err := RegisterMapper(new(M07PageMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("M07PageMapper").(M07PageMapper)

	p1, err := mp.SelectPage(&PageParam{PageNum: 1, PageSize: 10})
	if err != nil {
		t.Errorf("selectPage page 1 failed: %v", err)
		return
	}
	if p1.Total != 25 {
		t.Errorf("page 1 total = %d, want 25", p1.Total)
	}
	if len(p1.Records) != 10 {
		t.Errorf("page 1 records = %d, want 10", len(p1.Records))
	}
	if p1.PageNum != 1 || p1.PageSize != 10 {
		t.Errorf("page 1 metadata: PageNum=%d PageSize=%d", p1.PageNum, p1.PageSize)
	}

	p3, err := mp.SelectPage(&PageParam{PageNum: 3, PageSize: 10})
	if err != nil {
		t.Errorf("selectPage page 3 failed: %v", err)
		return
	}
	if p3.Total != 25 {
		t.Errorf("page 3 total = %d, want 25", p3.Total)
	}
	if len(p3.Records) != 5 {
		t.Errorf("page 3 records = %d, want 5", len(p3.Records))
	}

	p4, err := mp.SelectPage(&PageParam{PageNum: 4, PageSize: 10})
	if err != nil {
		t.Errorf("selectPage page 4 failed: %v", err)
		return
	}
	if len(p4.Records) != 0 {
		t.Errorf("page 4 should have 0 records, got %d", len(p4.Records))
	}
}

func Test_M07_SelectPageWithParams(t *testing.T) {
	dir := initM07Sqlite(t)
	if dir == "" {
		return
	}
	defer Close()
	if _, err := Execute(`CREATE TABLE m07_table (id INTEGER, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	for i := 1; i <= 15; i++ {
		if _, err := Execute(`INSERT INTO m07_table (id, name) VALUES (?, ?)`, i, fmt.Sprintf("user_%d", i)); err != nil {
			t.Errorf("insert failed: %v", err)
			return
		}
	}
	if err := RegisterMapper(new(M07PageMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("M07PageMapper").(M07PageMapper)

	p, err := mp.SelectPageWith(&PageParam{PageNum: 1, PageSize: 5}, map[string]interface{}{"name": "user_1%"})
	if err != nil {
		t.Errorf("selectPageWith failed: %v", err)
		return
	}
	if p.Total != 7 {
		t.Errorf("filtered total = %d, want 7 (user_1, user_10..user_15)", p.Total)
	}
	if len(p.Records) != 5 {
		t.Errorf("filtered records = %d, want 5", len(p.Records))
	}
}

func Test_M07_PageParamValidation(t *testing.T) {
	pp := &PageParam{PageNum: 0, PageSize: 10}
	if pp.Valid() {
		t.Error("PageNum=0 should be invalid")
	}
	pp = &PageParam{PageNum: 1, PageSize: 0}
	if pp.Valid() {
		t.Error("PageSize=0 should be invalid")
	}
	pp = &PageParam{PageNum: 1, PageSize: 10}
	if !pp.Valid() {
		t.Error("PageNum=1 PageSize=10 should be valid")
	}
	if pp.Offset() != 0 {
		t.Errorf("page 1 offset = %d, want 0", pp.Offset())
	}
	if pp.Limit() != 10 {
		t.Errorf("limit = %d, want 10", pp.Limit())
	}
	pp = &PageParam{PageNum: 3, PageSize: 10}
	if pp.Offset() != 20 {
		t.Errorf("page 3 offset = %d, want 20", pp.Offset())
	}
}

func Test_M07_PageParamNotExtractedAsSQLArg(t *testing.T) {
	pp := &PageParam{PageNum: 1, PageSize: 5}
	arg := NewProxyArg(nil, []reflect.Value{reflect.ValueOf(pp), reflect.ValueOf(map[string]interface{}{"x": 1})})
	if arg.PageParam != pp {
		t.Error("PageParam not extracted from args")
	}
	if arg.ArgsLen != 1 {
		t.Errorf("ArgsLen = %d, want 1 (PageParam should be filtered out)", arg.ArgsLen)
	}
}
