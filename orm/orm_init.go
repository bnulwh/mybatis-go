package orm

import (
	"database/sql"
	log "github.com/bnulwh/logrus"
	"sync"
	"time"
)

var (
	gDbConn *sql.DB
	gLock   sync.Mutex
	gDone   chan interface{}
)

func Initialize(filename string) {
	cm := LoadSettings(filename)
	InitializeFromSettings(cm)
}

func InitializeFromSettings(cm map[string]string) {
	dc := NewConfigFromSettings(cm)
	driverName, connStr := dc.DbConfig.GenerateConn()
	var err error
	gDbConn, err = sql.Open(driverName, connStr)
	if err != nil {
		panic(err)
	}
	log.Infof("successfully connected!")
	gDbConn.SetConnMaxLifetime(time.Minute * 5)
	gDbConn.SetMaxIdleConns(10)
	gDbConn.SetMaxOpenConns(10)

	err = gDbConn.Ping()
	if err != nil {
		panic(err)
	}
	gCache.initSqls(dc.MapperLocations)
}
func Close() {
	if gDbConn != nil {
		err := gDbConn.Close()
		if err != nil {
			log.Errorf("close db error: %v", err)
		}
	}
	//gDone <- "done"
}
