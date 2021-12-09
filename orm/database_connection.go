package orm

import (
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
	dc.database.SetMaxIdleConns(10)
	dc.database.SetMaxOpenConns(10)
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
