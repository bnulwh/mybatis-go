package orm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/orm/dialector"
)

var (
	explainSlowSQLEnabled bool
	slowSQLThreshold      time.Duration = 3000 * time.Millisecond
)

func SetExplainSlowSQL(on bool) { explainSlowSQLEnabled = on }
func ExplainSlowSQLEnabled() bool { return explainSlowSQLEnabled }

func SetSlowSQLThreshold(d time.Duration) {
	if d > 0 {
		slowSQLThreshold = d
	}
}
func SlowSQLThreshold() time.Duration { return slowSQLThreshold }

func explainSQLPrefix(family dialector.DatabaseFamily) string {
	switch family {
	case dialector.FamilyMySQL, dialector.FamilySQLite:
		return "EXPLAIN "
	case dialector.FamilyPostgres:
		return "EXPLAIN ANALYZE "
	case dialector.FamilyOracle:
		return "EXPLAIN PLAN FOR "
	case dialector.FamilyMSSQL:
		return "SET SHOWPLAN_TEXT ON; "
	case dialector.FamilyDB2:
		return "EXPLAIN ALL FOR "
	case dialector.FamilyInformix:
		return "SET EXPLAIN ON; "
	default:
		return "EXPLAIN "
	}
}

func explainSlowSQL(hc *HookContext) {
	if !explainSlowSQLEnabled {
		return
	}
	if hc.SqlType != "select" {
		return
	}
	if hc.Duration < slowSQLThreshold {
		return
	}
	if gDbConn == nil || gDbConn.ConnPool == nil {
		return
	}

	family := getDBFamily()
	prefix := explainSQLPrefix(family)
	explainSQL := prefix + hc.SQL

	ctx, cancel := context.WithTimeout(hc.Ctx, 10*time.Second)
	defer cancel()

	rows, err := gDbConn.QueryContext(ctx, explainSQL)
	if err != nil {
		log.Warnf("EXPLAIN failed for slow SQL (cost=%v): %v\n  SQL: %v", hc.Duration, err, hc.SQL)
		return
	}
	defer rows.Close()

	var b strings.Builder
	fmt.Fprintf(&b, "Slow SQL EXPLAIN (cost=%v, namespace=%v.%v):\n", hc.Duration, hc.Namespace, hc.SqlId)
	cols, _ := rows.Columns()
	for rows.Next() {
		values := make([]interface{}, len(cols))
		for i := range values {
			values[i] = new(string)
		}
		if err := rows.Scan(values...); err != nil {
			break
		}
		parts := make([]string, len(cols))
		for i, v := range values {
			s, _ := v.(*string)
			if s != nil {
				parts[i] = *s
			}
		}
		b.WriteString("  ")
		b.WriteString(strings.Join(parts, " | "))
		b.WriteByte('\n')
	}

	log.Warnf("%s", strings.TrimRight(b.String(), "\n"))
}

func getDBFamily() dialector.DatabaseFamily {
	if gDbConn != nil && gDbConn.Dialector != nil {
		return gDbConn.Dialector.Family()
	}
	return dialector.FamilyMySQL
}

func registerExplainSlowHook() {
	RegisterHook(HookAfterExecute, func(hc *HookContext) {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("explainSlowSQL hook panic: %v", r)
			}
		}()
		explainSlowSQL(hc)
	})
}
