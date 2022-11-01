package orm

import (
	"sync/atomic"
	"time"
)

type Statement struct {
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
	DBExecDuration     int64
	DBQueryDuration    int64
	DBExecMaxDuration  int64
	DBExecMinDuration  int64
	DBQueryMaxDuration int64
	DBQueryMinDuration int64
}

func (state *Statement) init() {
	atomic.SwapInt64(&state.QueryMinDuration, 60000)
	atomic.SwapInt64(&state.ExecuteMinDuration, 60000)
	atomic.SwapInt64(&state.DBExecMinDuration, 60000)
	atomic.SwapInt64(&state.DBQueryMinDuration, 60000)
}

func (state *Statement) updateExecStatement(start time.Time, success bool) {
	atomic.AddInt64(&state.ExecuteCount, 1)
	if !success {
		atomic.AddInt64(&state.ErrorCount, 1)
	}
	d := time.Since(start).Milliseconds()
	atomic.AddInt64(&state.DurationTotal, d)
	atomic.AddInt64(&state.ExecuteDuration, d)
	if d > state.ExecuteMaxDuration {
		atomic.SwapInt64(&state.ExecuteMaxDuration, d)
	}
	if d < state.ExecuteMinDuration {
		atomic.SwapInt64(&state.ExecuteMinDuration, d)
	}
}

func (state *Statement) updateQueryStatement(start time.Time, success bool) {
	atomic.AddInt64(&state.QueryCount, 1)
	if !success {
		atomic.AddInt64(&state.ErrorCount, 1)
	}
	d := time.Since(start).Milliseconds()
	atomic.AddInt64(&state.DurationTotal, d)
	atomic.AddInt64(&state.QueryDuration, d)
	if d > state.QueryMaxDuration {
		atomic.SwapInt64(&state.QueryMaxDuration, d)
	}
	if d < state.QueryMinDuration {
		atomic.SwapInt64(&state.QueryMinDuration, d)
	}
}
func (state *Statement) updateDBExecStatement(start time.Time) {

	d := time.Since(start).Milliseconds()
	atomic.AddInt64(&state.DBExecDuration, d)
	if d > state.DBExecMaxDuration {
		atomic.SwapInt64(&state.DBExecMaxDuration, d)
	}
	if d < state.DBExecMinDuration {
		atomic.SwapInt64(&state.DBExecMinDuration, d)
	}
}

func (state *Statement) updateDBQueryStatement(start time.Time) {
	d := time.Since(start).Milliseconds()
	atomic.AddInt64(&state.DBQueryDuration, d)
	if d > state.DBQueryMaxDuration {
		atomic.SwapInt64(&state.DBQueryMaxDuration, d)
	}
	if d < state.DBQueryMinDuration {
		atomic.SwapInt64(&state.DBQueryMinDuration, d)
	}
}
