package dialector

import (
	"fmt"
	"strings"

	_ "github.com/microsoft/go-mssqldb"
)

type MssqlDialector struct {
	BaseDialector
}

func NewMssqlDialector(cfg ConfigProvider) *MssqlDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generateMssqlDSN(params)
	}
	return &MssqlDialector{
		BaseDialector: BaseDialector{
			params:           params,
			name:             "mssql",
			driverName:       GetDriverName(params.Type),
			dsn:              dsn,
			conn:             cfg.GetConnPool(),
			family:           FamilyMSSQL,
			placeholderStyle: PlaceholderAtP,
			defaultMaxIdle:   DefaultMaxIdle,
			maxTimeout:       cfg.GetMaxTimeout(),
			maxOpen:          cfg.GetMaxOpen(),
		},
	}
}

func (d *MssqlDialector) NeedsReturning() bool {
	return false
}

func (d *MssqlDialector) SystemTablePrefixes() []string {
	return []string{"sys", "INFORMATION_SCHEMA"}
}

func (d *MssqlDialector) TableStructureSQL(table string) string {
	schema := d.effectiveSchema()
	return fmt.Sprintf(`SELECT C.COLUMN_NAME, C.DATA_TYPE AS COLUMN_TYPE,
    '' AS COLUMN_COMMENT,
    CASE WHEN PK.COLUMN_NAME IS NOT NULL THEN 'PRI' ELSE '' END AS COLUMN_KEY,
    COALESCE(C.CHARACTER_MAXIMUM_LENGTH, C.NUMERIC_PRECISION, -1) AS LENGTH,
    CASE C.IS_NULLABLE WHEN 'YES' THEN 1 ELSE 0 END AS IS_NULLABLE
    FROM INFORMATION_SCHEMA.COLUMNS C
    LEFT JOIN (
        SELECT CCU.COLUMN_NAME
        FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS TC
        JOIN INFORMATION_SCHEMA.CONSTRAINT_COLUMN_USAGE CCU
            ON TC.CONSTRAINT_NAME = CCU.CONSTRAINT_NAME
        WHERE TC.CONSTRAINT_TYPE = 'PRIMARY KEY'
            AND TC.TABLE_SCHEMA = '%s' AND TC.TABLE_NAME = '%s'
    ) PK ON C.COLUMN_NAME = PK.COLUMN_NAME
    WHERE C.TABLE_SCHEMA = '%s' AND C.TABLE_NAME = '%s'
    ORDER BY C.ORDINAL_POSITION`, schema, table, schema, table)
}

func (d *MssqlDialector) TableListSQL() string {
	schema := d.effectiveSchema()
	return fmt.Sprintf(
		"SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = '%s' AND TABLE_TYPE = 'BASE TABLE'",
		schema)
}

func (d *MssqlDialector) ApplyPagination(query string, limit, offset int) string {
	if limit <= 0 {
		return query
	}
	q := limitRe.ReplaceAllString(query, " ")
	q = offsetRe.ReplaceAllString(q, " ")
	q = strings.TrimRight(q, " \t\n\r;")
	if offset > 0 {
		return fmt.Sprintf("%s OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", q, offset, limit)
	}
	return fmt.Sprintf("%s OFFSET 0 ROWS FETCH NEXT %d ROWS ONLY", q, limit)
}
