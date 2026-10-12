package dialector

import (
	"database/sql"

	"github.com/lib/pq"
)

func init() {
	for _, name := range []string{"antdb"} {
		if isDriverRegistered(name) {
			continue
		}
		sql.Register(name, &pq.Driver{})
	}
}

type AntDBDialector struct {
	PostgresDialector
}

func NewAntDBDialector(cfg ConfigProvider) *AntDBDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generatePostgresDSN(params)
	}
	return &AntDBDialector{
		PostgresDialector: PostgresDialector{
			BaseDialector: BaseDialector{
				params:           params,
				name:             "antdb",
				driverName:       GetDriverName(params.Type),
				dsn:              dsn,
				conn:             cfg.GetConnPool(),
				family:           FamilyPostgres,
				placeholderStyle: PlaceholderDollar,
				defaultMaxIdle:   DefaultMaxIdle,
				maxTimeout:       cfg.GetMaxTimeout(),
				maxOpen:          cfg.GetMaxOpen(),
			},
		},
	}
}
