package orm

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bnulwh/mybatis-go/log"
)

var (
	poolStatsMu         sync.Mutex
	poolStatsInterval   time.Duration
	poolStatsStopChan   chan struct{}
	poolStatsRunning    bool
)

func PoolStats() sql.DBStats {
	if gDbConn == nil {
		return sql.DBStats{}
	}
	return gDbConn.Stats()
}

func PoolStatsFor(name string) (sql.DBStats, error) {
	db, ok := gDataSources.get(name)
	if !ok {
		return sql.DBStats{}, fmt.Errorf("datasource %s not found", name)
	}
	return db.Stats(), nil
}

func PoolStatsAll() map[string]sql.DBStats {
	result := map[string]sql.DBStats{}
	for _, name := range gDataSources.names() {
		db, ok := gDataSources.get(name)
		if ok {
			result[name] = db.Stats()
		}
	}
	return result
}

func PoolStatsString() string {
	all := PoolStatsAll()
	if len(all) == 0 {
		return "no active datasource"
	}
	var b strings.Builder
	for name, st := range all {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "[%s] MaxOpenConnections=%d OpenConnections=%d InUse=%d Idle=%d WaitCount=%d WaitDuration=%v MaxIdleClosed=%d MaxIdleTimeClosed=%d MaxLifetimeClosed=%d",
			name, st.MaxOpenConnections, st.OpenConnections, st.InUse, st.Idle,
			st.WaitCount, st.WaitDuration, st.MaxIdleClosed, st.MaxIdleTimeClosed, st.MaxLifetimeClosed)
	}
	return b.String()
}

func SetPoolStatsInterval(d time.Duration) {
	poolStatsMu.Lock()
	defer poolStatsMu.Unlock()
	poolStatsInterval = d
	if poolStatsRunning {
		stopPoolStatsLogger()
		if d > 0 {
			startPoolStatsLogger()
		}
	} else if d > 0 {
		startPoolStatsLogger()
	}
}

func PoolStatsInterval() time.Duration {
	poolStatsMu.Lock()
	defer poolStatsMu.Unlock()
	return poolStatsInterval
}

func StartPoolStatsLogger(interval time.Duration) {
	poolStatsMu.Lock()
	defer poolStatsMu.Unlock()
	if poolStatsRunning {
		stopPoolStatsLogger()
	}
	poolStatsInterval = interval
	startPoolStatsLogger()
}

func StopPoolStatsLogger() {
	poolStatsMu.Lock()
	defer poolStatsMu.Unlock()
	stopPoolStatsLogger()
	poolStatsInterval = 0
}

func startPoolStatsLogger() {
	poolStatsStopChan = make(chan struct{})
	poolStatsRunning = true
	interval := poolStatsInterval
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Infof("Pool Stats:\n%s", PoolStatsString())
			case <-poolStatsStopChan:
				return
			}
		}
	}()
}

func stopPoolStatsLogger() {
	if poolStatsStopChan != nil {
		close(poolStatsStopChan)
		poolStatsStopChan = nil
	}
	poolStatsRunning = false
}
