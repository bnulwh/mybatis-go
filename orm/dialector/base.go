package dialector

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	limitRe  = regexp.MustCompile(`(?is)\s+LIMIT\s+\S+`)
	offsetRe = regexp.MustCompile(`(?is)\s+OFFSET\s+\S+`)
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

func FormatPrepareSQLByStyle(src string, style PlaceholderStyle) string {
	src = strings.ReplaceAll(src, "\r", " ")
	src = strings.ReplaceAll(src, "\n", " ")
	src = strings.ReplaceAll(src, "\t", " ")
	return formatPlaceholders(src, style)
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
	switch style {
	case PlaceholderDollar:
		return formatDollarPlaceholders(src)
	case PlaceholderColon:
		return formatColonPlaceholders(src)
	case PlaceholderAtP:
		return formatAtPPlaceholders(src)
	default:
		return src
	}
}

func formatDollarPlaceholders(src string) string {
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

func formatColonPlaceholders(src string) string {
	arr := strings.Split(src, "?")
	var res []string
	for i, s := range arr {
		res = append(res, s)
		if i < len(arr)-1 {
			res = append(res, fmt.Sprintf(":%d", i+1))
		}
	}
	return strings.Join(res, "")
}

func formatAtPPlaceholders(src string) string {
	arr := strings.Split(src, "?")
	var res []string
	for i, s := range arr {
		res = append(res, s)
		if i < len(arr)-1 {
			res = append(res, fmt.Sprintf("@p%d", i+1))
		}
	}
	return strings.Join(res, "")
}

func ApplyPagination(query string, limit, offset int) string {
	if limit <= 0 {
		return query
	}
	q := strings.TrimRight(query, " \t\n\r;")
	q = limitRe.ReplaceAllString(q, "")
	q = offsetRe.ReplaceAllString(q, "")
	if offset > 0 {
		return fmt.Sprintf("%s LIMIT %d OFFSET %d", q, limit, offset)
	}
	return fmt.Sprintf("%s LIMIT %d", q, limit)
}
