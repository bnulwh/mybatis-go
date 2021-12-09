package orm

import (
	"github.com/bnulwh/mybatis-go/utils"
)

func Initialize(filename string) {
	cm := utils.LoadSettings(filename)
	InitializeFromSettings(cm)
}

func InitializeFromSettings(cm map[string]string) {
	dc := NewConfigFromSettings(cm)
	gDbConn = newDatabaseConnection(dc.DbConfig)
	if gDbConn != nil {
		gDbConn.connect2Database()
	}
	gCache.initSqls(dc.MapperLocations)
}
func InitializeDatabase(dbType, host string, port int, user, pwd, dbName string) {
	dc := newDatabaseConfig(dbType, host, port, user, pwd, dbName)
	gDbConn = newDatabaseConnection(dc)
	if gDbConn != nil {
		gDbConn.connect2Database()
	}
}

func Close() {
	gDbConn.close()
	//gDone <- "done"
}
