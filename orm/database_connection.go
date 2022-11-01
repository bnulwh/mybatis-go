package orm

import (
	"context"
	"database/sql"
	log "github.com/bnulwh/logrus"
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
}

func Open(cfg *Config) (db *DB, err error) {
	var dialector Dialector
	switch cfg.DriverName() {
	case "postgres":
		dialector = NewPostgresDialector(cfg)
	case "mysql":
		dialector = NewMySqlDialector(cfg)
	default:
		return nil, ErrInvalidDB
	}
	db = &DB{
		Config:    cfg,
		Error:     nil,
		Statement: &Statement{},
	}
	db.Statement.init()
	if dialector != nil {
		db.Dialector = dialector
	}
	if db.Dialector != nil {
		err = db.Dialector.Initialize(db)
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

	if sqldb, ok := db.ConnPool.(*sql.DB); ok {
		err := sqldb.Close()
		if err != nil {
			log.Errorf("close db error: %v", err)
		}
	}
}

func (db *DB) prepare(ctx context.Context, query string) (Stmt, error) {
	//if !dc.connected {
	//	dc.connect2Database()
	//}
	v, _ := db.cacheStore.Load(preparedStmtDBKey)
	preparedStmt := v.(*PreparedStmtDB)
	log.Debugf("conn stats: %#v", preparedStmt.Stats())
	return preparedStmt.prepare(ctx, db.ConnPool, query)
	//var err error
	//conn, err := dc.database.Conn(ctx)
	//if err != nil {
	//	log.Warnf("create conn failed. %v", err)
	//	return nil, nil, err
	//}
	//stmt, err := conn.PrepareContext(ctx, sqlStr)
	//return conn, stmt, err

	//err := dc.database.Ping()
	//if err != nil {
	//	log.Warnf("ping failed. %v", err)
	//}
	//return dc.database.Prepare(sqlStr)
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

func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	tx, err := db.DB()
	if err != nil {
		db.updateExecStatement(start, false)
		return nil, err
	}
	defer db.updateExecStatement(start, true)
	cur := time.Now()
	defer db.Statement.updateDBExecStatement(cur)
	return tx.ExecContext(ctx, query, args...)
}
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	tx, err := db.DB()
	if err != nil {
		db.updateQueryStatement(start, false)
		return nil, err
	}
	defer db.updateQueryStatement(start, true)
	cur := time.Now()
	defer db.Statement.updateDBQueryStatement(cur)
	return tx.QueryContext(ctx, query, args...)
}
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	tx, err := db.DB()
	if err != nil {
		db.updateQueryStatement(start, false)
		log.Errorf("get db failed: %v", err)
		return nil
	}
	defer db.updateQueryStatement(start, true)
	cur := time.Now()
	defer db.Statement.updateDBQueryStatement(cur)
	return tx.QueryRowContext(ctx, query, args...)
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
