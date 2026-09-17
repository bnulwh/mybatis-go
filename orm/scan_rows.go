package orm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/bnulwh/mybatis-go/log"
)

// 强类型扫描直通道（G0）：
//
// 在 fetchRows（rows → []map[string]interface{} → 反射 map→struct）之外，提供
// 「rows → struct/slice」的直通道：扫描缓冲一次分配、目标字段索引一次预编译，
// 每行 Scan 后直接写入目标字段，不经 map 中转。类型转换优先自定义 TypeHandler
// （P0-4），未注册回退内置 utils.ChangeType；列名匹配优先模型 db tag 显式列名
// （P0-2），未声明回退与 RowStream.Scan 相同的四策略（原名 → 首字母大写 →
// 蛇形转驼峰 → 大小写不敏感）。
//
// 该通道是 gormish 链式 API Find/Scan（可行性报告 §2.4/§4.2）与 sqlc 生成代码
// 可选运行时（§3.3）的地基。

// QueryTo 执行查询并把结果直接扫描到 dst（G0，不经 map 中转）。
// dst 支持：
//   - *struct：填充首行；无行返回 sql.ErrNoRows；
//   - *[]T / *[]*T（T 为 struct）：填充全部行，遵循全局行数上限（DefaultRowLimit）；
//   - *[]map[string]interface{} / *[]T（T 为标量，单列结果）：填充全部行（S1 扩展）；
//   - *map[string]interface{}：填充首行；无行返回 sql.ErrNoRows；
//   - 其他可寻址指针（*int64/*string/...）：单列首行；无行返回 sql.ErrNoRows。
//
// 自动叠加全局默认超时（P4-1）；ctx 携带 WithTx 事务时自动在事务内执行。
func QueryTo(dst interface{}, sqlStr string, args ...interface{}) error {
	return QueryToContext(context.Background(), dst, sqlStr, args...)
}

// QueryToContext 与 QueryTo 相同，但支持传入 context（超时控制 / WithTx 事务）。
func QueryToContext(ctx context.Context, dst interface{}, sqlStr string, args ...interface{}) error {
	if gDbConn == nil {
		return fmt.Errorf("connection not init.")
	}
	return gDbConn.QueryToContext(ctx, dst, sqlStr, args...)
}

// QueryToContext 实例级扫描直通道（G1）：作用于指定 DB 实例（多数据源 / gormish 场景），
// 自动应用表名前缀改写与占位符格式化，ctx 携带 WithTx 事务时自动在事务内执行。
func (db *DB) QueryToContext(ctx context.Context, dst interface{}, sqlStr string, args ...interface{}) error {
	log.Debugf("sql: %v", sqlStr)
	ctx, cancel := withExecTimeout(ctx)
	defer cancel()
	rows, err := db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		log.Errorf("query sql %v failed: %v", sqlStr, err)
		return err
	}
	defer rows.Close()
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		log.Errorf("fill sql %v result failed: %v", sqlStr, err)
		return err
	}
	schemaHints := buildSchemaHints(sqlStr, colTypes)
	return scanRowsInto(rows, colTypes, schemaHints, dst)
}

// scanRowsInto 按 dst 形态分派扫描策略（G0）。
func scanRowsInto(rows *sql.Rows, colTypes []*sql.ColumnType, schemaHints map[string]*columnSchemaHint, dst interface{}) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("QueryTo: dst must be a non-nil pointer, got %T", dst)
	}
	ev := rv.Elem()
	switch {
	case ev.Kind() == reflect.Ptr:
		return fmt.Errorf("QueryTo: dst must not be pointer-to-pointer, got %v", ev.Type())
	case ev.Kind() == reflect.Struct:
		return scanRowsToStruct(rows, colTypes, schemaHints, ev)
	case ev.Kind() == reflect.Slice:
		return scanRowsToSlice(rows, colTypes, schemaHints, ev)
	case ev.Kind() == reflect.Map && ev.Type().Key().Kind() == reflect.String:
		return scanRowsToMapRow(rows, colTypes, schemaHints, ev)
	default:
		return scanRowsToScalar(rows, colTypes, schemaHints, ev)
	}
}

