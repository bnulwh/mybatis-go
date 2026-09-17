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
	// P0-1：显式 ${ew} 占位模式，Wrapper 以 ew 键并入渲染参数（生成 SQL 前注入）
	wrapperExplicit := arg.Wrapper != nil && hasNamedSlot(sqlFunc, "ew")
	if wrapperExplicit {
		args = injectWrapperArg(args, arg.Wrapper)
	}
	sqlStr, sqlargs, err := sqlFunc.GenerateSQL(args...)
	sqlStr = normalizeSQL(sqlStr)
	if err != nil {
		log.Warnf("generate sql failed: %v", err)
		return reflect.Value{}, err
	}
	// P0-1：自动注入模式（语句无 ${ew} 占位时改写 SQL；stream 仅 select）
	if arg.Wrapper != nil && !wrapperExplicit {
		sqlStr = applyQueryWrapper(sqlStr, arg.Wrapper)
	}
	log.Debugf("sql: %v", sqlStr)
	// P0-5：Before 钩子（可改写 SQL），After 钩子与 UpdateUsage defer 并列
	// G0：ctx 来自 Mapper 方法的 context.Context 参数（无则 Background，可携带 WithTx 事务）
	sqlStr = runBeforeHooks(arg.Ctx, in.Namespace, sqlFunc.Id, string(sqlFunc.Type), sqlStr, sqlargs)
	defer func() {
		runAfterHooks(arg.Ctx, in.Namespace, sqlFunc.Id, string(sqlFunc.Type), sqlStr, sqlargs, err, time.Since(start))
	}()
	stream, err := QueryStream(arg.Ctx, sqlStr, sqlargs...)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(stream), nil
}

// needsReturning 判断当前数据库是否需要通过 RETURNING 子句获取自增主键。
// PostgreSQL / KingbaseES 的 LastInsertId() 返回 error，需改用 INSERT ... RETURNING。
func needsReturning() bool {
	if gDbConn == nil || gDbConn.Dialector == nil {
		return false
	}
	return gDbConn.Dialector.NeedsReturning()
}

// keyColumnToSnake 把 keyProperty 驼峰名转为下划线列名（id → id, jobId → job_id, CreateTime → create_time）。
// 若 keyColumn 已显式指定则不走此函数（直接用 keyColumn）。
func keyColumnToSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteByte(byte(r + 32)) // toLower
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
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

// backfillGeneratedKeyFromRows 把 RETURNING 子句返回的自增主键回填到入参的 keyProperty 字段（M-03）。
// PostgreSQL / KingbaseES 不支持 LastInsertId()，改用 INSERT ... RETURNING col 读取生成的 ID。
func backfillGeneratedKeyFromRows(arg ProxyArg, keyProperty string, rows *sql.Rows) {
	if arg.ArgsLen == 0 || !rows.Next() {
		return
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil || len(colTypes) == 0 {
		log.Warnf("RETURNING column types failed: %v", err)
		return
	}
	scanTarget := newInstance(scanTargetType(colTypes[0]))
	converters := buildConvertersBasic(colTypes)
	if err := rows.Scan(scanTarget); err != nil {
		log.Warnf("RETURNING scan failed: %v", err)
		return
	}
	id, err := converters[0](scanTarget)
	if err != nil {
		log.Warnf("RETURNING convert failed: %v", err)
		return
	}
	idInt, ok := toInt64(id)
	if !ok {
		log.Warnf("RETURNING value %v cannot convert to int64", id)
		return
	}
	if idInt <= 0 {
		log.Warnf("RETURNING id is %v, skip keyProperty %v backfill", idInt, keyProperty)
		return
	}
	val := arg.Args[0]
	if !val.IsValid() {
		return
	}
	iv := reflect.Indirect(val)
	if iv.Kind() == reflect.Map {
		iv.SetMapIndex(reflect.ValueOf(types.UpperFirst(keyProperty)), reflect.ValueOf(idInt))
		iv.SetMapIndex(reflect.ValueOf(keyProperty), reflect.ValueOf(idInt))
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
		log.Warnf("keyProperty %v backfill: field not found or not settable", keyProperty)
		return
	}
	rval, err := utils.ChangeType(idInt, field.Type())
	if err != nil {
		log.Warnf("keyProperty %v backfill: change type failed: %v", keyProperty, err)
		return
	}
	field.Set(reflect.ValueOf(rval))
}

// toInt64 将转换函数返回的接口值转为 int64（RETURNING 列为整数类型时）。
func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case int8:
		return int64(n), true
	case int16:
		return int64(n), true
	case uint:
		return int64(n), true
	case uint64:
		return int64(n), true
	case uint8:
		return int64(n), true
	case uint16:
		return int64(n), true
	case uint32:
		return int64(n), true
	case float64:
		return int64(n), true
	case float32:
		return int64(n), true
	}
	return 0, false
}

