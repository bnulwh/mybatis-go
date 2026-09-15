package dialector

import (
	"database/sql"

	"github.com/lib/pq"
)

func init() {
	for _, name := range []string{"highgo"} {
		if isDriverRegistered(name) {
			continue
		}
		sql.Register(name, &pq.Driver{})
	}
}

type HighGoDialector struct {
	PostgresDialector
}

func NewHighGoDialector(cfg ConfigProvider) *HighGoDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generatePostgresDSN(params)
	}
	return &HighGoDialector{
		PostgresDialector: PostgresDialector{
			BaseDialector: BaseDialector{
				params:           params,
				name:             "highgo",
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
