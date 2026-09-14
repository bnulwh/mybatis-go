package orm

import (
	"database/sql"
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/go-sql-driver/mysql"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func newInstance(typ reflect.Type) interface{} {
	switch typ.String() {
	case "string":
		return new(sql.NullString)
	case "bool":
		return new(sql.NullBool)
	case "int", "int8", "int16", "int32",
		"uint", "uint8", "uint16", "uint32":
		return new(sql.NullInt32)
	case "int64", "uint64", "sql.NullInt64":
		return new(sql.NullInt64)
	case "float32", "float64":
		return new(sql.NullFloat64)
	case "time.Time", "sql.NullTime":
		return new(sql.NullTime)
	case "mysql.NullTime":
		return new(mysql.NullTime)
	case "sql.RawBytes":
		return new(sql.RawBytes)
	case "[]uint8", "[]byte":
		// MySQL 未开 parseTime 时 DATETIME/文本列 ScanType 为 []uint8，与 sql.RawBytes 同样按原始字节扫描
		return new(sql.RawBytes)
	case "interface {}":
		return new(sql.NullString)
	}
	log.Debugf("not support  type %v", typ)
	return new(sql.NullString)
}

func convertSqlString2String(ptr interface{}) (string, error) {
	pval, ok := ptr.(*sql.NullString)
	if ok && pval.Valid {
		return pval.String, nil
	}
	return "", nil
}

func convertRawBytes2String(ptr interface{}) (string, error) {
	pval, ok := ptr.(*sql.RawBytes)
	if ok {
		return string(*pval), nil
	}
	return "", nil
}
func convertRawBytes2Bool(ptr interface{}) (bool, error) {
	pval, ok := ptr.(*sql.RawBytes)
	if ok {
		sval := string(*pval)
		if sval == "\x00" {
			return false, nil
		}
		return strconv.ParseBool(sval)
	}
	return false, nil
}

func convertSqlBool2Bool(ptr interface{}) (bool, error) {
	pval, ok := ptr.(*sql.NullBool)
	if ok && pval.Valid {
		return pval.Bool, nil
	}
	return false, nil
}

func convertSqlInt32ToInt(ptr interface{}) (int, error) {
	pval, ok := ptr.(*sql.NullInt32)
	if ok && pval.Valid {
		return int(pval.Int32), nil
	}
	return 0, nil
}

func convertSqlInt32ToInt8(ptr interface{}) (int8, error) {
	pval, ok := ptr.(*sql.NullInt32)
	if ok && pval.Valid {
		return int8(pval.Int32), nil
	}
	return int8(0), nil
}

func convertSqlInt32ToInt16(ptr interface{}) (int16, error) {
	pval, ok := ptr.(*sql.NullInt32)
	if ok && pval.Valid {
		return int16(pval.Int32), nil
	}
	return int16(0), nil
}

func convertSqlInt32ToInt32(ptr interface{}) (int32, error) {
	pval, ok := ptr.(*sql.NullInt32)
	if ok && pval.Valid {
		return pval.Int32, nil
	}
	return int32(0), nil
}

func convertSqlInt32ToUInt(ptr interface{}) (uint, error) {
	pval, ok := ptr.(*sql.NullInt32)
	if ok && pval.Valid {
		return uint(pval.Int32), nil
	}
	return uint(0), nil
}

func convertSqlInt32ToUInt8(ptr interface{}) (uint8, error) {
	pval, ok := ptr.(*sql.NullInt32)
	if ok && pval.Valid {
		return uint8(pval.Int32), nil
	}
	return uint8(0), nil
}

func convertSqlInt32ToUInt16(ptr interface{}) (uint16, error) {
	pval, ok := ptr.(*sql.NullInt32)
	if ok && pval.Valid {
		return uint16(pval.Int32), nil
	}
	return uint16(0), nil
}

func convertSqlInt32ToUInt32(ptr interface{}) (uint32, error) {
	pval, ok := ptr.(*sql.NullInt32)
	if ok && pval.Valid {
		return uint32(pval.Int32), nil
	}
	return uint32(0), nil
}

func convertSqlInt64ToInt64(ptr interface{}) (int64, error) {
	pval, ok := ptr.(*sql.NullInt64)
	if ok && pval.Valid {
		return pval.Int64, nil
	}
	return int64(0), nil
}

func convertSqlInt64ToUInt64(ptr interface{}) (uint64, error) {
	pval, ok := ptr.(*sql.NullInt64)
	if ok && pval.Valid {
		return uint64(pval.Int64), nil
	}
	return uint64(0), nil
}

func convertSqlFloat64ToFloat32(ptr interface{}) (float32, error) {
	pval, ok := ptr.(*sql.NullFloat64)
	if ok && pval.Valid {
		return float32(pval.Float64), nil
	}
	return float32(0.0), nil
}