// scanBinding 单列扫描绑定（G0）：扫描缓冲 + 转换函数 + 目标字段索引一次预编译。
type scanBinding struct {
	buf       interface{}   // prepareColumns 分配的 *sql.Null* 扫描缓冲（复用一次，逐行覆写）
	converter convertFn     // 缓冲值 → Go 值（buildConverters，schemaHint 感知）
	fieldIdx  []int         // 目标字段索引；nil 表示该列无匹配字段（仍需扫描占位）
	fieldType reflect.Type  // 目标字段类型（convertFieldValue 的目标）
}

// buildScanBindings 为「列 → struct 字段」构建预编译绑定表（G0）。
func buildScanBindings(structType reflect.Type, colTypes []*sql.ColumnType, schemaHints map[string]*columnSchemaHint, tagIndex map[string][]int) []scanBinding {
	bufs := prepareColumns(colTypes)
	converters := buildConverters(colTypes, schemaHints)
	bindings := make([]scanBinding, len(colTypes))
	for i, ct := range colTypes {
		b := scanBinding{buf: bufs[i], converter: converters[i]}
		if idx := findScanFieldIndex(structType, ct.Name(), tagIndex); idx != nil {
			b.fieldIdx = idx
			b.fieldType = structType.FieldByIndex(idx).Type
		}
		bindings[i] = b
	}
	return bindings
}

// scanBuffers 提取绑定表的扫描缓冲列表。
func scanBuffers(bindings []scanBinding) []interface{} {
	bufs := make([]interface{}, len(bindings))
	for i, b := range bindings {
		bufs[i] = b.buf
	}
	return bufs
}

// fillStructFromBindings 把一行扫描缓冲写入目标 struct（G0）。
// 列级失败聚合为一条错误返回（M-05 / RowStream.Scan 风格），不静默丢列。
func fillStructFromBindings(bindings []scanBinding, colTypes []*sql.ColumnType, ev reflect.Value) error {
	var errs []string
	for i, b := range bindings {
		if b.fieldIdx == nil {
			continue
		}
		val, err := b.converter(b.buf)
		if err != nil {
			errs = append(errs, fmt.Sprintf("column %q: %v", colTypes[i].Name(), err))
			continue
		}
		rval, err := convertFieldValue(val, b.fieldType)
		if err != nil {
			errs = append(errs, fmt.Sprintf("column %q -> field %s: %v", colTypes[i].Name(), structTypeFieldName(ev.Type(), b.fieldIdx), err))
			continue
		}
		ev.FieldByIndex(b.fieldIdx).Set(reflect.ValueOf(rval))
	}
	if len(errs) > 0 {
		return fmt.Errorf("scan row: %v", strings.Join(errs, "; "))
	}
	return nil
}

// structTypeFieldName 返回字段索引对应的字段名（错误信息用）。
func structTypeFieldName(typ reflect.Type, idx []int) string {
	if len(idx) == 0 {
		return "<unknown>"
	}
	return typ.FieldByIndex(idx).Name
}

// buildTagColumnIndex 从模型 db tag 元数据构建「列名 → 字段索引」（G0）。
// 无 db tag 的模型返回 nil，字段匹配回退 findStreamField 四策略。
func buildTagColumnIndex(structType reflect.Type) map[string][]int {
	info := GetModelInfo(reflect.New(structType).Interface())
	if info == nil {
		return nil
	}
	idx := make(map[string][]int, len(info.Fields))
	for _, f := range info.Fields {
		if sf, ok := structType.FieldByName(f.Name); ok {
			idx[f.Column] = sf.Index
		}
	}
	if len(idx) == 0 {
		return nil
	}
	return idx
}

