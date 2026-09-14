package dialector

import "fmt"

type MySqlDialector struct {
	BaseDialector
}

func NewMySqlDialector(cfg ConfigProvider) *MySqlDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generateMySQLDSN(params)
	}
	return &MySqlDialector{
		BaseDialector: BaseDialector{
			params:           params,
			name:             "mysql",
			driverName:       GetDriverName(params.Type),
			dsn:              dsn,
			conn:             cfg.GetConnPool(),
			family:           FamilyMySQL,
			placeholderStyle: PlaceholderQuestion,
			defaultMaxIdle:   DefaultMaxIdle,
			maxTimeout:       cfg.GetMaxTimeout(),
			maxOpen:          cfg.GetMaxOpen(),
		},
	}
}

func (d *MySqlDialector) SystemTablePrefixes() []string {
	return []string{"information_schema", "mysql", "performance_schema", "sys"}
}

func (d *MySqlDialector) TableStructureSQL(table string) string {
	schema := d.effectiveSchema()
	return fmt.Sprintf(`select TABLE_NAME as table_name,COLUMN_NAME as column_name,
    COLUMN_TYPE as column_type,COLUMN_COMMENT as column_comment,COLUMN_KEY as column_key 
    from information_schema.COLUMNS WHERE TABLE_SCHEMA='%s' AND TABLE_NAME='%s'
    ORDER BY ORDINAL_POSITION ASC`, schema, table)
}

func (d *MySqlDialector) TableListSQL() string {
	schema := d.effectiveSchema()
	return fmt.Sprintf("select DISTINCT TABLE_NAME as table_name from information_schema.COLUMNS WHERE TABLE_SCHEMA='%s'", schema)
}
