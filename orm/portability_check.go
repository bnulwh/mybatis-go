package orm

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/orm/dialector"
)

var portabilityCheck atomic.Bool
var portabilityWarned sync.Map

// SetPortabilityCheck 开启/关闭 || 跨模式语义差异检测（F2，
// mybatis.configuration.portability-check，默认关）。
func SetPortabilityCheck(on bool) { portabilityCheck.Store(on) }

// PortabilityCheck 返回 || 跨模式语义差异检测开关状态。
func PortabilityCheck() bool { return portabilityCheck.Load() }

// checkPortability 检测执行 SQL 中 || 操作符的跨模式语义差异，每条 SQL 仅告警一次：
//   - MySQL 族：|| 默认是逻辑或（除非 PIPES_AS_CONCAT），与 PG/Oracle 的字符串拼接语义不同；
//   - KingbaseES：|| 语义随兼容模式变化（MySQL 模式=逻辑或，PG/Oracle 模式=拼接），
//     框架无法感知实际 database_mode。
//
// 仅诊断告警，不改写 SQL；引号与注释内的 || 不计。
func (db *DB) checkPortability(query string) {
	if !portabilityCheck.Load() {
		return
	}
	if db == nil || db.Config == nil || db.Config.Dialector == nil {
		return
	}
	if !strings.Contains(query, "||") || !dialector.ContainsConcatOp(query) {
		return
	}
	var tip string
	switch {
	case db.Config.Dialector.Family() == dialector.FamilyMySQL:
		tip = "MySQL dialect || is logical OR by default (unless PIPES_AS_CONCAT), not string concat as in PG/Oracle"
	case db.Config.Dialector.Name() == "kingbase":
		tip = "KingbaseES || semantics depend on compatibility mode (logical OR in MySQL mode), actual database_mode unknown"
	default:
		return
	}
	if _, dup := portabilityWarned.LoadOrStore(query, struct{}{}); dup {
		return
	}
	log.Warnf("portability: %s: %s", tip, truncateSQLForLog(query))
}

const portabilitySQLLogMax = 160

func truncateSQLForLog(sql string) string {
	if len(sql) <= portabilitySQLLogMax {
		return sql
	}
	return sql[:portabilitySQLLogMax] + "..."
}
