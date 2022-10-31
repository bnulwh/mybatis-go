package orm

import (
	"fmt"
	log "github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/utils"
	"io/ioutil"
	"strings"
)

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
	if gDbConn != nil {
		return gDbConn.Dialector.Initialize(gDbConn)
	}
	return fmt.Errorf("connection not init.")
}

func InitializeDatabase(dbType, host string, port int, user, pwd, dbName string) error {
	cfg := newDatabaseConfig(dbType, host, port, user, pwd, dbName)
	db, err2 := Open(cfg)
	if err2 == nil {
		gDbConn = db
	}
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
