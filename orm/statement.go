package orm

import "database/sql"

type Statement struct {
	sql.DBStats
	QueryCount         int64
	ExecuteCount       int64
	ErrorCount         int64
	DurationTotal      int64
	QueryDuration      int64
	ExecuteDuration    int64
	QueryMaxDuration   int64
	QueryMinDuration   int64
	ExecuteMaxDuration int64
	ExecuteMinDuration int64
}
