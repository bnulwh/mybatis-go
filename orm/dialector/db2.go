package dialector

import (
	"fmt"
	"strings"
)

type Db2Dialector struct {
	BaseDialector
}

func NewDb2Dialector(cfg ConfigProvider) *Db2Dialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generateDb2DSN(params)
	}
	return &Db2Dialector{
		BaseDialector: BaseDialector{
			params:           params,
			name:             "db2",
			driverName:       GetDriverName(params.Type),
			dsn:              dsn,
			conn:             cfg.GetConnPool(),
			family:           FamilyDB2,
			placeholderStyle: PlaceholderColon,
			defaultMaxIdle:   DefaultMaxIdle,
			maxTimeout:       cfg.GetMaxTimeout(),
			maxOpen:          cfg.GetMaxOpen(),
		},
	}
}

func (d *Db2Dialector) NeedsReturning() bool {
	return false
}

func (d *Db2Dialector) SystemTablePrefixes() []string {
	return []string{"SYSIBM", "SYSCAT", "SYSSTAT", "SYSTOOLS", "SYSFUN", "SYSPROC", "SYSIBMADM"}
}

func (d *Db2Dialector) TableStructureSQL(table string) string {
	schema := d.effectiveSchema()
	return fmt.Sprintf(`SELECT C.COLNAME AS COLUMN_NAME, C.TYPENAME AS COLUMN_TYPE,
    C.REMARKS AS COLUMN_COMMENT,
    CASE WHEN PK.COLNAME IS NOT NULL THEN 'PRI' ELSE '' END AS COLUMN_KEY,
    C.LENGTH AS LENGTH,
    CASE C.NULLS WHEN 'Y' THEN 1 ELSE 0 END AS IS_NULLABLE
    FROM SYSCAT.COLUMNS C
    LEFT JOIN (
        SELECT K.COLNAME
        FROM SYSCAT.KEYCOLUSE K
        JOIN SYSCAT.INDEXES I ON K.TABSCHEMA=I.TABSCHEMA AND K.TABNAME=I.TABNAME AND K.INDEXNAME=I.INDEXNAME
        WHERE I.TABSCHEMA='%s' AND I.TABNAME='%s' AND I.UNIQUERULE='P'
    ) PK ON C.COLNAME=PK.COLNAME
    WHERE C.TABSCHEMA='%s' AND C.TABNAME='%s'
    ORDER BY C.COLNO`, schema, table, schema, table)
}

func (d *Db2Dialector) TableListSQL() string {
	schema := d.effectiveSchema()
	return fmt.Sprintf("SELECT TABNAME AS TABLE_NAME FROM SYSCAT.TABLES WHERE TABSCHEMA='%s' AND TYPE='T'", schema)
}

func (d *Db2Dialector) ApplyPagination(query string, limit, offset int) string {
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

func generateDb2DSN(p ConnectParams) string {
	return fmt.Sprintf("HOSTNAME=%s;PORT=%d;DATABASE=%s;UID=%s;PWD=%s",
		p.Host, p.Port, p.DBName, p.Username, p.Password)
}
