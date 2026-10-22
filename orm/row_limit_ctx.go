package orm

import (
	"context"
	"sync/atomic"
	"time"
)

// B3：语句级行数上限覆盖 + 截断可观测。
//
// 语句级覆盖经 context 传递（不改公共 API 签名）：<select maxRows="N"> 解析到
// SqlFunction.MaxRows 后，Mapper 代理执行路径（executeMethod/executeStream/
// executePage）把「生效上限 + 语句标识」挂到 arg.Ctx，下游 fetchRows /
// scanRowsTo* / RowStream 统一经 effectiveRowLimit 解析；原生 SQL 查询
// （Query/QueryTo/QueryStream）ctx 未携带时沿用全局 DefaultRowLimit。

// rowLimitInfo ctx 携带的语句级行数上限信息（B3）。
type rowLimitInfo struct {
	limit     int    // 语句级覆盖：0=未设置（沿用全局），-1 不限，>0 语句级上限
	statement string // 语句标识 namespace.id（截断记录用），原生 SQL 查询为空
}

type rowLimitCtxKey struct{}

// withStatementRowLimit 为 ctx 附加语句级行数上限与语句标识（B3）。
// maxRows 0=未设置（沿用全局 DefaultRowLimit），-1 不限，>0 语句级上限。
func withStatementRowLimit(ctx context.Context, statement string, maxRows int64) context.Context {
	return context.WithValue(ctx, rowLimitCtxKey{}, &rowLimitInfo{limit: int(maxRows), statement: statement})
}

// effectiveRowLimit 解析生效的行数上限与语句标识（B3）：
// ctx 未携带 → 全局 DefaultRowLimit；携带且语句级覆盖非 0 → 语句级优先。
func effectiveRowLimit(ctx context.Context) (int, string) {
	limit := DefaultRowLimit()
	if ctx == nil {
		return limit, ""
	}
	if v := ctx.Value(rowLimitCtxKey{}); v != nil {
		if info, ok := v.(*rowLimitInfo); ok {
			if info.limit != 0 {
				limit = info.limit
			}
			return limit, info.statement
		}
	}
	return limit, ""
}

// QueryTruncation 最近一次因行数上限被截断（或严格模式拦截）的查询记录（B3）。
type QueryTruncation struct {
	Statement string    // 语句标识 namespace.id，原生 SQL 查询为空
	Limit     int       // 触发的行数上限
	Time      time.Time // 发生时间
}

var lastQueryTruncation atomic.Pointer[QueryTruncation]

// recordQueryTruncation 记录一次触发行数上限事件（静默截断与严格拦截均记录）。
func recordQueryTruncation(statement string, limit int) {
	lastQueryTruncation.Store(&QueryTruncation{Statement: statement, Limit: limit, Time: time.Now()})
}

// LastQueryTruncated 返回最近一次因行数上限被截断/拦截的查询记录（B3 可观测）：
// 进程内未发生过返回 nil。并发安全（总是最近一次）；截断发生时另有 Warn 日志，
// 严格模式下查询错误本身已含 ErrRowLimitExceeded。
func LastQueryTruncated() *QueryTruncation {
	return lastQueryTruncation.Load()
}

// ResetQueryTruncation 清空最近一次截断记录（测试用）。
func ResetQueryTruncation() {
	lastQueryTruncation.Store(nil)
}
