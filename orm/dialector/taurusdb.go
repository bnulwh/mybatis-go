package dialector

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
)

func init() {
	for _, name := range []string{"taurusdb"} {
		if isDriverRegistered(name) {
			continue
		}
		sql.Register(name, &mysql.MySQLDriver{})
	}
}

type TaurusDBDialector struct {
	MySqlDialector
}

func NewTaurusDBDialector(cfg ConfigProvider) *TaurusDBDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generateMySQLDSN(params)
	}
	return &TaurusDBDialector{
		MySqlDialector: MySqlDialector{
			BaseDialector: BaseDialector{
				params:           params,
				name:             "taurusdb",
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
