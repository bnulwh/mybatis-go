package orm

import (
	"database/sql"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
	"github.com/bnulwh/mybatis-go/utils"
	"reflect"
)

func convert2Results(rows []map[string]interface{}, resInfo types.SqlResult) reflect.Value {
	//var results []interface{}
	itemTyp := getResultType(resInfo)
	itemsTyp := reflect.SliceOf(itemTyp)
	resultsPtr := reflect.New(itemsTyp)
	results := reflect.Indirect(resultsPtr)
	// P1-4：resultMap 的 property -> 字段索引 只预编译一次，跨行复用
	var fieldIdx map[string][]int
	if resInfo.ResultM != nil {
		fieldIdx = buildFieldIndexMap(itemTyp, resInfo.ResultM)
	}
	for _, row := range rows {
		result, err := convertMap2Result(row, resInfo, fieldIdx)
		if err != nil {
			log.Warnf("fill result failed: %v", err)
			continue
		}
		results = reflect.Append(results, reflect.ValueOf(result))
	}
	log.Debugf("results: %v", types.ToJson(results.Interface()))
	//log.Infof("results ptr: %v", types.ToJson(reflect.Indirect(resultsPtr).Interface()))
	return results
}

// scanTargetType 返回列的扫描目标类型；部分驱动在首行前不填充 ScanType，
// 此时回退到通用的 sql.NullString 目标（newInstance 对未知类型同样返回 NullString）。
func scanTargetType(coltyp *sql.ColumnType) reflect.Type {
	if st := coltyp.ScanType(); st != nil {
		return st
	}
	return reflect.TypeOf(sql.NullString{})
}

func prepareColumns(colTypes []*sql.ColumnType) []interface{} {
	var ptrs []interface{}
	for _, coltyp := range colTypes {
		log.Debugf("name: %v,dbtype: %v,scan type: %v",
			coltyp.Name(), coltyp.DatabaseTypeName(), coltyp.ScanType())
		ptrs = append(ptrs, newInstance(scanTargetType(coltyp)))
	}
	return ptrs
}
func createMap(ptrs []interface{}, colTypes []*sql.ColumnType) map[string]interface{} {
	return createMapWithConverters(ptrs, colTypes, buildConverters(colTypes))
}

// createMapWithConverters 使用预编译的转换函数表构造行 map（P1-4）。
func createMapWithConverters(ptrs []interface{}, colTypes []*sql.ColumnType, converters []convertFn) map[string]interface{} {
	mp := map[string]interface{}{}
	for i, coltyp := range colTypes {
		v, err := converters[i](ptrs[i])
		if err != nil {
			log.Warnf("convert %v to %v %v failed: %v", ptrs[i], coltyp.Name(), coltyp.ScanType(), err)
			continue
		}
		mp[coltyp.Name()] = v
	}
	return mp
}

func convert2Result(mp map[string]interface{}, rmp *types.ResultMap, fieldIdx map[string][]int) (interface{}, error) {
	name := types.GetShortName(rmp.TypeName)
	inst, err := gCache.createModel(name)
	if err != nil {
		log.Errorf("convert to result %v failed: %v", rmp.TypeName, err)
		return nil, err
	}
	setColumnValuesPrepared(inst, rmp, mp, fieldIdx)
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
func convertMap2Result(mp map[string]interface{}, resInfo types.SqlResult, fieldIdx map[string][]int) (interface{}, error) {
	if resInfo.ResultM != nil {
		return convert2Result(mp, resInfo.ResultM, fieldIdx)
	}
	if resInfo.ResultT.Kind() != reflect.Map {
		for _, v := range mp {
			return utils.ChangeType(v, resInfo.ResultT)
		}
	}
	return mp, nil
}

// buildFieldIndexMap 预编译 resultMap 的 property -> 字段索引 映射，
// 避免每行两次 FieldByName O(N) 名称匹配（P1-4）。
func buildFieldIndexMap(outTyp reflect.Type, rmp *types.ResultMap) map[string][]int {
	idxMap := make(map[string][]int, len(rmp.ColumnMap))
	for _, ritem := range rmp.ColumnMap {
		if f, ok := outTyp.FieldByName(ritem.Property); ok {
			idxMap[ritem.Property] = f.Index
			continue
		}
		pname := types.UpperFirst(ritem.Property)
		if f, ok := outTyp.FieldByName(pname); ok {
			idxMap[ritem.Property] = f.Index
		}
	}
	return idxMap
}

func setColumnValues(value reflect.Value, rmp *types.ResultMap, mp map[string]interface{}) {
	setColumnValuesPrepared(value, rmp, mp, buildFieldIndexMap(value.Elem().Type(), rmp))
}

func setColumnValuesPrepared(value reflect.Value, rmp *types.ResultMap, mp map[string]interface{}, fieldIdx map[string][]int) {
	outVal := value.Elem()
	for col, val := range mp {
		ritem, ok := rmp.ColumnMap[col]
		if !ok {
			log.Warnf("result map %v dos not contains column %v", rmp.TypeName, col)
			continue
		}
		idx, ok := fieldIdx[ritem.Property]
		if !ok {
			log.Warnf("not found result map %v binding property %v", rmp.TypeName, ritem.Property)
			continue
		}
		fval := outVal.FieldByIndex(idx)
		rval, err := utils.ChangeType(val, fval.Type())
		if err != nil {
			log.Warnf("change `%v`to type %v failed: %v", val, fval.Type(), err)
			continue
		}
		fval.Set(reflect.ValueOf(rval))
	}
}
