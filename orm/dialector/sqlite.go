package dialector

import "fmt"

type SqliteDialector struct {
	BaseDialector
}

func NewSqliteDialector(cfg ConfigProvider) *SqliteDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generateSQLiteDSN(params)
	}
	return &SqliteDialector{
		BaseDialector: BaseDialector{
			params:           params,
			name:             "sqlite",
			driverName:       GetDriverName(params.Type),
			dsn:              dsn,
			conn:             cfg.GetConnPool(),
			family:           FamilySQLite,
			placeholderStyle: PlaceholderQuestion,
			defaultMaxIdle:   1,
			maxTimeout:       cfg.GetMaxTimeout(),
			maxOpen:          cfg.GetMaxOpen(),
		},
	}
}

func (d *SqliteDialector) SystemTablePrefixes() []string {
	return []string{"sqlite_", "pragma_"}
}

func (d *SqliteDialector) TableStructureSQL(table string) string {
	return fmt.Sprintf(`SELECT name AS column_name, type AS column_type,
    '' AS column_comment, CASE WHEN pk > 0 THEN 'PRI' ELSE '' END AS column_key
    FROM pragma_table_info('%s')`, table)
}

func (d *SqliteDialector) TableListSQL() string {
	return "select name as table_name from sqlite_master where type='table' and name not like 'sqlite_%'"
}
