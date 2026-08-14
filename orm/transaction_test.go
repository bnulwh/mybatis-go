package orm

import (
	"fmt"
	"testing"
	"time"
)

func initTxTest(t *testing.T) bool {
	if initSqliteTest(t) == "" {
		return false
	}
	if _, err := Execute(`CREATE TABLE t_sqlite (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		create_time DATETIME)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return false
	}
	return true
}

func rowCount(t *testing.T) int64 {
	res, err := Query(`SELECT count(*) as c FROM t_sqlite`)
	if err != nil {
		t.Errorf("query count failed: %v", err)
		return -1
	}
	var n int64
	fmt.Sscanf(fmt.Sprintf("%v", res[0]["c"]), "%d", &n)
	return n
}

func Test_TransactionCommit(t *testing.T) {
	if !initTxTest(t) {
		return
	}
	defer Close()

	tx, err := Begin()
	if err != nil {
		t.Errorf("Begin failed: %v", err)
		return
	}
	if _, err := Execute(`INSERT INTO t_sqlite (name) VALUES (?)`, "in_tx"); err != nil {
		t.Errorf("insert in tx failed: %v", err)
		return
	}
	// 事务内可见
	if n := rowCount(t); n != 1 {
		t.Errorf("row count inside tx should be 1, got %d", n)
	}
	if err := tx.Commit(); err != nil {
		t.Errorf("Commit failed: %v", err)
		return
	}
	// 提交后仍可见
	if n := rowCount(t); n != 1 {
		t.Errorf("row count after commit should be 1, got %d", n)
	}
}

func Test_TransactionRollback(t *testing.T) {
	if !initTxTest(t) {
		return
	}
	defer Close()

	tx, err := Begin()
	if err != nil {
		t.Errorf("Begin failed: %v", err)
		return
	}
	if _, err := Execute(`INSERT INTO t_sqlite (name) VALUES (?)`, "rolled_back"); err != nil {
		t.Errorf("insert in tx failed: %v", err)
		return
	}
	if err := tx.Rollback(); err != nil {
		t.Errorf("Rollback failed: %v", err)
		return
	}
	// 回滚后不可见
	if n := rowCount(t); n != 0 {
		t.Errorf("row count after rollback should be 0, got %d", n)
	}
}

func Test_TransactionDoubleBegin(t *testing.T) {
	if !initTxTest(t) {
		return
	}
	defer Close()

	tx, err := Begin()
	if err != nil {
		t.Errorf("Begin failed: %v", err)
		return
	}
	defer tx.Rollback()
	if _, err := Begin(); err == nil {
		t.Error("second Begin should fail while a transaction is active")
	}
}

func Test_TransactionCommitIdempotent(t *testing.T) {
	if !initTxTest(t) {
		return
	}
	defer Close()

	tx, err := Begin()
	if err != nil {
		t.Errorf("Begin failed: %v", err)
		return
	}
	if err := tx.Commit(); err != nil {
		t.Errorf("first Commit failed: %v", err)
		return
	}
	// 重复 Commit 不应报错
	if err := tx.Commit(); err != nil {
		t.Errorf("second Commit should be no-op, got: %v", err)
	}
}

func Test_TransactionMapper(t *testing.T) {
	if !initTxTest(t) {
		return
	}
	defer Close()
	RegisterModel(new(SqliteTestModel))
	if err := RegisterMapper(new(SqliteTestMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("SqliteTestMapper").(SqliteTestMapper)
	tm := time.Date(2026, 8, 13, 17, 0, 0, 0, time.UTC)

	tx, err := Begin()
	if err != nil {
		t.Errorf("Begin failed: %v", err)
		return
	}
	if _, err := mp.Insert(SqliteTestModel{Name: "tx_mapper", CreateTime: tm}); err != nil {
		t.Errorf("mapper insert in tx failed: %v", err)
		return
	}
	// 事务内通过 Mapper 查询可见
	rs, err := mp.SelectAll()
	if err != nil {
		t.Errorf("mapper select in tx failed: %v", err)
		return
	}
	if len(rs) != 1 {
		t.Errorf("rows inside tx should be 1, got %d", len(rs))
		return
	}
	// 回滚后不可见
	if err := tx.Rollback(); err != nil {
		t.Errorf("Rollback failed: %v", err)
		return
	}
	rs, err = mp.SelectAll()
	if err != nil {
		t.Errorf("mapper select after rollback failed: %v", err)
		return
	}
	if len(rs) != 0 {
		t.Errorf("rows after rollback should be 0, got %d", len(rs))
	}
}

func Test_TransactionExecDirect(t *testing.T) {
	if !initTxTest(t) {
		return
	}
	defer Close()

	tx, err := Begin()
	if err != nil {
		t.Errorf("Begin failed: %v", err)
		return
	}
	if _, err := tx.Exec(`INSERT INTO t_sqlite (name) VALUES (?)`, "direct"); err != nil {
		t.Errorf("tx.Exec failed: %v", err)
		return
	}
	var name string
	if err := tx.QueryRow(`SELECT name FROM t_sqlite WHERE name = ?`, "direct").Scan(&name); err != nil {
		t.Errorf("tx.QueryRow failed: %v", err)
		return
	}
	if name != "direct" {
		t.Errorf("tx.QueryRow got %q, want direct", name)
	}
	if err := tx.Commit(); err != nil {
		t.Errorf("Commit failed: %v", err)
	}
}