func (in *BaseMapper) executePage(sqlFunc *types.SqlFunction, arg ProxyArg) (val reflect.Value, err error) {
	pp := arg.PageParam
	if pp == nil || !pp.Valid() {
		return reflect.Value{}, fmt.Errorf("PageParam is nil or invalid (PageNum=%d, PageSize=%d)", pp.PageNum, pp.PageSize)
	}
	start := time.Now()
	defer sqlFunc.UpdateUsage(start, err == nil)
	args := arg.buildArgs()
	// P0-1：显式 ${ew} 占位模式，Wrapper 以 ew 键并入渲染参数（生成 SQL 前注入）
	wrapperExplicit := arg.Wrapper != nil && hasNamedSlot(sqlFunc, "ew")
	if wrapperExplicit {
		args = injectWrapperArg(args, arg.Wrapper)
	}
	sqlStr, sqlargs, err := sqlFunc.GenerateSQL(args...)
	sqlStr = normalizeSQL(sqlStr)
	if err != nil {
		log.Warnf("generate sql failed: %v", err)
		return reflect.Value{}, err
	}
	// P0-1：自动注入模式，在 buildCountSQL 之前注入（count/page 天然一致）
	if arg.Wrapper != nil && !wrapperExplicit {
		sqlStr = applyQueryWrapper(sqlStr, arg.Wrapper)
	}
	// P0-5：Before 钩子一对包裹 count+page 全程（在 buildCountSQL 之前注入，保证 count/page 一致）
	sqlStr = runBeforeHooks(arg.Ctx, in.Namespace, sqlFunc.Id, string(sqlFunc.Type), sqlStr, sqlargs)
	defer func() {
		runAfterHooks(arg.Ctx, in.Namespace, sqlFunc.Id, string(sqlFunc.Type), sqlStr, sqlargs, err, time.Since(start))
	}()
	countSQL := normalizeSQL(buildCountSQL(sqlStr))
	log.Debugf("page count sql: %v", countSQL)
	var total int64
	crows, cerr := queryRows(arg.Ctx, countSQL, sqlargs...)
	if cerr != nil {
		log.Warnf("page count query failed: %v", cerr)
	} else if len(crows) > 0 {
		for _, v := range crows[0] {
			if n, ok := toInt64(v); ok {
				total = n
				break
			}
		}
	}
	pageSQL := applyPagination(sqlStr, pp.Limit(), pp.Offset())
	log.Debugf("page sql: %v", pageSQL)
	rows, err := queryRows(arg.Ctx, pageSQL, sqlargs...)
	if err != nil {
		return reflect.Value{}, err
	}
	page := &Page{
		Total:    total,
		Records:  rows,
		PageNum:  pp.PageNum,
		PageSize: pp.PageSize,
	}
	return reflect.ValueOf(page), nil
}

func (in *BaseMapper) executeMethod(sqlFunc *types.SqlFunction, arg ProxyArg) (val reflect.Value, err error) {
	//in.lock.Lock()
	//defer in.lock.Unlock()
	start := time.Now()
	defer sqlFunc.UpdateUsage(start, err == nil)
	log.Debugf("func: %v ,state : %v", sqlFunc, gDbConn.Statement)
	//log.Debugf("state: %v", gDbConn.Statement)
	args := arg.buildArgs()
	// P0-1：显式 ${ew} 占位模式，Wrapper 以 ew 键并入渲染参数（生成 SQL 前注入）
	wrapperExplicit := arg.Wrapper != nil && hasNamedSlot(sqlFunc, "ew")
	if wrapperExplicit {
		args = injectWrapperArg(args, arg.Wrapper)
	}
	sqlStr, sqlargs, err := sqlFunc.GenerateSQL(args...)
	sqlStr = normalizeSQL(sqlStr)
	if err != nil {
		log.Warnf("generate sql failed: %v", err)
		return reflect.Value{}, err
	}
	// P0-1：自动注入模式（语句无 ${ew} 占位时改写 SQL；P0 限定 select）
	if arg.Wrapper != nil && !wrapperExplicit {
		if sqlFunc.Type != types.SelectFunction {
			return reflect.Value{}, fmt.Errorf("%v.%v: QueryWrapper only support select functions", in.Namespace, sqlFunc.Id)
		}
		sqlStr = applyQueryWrapper(sqlStr, arg.Wrapper)
	}
	log.Debugf("sql: %v", sqlStr)
	// P0-5：Before 钩子（可改写 SQL），After 钩子与 UpdateUsage defer 并列（覆盖全部执行路径）
	sqlStr = runBeforeHooks(arg.Ctx, in.Namespace, sqlFunc.Id, string(sqlFunc.Type), sqlStr, sqlargs)
	defer func() {
		runAfterHooks(arg.Ctx, in.Namespace, sqlFunc.Id, string(sqlFunc.Type), sqlStr, sqlargs, err, time.Since(start))
	}()
	switch sqlFunc.Type {
	case types.InsertFunction, types.DeleteFunction, types.UpdateFunction:
		// M-03：PostgreSQL / KingbaseES 不支持 LastInsertId()，
		// 通过 RETURNING 子句读取自增主键（INSERT ... RETURNING col → Query + Scan）。
		// MySQL / SQLite 仍走 LastInsertId() 路径，行为不变。
		if sqlFunc.Type == types.InsertFunction && sqlFunc.UseGeneratedKeys && sqlFunc.KeyProperty != "" && needsReturning() {
			keyCol := sqlFunc.KeyColumn
			if keyCol == "" {
				keyCol = keyColumnToSnake(sqlFunc.KeyProperty)
			}
			returningSQL := sqlStr + " RETURNING " + keyCol
			ctx, cancel := withExecTimeout(arg.Ctx)
			defer cancel()
			rows, qErr := gDbConn.QueryContext(ctx, returningSQL, sqlargs...)
			if qErr != nil {
				log.Errorf("execute RETURNING query failed: %v", qErr)
				return reflect.Value{}, qErr
			}
			defer rows.Close()
			backfillGeneratedKeyFromRows(arg, sqlFunc.KeyProperty, rows)
			return reflect.ValueOf(int64(1)), nil
		}
		result, err := executeWithResult(arg.Ctx, sqlStr, sqlargs...)
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
		rows, err := queryRows(arg.Ctx, sqlStr, sqlargs...)
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
