package orm

import (
	"fmt"

	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/utils"
	"os"
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
		gDataSources.closeAll()
		gDataSources.reset()
		gDataSources.add(defaultDataSourceName, db)
	}
	return combineErrors(err1, err2)
}

// ReConnect 重连当前活跃数据源（兼容旧 API）。
func ReConnect() error {
	if gDbConn == nil {
		return fmt.Errorf("connection not init.")
	}
	return gDbConn.Reconnect()
}

// Reconnect 重连当前 DB 的连接池（预编译缓存重建、旧连接关闭、Ping 验证）。
func (db *DB) Reconnect() error {
	oldSQLDB, _ := db.DB()
	if err := db.Dialector.Initialize(db); err != nil {
		return err
	}
	// 预编译缓存绑定的是旧连接，重连后必须重置缓存并指向新连接池
	if v, ok := db.cacheStore.Load(preparedStmtDBKey); ok {
		preparedStmt := v.(*PreparedStmtDB)
		preparedStmt.Reset()
		preparedStmt.ConnPool = db.ConnPool
		if db.PreparedStmt {
			db.ConnPool = preparedStmt
		}
	}
	// 仅当底层连接确实被替换时才关闭旧连接，避免泄漏
	if newSQLDB, err := db.DB(); err == nil && oldSQLDB != nil && oldSQLDB != newSQLDB {
		_ = oldSQLDB.Close()
	}
	// 验证新连接可用
	if pinger, ok := db.ConnPool.(interface{ Ping() error }); ok {
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
	gDataSources.closeAll()
	gDataSources.reset()
	gDataSources.add(defaultDataSourceName, db)
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
	body, err := os.ReadFile(filename)
	if err != nil {
		log.Warnf("load file %v failed: %v", filename, err)
		return map[string]string{}
	}
	envMap := map[string]string{}
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)

		if len(line) == 0 || line[0] == '!' || line[0] == '#' {
			continue
		}
		pos := strings.Index(line, "=")
		if pos <= 0 {
			pos = strings.Index(line, ":")
		}
		if pos <= 0 {
			continue
		}
		key := strings.TrimSpace(line[0:pos])
		val := strings.Trim(line[pos+1:], "'\" ")
		envMap[key] = val
	}
	return envMap
}

func getRealValue(val string, em map[string]string) string {
	// 边界防护："${"、"${"等非法输入直接原样返回，避免 val[2:...] 越界 panic
	if !strings.HasPrefix(val, "${") || len(val) <= 3 {
		return val
	}
	inner := val[2 : len(val)-1]
	var key, def string
	if pos := strings.Index(inner, ":"); pos >= 0 {
		key = inner[:pos]
		def = inner[pos+1:]
	} else {
		key = inner
	}
	// 空 key（如 "${:}"、"${:default}"）视为非法，原样返回
	if key == "" {
		return val
	}
	if rv, ok := em[key]; ok {
		return rv
	}
	return def
}

func Close() {
	gDataSources.closeAll()
	gDataSources.reset()
	gDbConn = nil
}
