package orm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type HookMapper struct {
	BaseMapper
	SelectAll  func() ([]map[string]interface{}, error)
	SelectPage func(page *PageParam) (*Page, error)
	SelectErr  func() ([]map[string]interface{}, error)
}

// initHookTest 初始化 SQLite 数据源（Hook 专用 mapper：全量查询 / 分页 / 故意失败的查询）。
func initHookTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="HookMapper">
  <select id="selectAll" resultType="map">
    select id, name from hook_user order by id
  </select>
  <select id="selectPage" resultType="map">
    select id, name from hook_user order by id
  </select>
  <select id="selectErr" resultType="map">
    select id from hook_no_such_table
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "HookMapper.xml"), []byte(xml), 0644); err != nil {
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
	return dir
}

func initHookUserData(t *testing.T) {
	t.Helper()
	if _, err := Execute(`CREATE TABLE hook_user (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	for i := 1; i <= 3; i++ {
		if _, err := Execute(`INSERT INTO hook_user (id, name) VALUES (?, ?)`, i, strings.Repeat("u", i)); err != nil {
			t.Errorf("insert failed: %v", err)
			return
		}
	}
}

// Test_RegisterHook 注册 / 执行 / 清理：nil 钩子忽略，ClearHooks 后不再触发。
func Test_RegisterHook(t *testing.T) {
	ClearHooks()
	defer ClearHooks()
	var beforeCnt, afterCnt int
	RegisterHook(HookBeforeExecute, func(hc *HookContext) { beforeCnt++ })
	RegisterHook(HookAfterExecute, func(hc *HookContext) { afterCnt++ })
	RegisterHook(HookBeforeExecute, nil) // nil 忽略
	sql := runBeforeHooks(context.Background(), "ns", "selectAll", "select", "select 1", nil)
	if sql != "select 1" {
		t.Error("before hooks should keep SQL unchanged when not modified, got", sql)
	}
	if beforeCnt != 1 {
		t.Errorf("before hook count = %d, want 1", beforeCnt)
	}
	runAfterHooks(context.Background(), "ns", "selectAll", "select", "select 1", nil, nil, time.Millisecond)
	if afterCnt != 1 {
		t.Errorf("after hook count = %d, want 1", afterCnt)
	}
	ClearHooks()
	runBeforeHooks(context.Background(), "ns", "selectAll", "select", "select 1", nil)
	runAfterHooks(context.Background(), "ns", "selectAll", "select", "select 1", nil, nil, 0)
	if beforeCnt != 1 || afterCnt != 1 {
		t.Error("hooks should not run after ClearHooks")
	}
}

// Test_SqliteBeforeHookRewriteSQL Before 钩子改写 SQL：注入 where id <= 2，
// 全量查询结果由 3 行变 2 行（执行器取改写后的 SQL）。
func Test_SqliteBeforeHookRewriteSQL(t *testing.T) {
	ClearHooks()
	defer ClearHooks()
	dir := initHookTest(t)
	if dir == "" {
		return
	}
	defer Close()
	initHookUserData(t)
	if err := RegisterMapper(new(HookMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("HookMapper").(HookMapper)

	// 改写前基线：3 行
	rows, err := mp.SelectAll()
	if err != nil {
		t.Error("select all failed:", err)
		return
	}
	if len(rows) != 3 {
		t.Errorf("baseline rows = %d, want 3", len(rows))
		return
	}

	RegisterHook(HookBeforeExecute, func(hc *HookContext) {
		if hc.SqlId == "selectAll" {
			hc.SQL = strings.Replace(hc.SQL, "order by id", "where id <= 2 order by id", 1)
		}
	})
	rows2, err := mp.SelectAll()
	if err != nil {
		t.Error("select all after rewrite failed:", err)
		return
	}
	if len(rows2) != 2 {
		t.Errorf("rewritten rows = %d, want 2", len(rows2))
		return
	}
	for _, r := range rows2 {
		if id, _ := r["id"].(int64); id > 2 {
			t.Errorf("rewritten result contains id = %v, want <= 2", r["id"])
		}
	}
}

// Test_SqliteAfterHookCapture After 钩子捕获 namespace/sqlId/sqlType/duration/error：
// 成功路径 Error 为 nil，失败路径（查询不存在的表）Error 传递。
func Test_SqliteAfterHookCapture(t *testing.T) {
	ClearHooks()
	defer ClearHooks()
	dir := initHookTest(t)
	if dir == "" {
		return
	}
	defer Close()
	initHookUserData(t)
	if err := RegisterMapper(new(HookMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("HookMapper").(HookMapper)

	var captured []*HookContext
	RegisterHook(HookAfterExecute, func(hc *HookContext) {
		captured = append(captured, hc)
	})

	if _, err := mp.SelectAll(); err != nil {
		t.Error("select all failed:", err)
		return
	}
	if _, err := mp.SelectErr(); err == nil {
		t.Error("selectErr should return error")
		return
	}
	if len(captured) != 2 {
		t.Fatalf("captured hooks = %d, want 2", len(captured))
	}
	okCtx := captured[0]
	if okCtx.Namespace != "HookMapper" || okCtx.SqlId != "selectAll" || okCtx.SqlType != "select" {
		t.Errorf("captured ctx = %v.%v type %v", okCtx.Namespace, okCtx.SqlId, okCtx.SqlType)
	}
	if okCtx.Error != nil {
		t.Error("successful execution should capture nil error, got", okCtx.Error)
	}
	if okCtx.Duration <= 0 {
		t.Error("captured duration should be positive")
	}
	errCtx := captured[1]
	if errCtx.SqlId != "selectErr" {
		t.Error("failed ctx SqlId =", errCtx.SqlId, "want selectErr")
	}
	if errCtx.Error == nil {
		t.Error("failed execution should capture error")
	}
}

// Test_HookPageConsistency selectPage 场景：Before 钩子注入的过滤条件
// 同时作用于 count（Total）与 records（分页查询）。
func Test_HookPageConsistency(t *testing.T) {
	ClearHooks()
	defer ClearHooks()
	dir := initHookTest(t)
	if dir == "" {
		return
	}
	defer Close()
	initHookUserData(t)
	if err := RegisterMapper(new(HookMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("HookMapper").(HookMapper)

	RegisterHook(HookBeforeExecute, func(hc *HookContext) {
		if hc.SqlId == "selectPage" {
			hc.SQL = strings.Replace(hc.SQL, "order by id", "where id <= 2 order by id", 1)
		}
	})
	page, err := mp.SelectPage(&PageParam{PageNum: 1, PageSize: 10})
	if err != nil {
		t.Error("select page failed:", err)
		return
	}
	if page.Total != 2 {
		t.Errorf("page total = %d, want 2 (count should be filtered by hook)", page.Total)
	}
	if len(page.Records) != 2 {
		t.Errorf("page records = %d, want 2 (records should be filtered by hook)", len(page.Records))
	}
}
