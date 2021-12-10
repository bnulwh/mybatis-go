package orm

import (
	"database/sql"
	log "github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/types"
)

func Execute(sqlStr string, args ...interface{}) (int64, error) {
	gDbConn.lock.Lock()
	defer gDbConn.lock.Unlock()
	return execute(sqlStr, args...)
}
func Query(sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	gDbConn.lock.Lock()
	defer gDbConn.lock.Unlock()
	log.Debugf("sql: %v", sqlStr)
	return queryRows(sqlStr, args...)
}
func execute(sqlStr string, args ...interface{}) (int64, error) {
	log.Debugf("sql: %v", sqlStr)
	stmt, err := gDbConn.prepare(sqlStr)
	if err != nil {
		log.Errorf("prepare sql %v failed: %v", sqlStr, err)
		return 0, err
	}
	defer closeStmt(stmt)
	result, err := stmt.Exec(args...)
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
		log.Warnf("close warning: %v", err)
	}
	if gDbConn.conn != nil {
		err = gDbConn.conn.Close()
		if err != nil {
			log.Warnf("close warning: %v", err)
		}
	}
	//err = gDbConn.database.Close()
	//if err != nil {
	//	log.Warnf("close warning: %v", err)
	//}
}

func queryRows(sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	stmt, err := gDbConn.prepare(sqlStr)
	if err != nil {
		log.Errorf("prepare sql %v failed: %v", sqlStr, err)
		return nil, err
	}
	defer closeStmt(stmt)
	rows, err := stmt.Query(args...)
	if err != nil {
		log.Errorf("query sql %v failed: %v", sqlStr, err)
		return nil, err
	}
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