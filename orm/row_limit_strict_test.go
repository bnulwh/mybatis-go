package orm

import (
	"context"
	"errors"
	"testing"
)

// B3：严格行数上限模式 —— Query 达到上限返回 ErrRowLimitExceeded（errors.Is 可判），
// 关闭严格模式时保持旧的静默截断行为。
func Test_QueryRowLimitStrict(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	oldLimit := DefaultRowLimit()
	defer SetDefaultRowLimit(oldLimit)
	SetRowLimitStrict(false)
	defer SetRowLimitStrict(false)

	if _, err := Execute(`CREATE TABLE t_strict (id INTEGER PRIMARY KEY AUTOINCREMENT, val TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	for i := 0; i < 25; i++ {
		if _, err := Execute(`INSERT INTO t_strict (val) VALUES (?)`, "x"); err != nil {
			t.Errorf("insert failed: %v", err)
			return
		}
	}

	// 非严格：静默截断为 10 行，不报错
	SetDefaultRowLimit(10)
	res, err := Query(`SELECT id, val FROM t_strict`)
	if err != nil {
		t.Errorf("query without strict failed: %v", err)
		return
	}
	if len(res) != 10 {
		t.Errorf("expected 10 rows without strict, got %d", len(res))
	}

	// 严格：达到上限返回 ErrRowLimitExceeded
	SetRowLimitStrict(true)
	_, err = Query(`SELECT id, val FROM t_strict`)
	if !errors.Is(err, ErrRowLimitExceeded) {
		t.Errorf("expected ErrRowLimitExceeded with strict, got %v", err)
	}

	// 严格但行数低于上限：正常返回，不报错
	SetDefaultRowLimit(30)
	res, err = Query(`SELECT id, val FROM t_strict`)
	if err != nil {
		t.Errorf("query below limit with strict failed: %v", err)
		return
	}
	if len(res) != 25 {
		t.Errorf("expected 25 rows below limit with strict, got %d", len(res))
	}

	// 严格 + 负数上限（不限）：全部返回
	SetDefaultRowLimit(-1)
	res, err = Query(`SELECT id, val FROM t_strict`)
	if err != nil {
		t.Errorf("query with negative limit and strict failed: %v", err)
		return
	}
	if len(res) != 25 {
		t.Errorf("expected 25 rows with negative limit, got %d", len(res))
	}
}

// B3：QueryTo（struct 切片扫描路径）同样遵循严格模式。
func Test_QueryToRowLimitStrict(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	oldLimit := DefaultRowLimit()
	defer SetDefaultRowLimit(oldLimit)
	SetRowLimitStrict(false)
	defer SetRowLimitStrict(false)

	if _, err := Execute(`CREATE TABLE t_strict2 (id INTEGER PRIMARY KEY AUTOINCREMENT, val TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	for i := 0; i < 15; i++ {
		if _, err := Execute(`INSERT INTO t_strict2 (val) VALUES (?)`, "x"); err != nil {
			t.Errorf("insert failed: %v", err)
			return
		}
	}

	type row struct {
		Id  int64  `column:"id"`
		Val string `column:"val"`
	}
	var rows []row
	SetDefaultRowLimit(10)
	SetRowLimitStrict(true)
	err := QueryTo(&rows, `SELECT id, val FROM t_strict2`)
	if !errors.Is(err, ErrRowLimitExceeded) {
		t.Errorf("QueryTo expected ErrRowLimitExceeded with strict, got %v", err)
	}
}

// B3：RowStream 严格模式 —— 打开时快照 limit 与 strict，超限时 Err() 返回
// ErrRowLimitExceeded；非严格模式 Err() 为 nil（仅计数截断）。
func Test_QueryStreamRowLimitStrict(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	oldLimit := DefaultRowLimit()
	defer SetDefaultRowLimit(oldLimit)
	SetRowLimitStrict(false)
	defer SetRowLimitStrict(false)

	if _, err := Execute(`CREATE TABLE t_strict3 (id INTEGER PRIMARY KEY AUTOINCREMENT, val TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	for i := 0; i < 12; i++ {
		if _, err := Execute(`INSERT INTO t_strict3 (val) VALUES (?)`, "x"); err != nil {
			t.Errorf("insert failed: %v", err)
			return
		}
	}

	// 严格：达到上限停止，Err() 报 ErrRowLimitExceeded
	SetDefaultRowLimit(5)
	SetRowLimitStrict(true)
	st, err := QueryStream(context.Background(), `SELECT id, val FROM t_strict3`)
	if err != nil {
		t.Errorf("QueryStream failed: %v", err)
		return
	}
	for st.Next() {
	}
	if st.Count() != 5 {
		t.Errorf("expected 5 rows before limit, got %d", st.Count())
	}
	if !errors.Is(st.Err(), ErrRowLimitExceeded) {
		t.Errorf("stream Err() expected ErrRowLimitExceeded with strict, got %v", st.Err())
	}
	st.Close()

	// 非严格：同样截断 5 行，但 Err() 为 nil
	SetRowLimitStrict(false)
	st2, err := QueryStream(context.Background(), `SELECT id, val FROM t_strict3`)
	if err != nil {
		t.Errorf("QueryStream failed: %v", err)
		return
	}
	for st2.Next() {
	}
	if st2.Count() != 5 {
		t.Errorf("expected 5 rows without strict, got %d", st2.Count())
	}
	if st2.Err() != nil {
		t.Errorf("stream Err() expected nil without strict, got %v", st2.Err())
	}
	st2.Close()
}

// B3：mybatis.max-rows / mybatis.configuration.row-limit-strict 配置解析。
func Test_ParseRowLimitSettings(t *testing.T) {
	cm := map[string]string{
		"spring.datasource.url":                  "jdbc:sqlite:test.db",
		"mybatis.max-rows":                       "7",
		"mybatis.configuration.row-limit-strict": "true",
	}
	cfg := NewConfigFromSettings(cm)
	if cfg.Setting.MaxRows != 7 {
		t.Errorf("mybatis.max-rows = %d, want 7", cfg.Setting.MaxRows)
	}
	if !cfg.Setting.RowLimitStrict {
		t.Error("mybatis.configuration.row-limit-strict should be true")
	}

	// 缺省：MaxRows 0（未设置）、RowLimitStrict false
	cm2 := map[string]string{
		"spring.datasource.url": "jdbc:sqlite:test.db",
	}
	cfg2 := NewConfigFromSettings(cm2)
	if cfg2.Setting.MaxRows != 0 {
		t.Errorf("default mybatis.max-rows = %d, want 0", cfg2.Setting.MaxRows)
	}
	if cfg2.Setting.RowLimitStrict {
		t.Error("default row-limit-strict should be false")
	}
}
