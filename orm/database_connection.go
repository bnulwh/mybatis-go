package orm

import (
	"context"
	"database/sql"
	log "github.com/bnulwh/logrus"
	"sync"
	"time"
)

var (
	gDbConn *databaseConnection
)

const (
	preparedStmtDBKey = "preparedStmt"
)

type databaseConnection struct {
	ConnPool   ConnPool
	connStr    string
	driver     string
	dbName     string
	dbType     DatabaseType
	config     *DatabaseConfig
	connected  bool
	cacheStore *sync.Map
	//lock     sync.Mutex
}

func newDatabaseConnection(dc *DatabaseConfig) *databaseConnection {

	return &databaseConnection{
		connStr:    dc.generateConn(),
		driver:     dc.getDriver(),
		dbType:     dc.DbType,
		config:     dc,
		dbName:     dc.DbName,
		connected:  false,
		cacheStore: &sync.Map{},
	}
}

func (dc *databaseConnection) connect2Database() error {
	if dc.connected {
		return nil
	}
	var err error
	sqldb, err := sql.Open(dc.driver, dc.connStr)
	if err != nil {
		return err
	}
	log.Infof("successfully connected! config: %#v", *dc.config)
	timeout := int(time.Second) * dc.config.MaxTimeout

	sqldb.SetConnMaxLifetime(time.Duration(timeout))
	sqldb.SetMaxIdleConns(dc.config.MaxIdle)
	sqldb.SetMaxOpenConns(dc.config.MaxOpen)
	err = sqldb.Ping()
	if err != nil {
		log.Errorf("ping error : %v", err)
		return err
	}
	dc.ConnPool = sqldb
	preparedStmt := &PreparedStmtDB{
		ConnPool:    dc.ConnPool,
		Stmts:       map[string]*Stmt{},
		Mux:         &sync.RWMutex{},
		PreparedSQL: make([]string, 0, 100),
	}
	dc.cacheStore.Store(preparedStmtDBKey, preparedStmt)
	dc.connected = true
	return nil
}

func (dc *databaseConnection) close() {
	if dc.connected {
		if v, ok := dc.cacheStore.Load(preparedStmtDBKey); ok {
			preparedStmt := v.(*PreparedStmtDB)
			preparedStmt.Close()
		}

		if sqldb, ok := dc.ConnPool.(*sql.DB); ok {
			err := sqldb.Close()
			if err != nil {
				log.Errorf("close db error: %v", err)
			}
		}
	}
}

func (dc *databaseConnection) prepare(ctx context.Context, query string) (Stmt, error) {
	//if !dc.connected {
	//	dc.connect2Database()
	//}
	v, _ := dc.cacheStore.Load(preparedStmtDBKey)
	preparedStmt := v.(*PreparedStmtDB)
	log.Debugf("conn stats: %#v", preparedStmt.Stats())
	return preparedStmt.prepare(ctx, dc.ConnPool, query)
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
