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
	"unicode"
)

// rowStreamType *RowStream 的反射类型，供 Mapper 代理判断流式返回（P4-2）。
var rowStreamType = reflect.TypeOf((*RowStream)(nil))

// RowStream 大结果集流式读取（P4-2）。
//
// 与 Query/QueryContext 全量读进内存不同，RowStream 保持查询游标打开，由调用方
// 通过 Next() 逐行消费：任意时刻内存中只有当前一行，10 万行以上的大结果集不再
// 一次性占满内存。典型的用法：
//
//	st, err := orm.QueryStream(ctx, sql, args...)
//	if err != nil {
//		return err
//	}
//	defer st.Close()
//	for st.Next() {
//		row := st.Row()            // 当前行 map（与 Query 的行结构一致）
//		// 或 st.Scan(&dest)        // 填充到结构体 / map
//	}
//	if err := st.Err(); err != nil { ... }
//
// 约束：
//   - 无论是否读完，都必须调用 Close() 释放连接（幂等，可 defer）；
//   - 行数上限遵循全局 DefaultRowLimit（P4-3）：达到上限停止读取并 Warn 提示，
//     负数不限制（返回全部）；上限在 QueryStream 打开时快照；
//   - ctx 无 deadline 时叠加全局默认超时（P4-1）。
type RowStream struct {
	rows       *sql.Rows
	colTypes   []*sql.ColumnType
	tempItems  []interface{}
	converters []convertFn
	limit      int
	count      int
	row        map[string]interface{}
	err        error
	warned     bool
	closed     bool
	cancel     context.CancelFunc
}

// QueryStream 执行查询并以流式方式逐行返回结果（P4-2）。
// 调用方必须负责在遍历结束后调用 Close() 释放连接（defer 即可）。
func QueryStream(ctx context.Context, sqlStr string, args ...interface{}) (*RowStream, error) {
	log.Debugf("sql: %v", sqlStr)
	ctx, cancel := withExecTimeout(ctx)
	rows, err := gDbConn.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		cancel()
		log.Errorf("query sql %v failed: %v", sqlStr, err)
		return nil, err
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		cancel()
		_ = rows.Close()
		log.Errorf("fill sql %v result failed: %v", sqlStr, err)
		return nil, err
	}
	return &RowStream{
		rows:     rows,
		colTypes: colTypes,
		limit:    DefaultRowLimit(),
		cancel:   cancel,
	}, nil
}

// Next 前进到下一行；返回 false 表示遍历结束。
// 结束原因可通过 Err() 区分：Err() 非 nil 为读取错误，nil 为正常读完（或触发行数上限）。
func (s *RowStream) Next() bool {
	if s.closed || s.err != nil {
		return false
	}
	// 全局行数上限（P4-3）：达到上限即停止读取（连接仍由调用方 Close 释放）
	if s.limit >= 0 && s.count >= s.limit {
		if !s.warned {
			s.warned = true
			log.Warnf("row limit reached: %d, truncate result set (SetDefaultRowLimit(-1) to return all)", s.limit)
		}
		return false
	}
	if !s.rows.Next() {
		if err := s.rows.Err(); err != nil {
			s.err = err
		}
		return false
	}
	// 与 fetchRows 一致：部分驱动（如 modernc sqlite）在首行之后才填充 ScanType，
	// 因此扫描目标延迟到首个成功 Next() 之后再构建。
	if s.tempItems == nil {
		s.tempItems = prepareColumns(s.colTypes)
		s.converters = buildConverters(s.colTypes)
	}
	if err := s.rows.Scan(s.tempItems...); err != nil {
		// 流式场景不静默丢行（M-05 精神）：扫描失败立即终止，Err() 返回行号明细
		s.err = fmt.Errorf("scan row %d failed: %w", s.count+1, err)
		return false
	}
	s.row = createMapWithConverters(s.tempItems, s.colTypes, s.converters)
	s.count++
	return true
}

// Row 返回当前行的 map 表示（未调用 Next 或遍历结束后为 nil）。
// 返回的 map 复用流内部缓冲，下一次 Next() 会覆盖其内容；需要长期保留请自行拷贝。
func (s *RowStream) Row() map[string]interface{} {
	return s.row
}

// Count 返回已成功读取的行数（含触发行数上限前已读的行）。
func (s *RowStream) Count() int {
	return s.count
}

// Err 返回遍历期间的错误（游标错误 / 扫描失败）；正常结束返回 nil。
func (s *RowStream) Err() error {
	return s.err
}

// Close 释放查询连接并取消内部 context；幂等，可安全重复调用（defer 场景）。
// 未读完就提前退出（如调用方 break）也必须 Close，否则连接会被占住。
func (s *RowStream) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	s.cancel()
	return s.rows.Close()
}

// Scan 把当前行填充到 dest：
//   - *map[string]interface{}：拷贝当前行到目标 map；
//   - struct 指针：按列名匹配字段填充（匹配顺序：原名 → 首字母大写 → 下划线/连字符转驼峰
//     → 大小写不敏感），类型用 utils.ChangeType 转换，列级失败聚合为一条错误返回。
func (s *RowStream) Scan(dest interface{}) error {
	if s.row == nil {
		return fmt.Errorf("RowStream.Scan: no current row, call Next() first")
	}
	if mp, ok := dest.(*map[string]interface{}); ok {
		copied := make(map[string]interface{}, len(s.row))
		for k, v := range s.row {
			copied[k] = v
		}
		*mp = copied
		return nil
	}
	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("RowStream.Scan: dest must be *map[string]interface{} or struct pointer, got %T", dest)
	}
	ev := rv.Elem()
	if ev.Kind() != reflect.Struct {
		return fmt.Errorf("RowStream.Scan: dest must be *map[string]interface{} or struct pointer, got %T", dest)
	}
	typ := ev.Type()
	var errs []string
	for col, val := range s.row {
		field := findStreamField(typ, col)
		if field == nil || !field.IsExported() {
			continue
		}
		fval := ev.FieldByIndex(field.Index)
		if !fval.CanSet() {
			continue
		}
		rval, err := utils.ChangeType(val, fval.Type())
		if err != nil {
			errs = append(errs, fmt.Sprintf("column %q -> field %s: %v", col, field.Name, err))
			continue
		}
		fval.Set(reflect.ValueOf(rval))
	}
	if len(errs) > 0 {
		return fmt.Errorf("RowStream.Scan: %v", strings.Join(errs, "; "))
	}
	return nil
}

// findStreamField 按列名在结构体上查找字段（Scan 用）。
func findStreamField(typ reflect.Type, col string) *reflect.StructField {
	if f, ok := typ.FieldByName(col); ok {
		return &f
	}
	if f, ok := typ.FieldByName(types.UpperFirst(col)); ok {
		return &f
	}
	if f, ok := typ.FieldByName(snakeToCamel(col)); ok {
		return &f
	}
	lower := strings.ToLower(col)
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if strings.ToLower(f.Name) == lower {
			return &f
		}
	}
	return nil
}

// snakeToCamel 把下划线/连字符命名转为首字母大写的驼峰（create_time → CreateTime）。
func snakeToCamel(s string) string {
	var b strings.Builder
	upper := true
	for _, r := range s {
		if r == '_' || r == '-' {
			upper = true
			continue
		}
		if upper {
			b.WriteRune(unicode.ToUpper(r))
			upper = false
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
