package orm

import (
	"context"
	"testing"
	"time"
)

// P4-1：全局默认超时设置与取值
func Test_DefaultTimeout(t *testing.T) {
	old := DefaultTimeout()
	defer SetDefaultTimeout(old)

	// 默认 5 分钟
	if DefaultTimeout() != 5*time.Minute {
		t.Errorf("expected default timeout 5m, got %v", DefaultTimeout())
	}
	SetDefaultTimeout(7 * time.Second)
	if DefaultTimeout() != 7*time.Second {
		t.Errorf("expected 7s after SetDefaultTimeout, got %v", DefaultTimeout())
	}
	SetDefaultTimeout(0)
	if DefaultTimeout() != 0 {
		t.Errorf("expected 0 after disabling, got %v", DefaultTimeout())
	}
	SetDefaultTimeout(-1 * time.Second)
	if DefaultTimeout() != -1*time.Second {
		t.Errorf("expected -1s after negative set, got %v", DefaultTimeout())
	}
}

// P4-1：withExecTimeout 语义
func Test_withExecTimeout(t *testing.T) {
	old := DefaultTimeout()
	defer SetDefaultTimeout(old)
	SetDefaultTimeout(5 * time.Minute)

	// 无 deadline → 叠加默认超时（deadline 在 ~5 分钟后）
	bgCtx, bgCancel := withExecTimeout(context.Background())
	defer bgCancel()
	if d, ok := bgCtx.Deadline(); !ok {
		t.Error("expected deadline after withExecTimeout(background)")
	} else {
		until := time.Until(d)
		if until > 5*time.Minute+time.Second || until < 4*time.Minute+50*time.Second {
			t.Errorf("expected deadline ~5m from now, got until=%v", until)
		}
	}

	// 已有 deadline → 保持原样（不叠加默认超时）
	baseCtx, baseCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer baseCancel()
	wrapped, wrappedCancel := withExecTimeout(baseCtx)
	defer wrappedCancel()
	if wrapped != baseCtx {
		t.Error("expected original ctx returned when deadline exists")
	}

	// 全局超时为 0（关闭）→ 无 deadline
	SetDefaultTimeout(0)
	noDeadlineCtx, noDeadlineCancel := withExecTimeout(context.Background())
	defer noDeadlineCancel()
	if _, ok := noDeadlineCtx.Deadline(); ok {
		t.Error("expected no deadline when default timeout disabled (0)")
	}
}

// P4-1：带上下文的 Mapper 代理查询正常工作
func Test_ExecuteQueryContext(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	old := DefaultTimeout()
	defer SetDefaultTimeout(old)
	SetDefaultTimeout(2 * time.Second)

	if _, err := Execute(`CREATE TABLE t_p41 (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT
	)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}

	// ExecuteContext
	if _, err := ExecuteContext(context.Background(), `INSERT INTO t_p41 (name) VALUES (?)`, "hello"); err != nil {
		t.Errorf("execute context insert failed: %v", err)
		return
	}

	// QueryContext
	res, err := QueryContext(context.Background(), `SELECT id, name FROM t_p41`)
	if err != nil {
		t.Errorf("query context failed: %v", err)
		return
	}
	if len(res) != 1 {
		t.Errorf("expected 1 row, got %d", len(res))
		return
	}
	if res[0]["name"] != "hello" {
		t.Errorf("expected name 'hello', got %v", res[0]["name"])
	}

	// 普通 Execute/Query（通过默认超时路径）
	if _, err := Execute(`INSERT INTO t_p41 (name) VALUES (?)`, "world"); err != nil {
		t.Errorf("execute insert failed: %v", err)
		return
	}
	res2, err := Query(`SELECT name FROM t_p41 ORDER BY id`)
	if err != nil {
		t.Errorf("query failed: %v", err)
		return
	}
	if len(res2) != 2 || res2[0]["name"] != "hello" || res2[1]["name"] != "world" {
		t.Errorf("query result mismatch: %v", res2)
	}
}

// P4-1：已取消 context → 立即报错，不会挂起
func Test_ExecuteContextCanceled(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	// 已取消 → 立即报错
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ExecuteContext(ctx, `SELECT 1`); err == nil {
		t.Error("expected error with canceled context")
	}

	// 已过期 → 立即报错
	tctx, tcancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer tcancel()
	time.Sleep(5 * time.Millisecond) // 确保过期
	if _, err := QueryContext(tctx, `SELECT 1`); err == nil {
		t.Error("expected error with expired context")
	}
}

// P4-1：自定义超时的 context → 超时后报错（短超时 + 慢查询模拟）
func Test_ExecuteContextShortTimeout(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	old := DefaultTimeout()
	defer SetDefaultTimeout(old)
	SetDefaultTimeout(0) // 关闭默认超时，避免干扰

	// 创建大表
	if _, err := Execute(`CREATE TABLE t_short (id INTEGER PRIMARY KEY, val TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	// 插入数据
	for i := 0; i < 10; i++ {
		Execute(`INSERT INTO t_short (val) VALUES (?)`, "x")
	}

	// 用极短超时查询 + 大量重复子查询模拟慢 SQL
	shortCtx, shortCancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer shortCancel()
	time.Sleep(2 * time.Millisecond) // 确保过期

	if _, err := QueryContext(shortCtx, `SELECT * FROM t_short`); err == nil {
		t.Error("expected timeout error")
	}
}

// P4-1：超时 0（关闭）时 context.Background() 不设 deadline
func Test_ExecuteNoTimeout(t *testing.T) {
	old := DefaultTimeout()
	defer SetDefaultTimeout(old)
	SetDefaultTimeout(0)

	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	// 默认 0 → 无超时限制，正常执行
	if _, err := Execute(`CREATE TABLE t_no_timeout (id INTEGER PRIMARY KEY)`); err != nil {
		t.Errorf("create table failed: %v", err)
	}
}
