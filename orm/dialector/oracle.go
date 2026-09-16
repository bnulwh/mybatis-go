package dialector

import (
	"fmt"
	"strings"

	_ "github.com/sijms/go-ora/v2"
)

type OracleDialector struct {
	BaseDialector
}

func NewOracleDialector(cfg ConfigProvider) *OracleDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generateOracleDSN(params)
	}
	return &OracleDialector{
		BaseDialector: BaseDialector{
			params:           params,
			name:             "oracle",
			driverName:       GetDriverName(params.Type),
			dsn:              dsn,
			conn:             cfg.GetConnPool(),
			family:           FamilyOracle,
			placeholderStyle: PlaceholderColon,
			defaultMaxIdle:   DefaultMaxIdle,
			maxTimeout:       cfg.GetMaxTimeout(),
			maxOpen:          cfg.GetMaxOpen(),
		},
	}
}

func (d *OracleDialector) NeedsReturning() bool {
	return false
}

func (d *OracleDialector) SystemTablePrefixes() []string {
	return []string{"SYS", "SYSTEM", "CTXSYS", "MDSYS", "OLAPSYS", "ORDSYS", "OUTLN", "WMSYS", "XDB", "APEX"}
}

func (d *OracleDialector) TableStructureSQL(table string) string {
	schema := d.effectiveSchema()
	return fmt.Sprintf(`SELECT C.COLUMN_NAME, C.DATA_TYPE AS COLUMN_TYPE,
    CC.COMMENTS AS COLUMN_COMMENT,
    CASE WHEN PK.COLUMN_NAME IS NOT NULL THEN 'PRI' ELSE '' END AS COLUMN_KEY,
    C.DATA_LENGTH AS LENGTH,
    CASE C.NULLABLE WHEN 'Y' THEN 1 ELSE 0 END AS IS_NULLABLE
    FROM ALL_TAB_COLUMNS C
    LEFT JOIN ALL_COL_COMMENTS CC ON C.OWNER=CC.OWNER AND C.TABLE_NAME=CC.TABLE_NAME AND C.COLUMN_NAME=CC.COLUMN_NAME
    LEFT JOIN (SELECT CC2.OWNER,CC2.TABLE_NAME,CC2.COLUMN_NAME FROM ALL_CONSTRAINTS AC
        JOIN ALL_CONS_COLUMNS CC2 ON AC.OWNER=CC2.OWNER AND AC.CONSTRAINT_NAME=CC2.CONSTRAINT_NAME
        WHERE AC.CONSTRAINT_TYPE='P') PK ON C.OWNER=PK.OWNER AND C.TABLE_NAME=PK.TABLE_NAME AND C.COLUMN_NAME=PK.COLUMN_NAME
    WHERE C.OWNER='%s' AND C.TABLE_NAME='%s'
    ORDER BY C.COLUMN_ID`, schema, table)
}

func (d *OracleDialector) TableListSQL() string {
	schema := d.effectiveSchema()
	return fmt.Sprintf("SELECT TABLE_NAME FROM ALL_TABLES WHERE OWNER='%s'", schema)
}

func (d *OracleDialector) ApplyPagination(query string, limit, offset int) string {
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
