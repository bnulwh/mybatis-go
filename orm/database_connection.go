package orm

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/orm/dialector"
	"sync"
	"time"
)

var (
	gDbConn *DB
)

const (
	preparedStmtDBKey = "preparedStmt"
)

type DB struct {
	*Config
	Error     error
	Statement *Statement
	txMu      sync.RWMutex
	curTx     *Transaction
}

func Open(cfg *Config) (db *DB, err error) {
	d, err := dialector.NewForType(cfg.Setting.Type, cfg)
	if err != nil {
		return nil, ErrInvalidDB
	}
	db = &DB{
		Config:    cfg,
		Error:     nil,
		Statement: &Statement{},
	}
	db.Statement.init()
	if d != nil {
		db.Dialector = d
	}
	if db.Dialector != nil {
		pool, initErr := db.Dialector.Initialize()
		if initErr != nil {
			return nil, initErr
		}
		db.ConnPool = pool
	}

	preparedStmt := &PreparedStmtDB{
		ConnPool:    db.ConnPool,
		Stmts:       map[string]*Stmt{},
		Mux:         &sync.RWMutex{},
		PreparedSQL: make([]string, 0, 100),
	}
	db.cacheStore.Store(preparedStmtDBKey, preparedStmt)

	if db.PreparedStmt {
		db.ConnPool = preparedStmt
	}
	if err == nil {
		if pinger, ok := db.ConnPool.(interface{ Ping() error }); ok {
			err = pinger.Ping()
		}
	}
	return
}

func (db *DB) close() {

	if v, ok := db.cacheStore.Load(preparedStmtDBKey); ok {
		preparedStmt := v.(*PreparedStmtDB)
		preparedStmt.Close()
	}

	if sqldb, err := db.DB(); err == nil && sqldb != nil {
		if err := sqldb.Close(); err != nil {
			log.Errorf("close db error: %v", err)
		}
	}
}

func (db *DB) formatSQL(query string, args []interface{}) string {
	if len(args) == 0 || db.Dialector == nil {
		return query
	}
	return db.Dialector.FormatPrepareSQL(query)
}

func (db *DB) DB() (*sql.DB, error) {
	connPool := db.ConnPool
	if dbConnector, ok := connPool.(GetDBConnector); ok {
		return dbConnector.GetDBConn()
	}
	if sqldb, ok := connPool.(*sql.DB); ok {
		return sqldb, nil
	}
	return nil, ErrInvalidDB
}

func (db *DB) currentTx() *Transaction {
	db.txMu.RLock()
	defer db.txMu.RUnlock()
	return db.curTx
}

func (db *DB) setCurTx(t *Transaction) error {
	db.txMu.Lock()
	defer db.txMu.Unlock()
	if db.curTx != nil {
		return fmt.Errorf("a transaction is already active, commit or rollback it first")
	}
	db.curTx = t
	return nil
}

func (db *DB) clearCurTx(t *Transaction) {
	db.txMu.Lock()
	defer db.txMu.Unlock()
	if db.curTx == t {
		db.curTx = nil
	}
}

func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	query = db.applyTablePrefix(query)
	query = db.formatSQL(query, args)
	if isDDLStatement(query) {
		defer db.invalidateTableNames()
		defer db.invalidateTableStructures()
	}
	// G0：context 携带的事务（WithTx）优先于 TCC 全局事务槽
	if t := TxFromContext(ctx); t != nil {
		defer db.updateExecStatement(start, true)
		cur := time.Now()
		defer db.Statement.updateDBExecStatement(cur)
		return t.tx.ExecContext(ctx, query, args...)
	}
	if t := db.currentTx(); t != nil {
		defer db.updateExecStatement(start, true)
		cur := time.Now()
		defer db.Statement.updateDBExecStatement(cur)
		return t.tx.ExecContext(ctx, query, args...)
	}
	if db.ConnPool == nil {
		db.updateExecStatement(start, false)
		return nil, ErrInvalidDB
	}
	defer db.updateExecStatement(start, true)
	cur := time.Now()
	defer db.Statement.updateDBExecStatement(cur)
	return db.ConnPool.ExecContext(ctx, query, args...)
}
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	query = db.applyTablePrefix(query)
	query = db.formatSQL(query, args)
	// G0：context 携带的事务（WithTx）优先于 TCC 全局事务槽
	if t := TxFromContext(ctx); t != nil {
		defer db.updateQueryStatement(start, true)
		cur := time.Now()
		defer db.Statement.updateDBQueryStatement(cur)
		return t.tx.QueryContext(ctx, query, args...)
	}
	if t := db.currentTx(); t != nil {
		defer db.updateQueryStatement(start, true)
		cur := time.Now()
		defer db.Statement.updateDBQueryStatement(cur)
		return t.tx.QueryContext(ctx, query, args...)
	}
	if db.ConnPool == nil {
		db.updateQueryStatement(start, false)
		return nil, ErrInvalidDB
	}
	defer db.updateQueryStatement(start, true)
	cur := time.Now()
	defer db.Statement.updateDBQueryStatement(cur)
	return db.ConnPool.QueryContext(ctx, query, args...)
}
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	query = db.applyTablePrefix(query)
	query = db.formatSQL(query, args)
	// G0：context 携带的事务（WithTx）优先于 TCC 全局事务槽
	if t := TxFromContext(ctx); t != nil {
		defer db.updateQueryStatement(start, true)
		cur := time.Now()
		defer db.Statement.updateDBQueryStatement(cur)
		return t.tx.QueryRowContext(ctx, query, args...)
	}
	if t := db.currentTx(); t != nil {
		defer db.updateQueryStatement(start, true)
		cur := time.Now()
		defer db.Statement.updateDBQueryStatement(cur)
		return t.tx.QueryRowContext(ctx, query, args...)
	}
	if db.ConnPool == nil {
		db.updateQueryStatement(start, false)
		log.Errorf("get db failed: %v", ErrInvalidDB)
		return nil
	}
	defer db.updateQueryStatement(start, true)
	cur := time.Now()
	defer db.Statement.updateDBQueryStatement(cur)
	return db.ConnPool.QueryRowContext(ctx, query, args...)
}
func (db *DB) Stats() sql.DBStats {
	tx, err := db.DB()
	if err != nil {
		log.Errorf("get db failed: %v", err)
		return sql.DBStats{}
	}
	return tx.Stats()
}

func (db *DB) updateExecStatement(tm time.Time, success bool) {
	db.Statement.updateExecStatement(tm, success)
}

func (db *DB) updateQueryStatement(tm time.Time, success bool) {
	db.Statement.updateQueryStatement(tm, success)
}
