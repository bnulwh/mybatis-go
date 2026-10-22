package orm

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// B3：语句级行数上限（<select maxRows="N">）端到端 + 截断可观测（LastQueryTruncated）。

type b3MaxRowsMapper struct {
	BaseMapper
	SelectAll       func() ([]map[string]interface{}, error)
	SelectLimited   func() ([]map[string]interface{}, error)
	SelectUnlimited func() ([]map[string]interface{}, error)
	BadMaxRows      func() ([]map[string]interface{}, error)
	StreamLimited   func() (*RowStream, error)
}

func initB3MaxRowsTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="b3MaxRowsMapper">
  <select id="selectAll" resultType="map">
    select id, name from t_b3 order by id
  </select>
  <select id="selectLimited" resultType="map" maxRows="3">
    select id, name from t_b3 order by id
  </select>
  <select id="selectUnlimited" resultType="map" maxRows="-1">
    select id, name from t_b3 order by id
  </select>
  <select id="badMaxRows" resultType="map" maxRows="abc">
    select id, name from t_b3 order by id
  </select>
  <select id="streamLimited" resultType="map" maxRows="4">
    select id, name from t_b3 order by id
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "b3MaxRowsMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	dbPath := filepath.Join(dir, "test.db")
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return ""
	}
	if _, err := Execute(`CREATE TABLE t_b3 (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return ""
	}
	for i := 0; i < 10; i++ {
		if _, err := Execute(`INSERT INTO t_b3 (name) VALUES (?)`, "x"); err != nil {
			t.Errorf("insert failed: %v", err)
			return ""
		}
	}
	if err := RegisterMapper(new(b3MaxRowsMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return ""
	}
	return dir
}

// Test_B3_StatementMaxRows_E2E 语句级 maxRows 覆盖全局上限（-1 不限、N 语句上限、
// 非法值回退全局），截断经 orm.LastQueryTruncated() 可观测。
func Test_B3_StatementMaxRows_E2E(t *testing.T) {
	dir := initB3MaxRowsTest(t)
	if dir == "" {
		return
	}
	defer Close()

	oldLimit := DefaultRowLimit()
	defer SetDefaultRowLimit(oldLimit)
	SetRowLimitStrict(false)
	defer SetRowLimitStrict(false)
	SetDefaultRowLimit(5)
	ResetQueryTruncation()
	defer ResetQueryTruncation()

	mp := NewMapper("b3MaxRowsMapper").(b3MaxRowsMapper)

	// 无 maxRows：全局上限 5，截断可观测
	rows, err := mp.SelectAll()
	if err != nil {
		t.Errorf("selectAll failed: %v", err)
		return
	}
	if len(rows) != 5 {
		t.Errorf("selectAll want 5 rows (global limit), got %d", len(rows))
	}
	tr := LastQueryTruncated()
	if tr == nil {
		t.Error("LastQueryTruncated should record truncation on selectAll")
	} else if tr.Limit != 5 || tr.Statement == "" {
		t.Errorf("truncation record = %+v, want limit=5 with statement", tr)
	}

	// maxRows="3"：语句级上限优先于全局
	ResetQueryTruncation()
	rows, err = mp.SelectLimited()
	if err != nil {
		t.Errorf("selectLimited failed: %v", err)
		return
	}
	if len(rows) != 3 {
		t.Errorf("selectLimited want 3 rows (statement maxRows), got %d", len(rows))
	}
	if tr = LastQueryTruncated(); tr == nil || tr.Limit != 3 {
		t.Errorf("selectLimited truncation record = %+v, want limit=3", tr)
	}

	// maxRows="-1"：不限制，全部返回且无截断记录
	ResetQueryTruncation()
	rows, err = mp.SelectUnlimited()
	if err != nil {
		t.Errorf("selectUnlimited failed: %v", err)
		return
	}
	if len(rows) != 10 {
		t.Errorf("selectUnlimited want 10 rows (unlimited), got %d", len(rows))
	}
	if tr = LastQueryTruncated(); tr != nil {
		t.Errorf("selectUnlimited should not truncate, got record %+v", tr)
	}

	// maxRows="abc"：非法值回退全局上限
	rows, err = mp.BadMaxRows()
	if err != nil {
		t.Errorf("badMaxRows failed: %v", err)
		return
	}
	if len(rows) != 5 {
		t.Errorf("badMaxRows want 5 rows (fallback to global), got %d", len(rows))
	}

	// 流式路径：stream 打开时快照语句级上限
	ResetQueryTruncation()
	st, err := mp.StreamLimited()
	if err != nil {
		t.Errorf("streamLimited failed: %v", err)
		return
	}
	for st.Next() {
	}
	if st.Count() != 4 {
		t.Errorf("streamLimited want 4 rows (statement maxRows), got %d", st.Count())
	}
	if tr = LastQueryTruncated(); tr == nil || tr.Limit != 4 {
		t.Errorf("streamLimited truncation record = %+v, want limit=4", tr)
	}
	st.Close()
}

// Test_B3_StatementMaxRows_Strict 严格模式叠加语句级上限：
// 语句级超限返回 ErrRowLimitExceeded；maxRows=-1 语句在严格模式下不报错。
func Test_B3_StatementMaxRows_Strict(t *testing.T) {
	dir := initB3MaxRowsTest(t)
	if dir == "" {
		return
	}
	defer Close()

	oldLimit := DefaultRowLimit()
	defer SetDefaultRowLimit(oldLimit)
	SetDefaultRowLimit(5)
	SetRowLimitStrict(true)
	defer SetRowLimitStrict(false)
	ResetQueryTruncation()
	defer ResetQueryTruncation()

	mp := NewMapper("b3MaxRowsMapper").(b3MaxRowsMapper)

	_, err := mp.SelectLimited()
	if !errors.Is(err, ErrRowLimitExceeded) {
		t.Errorf("selectLimited with strict want ErrRowLimitExceeded, got %v", err)
	}
	if tr := LastQueryTruncated(); tr == nil || tr.Limit != 3 {
		t.Errorf("strict truncation record = %+v, want limit=3", tr)
	}

	rows, err := mp.SelectUnlimited()
	if err != nil {
		t.Errorf("selectUnlimited with strict failed: %v", err)
		return
	}
	if len(rows) != 10 {
		t.Errorf("selectUnlimited with strict want 10 rows, got %d", len(rows))
	}
}
