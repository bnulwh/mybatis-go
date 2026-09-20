package orm

import (
	"database/sql"
	"testing"
	"time"
)

func Test_PoolStats_NoConn(t *testing.T) {
	st := PoolStats()
	if st != (sql.DBStats{}) {
		t.Errorf("expected zero stats without conn, got %+v", st)
	}
}

func Test_PoolStatsFor_NotFound(t *testing.T) {
	_, err := PoolStatsFor("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent datasource")
	}
}

func Test_PoolStatsAll_NoConn(t *testing.T) {
	all := PoolStatsAll()
	if len(all) != 0 {
		t.Errorf("expected empty map, got %d entries", len(all))
	}
}

func Test_PoolStatsString_NoConn(t *testing.T) {
	s := PoolStatsString()
	if s != "no active datasource" {
		t.Errorf("got %q, want no active datasource", s)
	}
}

func Test_PoolStatsInterval_SetGet(t *testing.T) {
	orig := PoolStatsInterval()
	defer SetPoolStatsInterval(orig)

	SetPoolStatsInterval(30 * time.Second)
	if PoolStatsInterval() != 30*time.Second {
		t.Errorf("interval = %v, want 30s", PoolStatsInterval())
	}
	SetPoolStatsInterval(0)
	if PoolStatsInterval() != 0 {
		t.Errorf("interval = %v, want 0", PoolStatsInterval())
	}
}

func Test_PoolStatsLogger_StartStop(t *testing.T) {
	StartPoolStatsLogger(1 * time.Hour)
	poolStatsMu.Lock()
	running := poolStatsRunning
	poolStatsMu.Unlock()
	if !running {
		t.Error("expected logger running")
	}
	StopPoolStatsLogger()
	poolStatsMu.Lock()
	running = poolStatsRunning
	poolStatsMu.Unlock()
	if running {
		t.Error("expected logger stopped")
	}
}
