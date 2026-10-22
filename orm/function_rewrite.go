package orm

import (
	"sync/atomic"

	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/orm/dialector"
)

// F1：MySQL→PG 族函数/语法翻译开关（默认关）。
// 开启后对 PG 族数据源执行前的最终 SQL 做方言翻译（ifnull→coalesce、
// find_in_set→position、date_format→to_char、反引号→双引号标识符），
// 共享 Java XML 无需手改副本即可跑 PG / 金仓 PG 模式。
var functionRewrite atomic.Bool

// SetFunctionRewrite 设置 MySQL→PG 族函数翻译开关（F1，默认关）。
func SetFunctionRewrite(on bool) { functionRewrite.Store(on) }

// FunctionRewrite 返回当前函数翻译开关状态。
func FunctionRewrite() bool { return functionRewrite.Load() }

// applyFunctionRewrite 对 SQL 执行前的最终语句做 MySQL 方言函数/语法翻译（F1）：
// 仅在开关开启且当前数据源为 PG 族时生效（MySQL 族原生支持这些语法，无需翻译）；
// 其余情况原样返回。挂在 ExecContext/QueryContext/QueryRowContext 三条执行收口路径上，
// COUNT 派生、分页注入、RowStream 等上游产物均经由此处统一翻译。
func (db *DB) applyFunctionRewrite(query string) string {
	if !functionRewrite.Load() || db == nil || db.Dialector == nil {
		return query
	}
	if db.Dialector.Family() != dialector.FamilyPostgres {
		return query
	}
	out := dialector.RewriteMySqlCompat(query)
	if log.IsDebugEnabled() && out != query {
		name := ""
		if db.Config != nil {
			name = db.Setting.Name
		}
		log.Debugf("[function-rewrite] %s: rewrite %q -> %q", name, query, out)
	}
	return out
}
