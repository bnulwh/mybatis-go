package dialector

import (
	"database/sql"

	"github.com/lib/pq"
)

func init() {
	for _, name := range []string{"gaussdb"} {
		if isDriverRegistered(name) {
			continue
		}
		sql.Register(name, &pq.Driver{})
	}
}

type GaussDBDialector struct {
	PostgresDialector
}

func NewGaussDBDialector(cfg ConfigProvider) *GaussDBDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generatePostgresDSN(params)
	}
	return &GaussDBDialector{
		PostgresDialector: PostgresDialector{
			BaseDialector: BaseDialector{
				params:           params,
				name:             "gaussdb",
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
