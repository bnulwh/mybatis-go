package dialector

import (
	"database/sql"

	"github.com/lib/pq"
)

func init() {
	for _, name := range []string{"vastbase"} {
		if isDriverRegistered(name) {
			continue
		}
		sql.Register(name, &pq.Driver{})
	}
}

type VastbaseDialector struct {
	PostgresDialector
}

func NewVastbaseDialector(cfg ConfigProvider) *VastbaseDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generatePostgresDSN(params)
	}
	return &VastbaseDialector{
		PostgresDialector: PostgresDialector{
			BaseDialector: BaseDialector{
				params:           params,
				name:             "vastbase",
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
