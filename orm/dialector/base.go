package dialector

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	DefaultMaxIdle    = 100
	DefaultMaxOpen    = 100
	DefaultMaxTimeout = 300
)

type BaseDialector struct {
	params           ConnectParams
	name             string
	driverName       string
	dsn              string
	conn             ConnPool
	family           DatabaseFamily
	placeholderStyle PlaceholderStyle
	defaultMaxIdle   int
	maxTimeout       int
	maxOpen          int
}

func (d *BaseDialector) Name() string {
	return d.name
}

func (d *BaseDialector) DSN() string {
	return d.dsn
}

func (d *BaseDialector) effectiveSchema() string {
	return EffectiveSchema(d.params)
}

func (d *BaseDialector) Initialize() (ConnPool, error) {
	if d.conn != nil {
		return d.conn, nil
	}
	sqldb, err := sql.Open(d.driverName, d.dsn)
	if err != nil {
		return nil, err
	}
	timeout := int(time.Second) * d.maxTimeout
	sqldb.SetConnMaxLifetime(time.Duration(timeout))
	sqldb.SetMaxIdleConns(d.defaultMaxIdle)
	sqldb.SetMaxOpenConns(d.maxOpen)
	return sqldb, nil
}

func (d *BaseDialector) FormatPrepareSQL(src string) string {
	return FormatPrepareSQLByStyle(src, d.placeholderStyle)
}

// FormatPrepareSQLByStyle 按占位符风格改写 SQL：
// 只有代码区的 `?` 会改写为方言占位符（$n / :n / @pn），字符串字面量、
// 引号标识符、行/块注释、Postgres 美元引用字符串以及 jsonb 的 ? / ?| / ?&
// 操作符原样保留（P0-1，见 placeholder.go）。
func FormatPrepareSQLByStyle(src string, style PlaceholderStyle) string {
	return rewritePlaceholders(src, style)
}

func (d *BaseDialector) PlaceholderStyle() PlaceholderStyle {
	return d.placeholderStyle
}

func (d *BaseDialector) NeedsReturning() bool {
	return false
}

func (d *BaseDialector) Family() DatabaseFamily {
	return d.family
}

func (d *BaseDialector) ApplyPagination(query string, limit, offset int) string {
	return ApplyPagination(query, limit, offset)
}

func (d *BaseDialector) SystemTablePrefixes() []string {
	return nil
}

// ReservedTableNames 默认无保留名；具体方言按需覆盖。
func (d *BaseDialector) ReservedTableNames() *ReservedNames {
	return nil
}

func (d *BaseDialector) TableStructureSQL(table string) string {
	return ""
}

func (d *BaseDialector) TableListSQL() string {
	return ""
}

func (d *BaseDialector) DefaultMaxIdle() int {
	return d.defaultMaxIdle
}

func formatPlaceholders(src string, style PlaceholderStyle) string {
	return rewritePlaceholders(src, style)
}

// ApplyPagination 把分页子句拼到 query 尾部（B2 修复）：
// 只剥离「顶层」（括号深度 0）已存在的 LIMIT/OFFSET 再拼接，
// 子查询里的 LIMIT/OFFSET（如 `... (SELECT ... LIMIT 1) AS x`）原样保留。
func ApplyPagination(query string, limit, offset int) string {
	if limit <= 0 {
		return query
	}
	q := strings.TrimRight(query, " \t\n\r;")
	q = StripTopLevelTailClauses(q, "limit", "offset")
	if offset > 0 {
		return fmt.Sprintf("%s LIMIT %d OFFSET %d", q, limit, offset)
	}
	return fmt.Sprintf("%s LIMIT %d", q, limit)
}
