package orm

import (
	"fmt"
	log "github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/types"
	"reflect"
	"strings"
)

type BaseMapper struct {
	mapper *types.SqlMapper
	//lock   sync.Mutex
}

func (in *BaseMapper) fetchSqlFunction(name string) (*types.SqlFunction, error) {
	item, ok := in.mapper.NamedFunctions[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("%s not contains function %s", in.mapper.Namespace, name)
	}
	return item, nil
}

func (in *BaseMapper) executeMethod(sqlFunc *types.SqlFunction, arg ProxyArg) (reflect.Value, error) {
	//in.lock.Lock()
	//defer in.lock.Unlock()
	args := arg.buildArgs()
	sqlStr, items, err := sqlFunc.GenerateSQL(args...)
	if err != nil {
		log.Warnf("generate sql failed: %v", err)
		return reflect.Value{}, err
	}
	log.Debugf("sql: %v", sqlStr)
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
		log.Debugf("results: %v", types.ToJson(results.Interface()))
		log.Debugf("results: %v", types.ToJson(reflect.Indirect(results).Interface()))
		return results, nil
	}
	return reflect.Value{}, nil
}
