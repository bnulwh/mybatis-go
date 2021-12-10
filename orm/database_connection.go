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

type databaseConnection struct {
	database *sql.DB
	conn     *sql.Conn
	connStr  string
	driver   string
	dbName   string
	dbType   DatabaseType
	config   *DatabaseConfig
	lock     sync.Mutex
}

func newDatabaseConnection(dc *DatabaseConfig) *databaseConnection {

	return &databaseConnection{
		connStr: dc.generateConn(),
		driver:  dc.getDriver(),
		dbType:  dc.DbType,
		config:  dc,
		dbName:  dc.DbName,
	}
}

func (dc *databaseConnection) connect2Database() {
	var err error
	dc.database, err = sql.Open(dc.driver, dc.connStr)
	if err != nil {
		panic(err)
	}
	log.Infof("successfully connected!")
	dc.database.SetConnMaxLifetime(time.Minute * 5)
	dc.database.SetMaxIdleConns(100)
	dc.database.SetMaxOpenConns(100)
	err = dc.database.Ping()
	if err != nil {
		panic(err)
	}
}

func (dc *databaseConnection) close() {
	if dc.database != nil {
		err := dc.database.Close()
		if err != nil {
			log.Errorf("close db error: %v", err)
		}
	}
}

func (dc *databaseConnection) prepare(sqlStr string) (*sql.Stmt, error) {
	var err error
	dc.conn, err = dc.database.Conn(context.Background())
	if err != nil {
		log.Warnf("create conn failed.", err)
		return nil, err
	}
	return dc.conn.PrepareContext(context.Background(), sqlStr)

	//err := dc.database.Ping()
	//if err != nil {
	//	log.Warnf("ping failed. %v", err)
	//}
	//return dc.database.Prepare(sqlStr)
}
