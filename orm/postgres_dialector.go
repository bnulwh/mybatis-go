package orm

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type PostgresConfig struct {
	Config
	DriverName string
	DSN        string
	Conn       ConnPool
}

type PostgresDialector struct {
	*PostgresConfig
}

func NewPostgresDialector(cfg *Config) *PostgresDialector {
	return &PostgresDialector{
		PostgresConfig: &PostgresConfig{
			Config:     *cfg,
			DriverName: cfg.DriverName(),
			DSN:        cfg.GenerateDSN(),
		},
	}
}

func (dialector *PostgresDialector) Name() string {
	return "postgres"
}

func (dialector *PostgresDialector) Initialize(db *DB) (err error) {
	if dialector.DriverName == "" {
		dialector.DriverName = "postgres"
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

func (dialector *PostgresDialector) FormatPrepareSQL(src string) string {
	src = strings.ReplaceAll(src, "\r", " ")
	src = strings.ReplaceAll(src, "\n", " ")
	src = strings.ReplaceAll(src, "\t", " ")
	arr := strings.Split(src, "?")
	var res []string
	for i, s := range arr {
		res = append(res, s)
		if i < len(arr)-1 {
			res = append(res, fmt.Sprintf("$%d", i+1))
		}
	}
	return strings.Join(res, "")
}
