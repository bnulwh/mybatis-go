package orm

import (
	"context"
	"database/sql"
	log "github.com/bnulwh/logrus"
	"sync"
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
