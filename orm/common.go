package orm

import (
	"database/sql"
	"fmt"
	log "github.com/bnulwh/logrus"
	"github.com/go-sql-driver/mysql"
	"reflect"
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
	case "time.Time", "sql.NullTime", "mysql.NullTime":
		return new(mysql.NullTime)
	case "sql.RawBytes":
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

func convertMySqlTime2Time(ptr interface{}) (time.Time, error) {
	pval, ok := ptr.(*mysql.NullTime)
	if ok && pval.Valid {
		return pval.Time, nil
	}
	return time.Time{}, nil
}

func convertInstanceType(ptr interface{}, typ reflect.Type) (interface{}, error) {
	switch typ.String() {
	case "string":
		return convertSqlString2String(ptr)
	case "sql.RawBytes":
		return convertRawBytes2String(ptr)
	case "bool":
		return convertSqlBool2Bool(ptr)
	case "int":
		return convertSqlInt32ToInt(ptr)
	case "int8":
		return convertSqlInt32ToInt8(ptr)
	case "int16":
		return convertSqlInt32ToInt16(ptr)
	case "int32":
		return convertSqlInt32ToInt32(ptr)
	case "uint":
		return convertSqlInt32ToUInt(ptr)
	case "uint8":
		return convertSqlInt32ToUInt8(ptr)
	case "uint16":
		return convertSqlInt32ToUInt16(ptr)
	case "uint32":
		return convertSqlInt32ToUInt32(ptr)
	case "int64", "sql.NullInt64":
		return convertSqlInt64ToInt64(ptr)
	case "uint64":
		return convertSqlInt64ToUInt64(ptr)
	case "float32":
		return convertSqlFloat64ToFloat32(ptr)
	case "float64":
		return convertSqlFloat64ToFloat64(ptr)
	case "time.Time", "sql.NullTime":
		return convertSqlTime2Time(ptr)
	case "mysql.NullTime":
		return convertMySqlTime2Time(ptr)
	case "interface {}":
		return convertSqlString2String(ptr)
	}
	log.Warnf("not support convert type: %v ,value: %v", typ, ptr)
	return nil, fmt.Errorf("not support convert type: %v ,value: %v", typ, ptr)
}

func combineErrors(errs ...error) error {
	var es []string
	for _, err := range errs {
		es = append(es, fmt.Sprintf("%v", err))
	}
	return fmt.Errorf("%v", strings.Join(es, "\n"))
}
