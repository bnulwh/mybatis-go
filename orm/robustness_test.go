package orm

import (
	"context"
	"fmt"
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
