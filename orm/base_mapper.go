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
	sqlStr, err := sqlFunc.GenerateSQL(in.mapper, args)
	if err != nil {
		log.Warn("generate sql failed: %v", err)
		return reflect.Value{}, err
	}
	log.Info("sql: %v", sqlStr)
	stmt, err := gDbConn.Prepare(sqlStr)
	if err != nil {
		log.Error("prepare sql %v failed: %v", sqlStr, err)
		return reflect.Value{}, err
	}
	defer closeStmt(stmt)
	switch sqlFunc.Type {
	case types.InsertSQL, types.DeleteSQL, types.UpdateSQL:
		result, err := stmt.Exec()
		if err != nil {
			log.Error("execute sql %v failed: %v", sqlStr, err)
			return reflect.Value{}, err
		}
		rf, _ := result.RowsAffected()
		return reflect.ValueOf(rf), nil
	case types.SelectSQL:
		results, err := queryRows(sqlStr, sqlFunc.Result)
		if err != nil {
			return reflect.Value{}, err
		}
		log.Info("results: %v", types.ToJson(results.Interface()))
		log.Info("results: %v", types.ToJson(reflect.Indirect(results).Interface()))
		return results, nil
	}
	return reflect.Value{}, nil
}

// func (in *BaseMapper) fetchExecuteFunc(name string) ExecuteFunc {
// 	item, ok := in.mapper.NamedFunctions[strings.ToLower(name)]
// 	if !ok {
// 		panic(fmt.Sprintf("%s not contains function %s", in.mapper.Namespace, name))
// 	}
// 	if item.Type == types.SelectSQL {
// 		panic(fmt.Sprintf("%s.%s  function define select operation", in.mapper.Namespace, name))
// 	}
// 	return func(args ...interface{}) (int64, int64, error) {
// 		gLock.Lock()
// 		defer gLock.Unlock()
// 		sqlStr, err := item.GenerateSQL(in.mapper, args)
// 		if err != nil {
// 			return 0, 0, err
// 		}
// 		log.Debug("sql: %v", sqlStr)
// 		if len(sqlStr) == 0 {
// 			return 0, 0, fmt.Errorf("generate sql failed. args: %v", args)
// 		}
// 		stmt, err := gDbConn.Prepare(sqlStr)
// 		if err != nil {
// 			log.Error("prepare sql %v failed: %v", sqlStr, err)
// 			return 0, 0, err
// 		}
// 		defer closeStmt(stmt)
// 		result, err := stmt.Exec()
// 		if err != nil {
// 			log.Error("execute sql %v failed: %v", sqlStr, err)
// 			return 0, 0, err
// 		}
// 		affected, _ := result.RowsAffected()
// 		id, _ := result.LastInsertId()
// 		return affected, id, nil
// 	}
// }

//func (in *BaseMapper) fetchQueryRowFunc(name string) QueryRowFunc {
//	item, ok := in.mapper.NamedFunctions[strings.ToLower(name)]
//	if !ok {
//		panic(fmt.Sprintf("%s not contains function %s", in.mapper.Namespace, name))
//	}
//	if item.Type != types.SelectSQL {
//		panic(fmt.Sprintf("%s.%s  function not define select operation", in.mapper.Namespace, name))
//	}
//	return func(args ...interface{}) (interface{}, error) {
//		gLock.Lock()
//		defer gLock.Unlock()
//		sqlStr, err := item.GenerateSQL(in.mapper, args)
//		if err != nil {
//			return nil, err
//		}
//		log.Debug("sql: %v", sqlStr)
//		if len(sqlStr) == 0 {
//			return nil, fmt.Errorf("generate sql failed. args: %v", args)
//		}
//		results, err := queryRows(sqlStr, item.Result)
//		if err != nil {
//			return nil, err
//		}
//		return results.Index(0).Interface(), err
//	}
//}
//func (in *BaseMapper) fetchQueryRowsFunc(name string) QueryRowFunc {
//	item, ok := in.mapper.NamedFunctions[strings.ToLower(name)]
//	if !ok {
//		panic(fmt.Sprintf("%s not contains function %s", in.mapper.Namespace, name))
//	}
//	if item.Type != types.SelectSQL {
//		panic(fmt.Sprintf("%s.%s  function not define select operation", in.mapper.Namespace, name))
//	}
//	return func(args ...interface{}) (interface{}, error) {
//		gLock.Lock()
//		defer gLock.Unlock()
//		sqlStr, err := item.GenerateSQL(in.mapper, args)
//		if err != nil {
//			return nil, err
//		}
//		log.Debug("sql: %v", sqlStr)
//		if len(sqlStr) == 0 {
//			return nil, fmt.Errorf("generate sql failed. args: %v", args)
//		}
//		return queryRows(sqlStr, item.Result)
//	}
//}

//func fillReturnValueWithSlice(results reflect.Value, returnValue *reflect.Value) {
//	returnValue = &results
//}
//
//func fillReturnValueWithSqlResult(result sql.Result, returnValue *reflect.Value) {
//	rf, _ := result.RowsAffected()
//	*returnValue = reflect.ValueOf(&rf)
//}

func closeStmt(stmt *sql.Stmt) {
	err := stmt.Close()
	if err != nil {
		log.Warn("close warning: %v", err)
	}
}

func queryRows(sqlStr string, resInfo types.SqlResult) (reflect.Value, error) {
	stmt, err := gDbConn.Prepare(sqlStr)
	if err != nil {
		log.Error("prepare sql %v failed: %v", sqlStr, err)
		return reflect.Value{}, err
	}
	defer closeStmt(stmt)
	rows, err := stmt.Query()
	if err != nil {
		log.Error("query sql %v failed: %v", sqlStr, err)
		return reflect.Value{}, err
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		log.Error("fill sql %v result failed: %v", sqlStr, err)
		return reflect.Value{}, err
	}
	results := fetchRows(rows, colTypes, resInfo)
	return results, nil
}
func fetchRows(rows *sql.Rows, colTypes []*sql.ColumnType, resInfo types.SqlResult) reflect.Value {
	//var results []interface{}
	itemTyp := getResultType(resInfo)
	itemsTyp := reflect.SliceOf(itemTyp)
	resultsPtr := reflect.New(itemsTyp)
	results := reflect.Indirect(resultsPtr)
	for rows.Next() {
		tempItems := prepareColumns(colTypes)
		err := rows.Scan(tempItems...)
		if err != nil {
			log.Warn("scan error: %v", err)
			continue
		}
		result, err := createResult(colTypes, resInfo, tempItems)
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
func createMap(colTypes []*sql.ColumnType, ptrs []interface{}) map[string]interface{} {
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

func convert2Result(rmp *types.ResultMap, mp map[string]interface{}) (interface{}, error) {
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
func createResult(colTypes []*sql.ColumnType, resInfo types.SqlResult, ptrs []interface{}) (interface{}, error) {
	mp := createMap(colTypes, ptrs)
	if len(ptrs) == 1 {
		return mp[colTypes[0].Name()], nil
	}
	if resInfo.ResultM != nil {
		return convert2Result(resInfo.ResultM, mp)
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
