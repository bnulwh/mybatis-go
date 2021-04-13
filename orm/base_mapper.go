package orm

import (
	"database/sql"
	"fmt"
	log "github.com/astaxie/beego/logs"
	"github.com/bnulwh/mybatis-go/types"
	"reflect"
	"strings"
)

type BaseMapper struct {
	mapper *types.SqlMapper
}

func Execute(sqlStr string, args ...interface{}) (int64, error) {
	gLock.Lock()
	defer gLock.Unlock()
	return execute(sqlStr, args...)
}
func Query(sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	gLock.Lock()
	defer gLock.Unlock()
	log.Info("sql: %v", sqlStr)
	return queryRows(sqlStr, args...)
}
func execute(sqlStr string, args ...interface{}) (int64, error) {
	log.Info("sql: %v", sqlStr)
	stmt, err := gDbConn.Prepare(sqlStr)
	if err != nil {
		log.Error("prepare sql %v failed: %v", sqlStr, err)
		return 0, err
	}
	defer closeStmt(stmt)
	result, err := stmt.Exec(args...)
	if err != nil {
		log.Error("execute sql %v failed: %v", sqlStr, err)
		return 0, err
	}
	rf, _ := result.RowsAffected()
	return rf, nil
}
func (in *BaseMapper) fetchSqlFunction(name string) (*types.SqlFunction, error) {
	item, ok := in.mapper.NamedFunctions[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("%s not contains function %s", in.mapper.Namespace, name)
	}
	return item, nil
}

func (in *BaseMapper) executeMethod(sqlFunc *types.SqlFunction, arg ProxyArg) (reflect.Value, error) {
	args := arg.buildArgs()
	gLock.Lock()
	defer gLock.Unlock()
	sqlStr, items, err := sqlFunc.GenerateSQL(args...)
	if err != nil {
		log.Warn("generate sql failed: %v", err)
		return reflect.Value{}, err
	}
	log.Info("sql: %v", sqlStr)
	switch sqlFunc.Type {
	case types.InsertFunction, types.DeleteFunction, types.UpdateFunction:
		rf, err := execute(sqlStr, items...)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(rf), nil
	case types.SelectFunction:
		rows, err := queryRows(sqlStr, items...)
		if err != nil {
			return reflect.Value{}, err
		}
		results := convert2Results(rows, sqlFunc.Result)
		log.Info("results: %v", types.ToJson(results.Interface()))
		log.Info("results: %v", types.ToJson(reflect.Indirect(results).Interface()))
		return results, nil
	}
	return reflect.Value{}, nil
}

func closeStmt(stmt *sql.Stmt) {
	err := stmt.Close()
	if err != nil {
		log.Warn("close warning: %v", err)
	}
}

func queryRows(sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	stmt, err := gDbConn.Prepare(sqlStr)
	if err != nil {
		log.Error("prepare sql %v failed: %v", sqlStr, err)
		return nil, err
	}
	defer closeStmt(stmt)
	rows, err := stmt.Query(args...)
	if err != nil {
		log.Error("query sql %v failed: %v", sqlStr, err)
		return nil, err
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		log.Error("fill sql %v result failed: %v", sqlStr, err)
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
			log.Warn("scan error: %v", err)
			continue
		}
		mp := createMap(tempItems, colTypes)
		results = append(results, mp)
	}
	log.Debug("results: %v", types.ToJson(results))
	return results
}
func convert2Results(rows []map[string]interface{}, resInfo types.SqlResult) reflect.Value {
	//var results []interface{}
	itemTyp := getResultType(resInfo)
	itemsTyp := reflect.SliceOf(itemTyp)
	resultsPtr := reflect.New(itemsTyp)
	results := reflect.Indirect(resultsPtr)
	for _, row := range rows {
		result, err := createResult(row, resInfo)
		if err != nil {
			log.Warn("fill result failed: %v", err)
			continue
		}
		results = reflect.Append(results, reflect.ValueOf(result))
	}
	log.Debug("results: %v", types.ToJson(results.Interface()))
	//log.Info("results ptr: %v", types.ToJson(reflect.Indirect(resultsPtr).Interface()))
	return results
}
func prepareColumns(colTypes []*sql.ColumnType) []interface{} {
	var ptrs []interface{}
	for _, coltyp := range colTypes {
		log.Debug("name: %v,dbtype: %v,scan type: %v", coltyp.Name(), coltyp.DatabaseTypeName(), coltyp.ScanType())
		ptrs = append(ptrs, getSqlPtrType(coltyp.ScanType()))
	}
	return ptrs
}
func createMap(ptrs []interface{}, colTypes []*sql.ColumnType) map[string]interface{} {
	mp := map[string]interface{}{}
	for i, coltyp := range colTypes {
		v, err := convertValue(ptrs[i], coltyp.ScanType())
		if err != nil {
			log.Warn("convert %v to %v failed: %v", ptrs[i], coltyp.ScanType(), err)
			continue
		}
		mp[coltyp.Name()] = v
	}
	return mp
}

func convert2Result(mp map[string]interface{}, rmp *types.ResultMap) (interface{}, error) {
	name := types.GetShortName(rmp.TypeName)
	inst, err := gCache.createModel(name)
	if err != nil {
		log.Error("convert to result %v failed: %v", rmp.TypeName, err)
		return nil, err
	}
	setColumnValues(inst, rmp, mp)
	return reflect.Indirect(inst).Interface(), nil
}
func getResultType(resInfo types.SqlResult) reflect.Type {
	if resInfo.ResultM != nil {
		name := types.GetShortName(resInfo.ResultM.TypeName)
		inst, _ := gCache.createModel(name)
		return reflect.Indirect(inst).Type()
	}
	return resInfo.ResultT
}
func createResult(mp map[string]interface{}, resInfo types.SqlResult) (interface{}, error) {
	if resInfo.ResultM != nil {
		return convert2Result(mp, resInfo.ResultM)
	}
	if resInfo.ResultT.Kind() != reflect.Map {
		for _, v := range mp {
			return v, nil
		}
	}
	return mp, nil
}

func setColumnValues(value reflect.Value, rmp *types.ResultMap, mp map[string]interface{}) {
	outVal := value.Elem()
	outTyp := outVal.Type()
	for col, val := range mp {
		ritem, ok := rmp.ColumnMap[col]
		if !ok {
			log.Warn("result map %v dos not contains column %v", rmp.TypeName, col)
			continue
		}
		ftyp, ok := outTyp.FieldByName(ritem.Property)
		fval := outVal.FieldByName(ritem.Property)
		if !ok {
			pname := types.UpperFirst(ritem.Property)
			ftyp, ok = outTyp.FieldByName(pname)
			fval = outVal.FieldByName(pname)
			if !ok {
				log.Warn("not found result map %v binding property %v %v", rmp.TypeName, ritem.Property)
				continue
			}
		}
		rval, err := changeType(val, ftyp.Type)
		if err != nil {
			log.Warn("change `%v`to type %v failed: %v", val, ftyp.Type, err)
			continue
		}
		fval.Set(reflect.ValueOf(rval))
	}
}
