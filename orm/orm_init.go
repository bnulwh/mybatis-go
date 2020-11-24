package orm

import (
	"database/sql"
	log "github.com/astaxie/beego/logs"
	"github.com/bnulwh/mybatis-go/types"
	"sync"
	"time"
)

var (
	gDbConn  *sql.DB
	gLock    sync.Mutex
	gDone    chan interface{}
	gMappers *types.SqlMappers
)

func Initialize(filename string) {
	dc := NewConfig(filename)
	driverName, connStr := dc.DbConfig.GenerateConn()
	var err error
	gDbConn, err = sql.Open(driverName, connStr)
	if err != nil {
		panic(err)
	}
	log.Info("successfully connected!")
	gDbConn.SetConnMaxLifetime(time.Minute * 5)
	gDbConn.SetMaxIdleConns(10)
	gDbConn.SetMaxOpenConns(10)

	err = gDbConn.Ping()
	if err != nil {
		panic(err)
	}
	gMappers = types.NewSqlMappers(dc.MapperLocations)
}

func Close() {
	if gDbConn != nil {
		gDbConn.Close()
	}
}
