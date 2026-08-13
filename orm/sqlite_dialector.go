package orm

import (
	"database/sql"
	"strings"
	"time"
)

type SqliteConfig struct {
	Config
	DriverName string
	DSN        string
	Conn       ConnPool
}

type SqliteDialector struct {
	*SqliteConfig
}

func NewSqliteDialector(cfg *Config) *SqliteDialector {
	return &SqliteDialector{
		SqliteConfig: &SqliteConfig{
			Config:     *cfg,
			DriverName: cfg.DriverName(),
			DSN:        cfg.GenerateDSN(),
		},
	}
}

func (dialector *SqliteDialector) Name() string {
	return "sqlite"
}

func (dialector *SqliteDialector) Initialize(db *DB) (err error) {
	if dialector.DriverName == "" {
		dialector.DriverName = "sqlite"
	}
	if dialector.DSN == "" {
		dialector.DSN = dialector.GenerateDSN()
	}
	if dialector.Conn != nil {
		db.ConnPool = dialector.Conn
	} else {
		sqldb, err := sql.Open(dialector.DriverName, dialector.DSN)
		if err != nil {
			return err
		}
		timeout := int(time.Second) * dialector.MaxTimeout
		sqldb.SetConnMaxLifetime(time.Duration(timeout))
		// SQLite 单写者模型，空闲连接保持 1 避免文件锁竞争
		sqldb.SetMaxIdleConns(1)
		sqldb.SetMaxOpenConns(dialector.MaxOpen)
		db.ConnPool = sqldb
	}
	return nil
}

func (dialector *SqliteDialector) FormatPrepareSQL(src string) string {
	src = strings.ReplaceAll(src, "\r", " ")
	src = strings.ReplaceAll(src, "\n", " ")
	src = strings.ReplaceAll(src, "\t", " ")
	return src
}
