package orm

import (
	"context"
	"database/sql"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
)

func Execute(sqlStr string, args ...interface{}) (int64, error) {
	//gDbConn.lock.Lock()
	//defer gDbConn.lock.Unlock()
	return execute(sqlStr, args...)
}
func Query(sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	//gDbConn.lock.Lock()
	//defer gDbConn.lock.Unlock()
	log.Debugf("sql: %v", sqlStr)
	return queryRows(sqlStr, args...)
}
func execute(sqlStr string, args ...interface{}) (int64, error) {
	log.Debugf("sql: %v", sqlStr)
	ctx := context.Background()
	//stmt, err := gDbConn.prepare(ctx, sqlStr)
	//if err != nil {
	//	log.Errorf("prepare sql %v failed: %v", sqlStr, err)
	//	return 0, err
	//}
	//defer closeStmt(conn, stmt)
	result, err := gDbConn.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		log.Errorf("execute sql %v failed: %v", sqlStr, err)
		return 0, err
	}
	rf, _ := result.RowsAffected()
	return rf, nil
}

func closeStmt(stmt *sql.Stmt) {
	err := stmt.Close()
	if err != nil {
		log.Warnf("close stmt warning: %v", err)
	}
	//if gDbConn.conn != nil {
	//err = conn.Close()
	//if err != nil {
	//	log.Warnf("close conn warning: %v", err)
	//}
	//}
	//err = gDbConn.database.Close()
	//if err != nil {
	//	log.Warnf("close warning: %v", err)
	//}
}

func queryRows(sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	ctx := context.Background()
	//stmt, err := gDbConn.prepare(ctx, sqlStr)
	//if err != nil {
	//	log.Errorf("prepare sql %v failed: %v", sqlStr, err)
	//	return nil, err
	//}
	//defer closeStmt(conn, stmt)
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
	for rows.Next() {
		tempItems := prepareColumns(colTypes)
		err := rows.Scan(tempItems...)
		if err != nil {
			log.Warnf("scan error: %v", err)
			continue
		}
		mp := createMap(tempItems, colTypes)
		results = append(results, mp)
	}
	log.Debugf("results: %v", types.ToJson(results))
	return results
}
