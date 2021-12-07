package orm

import (
	"database/sql"
	log "github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/types"
)

func Execute(sqlStr string, args ...interface{}) (int64, error) {
	gLock.Lock()
	defer gLock.Unlock()
	return execute(sqlStr, args...)
}
func Query(sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	gLock.Lock()
	defer gLock.Unlock()
	log.Debugf("sql: %v", sqlStr)
	return queryRows(sqlStr, args...)
}

func closeStmt(stmt *sql.Stmt) {
	err := stmt.Close()
	if err != nil {
		log.Warnf("close warning: %v", err)
	}
}

func queryRows(sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	stmt, err := gDbConn.Prepare(sqlStr)
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
