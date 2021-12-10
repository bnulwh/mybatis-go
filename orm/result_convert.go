package orm

import (
	"database/sql"
	log "github.com/bnulwh/logrus"
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
	for _, row := range rows {
		result, err := convertMap2Result(row, resInfo)
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
func prepareColumns(colTypes []*sql.ColumnType) []interface{} {
	var ptrs []interface{}
	for _, coltyp := range colTypes {
		log.Debugf("name: %v,dbtype: %v,scan type: %v %v %v %v",
			coltyp.Name(),
			coltyp.DatabaseTypeName(),
			coltyp.ScanType(),
			coltyp.ScanType().Kind(),
			coltyp.ScanType().Name(),
			coltyp.ScanType().String(),
		)
		ptrs = append(ptrs, newInstance(coltyp.ScanType()))
	}
	return ptrs
}
func createMap(ptrs []interface{}, colTypes []*sql.ColumnType) map[string]interface{} {
	mp := map[string]interface{}{}
	for i, coltyp := range colTypes {
		v, err := convertInstanceType(ptrs[i], coltyp.ScanType())
		if err != nil {
			log.Warnf("convert %v to %v %v failed: %v", ptrs[i], coltyp.Name(), coltyp.ScanType(), err)
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
		log.Errorf("convert to result %v failed: %v", rmp.TypeName, err)
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
func convertMap2Result(mp map[string]interface{}, resInfo types.SqlResult) (interface{}, error) {
	if resInfo.ResultM != nil {
		return convert2Result(mp, resInfo.ResultM)
	}
	if resInfo.ResultT.Kind() != reflect.Map {
		for _, v := range mp {
			return utils.ChangeType(v, resInfo.ResultT)
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
			log.Warnf("result map %v dos not contains column %v", rmp.TypeName, col)
			continue
		}
		ftyp, ok := outTyp.FieldByName(ritem.Property)
		fval := outVal.FieldByName(ritem.Property)
		if !ok {
			pname := types.UpperFirst(ritem.Property)
			ftyp, ok = outTyp.FieldByName(pname)
			fval = outVal.FieldByName(pname)
			if !ok {
				log.Warnf("not found result map %v binding property %v", rmp.TypeName, ritem.Property)
				continue
			}
		}
		rval, err := utils.ChangeType(val, ftyp.Type)
		if err != nil {
			log.Warnf("change `%v`to type %v failed: %v", val, ftyp.Type, err)
			continue
		}
		fval.Set(reflect.ValueOf(rval))
	}
}
