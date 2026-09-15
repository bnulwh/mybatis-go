package dialector

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
)

func init() {
	for _, name := range []string{"tdsql"} {
		if isDriverRegistered(name) {
			continue
		}
		sql.Register(name, &mysql.MySQLDriver{})
	}
}

type TDSQLDialector struct {
	MySqlDialector
}

func NewTDSQLDialector(cfg ConfigProvider) *TDSQLDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generateMySQLDSN(params)
	}
	return &TDSQLDialector{
		MySqlDialector: MySqlDialector{
			BaseDialector: BaseDialector{
				params:           params,
				name:             "tdsql",
				driverName:       GetDriverName(params.Type),
				dsn:              dsn,
				conn:             cfg.GetConnPool(),
				family:           FamilyMySQL,
				placeholderStyle: PlaceholderQuestion,
				defaultMaxIdle:   DefaultMaxIdle,
				maxTimeout:       cfg.GetMaxTimeout(),
				maxOpen:          cfg.GetMaxOpen(),
			},
		},
	}
}
