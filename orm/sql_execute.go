package orm

import (
	"context"
	"database/sql"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
	"sync"
	"time"
)

var (
	// 默认执行超时（P4-1）：防止慢 SQL 无限挂起占住连接。
	// Execute/Query 及无 deadline 的 Context 变体均叠加该超时；0 或负数表示不设限。
	defaultExecTimeout = 5 * time.Minute
	execTimeoutMu      sync.RWMutex

	// 全局查询行数上限：防止大结果集 OOM（P4-2 相关）。
	// 0 表示不返回任何行，负数表示不限制（返回全部）。
	defaultRowLimit = 10000
	rowLimitMu      sync.RWMutex
)

// DefaultTimeout 返回当前全局默认执行超时。
func DefaultTimeout() time.Duration {
	execTimeoutMu.RLock()
	defer execTimeoutMu.RUnlock()
	return defaultExecTimeout
}

// SetDefaultTimeout 全局调整默认执行超时（P4-1，系统设置，默认 5 分钟）。
// 传 0 或负数可关闭默认超时（不推荐：慢 SQL 将无限挂起）。
func SetDefaultTimeout(d time.Duration) {
	execTimeoutMu.Lock()
	defer execTimeoutMu.Unlock()
	defaultExecTimeout = d
	log.Infof("set default exec timeout: %v", d)
}

// DefaultRowLimit 返回全局查询行数上限（默认 10000）。
func DefaultRowLimit() int {
	rowLimitMu.RLock()
	defer rowLimitMu.RUnlock()
	return defaultRowLimit
}

// SetDefaultRowLimit 全局调整查询返回行数上限（系统设置，默认 10000 行）。
// 负数不限制（返回全部），0 表示不返回任何行。
func SetDefaultRowLimit(n int) {
	rowLimitMu.Lock()
	defer rowLimitMu.Unlock()
	defaultRowLimit = n
	log.Infof("set default row limit: %d", n)
}

// withExecTimeout 为 ctx 附加全局默认超时作为安全网（P4-1）：
//   - ctx 已带 deadline → 返回原 ctx（调用方显式控制，不叠加）
//   - ctx 无 deadline 且全局超时 > 0 → 附加 context.WithTimeout
//   - 全局超时 <= 0 → 返回原 ctx（不设超时）
func withExecTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	d := DefaultTimeout()
	if d <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

// Execute 执行 DML/DDL 并返回影响行数；自动叠加全局默认超时（P4-1）。
func Execute(sqlStr string, args ...interface{}) (int64, error) {
	return execute(sqlStr, args...)
}

// ExecuteContext 以调用方 context 执行 DML/DDL（P4-1）；
// ctx 无 deadline 时叠加全局默认超时。
func ExecuteContext(ctx context.Context, sqlStr string, args ...interface{}) (int64, error) {
	result, err := executeWithResult(ctx, sqlStr, args...)
	if err != nil {
		return 0, err
	}
	rf, _ := result.RowsAffected()
	return rf, nil
}

// Query 执行查询并返回行 map 列表；自动叠加全局默认超时（P4-1）。
func Query(sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	return queryRows(context.Background(), sqlStr, args...)
}

// QueryContext 以调用方 context 执行查询（P4-1）；
// ctx 无 deadline 时叠加全局默认超时。
func QueryContext(ctx context.Context, sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	return queryRows(ctx, sqlStr, args...)
}

func execute(sqlStr string, args ...interface{}) (int64, error) {
	result, err := executeWithResult(context.Background(), sqlStr, args...)
	if err != nil {
		return 0, err
	}
	rf, _ := result.RowsAffected()
	return rf, nil
}

// executeWithResult 返回原始 sql.Result，供 useGeneratedKeys 回填自增主键使用（S-11）。
// ctx 无 deadline 时叠加全局默认超时（P4-1）。
func executeWithResult(ctx context.Context, sqlStr string, args ...interface{}) (sql.Result, error) {
	log.Debugf("sql: %v", sqlStr)
	ctx, cancel := withExecTimeout(ctx)
	defer cancel()
	result, err := gDbConn.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		log.Errorf("execute sql %v failed: %v", sqlStr, err)
		return nil, err
	}
	return result, nil
}

// queryRows 执行查询并返回行 map 列表；ctx 无 deadline 时叠加全局默认超时（P4-1）。
func queryRows(ctx context.Context, sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	ctx, cancel := withExecTimeout(ctx)
	defer cancel()
	log.Debugf("sql: %v", sqlStr)
	rows, err := gDbConn.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		log.Errorf("query sql %v failed: %v", sqlStr, err)
		return nil, err
	}
	defer rows.Close()
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		log.Errorf("fill sql %v result failed: %v", sqlStr, err)
		return nil, err
	}
	results := fetchRows(rows, colTypes)
	return results, nil
}
func fetchRows(rows *sql.Rows, colTypes []*sql.ColumnType) []map[string]interface{} {
	//var results []interface{}
	var results []map[string]interface{}
	// P1-3：扫描目标与转换函数只建一次，跨行复用（Scan 覆盖值，NULL 由 Valid 标记）。
	// 注意：部分驱动（如 modernc sqlite）在首行 Next() 之后才填充 ScanType，
	// 因此延迟到首次进入循环后再构建，与旧实现的行为保持一致。
	var tempItems []interface{}
	var converters []convertFn
	limit := DefaultRowLimit()
	for rows.Next() {
		// 全局行数上限（默认 1 万，负数不限制）：
		// 达到上限即停止读取，避免大结果集拖垮内存/连接（调用方 rows.Close 负责清理游标）
		if limit >= 0 && len(results) >= limit {
			log.Warnf("row limit reached: %d, truncate result set (SetDefaultRowLimit(-1) to return all)", limit)
			break
		}
		if tempItems == nil {
			tempItems = prepareColumns(colTypes)
			converters = buildConverters(colTypes)
		}
		err := rows.Scan(tempItems...)
		if err != nil {
			log.Warnf("scan error: %v", err)
			continue
		}
		mp := createMapWithConverters(tempItems, colTypes, converters)
		results = append(results, mp)
	}
	// 仅在调试日志开启时序列化，避免日志级别关闭时仍整结果集 ToJson
	if log.IsDebugEnabled() {
		log.Debugf("results: %v", types.ToJson(results))
	}
	return results
}
