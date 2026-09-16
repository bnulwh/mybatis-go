package dialector

import (
	"fmt"
	"net/url"
	"strings"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

type ClickHouseDialector struct {
	BaseDialector
}

func NewClickHouseDialector(cfg ConfigProvider) *ClickHouseDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generateClickHouseDSN(params)
	}
	return &ClickHouseDialector{
		BaseDialector: BaseDialector{
			params:           params,
			name:             "clickhouse",
			driverName:       GetDriverName(params.Type),
			dsn:              dsn,
			conn:             cfg.GetConnPool(),
			family:           FamilyClickHouse,
			placeholderStyle: PlaceholderQuestion,
			defaultMaxIdle:   DefaultMaxIdle,
			maxTimeout:       cfg.GetMaxTimeout(),
			maxOpen:          cfg.GetMaxOpen(),
		},
	}
}

func (d *ClickHouseDialector) NeedsReturning() bool {
	return false
}

func (d *ClickHouseDialector) SystemTablePrefixes() []string {
	return []string{"system", "INFORMATION_SCHEMA"}
}

func (d *ClickHouseDialector) TableStructureSQL(table string) string {
	schema := d.effectiveSchema()
	return fmt.Sprintf(`SELECT C.name AS COLUMN_NAME, C.type AS COLUMN_TYPE,
    C.comment AS COLUMN_COMMENT,
    CASE WHEN PK.name IS NOT NULL THEN 'PRI' ELSE '' END AS COLUMN_KEY,
    0 AS LENGTH,
    1 AS IS_NULLABLE
    FROM system.columns C
    LEFT JOIN (
        SELECT name
        FROM system.table_columns
        WHERE database='%s' AND table='%s' AND is_in_primary_key=1
    ) PK ON C.name=PK.name
    WHERE C.database='%s' AND C.table='%s'
    ORDER BY C.position`, schema, table, schema, table)
}

func (d *ClickHouseDialector) TableListSQL() string {
	schema := d.effectiveSchema()
	return fmt.Sprintf("SELECT name AS TABLE_NAME FROM system.tables WHERE database='%s'", schema)
}

func (d *ClickHouseDialector) ApplyPagination(query string, limit, offset int) string {
	if limit <= 0 {
		return query
	}
	q := limitRe.ReplaceAllString(query, " ")
	q = offsetRe.ReplaceAllString(q, " ")
	q = strings.TrimRight(q, " \t\n\r;")
	if offset > 0 {
		return fmt.Sprintf("%s LIMIT %d OFFSET %d", q, limit, offset)
	}
	return fmt.Sprintf("%s LIMIT %d", q, limit)
}

func generateClickHouseDSN(p ConnectParams) string {
	return fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s",
		url.QueryEscape(p.Username),
		url.QueryEscape(p.Password),
		p.Host, p.Port, p.DBName)
}
