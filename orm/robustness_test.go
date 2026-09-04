package orm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

// P1-1：全局缓存 map 加锁后的并发冒烟测试（无死锁/panic）

func Test_ModelCacheConcurrent(t *testing.T) {
	mc := modelCache{Models: map[string]reflect.Type{}}
	type ts struct {
		Name string
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mc.registerModel(new(ts))
			if _, err := mc.createModel("ts"); err != nil {
				t.Errorf("createModel failed: %v", err)
			}
		}()
	}
	wg.Wait()
	if _, ok := mc.Models["ts"]; !ok {
		t.Error("model should be registered after concurrent access")
	}
}

type capMapper struct {
	BaseMapper
	SelectAll func() ([]int, error)
}

func Test_MapperCacheConcurrent(t *testing.T) {
	mc := mapperCache{Mappers: map[string]*mapperInfo{}}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mc.registerMapper(new(capMapper))
			if _, err := mc.createMapper("capMapper"); err != nil {
				t.Errorf("createMapper failed: %v", err)
			}
		}()
	}
	wg.Wait()
	if _, ok := mc.Mappers["capMapper"]; !ok {
		t.Error("mapper should be registered after concurrent access")
	}
}

// P1-2：预编译缓存超过上限后降级为直接执行且不再增长

func Test_PreparedStmtCacheCap(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	if _, err := Execute(`CREATE TABLE t_cap (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	raw, err := gDbConn.DB()
	if err != nil {
		t.Errorf("get raw db failed: %v", err)
		return
	}
	ps := &PreparedStmtDB{
		ConnPool:    raw,
		Stmts:       map[string]*Stmt{},
		Mux:         &sync.RWMutex{},
		PreparedSQL: []string{},
	}
	// 预填充到上限
	for i := 0; i < maxPreparedStmts; i++ {
		ps.Stmts[fmt.Sprintf("sql_%d", i)] = &Stmt{prepared: make(chan struct{})}
	}
	if !ps.cacheFull() {
		t.Fatal("cache should report full at the cap")
	}
	ctx := context.Background()
	// 超过上限的参数化 SQL 应降级为直接执行（不 panic、不新增缓存条目）
	if _, err := ps.ExecContext(ctx, `INSERT INTO t_cap (name) VALUES (?)`, "over_cap"); err != nil {
		t.Errorf("exec over cap failed: %v", err)
		return
	}
	if len(ps.Stmts) != maxPreparedStmts {
		t.Errorf("cache should stay at cap %d, got %d", maxPreparedStmts, len(ps.Stmts))
	}
	rows, err := ps.QueryContext(ctx, `SELECT count(*) FROM t_cap`)
	if err != nil {
		t.Errorf("query over cap failed: %v", err)
		return
	}
	rows.Close()
	if len(ps.Stmts) != maxPreparedStmts {
		t.Errorf("cache should stay at cap after query, got %d", len(ps.Stmts))
	}
	// 未满时仍走缓存
	ps2 := &PreparedStmtDB{
		ConnPool:    raw,
		Stmts:       map[string]*Stmt{},
		Mux:         &sync.RWMutex{},
		PreparedSQL: []string{},
	}
	if _, err := ps2.ExecContext(ctx, `INSERT INTO t_cap (name) VALUES (?)`, "cached"); err != nil {
		t.Errorf("exec under cap failed: %v", err)
		return
	}
	if len(ps2.Stmts) != 1 {
		t.Errorf("cache should grow under cap, got %d", len(ps2.Stmts))
	}
}

// ---------------------------------------------------------------------------
// 5.1 宽松注册模式

type LaxMapper struct {
	BaseMapper
	SelectGood func() ([]map[string]interface{}, error)
	SelectBad  func() ([]map[string]interface{}, error)
}

func initLaxSqlite(t *testing.T) string {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	// XML 只含 selectGood；SelectBad 无对应语句
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="LaxMapper">
  <select id="selectGood" resultType="map">select 1 as g</select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "LaxMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	dbPath := filepath.Join(dir, "lax.db")
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

func Test_SetStrictRegister_Lax(t *testing.T) {
	dir := initLaxSqlite(t)
	if dir == "" {
		return
	}
	defer Close()

	// 严格（默认）：任一函数失败 → 整体失败
	if err := RegisterMapper(new(LaxMapper)); err == nil {
		t.Error("strict register should fail when a function has no sql statement")
	}
	// 宽松：失败函数跳过，其余正常注册
	SetStrictRegister(false)
	defer SetStrictRegister(true)
	if err := RegisterMapper(new(LaxMapper)); err != nil {
		t.Errorf("lax register should succeed, got: %v", err)
	}
	mp := NewMapper("LaxMapper").(LaxMapper)
	// 好函数可用
	rs, err := mp.SelectGood()
	if err != nil {
		t.Errorf("good function should still work in lax mode: %v", err)
	} else if len(rs) != 1 || rs[0]["g"] != int64(1) {
		t.Errorf("good function result wrong: %v", rs)
	}
	// 坏函数返回明确错误而非 panic（bindMapper 容错：fetchSqlFunction 失败 → error 代理）
	_, err = mp.SelectBad()
	if err == nil {
		t.Error("bad function should return error in lax mode (not panic)")
	}
}
