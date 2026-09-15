package dialector

import (
	"database/sql"

	"github.com/lib/pq"
)

func init() {
	for _, name := range []string{"opengauss"} {
		if isDriverRegistered(name) {
			continue
		}
		sql.Register(name, &pq.Driver{})
	}
}

type OpenGaussDialector struct {
	PostgresDialector
}

func NewOpenGaussDialector(cfg ConfigProvider) *OpenGaussDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generatePostgresDSN(params)
	}
	return &OpenGaussDialector{
		PostgresDialector: PostgresDialector{
			BaseDialector: BaseDialector{
				params:           params,
				name:             "opengauss",
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
