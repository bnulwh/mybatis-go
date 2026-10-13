package dialector

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
)

func init() {
	for _, name := range []string{"updrdb"} {
		if isDriverRegistered(name) {
			continue
		}
		sql.Register(name, &mysql.MySQLDriver{})
	}
}

type UPDRDBDialector struct {
	MySqlDialector
}

func NewUPDRDBDialector(cfg ConfigProvider) *UPDRDBDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generateMySQLDSN(params)
	}
	return &UPDRDBDialector{
		MySqlDialector: MySqlDialector{
			BaseDialector: BaseDialector{
				params:           params,
				name:             "updrdb",
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
