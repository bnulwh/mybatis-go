package dialector

import (
	"database/sql"

	"github.com/lib/pq"
)

func init() {
	for _, name := range []string{"kingbase", "kingbase8", "kingbase7", "kingbase6", "kingbase5"} {
		if isDriverRegistered(name) {
			continue
		}
		sql.Register(name, &pq.Driver{})
	}
}

func isDriverRegistered(name string) bool {
	for _, d := range sql.Drivers() {
		if d == name {
			return true
		}
	}
	return false
}

type KingbaseDialector struct {
	PostgresDialector
}

// kingbaseReservedTableNames KingbaseES Oracle 兼容模式下的哑表 dual；
// 官方驱动注册名列表见 init。
var kingbaseReservedTableNames = NewReservedNames("dual")

// ReservedTableNames Kingbase 兼容 PG 内置函数，同时包含 Oracle 兼容模式的 dual。
func (d *KingbaseDialector) ReservedTableNames() *ReservedNames {
	return postgresReservedTableNames.Merge(kingbaseReservedTableNames)
}

func NewKingbaseDialector(cfg ConfigProvider) *KingbaseDialector {
	params := cfg.GetConnectParams()
	dsn := cfg.GetDSN()
	if dsn == "" {
		dsn = generatePostgresDSN(params)
	}
	return &KingbaseDialector{
		PostgresDialector: PostgresDialector{
			BaseDialector: BaseDialector{
				params:           params,
				name:             "kingbase",
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
