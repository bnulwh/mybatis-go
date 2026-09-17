package orm

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/bnulwh/mybatis-go/log"
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

// beginDetachedTx 开启不占用全局事务槽的轻量事务（G0）：
// 生命周期完全由调用方（WithTx / TxFromContext）经 context 管理，
// 与 TCC API（Begin/Commit/Rollback 全局槽）并行不悖。
func (db *DB) beginDetachedTx(ctx context.Context, opts *sql.TxOptions) (*Transaction, error) {
	sqldb, err := db.DB()
	if err != nil {
		return nil, err
	}
	tx, err := sqldb.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx: tx, db: db}, nil
}

// txContextKey context 携带事务的私有键类型（G0）。
type txContextKey struct{}

// TxFromContext 提取 context 中携带的事务（WithTx 注入）；无事务返回 nil。
func TxFromContext(ctx context.Context) *Transaction {
	if ctx == nil {
		return nil
	}
	tx, _ := ctx.Value(txContextKey{}).(*Transaction)
	return tx
}

// WithTx 回调式事务（G0）：在全局连接上开启独立事务并注入 ctx 后执行 fn。
//
//   - fn 返回 nil → Commit；返回 error → Rollback 并原样返回该 error；
//   - fn panic → Rollback 后重新抛出；
//   - ctx 已携带事务（嵌套调用）→ 直接复用外层事务，内层不再独立提交/回滚
//     （事务边界由最外层 WithTx 决定）；
//   - 事务不占用 TCC 全局事务槽，多个 goroutine 可各自 WithTx 互不冲突；
//   - fn 内通过 orm.ExecuteContext / QueryContext / QueryToContext，或声明
//     context.Context 参数的 Mapper 方法自动加入本事务。
func WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if gDbConn == nil {
		return fmt.Errorf("connection not init.")
	}
	return gDbConn.WithTx(ctx, fn)
}

// WithTx 与包级 WithTx 相同，作用于指定 DB 实例（多数据源场景）。
func (db *DB) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if fn == nil {
		return fmt.Errorf("WithTx: fn cannot be nil")
	}
	if tx := TxFromContext(ctx); tx != nil {
		return fn(ctx)
	}
	tx, err := db.beginDetachedTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	txCtx := context.WithValue(ctx, txContextKey{}, tx)
	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Warnf("WithTx rollback failed: %v", rbErr)
		}
		return err
	}
	return tx.Commit()
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
	query = t.db.applyTablePrefix(query)
	query = t.db.formatSQL(query, args)
	return t.tx.ExecContext(ctx, query, args...)
}

func (t *Transaction) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	query = t.db.applyTablePrefix(query)
	query = t.db.formatSQL(query, args)
	return t.tx.QueryContext(ctx, query, args...)
}

func (t *Transaction) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	query = t.db.applyTablePrefix(query)
	query = t.db.formatSQL(query, args)
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