// findScanFieldIndex 按列名查找目标字段索引（G0）：
// 优先 db tag 显式列名（含非蛇形改名），未命中回退 findStreamField 四策略。
func findScanFieldIndex(typ reflect.Type, col string, tagIndex map[string][]int) []int {
	if tagIndex != nil {
		if idx, ok := tagIndex[col]; ok {
			return idx
		}
	}
	if f := findStreamField(typ, col); f != nil {
		return f.Index
	}
	return nil
}

// scanRowsToStruct 首行填充到 struct（G0）；无行返回 sql.ErrNoRows。
func scanRowsToStruct(rows *sql.Rows, colTypes []*sql.ColumnType, schemaHints map[string]*columnSchemaHint, ev reflect.Value) error {
	structType := ev.Type()
	bindings := buildScanBindings(structType, colTypes, schemaHints, buildTagColumnIndex(structType))
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	if err := rows.Scan(scanBuffers(bindings)...); err != nil {
		return fmt.Errorf("scan row 1 failed: %w", err)
	}
	return fillStructFromBindings(bindings, colTypes, ev)
}

// scanRowsToSlice 全部行填充到切片（G0；S1 扩展标量/map 元素）：
// 元素为 struct / *struct → 逐行反射填充；元素为 map[string]interface{} → 逐行转 map；
// 元素为标量（单列结果）→ 逐列扫描转换。遵循全局行数上限（DefaultRowLimit）。
func scanRowsToSlice(rows *sql.Rows, colTypes []*sql.ColumnType, schemaHints map[string]*columnSchemaHint, ev reflect.Value) error {
	sliceType := ev.Type()
	elemType := sliceType.Elem()
	structType := elemType
	if elemType.Kind() == reflect.Ptr {
		structType = elemType.Elem()
	}
	switch {
	case structType.Kind() == reflect.Struct:
		return scanRowsToStructSlice(rows, colTypes, schemaHints, ev, sliceType, elemType, structType)
	case elemType == reflect.TypeOf(map[string]interface{}{}):
		return scanRowsToMapSlice(rows, colTypes, schemaHints, ev)
	default:
		return scanRowsToScalarSlice(rows, colTypes, schemaHints, ev)
	}
}

