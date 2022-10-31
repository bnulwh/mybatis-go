package orm

import (
	"database/sql"
	"strings"
	"time"
)

type MySqlConfig struct {
	Config
	DriverName string
	DSN        string
	Conn       ConnPool
}

type MySqlDialector struct {
	*MySqlConfig
}

func NewMySqlDialector(cfg *Config) *MySqlDialector {
	return &MySqlDialector{
		MySqlConfig: &MySqlConfig{
			Config:     *cfg,
			DriverName: cfg.DriverName(),
			DSN:        cfg.GenerateDSN(),
		},
	}
}

func (dialector *MySqlDialector) Name() string {
	return "mysql"
}

func (dialector *MySqlDialector) Initialize(db *DB) (err error) {
	if dialector.DriverName == "" {
		dialector.DriverName = "mysql"
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
		sqldb.SetMaxIdleConns(dialector.MaxIdle)
		sqldb.SetMaxOpenConns(dialector.MaxOpen)
		db.ConnPool = sqldb
	}
	return nil
}

func (dialector *MySqlDialector) FormatPrepareSQL(src string) string {
	src = strings.ReplaceAll(src, "\r", " ")
	src = strings.ReplaceAll(src, "\n", " ")
	src = strings.ReplaceAll(src, "\t", " ")
	return src
}
