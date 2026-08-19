package orm

import (
	"context"
	"database/sql"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
)

func Execute(sqlStr string, args ...interface{}) (int64, error) {
	return execute(sqlStr, args...)
}
func Query(sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	log.Debugf("sql: %v", sqlStr)
	return queryRows(sqlStr, args...)
}
func execute(sqlStr string, args ...interface{}) (int64, error) {
	result, err := executeWithResult(sqlStr, args...)
	if err != nil {
		return 0, err
	}
	rf, _ := result.RowsAffected()
	return rf, nil
}

// executeWithResult 返回原始 sql.Result，供 useGeneratedKeys 回填自增主键使用（S-11）。
func executeWithResult(sqlStr string, args ...interface{}) (sql.Result, error) {
	log.Debugf("sql: %v", sqlStr)
	ctx := context.Background()
	result, err := gDbConn.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		log.Errorf("execute sql %v failed: %v", sqlStr, err)
		return nil, err
	}
	return result, nil
}

func queryRows(sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	ctx := context.Background()
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
	for rows.Next() {
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
