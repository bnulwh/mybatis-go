package dialector

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
)

func init() {
	for _, name := range []string{"tidb"} {
		if isDriverRegistered(name) {
			continue
		}
		sql.Register(name, &mysql.MySQLDriver{})
	}
}

type TiDBDialector struct {
	MySqlDialector
}

func NewTiDBDialector(cfg ConfigProvider) *TiDBDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generateMySQLDSN(params)
	}
	return &TiDBDialector{
		MySqlDialector: MySqlDialector{
			BaseDialector: BaseDialector{
				params:           params,
				name:             "tidb",
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