func convertSqlFloat64ToFloat64(ptr interface{}) (float64, error) {
	pval, ok := ptr.(*sql.NullFloat64)
	if ok && pval.Valid {
		return pval.Float64, nil
	}
	return float64(0.0), nil
}

func convertSqlTime2Time(ptr interface{}) (time.Time, error) {
	pval, ok := ptr.(*sql.NullTime)
	if ok && pval.Valid {
		return pval.Time, nil
	}
	return time.Time{}, nil
}

func convertTimeToTime(ptr interface{}) (time.Time, error) {
	switch pval := ptr.(type) {
	case *time.Time:
		return *pval, nil
	case *sql.NullTime:
		if pval.Valid {
			return pval.Time, nil
		}
	case *mysql.NullTime:
		if pval.Valid {
			return pval.Time, nil
		}
	}
	return time.Time{}, nil
}

// convertFn 列值转换函数（P1-4：一次查询只做一次类型分派，每行直接调用）
type convertFn func(ptr interface{}) (interface{}, error)

// convertToConvertFn 将返回具体类型的转换函数包装为 convertFn。
func convertToConvertFn[T any](fn func(ptr interface{}) (T, error)) convertFn {
	return func(ptr interface{}) (interface{}, error) { return fn(ptr) }
}

// resolveConverter 根据列类型解析单列转换函数。
// 当 schemaHint 非空且驱动 ScanType 不可靠时，用缓存的表结构类型作为补充推断。
func resolveConverter(colType *sql.ColumnType, schemaHint *columnSchemaHint) convertFn {
	typ := colType.ScanType()
	needSchemaFallback := false
	if typ == nil {
		needSchemaFallback = true
	} else if typ.String() == "interface {}" || typ.String() == "[]uint8" || typ.String() == "[]byte" || typ.String() == "sql.RawBytes" {
		needSchemaFallback = true
	}
	if needSchemaFallback && schemaHint != nil && schemaHint.goType != nil {
		return resolveConverterByGoType(colType, schemaHint.goType)
	}
	if typ == nil {
		return convertToConvertFn(convertSqlString2String)
	}
	switch typ.String() {
	case "string", "sql.NullString":
		return convertToConvertFn(convertSqlString2String)
	case "sql.RawBytes":
		if colType.DatabaseTypeName() == "BIT" {
			return convertToConvertFn(convertRawBytes2Bool)
		} else {
			return convertToConvertFn(convertRawBytes2String)
		}
	case "bool":
		return convertToConvertFn(convertSqlBool2Bool)

	case "int":
		return convertToConvertFn(convertSqlInt32ToInt)
	case "int8":
		return convertToConvertFn(convertSqlInt32ToInt8)
	case "int16":
		return convertToConvertFn(convertSqlInt32ToInt16)
	case "int32":
		return convertToConvertFn(convertSqlInt32ToInt32)
	case "uint":
		return convertToConvertFn(convertSqlInt32ToUInt)
	case "uint8":
		return convertToConvertFn(convertSqlInt32ToUInt8)
	case "uint16":
		return convertToConvertFn(convertSqlInt32ToUInt16)
	case "uint32":
		return convertToConvertFn(convertSqlInt32ToUInt32)
	case "int64", "sql.NullInt64":
		return convertToConvertFn(convertSqlInt64ToInt64)
	case "uint64":
		return convertToConvertFn(convertSqlInt64ToUInt64)
	case "float32":
		return convertToConvertFn(convertSqlFloat64ToFloat32)
	case "float64":
		return convertToConvertFn(convertSqlFloat64ToFloat64)
	case "time.Time", "sql.NullTime", "mysql.NullTime":
		return convertToConvertFn(convertTimeToTime)
	case "interface {}":
		return convertToConvertFn(convertSqlString2String)
	case "[]uint8", "[]byte":
		return convertToConvertFn(convertRawBytes2String)
	}
	log.Warnf("not support convert type: %v", typ)
	return func(ptr interface{}) (interface{}, error) {
		return nil, fmt.Errorf("not support convert type: %v ,value: %v", typ, ptr)
	}
}

