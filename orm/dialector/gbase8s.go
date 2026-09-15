package dialector

import "fmt"

type GBase8sDialector struct {
	BaseDialector
}

func NewGBase8sDialector(cfg ConfigProvider) *GBase8sDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generateInformixDSN(params)
	}
	return &GBase8sDialector{
		BaseDialector: BaseDialector{
			params:           params,
			name:             "gbase8s",
			driverName:       GetDriverName(params.Type),
			dsn:              dsn,
			conn:             cfg.GetConnPool(),
			family:           FamilyInformix,
			placeholderStyle: PlaceholderQuestion,
			defaultMaxIdle:   DefaultMaxIdle,
			maxTimeout:       cfg.GetMaxTimeout(),
			maxOpen:          cfg.GetMaxOpen(),
		},
	}
}

func (d *GBase8sDialector) NeedsReturning() bool {
	return false
}

func (d *GBase8sDialector) SystemTablePrefixes() []string {
	return []string{"SYS", "INFORMIX", "GBASE"}
}

func (d *GBase8sDialector) TableStructureSQL(table string) string {
	schema := d.effectiveSchema()
	return fmt.Sprintf(`SELECT C.COLNAME AS COLUMN_NAME, C.COLTYPE AS COLUMN_TYPE,
    '' AS COLUMN_COMMENT,
    CASE WHEN PK.COLNAME IS NOT NULL THEN 'PRI' ELSE '' END AS COLUMN_KEY,
    C.COLLNGTH AS LENGTH,
    CASE WHEN C.NULLS = 'Y' THEN 1 ELSE 0 END AS IS_NULLABLE
    FROM SYSOLUMNS C
    LEFT JOIN (SELECT RTSTRNAME, T2.TABNAME, COLNAME FROM SYSCONSTRAINTS T1
        JOIN SYSREFERENCES R ON T1.CONSTRAINTNAME = R.PRIMARY
        JOIN SYSCONSTRAINTS T2 ON R.PRIMARY = T2.CONSTRAINTNAME
        WHERE T1.CONSTTYPE = 'P') PK ON C.TABID = PK.TABID AND C.COLNAME = PK.COLNAME
    WHERE C.TABID = (SELECT TABID FROM SYSTABLES WHERE OWNER='%s' AND TABNAME='%s')
    ORDER BY C.COLNO`, schema, table)
}

func (d *GBase8sDialector) TableListSQL() string {
	schema := d.effectiveSchema()
	return fmt.Sprintf("SELECT TABNAME FROM SYSTABLES WHERE OWNER='%s' AND TABTYPE='T'", schema)
}
