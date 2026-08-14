package orm

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

// Transaction 封装 *sql.Tx，提供 Commit/Rollback 与直接的 SQL 执行方法。
// 事务通过 Begin 绑定到 DB 的当前事务槽：开启后，Mapper 代理方法、
// orm.Execute / orm.Query 都会自动在事务内执行，直到 Commit 或 Rollback。
type Transaction struct {
	tx   *sql.Tx
	db   *DB
	once sync.Once
}

// Begin 在全局连接上开启事务。
// 注意：事务绑定全局连接（单事务槽），请勿在多个 goroutine 中交错开启事务。
func Begin() (*Transaction, error) {
	if gDbConn == nil {
		return nil, fmt.Errorf("connection not init.")
	}
	return gDbConn.Begin()
}

// BeginTx 与 Begin 相同，但支持传入 context 与事务选项。
func BeginTx(ctx context.Context, opts *sql.TxOptions) (*Transaction, error) {
	if gDbConn == nil {
		return nil, fmt.Errorf("connection not init.")
	}
	return gDbConn.BeginTx(ctx, opts)
}

func (db *DB) Begin() (*Transaction, error) {
	return db.BeginTx(context.Background(), nil)
}

func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Transaction, error) {
	sqldb, err := db.DB()
	if err != nil {
		return nil, err
	}
	tx, err := sqldb.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	t := &Transaction{tx: tx, db: db}
	if err := db.setCurTx(t); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return t, nil
}

// Commit 提交事务。重复调用（含与 Rollback 交错）只生效一次；
// 无论成败都会释放当前事务槽，避免失效事务继续劫持后续 SQL。
func (t *Transaction) Commit() error {
	var err error
	t.once.Do(func() {
		err = t.tx.Commit()
	})
	t.db.clearCurTx(t)
	return err
}

// Rollback 回滚事务。重复调用（含与 Commit 交错）只生效一次；
// 无论成败都会释放当前事务槽。
func (t *Transaction) Rollback() error {
	var err error
	t.once.Do(func() {
		err = t.tx.Rollback()
	})
	t.db.clearCurTx(t)
	return err
}

func (t *Transaction) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

func (t *Transaction) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

func (t *Transaction) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}

// 便捷方法：自动使用 context.Background()

func (t *Transaction) Exec(query string, args ...interface{}) (sql.Result, error) {
	return t.ExecContext(context.Background(), query, args...)
}

func (t *Transaction) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return t.QueryContext(context.Background(), query, args...)
}

func (t *Transaction) QueryRow(query string, args ...interface{}) *sql.Row {
	return t.QueryRowContext(context.Background(), query, args...)
}
