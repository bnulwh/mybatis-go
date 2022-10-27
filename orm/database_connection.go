package orm

import (
	"context"
	"database/sql"
	log "github.com/bnulwh/logrus"
	"time"
)

var (
	gDbConn *databaseConnection
)

type databaseConnection struct {
	database  *sql.DB
	conn      *sql.Conn
	connStr   string
	driver    string
	dbName    string
	dbType    DatabaseType
	config    *DatabaseConfig
	connected bool
	//lock     sync.Mutex
}

func newDatabaseConnection(dc *DatabaseConfig) *databaseConnection {

	return &databaseConnection{
		connStr:   dc.generateConn(),
		driver:    dc.getDriver(),
		dbType:    dc.DbType,
		config:    dc,
		dbName:    dc.DbName,
		database:  nil,
		conn:      nil,
		connected: false,
	}
}

func (dc *databaseConnection) connect2Database() error {
	if dc.connected {
		return nil
	}
	var err error
	dc.database, err = sql.Open(dc.driver, dc.connStr)
	if err != nil {
		return err
	}
	log.Infof("successfully connected!")
	dc.database.SetConnMaxLifetime(time.Minute * 5)
	dc.database.SetMaxIdleConns(100)
	dc.database.SetMaxOpenConns(100)
	err = dc.database.Ping()
	if err != nil {
		return err
	}
	dc.connected = true
	return nil
}

func (dc *databaseConnection) close() {
	if dc.database != nil {
		err := dc.database.Close()
		if err != nil {
			log.Errorf("close db error: %v", err)
		}
	}
}

func (dc *databaseConnection) prepare(ctx context.Context, sqlStr string) (*sql.Conn, *sql.Stmt, error) {
	//var err error
	conn, err := dc.database.Conn(ctx)
	if err != nil {
		log.Warnf("create conn failed. %v", err)
		return nil, nil, err
	}
	stmt, err := conn.PrepareContext(ctx, sqlStr)
	return conn, stmt, err

	//err := dc.database.Ping()
	//if err != nil {
	//	log.Warnf("ping failed. %v", err)
	//}
	//return dc.database.Prepare(sqlStr)
}
