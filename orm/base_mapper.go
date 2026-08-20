package orm

import (
	"context"
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

// normalizeSQL 把生成 SQL 里的换行/制表符规整为空格，便于日志与调试。
func normalizeSQL(sqlStr string) string {
	sqlStr = strings.ReplaceAll(sqlStr, "\n", " ")
	sqlStr = strings.ReplaceAll(sqlStr, "\t", " ")
	sqlStr = strings.ReplaceAll(sqlStr, "\r", " ")
	return sqlStr
}

// executeStream 以流式方式执行 select（P4-2）：返回 *RowStream，
// 由调用方逐行 Next() 消费并负责 Close()，不把整个结果集读进内存。
func (in *BaseMapper) executeStream(sqlFunc *types.SqlFunction, arg ProxyArg) (val reflect.Value, err error) {
	if sqlFunc.Type != types.SelectFunction {
		return reflect.Value{}, fmt.Errorf("%v.%v is not select, cannot stream", in.Namespace, sqlFunc.Id)
	}
	start := time.Now()
	defer sqlFunc.UpdateUsage(start, err == nil)
	args := arg.buildArgs()
	sqlStr, sqlargs, err := sqlFunc.GenerateSQL(args...)
	sqlStr = normalizeSQL(sqlStr)
	if err != nil {
		log.Warnf("generate sql failed: %v", err)
		return reflect.Value{}, err
	}
	log.Debugf("sql: %v", sqlStr)
	stream, err := QueryStream(context.Background(), sqlStr, sqlargs...)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(stream), nil
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
	sqlStr = normalizeSQL(sqlStr)
	if err != nil {
		log.Warnf("generate sql failed: %v", err)
		return reflect.Value{}, err
	}
	log.Debugf("sql: %v", sqlStr)
	switch sqlFunc.Type {
	case types.InsertFunction, types.DeleteFunction, types.UpdateFunction:
		result, err := executeWithResult(context.Background(), sqlStr, sqlargs...)
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
		rows, err := queryRows(context.Background(), sqlStr, sqlargs...)
		if err != nil {
			return reflect.Value{}, err
		}
		results, report := convert2Results(rows, sqlFunc.Result)
		// M-05：转换失败的行不再静默丢弃，聚合错误（行号/列名）输出，便于排查「0 行但 SQL 有数据」
		if report.Skipped > 0 || len(report.Errors) > 0 {
			log.Errorf("result convert report [%v.%v]: total=%d converted=%d skipped=%d errors=%v",
				in.Namespace, sqlFunc.Id, report.Total, report.Converted, report.Skipped, types.ToJson(report.Errors))
		}
		if log.IsDebugEnabled() {
			log.Debugf("results: %v", types.ToJson(reflect.Indirect(results).Interface()))
		}
		return results, nil
	}
	return reflect.Value{}, fmt.Errorf("unsupport sql function type %v", sqlFunc.Type)
}
