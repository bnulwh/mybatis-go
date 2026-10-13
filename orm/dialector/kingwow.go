package dialector

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
)

func init() {
	for _, name := range []string{"kingwow"} {
		if isDriverRegistered(name) {
			continue
		}
		sql.Register(name, &mysql.MySQLDriver{})
	}
}

type KingwowDialector struct {
	MySqlDialector
}

func NewKingwowDialector(cfg ConfigProvider) *KingwowDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generateMySQLDSN(params)
	}
	return &KingwowDialector{
		MySqlDialector: MySqlDialector{
			BaseDialector: BaseDialector{
				params:           params,
				name:             "kingwow",
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
