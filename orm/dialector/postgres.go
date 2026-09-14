package dialector

import "fmt"

type PostgresDialector struct {
	BaseDialector
}

func NewPostgresDialector(cfg ConfigProvider) *PostgresDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generatePostgresDSN(params)
	}
	return &PostgresDialector{
		BaseDialector: BaseDialector{
			params:           params,
			name:             "postgres",
			driverName:       GetDriverName(params.Type),
			dsn:              dsn,
			conn:             cfg.GetConnPool(),
			family:           FamilyPostgres,
			placeholderStyle: PlaceholderDollar,
			defaultMaxIdle:   DefaultMaxIdle,
			maxTimeout:       cfg.GetMaxTimeout(),
			maxOpen:          cfg.GetMaxOpen(),
		},
	}
}

func (d *PostgresDialector) NeedsReturning() bool {
	return true
}

func (d *PostgresDialector) SystemTablePrefixes() []string {
	return []string{"pg_", "pg_catalog"}
}

func (d *PostgresDialector) TableStructureSQL(table string) string {
	schema := d.effectiveSchema()
	attrelid := table
	if schema != "public" {
		attrelid = schema + "." + table
	}
	return fmt.Sprintf(`SELECT
    A.ordinal_position,A.table_name,A.column_name,CASE A.is_nullable WHEN 'NO' THEN 0 ELSE 1 END AS is_nullable,
    col_description(B.attrelid,B.attnum) as column_comment,
    A.data_type as column_type,coalesce(A.character_maximum_length, A.numeric_precision, -1) as length,
    A.numeric_scale,CASE WHEN length(B.attname) > 0 THEN 'PRI' ELSE '' END AS column_key
    FROM information_schema.columns A,pg_attribute B
    WHERE A.column_name = B.attname AND B.attrelid = '%s' :: regclass   
          AND  A.table_schema = '%s'  AND A.table_name = '%s'
    ORDER BY A.ordinal_position ASC`, attrelid, schema, table)
}

func (d *PostgresDialector) TableListSQL() string {
	if schema := d.params.Schema; schema != "" {
		return fmt.Sprintf("select relname as TABLE_NAME from pg_class c join pg_namespace n on n.oid = c.relnamespace where c.relkind = 'r' and n.nspname = '%s' and c.relname not like 'pg_%%' and c.relname not like 'sql_%%'", schema)
	}
	return "select relname as TABLE_NAME from pg_class where  relkind = 'r' and relname not like 'pg_%' and relname not like 'sql_%'"
}
