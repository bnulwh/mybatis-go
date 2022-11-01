package orm

import (
	"fmt"
	log "github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/types"
	"reflect"
	"strings"
	"time"
)

type BaseMapper struct {
	*types.SqlMapper
	//lock   sync.Mutex
}

func (in *BaseMapper) fetchSqlFunction(name string) (*types.SqlFunction, error) {
	item, ok := in.NamedFunctions[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("%s not contains function %s", in.Namespace, name)
	}
	return item, nil
}

func (in *BaseMapper) executeMethod(sqlFunc *types.SqlFunction, arg ProxyArg) (val reflect.Value, err error) {
	//in.lock.Lock()
	//defer in.lock.Unlock()
	start := time.Now()
	defer sqlFunc.UpdateUsage(start, err == nil)
	log.Debugf("func: %v", sqlFunc)
	log.Debugf("state: %v", gDbConn.Statement)
	args := arg.buildArgs()
	sqlStr, sqlargs, err := sqlFunc.GenerateSQL(args...)
	sqlStr = strings.ReplaceAll(sqlStr, "\n", " ")
	sqlStr = strings.ReplaceAll(sqlStr, "\t", " ")
	sqlStr = strings.ReplaceAll(sqlStr, "\r", " ")
	//sqlStr = gDbConn.FormatPrepareSQL(sqlStr)
	//sqlargs := convert2Interfaces(items)
	if err != nil {
		log.Warnf("generate sql failed: %v", err)
		return reflect.Value{}, err
	}
	log.Debugf("sql: %v", sqlStr)
	switch sqlFunc.Type {
	case types.InsertFunction, types.DeleteFunction, types.UpdateFunction:
		rf, err := execute(sqlStr, sqlargs...)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(rf), nil
	case types.SelectFunction:
		rows, err := queryRows(sqlStr, sqlargs...)
		if err != nil {
			return reflect.Value{}, err
		}
		results := convert2Results(rows, sqlFunc.Result)
		log.Debugf("results: %v", types.ToJson(results.Interface()))
		log.Debugf("results: %v", types.ToJson(reflect.Indirect(results).Interface()))
		return results, nil
	}
	return reflect.Value{}, nil
}

func convert2Interfaces(arr []string) []interface{} {
	var results []interface{}
	for _, s := range arr {
		results = append(results, s)
	}
	return results
}