// scanRowsToStructSlice struct / *struct 元素切片填充（G0 原路径）。
func scanRowsToStructSlice(rows *sql.Rows, colTypes []*sql.ColumnType, schemaHints map[string]*columnSchemaHint, ev reflect.Value, sliceType, elemType, structType reflect.Type) error {
	isPtrElem := elemType.Kind() == reflect.Ptr
	bindings := buildScanBindings(structType, colTypes, schemaHints, buildTagColumnIndex(structType))
	bufs := scanBuffers(bindings)
	limit := DefaultRowLimit()
	out := reflect.MakeSlice(sliceType, 0, 16)
	count := 0
	warned := false
	for rows.Next() {
		if limit >= 0 && count >= limit {
			if !warned {
				warned = true
				log.Warnf("row limit reached: %d, truncate result set (SetDefaultRowLimit(-1) to return all)", limit)
			}
			break
		}
		if err := rows.Scan(bufs...); err != nil {
			return fmt.Errorf("scan row %d failed: %w", count+1, err)
		}
		item := reflect.New(structType)
		if err := fillStructFromBindings(bindings, colTypes, item.Elem()); err != nil {
			return fmt.Errorf("scan row %d: %w", count+1, err)
		}
		if isPtrElem {
			out = reflect.Append(out, item)
		} else {
			out = reflect.Append(out, item.Elem())
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	ev.Set(out)
	return nil
}

// scanRowsToMapSlice map[string]interface{} 元素切片填充（S1）：逐行转 map。
func scanRowsToMapSlice(rows *sql.Rows, colTypes []*sql.ColumnType, schemaHints map[string]*columnSchemaHint, ev reflect.Value) error {
	sliceType := ev.Type()
	bufs := prepareColumns(colTypes)
	converters := buildConverters(colTypes, schemaHints)
	limit := DefaultRowLimit()
	out := reflect.MakeSlice(sliceType, 0, 16)
	count := 0
	warned := false
	for rows.Next() {
		if limit >= 0 && count >= limit {
			if !warned {
				warned = true
				log.Warnf("row limit reached: %d, truncate result set (SetDefaultRowLimit(-1) to return all)", limit)
			}
			break
		}
		if err := rows.Scan(bufs...); err != nil {
			return fmt.Errorf("scan row %d failed: %w", count+1, err)
		}
		out = reflect.Append(out, reflect.ValueOf(createMapWithConverters(bufs, colTypes, converters)))
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	ev.Set(out)
	return nil
}

// scanRowsToScalarSlice 标量元素切片填充（S1）：要求单列结果，逐行扫描转换。
func scanRowsToScalarSlice(rows *sql.Rows, colTypes []*sql.ColumnType, schemaHints map[string]*columnSchemaHint, ev reflect.Value) error {
	if len(colTypes) != 1 {
		return fmt.Errorf("QueryTo: scalar slice dest requires exactly 1 column, got %d", len(colTypes))
	}
	sliceType := ev.Type()
	elemType := sliceType.Elem()
	bufs := prepareColumns(colTypes)
	converters := buildConverters(colTypes, schemaHints)
	limit := DefaultRowLimit()
	out := reflect.MakeSlice(sliceType, 0, 16)
	count := 0
	warned := false
	for rows.Next() {
		if limit >= 0 && count >= limit {
			if !warned {
				warned = true
				log.Warnf("row limit reached: %d, truncate result set (SetDefaultRowLimit(-1) to return all)", limit)
			}
			break
		}
		if err := rows.Scan(bufs...); err != nil {
			return fmt.Errorf("scan row %d failed: %w", count+1, err)
		}
		val, err := converters[0](bufs[0])
		if err != nil {
			return fmt.Errorf("scan row %d: %w", count+1, err)
		}
		rval, err := convertFieldValue(val, elemType)
		if err != nil {
			return fmt.Errorf("scan row %d: %w", count+1, err)
		}
		out = reflect.Append(out, reflect.ValueOf(rval))
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	ev.Set(out)
	return nil
}

// scanRowsToMapRow 首行填充到 map[string]interface{}（G0）；无行返回 sql.ErrNoRows。
func scanRowsToMapRow(rows *sql.Rows, colTypes []*sql.ColumnType, schemaHints map[string]*columnSchemaHint, ev reflect.Value) error {
	if ev.Type() != reflect.TypeOf(map[string]interface{}{}) {
		return fmt.Errorf("QueryTo: map dest must be *map[string]interface{}, got %v", ev.Type())
	}
	bufs := prepareColumns(colTypes)
	converters := buildConverters(colTypes, schemaHints)
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	if err := rows.Scan(bufs...); err != nil {
		return fmt.Errorf("scan row 1 failed: %w", err)
	}
	ev.Set(reflect.ValueOf(createMapWithConverters(bufs, colTypes, converters)))
	return nil
}

// scanRowsToScalar 单列首行填充到标量指针（G0）；无行返回 sql.ErrNoRows。
func scanRowsToScalar(rows *sql.Rows, colTypes []*sql.ColumnType, schemaHints map[string]*columnSchemaHint, ev reflect.Value) error {
	if len(colTypes) != 1 {
		return fmt.Errorf("QueryTo: scalar dest requires exactly 1 column, got %d", len(colTypes))
	}
	bufs := prepareColumns(colTypes)
	converters := buildConverters(colTypes, schemaHints)
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	if err := rows.Scan(bufs...); err != nil {
		return fmt.Errorf("scan row 1 failed: %w", err)
	}
	val, err := converters[0](bufs[0])
	if err != nil {
		return err
	}
	rval, err := convertFieldValue(val, ev.Type())
	if err != nil {
		return err
	}
	ev.Set(reflect.ValueOf(rval))
	return nil
}
