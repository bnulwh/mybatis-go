package orm

import (
	"database/sql"
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
	"github.com/bnulwh/mybatis-go/utils"
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

// backfillGeneratedKey 把 sql.Result.LastInsertId 回填到入参的 keyProperty 字段（S-11）。
// 入参为 struct 指针或 map 时回填生效；值传递的 struct 无法写回（Go 语义限制）。
func backfillGeneratedKey(arg ProxyArg, keyProperty string, result sql.Result) {
	id, err := result.LastInsertId()
	if err != nil {
		log.Warnf("get last insert id failed: %v", err)
		return
	}
	if id <= 0 {
		log.Warnf("last insert id is %v, skip keyProperty %v backfill", id, keyProperty)
		return
	}
	if arg.ArgsLen == 0 {
		return
	}
	val := arg.Args[0]
	if !val.IsValid() {
		return
	}
	iv := reflect.Indirect(val)
	if iv.Kind() == reflect.Map {
		iv.SetMapIndex(reflect.ValueOf(types.UpperFirst(keyProperty)), reflect.ValueOf(id))
		iv.SetMapIndex(reflect.ValueOf(keyProperty), reflect.ValueOf(id))
		return
	}
	if iv.Kind() != reflect.Struct {
		log.Warnf("keyProperty %v backfill: param kind %v not struct/map, skip", keyProperty, iv.Kind())
		return
	}
	field := iv.FieldByName(types.UpperFirst(keyProperty))
	if !field.IsValid() {
		field = iv.FieldByName(keyProperty)
	}
	if !field.IsValid() || !field.CanSet() {
		log.Warnf("keyProperty %v backfill: field not found or not settable (pass struct pointer to enable)", keyProperty)
		return
	}
	rval, err := utils.ChangeType(id, field.Type())
	if err != nil {
		log.Warnf("keyProperty %v backfill: change type failed: %v", keyProperty, err)
		return
	}
	field.Set(reflect.ValueOf(rval))
}

func (in *BaseMapper) executeMethod(sqlFunc *types.SqlFunction, arg ProxyArg) (val reflect.Value, err error) {
	//in.lock.Lock()
	//defer in.lock.Unlock()
	start := time.Now()
	defer sqlFunc.UpdateUsage(start, err == nil)
	log.Debugf("func: %v ,state : %v", sqlFunc, gDbConn.Statement)
	//log.Debugf("state: %v", gDbConn.Statement)
	args := arg.buildArgs()
	sqlStr, sqlargs, err := sqlFunc.GenerateSQL(args...)
	sqlStr = strings.ReplaceAll(sqlStr, "\n", " ")
	sqlStr = strings.ReplaceAll(sqlStr, "\t", " ")
	sqlStr = strings.ReplaceAll(sqlStr, "\r", " ")
	if err != nil {
		log.Warnf("generate sql failed: %v", err)
		return reflect.Value{}, err
	}
	log.Debugf("sql: %v", sqlStr)
	switch sqlFunc.Type {
	case types.InsertFunction, types.DeleteFunction, types.UpdateFunction:
		result, err := executeWithResult(sqlStr, sqlargs...)
		if err != nil {
			return reflect.Value{}, err
		}
		rf, _ := result.RowsAffected()
		// S-11：useGeneratedKeys + keyProperty 时把自增主键回填到入参
		if sqlFunc.Type == types.InsertFunction && sqlFunc.UseGeneratedKeys && sqlFunc.KeyProperty != "" {
			backfillGeneratedKey(arg, sqlFunc.KeyProperty, result)
		}
		return reflect.ValueOf(int64(rf)), nil
	case types.SelectFunction:
		rows, err := queryRows(sqlStr, sqlargs...)
		if err != nil {
			return reflect.Value{}, err
		}
		results := convert2Results(rows, sqlFunc.Result)
		if log.IsDebugEnabled() {
			log.Debugf("results: %v", types.ToJson(reflect.Indirect(results).Interface()))
		}
		return results, nil
	}
	return reflect.Value{}, fmt.Errorf("unsupport sql function type %v", sqlFunc.Type)
}
