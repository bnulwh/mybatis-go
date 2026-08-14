package orm

import (
	"fmt"

	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/utils"
	"io/ioutil"
	"strings"
)

func SetLogger(logger log.Logger) {
	log.SetLogger(logger)
}

func Initialize(filename string) error {
	cm := LoadSettings(filename)
	return InitializeFromSettings(cm)
}

func InitializeFromSettings(cm map[string]string) error {
	cfg := NewConfigFromSettings(cm)
	err1 := gCache.initSqls(cfg.Setting.MapperLocations)
	db, err2 := Open(cfg)
	if err2 == nil {
		gDbConn = db
	}
	return combineErrors(err1, err2)
}

func ReConnect() error {
	if gDbConn == nil {
		return fmt.Errorf("connection not init.")
	}
	oldSQLDB, _ := gDbConn.DB()
	if err := gDbConn.Dialector.Initialize(gDbConn); err != nil {
		return err
	}
	// 预编译缓存绑定的是旧连接，重连后必须重置缓存并指向新连接池
	if v, ok := gDbConn.cacheStore.Load(preparedStmtDBKey); ok {
		preparedStmt := v.(*PreparedStmtDB)
		preparedStmt.Reset()
		preparedStmt.ConnPool = gDbConn.ConnPool
		if gDbConn.PreparedStmt {
			gDbConn.ConnPool = preparedStmt
		}
	}
	// 仅当底层连接确实被替换时才关闭旧连接，避免泄漏
	if newSQLDB, err := gDbConn.DB(); err == nil && oldSQLDB != nil && oldSQLDB != newSQLDB {
		_ = oldSQLDB.Close()
	}
	// 验证新连接可用
	if pinger, ok := gDbConn.ConnPool.(interface{ Ping() error }); ok {
		return pinger.Ping()
	}
	return nil
}

func InitializeDatabase(dbType, host string, port int, user, pwd, dbName string) error {
	cfg := newDatabaseConfig(dbType, host, port, user, pwd, dbName)
	db, err := Open(cfg)
	if err != nil {
		return err
	}
	gDbConn = db
	return nil
}
func LoadSettings(filename string) map[string]string {
	m := LoadProperties(filename)
	em := utils.GetAllEnv()
	for k, v := range m {
		if strings.HasPrefix(v, "${") {
			v = getRealValue(v, em)
			m[k] = v
		}
	}
	for k, v := range em {
		m[k] = v
	}
	return m
}

func LoadProperties(filename string) map[string]string {
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Warnf("load file %v failed: %v", filename, err)
		return map[string]string{}
	}
	envMap := map[string]string{}
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)

		if len(line) == 0 || strings.Contains("!#", line[0:1]) {
			continue
		}
		pos := strings.Index(line, "=")
		if pos <= 0 {
			pos = strings.Index(line, ":")
		}
		if pos <= 0 {
			continue
		}
		key := line[0:pos]
		val := strings.Trim(line[pos+1:], "'\" ")
		envMap[key] = val
	}
	return envMap
}

func getRealValue(val string, em map[string]string) string {
	pos := strings.Index(val, ":")
	if pos < 0 {
		key := val[2 : len(val)-1]
		rv, ok := em[key]
		if ok {
			return rv
		}
		return ""
	}
	key := val[2:pos]
	rval := val[pos+1 : len(val)-1]
	rv, ok := em[key]
	if ok {
		return rv
	}
	return rval
}

func Close() {
	gDbConn.close()
	//gDone <- "done"
}