// resolveConverterByGoType 当驱动 ScanType 不可靠时，使用表结构缓存中的 Go 类型来选择扫描策略。
// 扫描目标仍按 sql.Null* 分配（兼容 NULL），但转换函数按表结构声明的 Go 类型输出。
func resolveConverterByGoType(colType *sql.ColumnType, goType reflect.Type) convertFn {
	dbTypeName := colType.DatabaseTypeName()
	switch goType.String() {
	case "string":
		return func(ptr interface{}) (interface{}, error) {
			if pval, ok := ptr.(*sql.NullString); ok && pval.Valid {
				return pval.String, nil
			}
			if pval, ok := ptr.(*sql.RawBytes); ok {
				return string(*pval), nil
			}
			return convertSqlString2String(ptr)
		}
	case "bool":
		if dbTypeName == "BIT" {
			return convertToConvertFn(convertRawBytes2Bool)
		}
		return func(ptr interface{}) (interface{}, error) {
			if pval, ok := ptr.(*sql.NullBool); ok && pval.Valid {
				return pval.Bool, nil
			}
			s, err := convertSqlString2String(ptr)
			if err != nil || s == "" {
				return false, nil
			}
			return strconv.ParseBool(s)
		}
	case "int":
		return func(ptr interface{}) (interface{}, error) {
			if pval, ok := ptr.(*sql.NullInt32); ok && pval.Valid {
				return int(pval.Int32), nil
			}
			if pval, ok := ptr.(*sql.RawBytes); ok {
				s := string(*pval)
				n, err := strconv.Atoi(s)
				if err != nil {
					return 0, err
				}
				return n, nil
			}
			return convertSqlInt32ToInt(ptr)
		}
	case "int64":
		return func(ptr interface{}) (interface{}, error) {
			if pval, ok := ptr.(*sql.NullInt64); ok && pval.Valid {
				return pval.Int64, nil
			}
			if pval, ok := ptr.(*sql.RawBytes); ok {
				s := string(*pval)
				n, err := strconv.ParseInt(s, 10, 64)
				if err != nil {
					return int64(0), err
				}
				return n, nil
			}
			return convertSqlInt64ToInt64(ptr)
		}
	case "float64":
		return func(ptr interface{}) (interface{}, error) {
			if pval, ok := ptr.(*sql.NullFloat64); ok && pval.Valid {
				return pval.Float64, nil
			}
			if pval, ok := ptr.(*sql.RawBytes); ok {
				s := string(*pval)
				f, err := strconv.ParseFloat(s, 64)
				if err != nil {
					return 0.0, err
				}
				return f, nil
			}
			return convertSqlFloat64ToFloat64(ptr)
		}
	case "time.Time":
		return func(ptr interface{}) (interface{}, error) {
			if pval, ok := ptr.(*sql.NullTime); ok && pval.Valid {
				return pval.Time, nil
			}
			if pval, ok := ptr.(*mysql.NullTime); ok && pval.Valid {
				return pval.Time, nil
			}
			if pval, ok := ptr.(*time.Time); ok {
				return *pval, nil
			}
			if pval, ok := ptr.(*sql.RawBytes); ok {
				s := string(*pval)
				for _, layout := range []string{
					time.RFC3339Nano, time.RFC3339,
					"2006-01-02 15:04:05.999999999",
					"2006-01-02 15:04:05",
					"2006-01-02",
				} {
					if t, err := time.Parse(layout, s); err == nil {
						return t, nil
					}
				}
				return time.Time{}, fmt.Errorf("cannot parse time from %q", s)
			}
			return convertTimeToTime(ptr)
		}
	}
	log.Debugf("schema hint: unsupported go type %v for column %s, fallback to string", goType, colType.Name())
	return func(ptr interface{}) (interface{}, error) {
		if pval, ok := ptr.(*sql.RawBytes); ok {
			return string(*pval), nil
		}
		return convertSqlString2String(ptr)
	}
}

// buildConverters 为一次查询的列预编译转换函数表。
// schemaHints 按列名提供表结构缓存的类型提示，当驱动 ScanType 不可靠时用于补充推断。
func buildConverters(colTypes []*sql.ColumnType, schemaHints map[string]*columnSchemaHint) []convertFn {
	converters := make([]convertFn, len(colTypes))
	for i, colType := range colTypes {
		hint := schemaHints[colType.Name()]
		converters[i] = resolveConverter(colType, hint)
	}
	return converters
}

// buildConvertersBasic 为无 schema hint 的场景构建转换函数（保持旧行为）。
func buildConvertersBasic(colTypes []*sql.ColumnType) []convertFn {
	converters := make([]convertFn, len(colTypes))
	for i, colType := range colTypes {
		converters[i] = resolveConverter(colType, nil)
	}
	return converters
}

func convertInstanceType(ptr interface{}, colType *sql.ColumnType) (interface{}, error) {
	return resolveConverter(colType, nil)(ptr)
}

func combineErrors(errs ...error) error {
	var es []string
	for _, err := range errs {
		if err != nil {
			es = append(es, fmt.Sprintf("%v", err))
		}
	}
	if len(es) == 0 {
		return nil
	}
	return fmt.Errorf("%v", strings.Join(es, "\n"))
}
