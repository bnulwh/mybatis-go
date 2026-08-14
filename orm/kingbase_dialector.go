package orm

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

// KingbaseES（人大金仓）与 PostgreSQL 采用相同的线协议，
// 这里复用 lib/pq 驱动并以 "kingbase" 等名称注册，避免依赖官方
// 未公开仓库 github.com/kingbase/kbgo。
func init() {
	for _, name := range []string{"kingbase", "kingbase8", "kingbase7", "kingbase6", "kingbase5"} {
		if isDriverRegistered(name) {
			continue
		}
		sql.Register(name, &pq.Driver{})
	}
}

func isDriverRegistered(name string) bool {
	for _, d := range sql.Drivers() {
		if d == name {
			return true
		}
	}
	return false
}

type KingbaseConfig struct {
	Config
	DriverName string
	DSN        string
	Conn       ConnPool
}

type KingbaseDialector struct {
	*KingbaseConfig
}

func NewKingbaseDialector(cfg *Config) *KingbaseDialector {
	return &KingbaseDialector{
		KingbaseConfig: &KingbaseConfig{
			Config:     *cfg,
			DriverName: cfg.DriverName(),
			DSN:        cfg.GenerateDSN(),
		},
	}
}

func (dialector *KingbaseDialector) Name() string {
	return "kingbase"
}

func (dialector *KingbaseDialector) Initialize(db *DB) (err error) {
	if dialector.DriverName == "" {
		dialector.DriverName = "kingbase"
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

// KingbaseES 占位符风格与 PostgreSQL 一致：? -> $1、$2 ...
func (dialector *KingbaseDialector) FormatPrepareSQL(src string) string {
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
