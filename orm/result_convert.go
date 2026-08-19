package orm

import (
	"database/sql"
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
	"github.com/bnulwh/mybatis-go/utils"
	"reflect"
)

// ResultConvertReport 汇总结果集转换期间的错误明细（M-05）。
// 修复前：转换失败的行仅 Warn 日志后静默丢弃，出现「0 行但 SQL 有数据」时难以排查；
// 修复后：按行号/列名聚合错误并返回，调用方（executeMethod）在 Skipped>0 或 Errors 非空时输出聚合日志。
type ResultConvertReport struct {
	Total     int                  // 原始行数
	Converted int                  // 成功转换的行数（含列级部分失败，字段保持零值）
	Skipped   int                  // 因转换失败被丢弃的行数
	Errors    []ResultConvertError // 错误明细（行级丢弃 + 列级类型不匹配）
}

// ResultConvertError 单条转换错误：Row 为 1-based 行号，Column 为列名（类型不匹配提示）。
type ResultConvertError struct {
	Row     int
	Column  string
	Message string
}

func convert2Results(rows []map[string]interface{}, resInfo types.SqlResult) (reflect.Value, *ResultConvertReport) {
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
	report := &ResultConvertReport{Total: len(rows)}
	for i, row := range rows {
		result, errs, err := convertMap2Result(row, resInfo, fieldIdx, i+1)
		if err != nil {
			report.Skipped++
			report.Errors = append(report.Errors, ResultConvertError{Row: i + 1, Message: err.Error()})
			continue
		}
		report.Converted++
		report.Errors = append(report.Errors, errs...)
		results = reflect.Append(results, reflect.ValueOf(result))
	}
	if log.IsDebugEnabled() {
		log.Debugf("results: %v", types.ToJson(results.Interface()))
	}
	//log.Infof("results ptr: %v", types.ToJson(reflect.Indirect(resultsPtr).Interface()))
	return results, report
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

func convert2Result(mp map[string]interface{}, rmp *types.ResultMap, fieldIdx map[string][]int, row int) (interface{}, []ResultConvertError, error) {
	name := types.GetShortName(rmp.TypeName)
	inst, err := gCache.createModel(name)
	if err != nil {
		log.Errorf("convert to result %v failed: %v", rmp.TypeName, err)
		return nil, nil, err
	}
	colErrs := setColumnValuesPrepared(inst, rmp, mp, fieldIdx)
	return reflect.Indirect(inst).Interface(), colErrs, nil
}
func getResultType(resInfo types.SqlResult) reflect.Type {
	if resInfo.ResultM != nil {
		name := types.GetShortName(resInfo.ResultM.TypeName)
		inst, _ := gCache.createModel(name)
		return reflect.Indirect(inst).Type()
	}
	return resInfo.ResultT
}
func convertMap2Result(mp map[string]interface{}, resInfo types.SqlResult, fieldIdx map[string][]int, row int) (interface{}, []ResultConvertError, error) {
	if resInfo.ResultM != nil {
		return convert2Result(mp, resInfo.ResultM, fieldIdx, row)
	}
	if resInfo.ResultT.Kind() != reflect.Map {
		for col, v := range mp {
			rval, err := utils.ChangeType(v, resInfo.ResultT)
			if err != nil {
				return nil, nil, fmt.Errorf("row %d column %q: %v", row, col, err)
			}
			return rval, nil, nil
		}
	}
	return mp, nil, nil
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

// setColumnValuesPrepared 按 resultMap 填充行对象；
// 列级类型转换失败的列跳过（行不丢弃），错误按列名聚合返回（M-05）。
func setColumnValuesPrepared(value reflect.Value, rmp *types.ResultMap, mp map[string]interface{}, fieldIdx map[string][]int) []ResultConvertError {
	outVal := value.Elem()
	var errs []ResultConvertError
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
			errs = append(errs, ResultConvertError{
				Column:  col,
				Message: fmt.Sprintf("change `%v` to type %v failed: %v", val, fval.Type(), err),
			})
			continue
		}
		fval.Set(reflect.ValueOf(rval))
	}
	return errs
}
