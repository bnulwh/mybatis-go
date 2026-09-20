package orm

import (
	"testing"
	"time"

	"github.com/bnulwh/mybatis-go/orm/dialector"
)

func Test_ExplainSQLPrefix(t *testing.T) {
	tests := []struct {
		family dialector.DatabaseFamily
		expect string
	}{
		{dialector.FamilyMySQL, "EXPLAIN "},
		{dialector.FamilySQLite, "EXPLAIN "},
		{dialector.FamilyPostgres, "EXPLAIN ANALYZE "},
		{dialector.FamilyOracle, "EXPLAIN PLAN FOR "},
		{dialector.FamilyMSSQL, "SET SHOWPLAN_TEXT ON; "},
		{dialector.FamilyDB2, "EXPLAIN ALL FOR "},
		{dialector.FamilyInformix, "SET EXPLAIN ON; "},
		{dialector.FamilyClickHouse, "EXPLAIN "},
	}
	for _, tt := range tests {
		got := explainSQLPrefix(tt.family)
		if got != tt.expect {
			t.Errorf("explainSQLPrefix(%v) = %q, want %q", tt.family, got, tt.expect)
		}
	}
}

func Test_ExplainSlowSQLConfig(t *testing.T) {
	SetExplainSlowSQL(true)
	if !ExplainSlowSQLEnabled() {
		t.Error("expected ExplainSlowSQLEnabled = true")
	}
	SetExplainSlowSQL(false)
	if ExplainSlowSQLEnabled() {
		t.Error("expected ExplainSlowSQLEnabled = false")
	}
}

func Test_SlowSQLThreshold(t *testing.T) {
	orig := SlowSQLThreshold()
	SetSlowSQLThreshold(5 * time.Second)
	if SlowSQLThreshold() != 5*time.Second {
		t.Errorf("threshold = %v, want 5s", SlowSQLThreshold())
	}
	SetSlowSQLThreshold(0)
	if SlowSQLThreshold() != 5*time.Second {
		t.Errorf("zero should not change threshold, got %v", SlowSQLThreshold())
	}
	SetSlowSQLThreshold(orig)
}

func Test_ExplainSlowSQL_SkipsNonSelect(t *testing.T) {
	SetExplainSlowSQL(true)
	defer SetExplainSlowSQL(false)
	hc := &HookContext{
		SqlType:  "insert",
		Duration: 5 * time.Second,
	}
	explainSlowSQL(hc)
}

func Test_ExplainSlowSQL_SkipsBelowThreshold(t *testing.T) {
	SetExplainSlowSQL(true)
	defer SetExplainSlowSQL(false)
	orig := SlowSQLThreshold()
	defer SetSlowSQLThreshold(orig)
	SetSlowSQLThreshold(10 * time.Second)
	hc := &HookContext{
		SqlType:  "select",
		Duration: 1 * time.Second,
	}
	explainSlowSQL(hc)
}

func Test_ExplainSlowSQL_SkipsWhenDisabled(t *testing.T) {
	SetExplainSlowSQL(false)
	hc := &HookContext{
		SqlType:  "select",
		Duration: 5 * time.Second,
	}
	explainSlowSQL(hc)
}

func Test_GetDBFamily_NoConn(t *testing.T) {
	f := getDBFamily()
	if f != dialector.FamilyMySQL {
		t.Errorf("expected mysql fallback, got %v", f)
	}
}
