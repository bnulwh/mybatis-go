package orm

import (
	"database/sql"
	"fmt"
	log "github.com/astaxie/beego/logs"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"time"
)

type ExecuteFunc func(args ...interface{}) (int64, int64, error)
type QueryRowFunc func(args ...interface{}) (interface{}, error)
type QueryRowsFunc func(args ...interface{}) ([]interface{}, error)

func GetAllEnv() map[string]string {
	envMap := map[string]string{}
	for _, envLine := range os.Environ() {
		kv := strings.Split(envLine, "=")
		envMap[kv[0]] = kv[1]
	}
	return envMap
}

func LoadSettings(filename string) map[string]string {
	m := LoadProperties(filename)
	em := GetAllEnv()
	for k, v := range m {
		if strings.Compare(v[0:2], "${") == 0 {
			v = getRealValue(v, em)
			m[k] = v
		}
	}
	for k, v := range em {
		m[k] = v
	}
	return m
}

func LoadProperties(filename string) map[string]string {
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Warn("load file %v failed: %v", filename, err)
		return map[string]string{}
	}
	envMap := map[string]string{}
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)

		if len(line) == 0 || strings.Contains("!#", line[0:1]) {
			continue
		}
		pos := strings.Index(line, "=")
		if pos <= 0 {
			pos = strings.Index(line, ":")
		}
		if pos <= 0 {
			continue
		}
		key := line[0:pos]
		val := strings.Trim(line[pos+1:], "'\" ")
		envMap[key] = val
	}
	return envMap
}

func getRealValue(val string, em map[string]string) string {
	pos := strings.Index(val, ":")
	if pos < 0 {
		key := val[2 : len(val)-1]
		rv, ok := em[key]
		if ok {
			return rv
		}
		return ""
	}
	key := val[2:pos]
	rval := val[pos+1 : len(val)-1]
	rv, ok := em[key]
	if ok {
		return rv
	}
	return rval
}
func getSqlPtrType(typ reflect.Type) interface{} {
	switch typ.String() {
	case "string":
		return new(sql.NullString)
	case "bool":
		return new(sql.NullBool)
	case "int", "int8", "int16", "int32",
		"uint", "uint8", "uint16", "uint32":
		return new(sql.NullInt32)
	case "int64", "uint64":
		return new(sql.NullInt64)
	case "float32", "float64":
		return new(sql.NullFloat64)
	case "time.Time":
		return new(sql.NullTime)
	}
	log.Info("not support  type %v", typ)
	return new(sql.NullString)
}

func convertValue(ptr interface{}, typ reflect.Type) (interface{}, error) {
	switch typ.String() {
	case "string":
		pval, ok := ptr.(*sql.NullString)
		if ok && pval.Valid {
			return pval.String, nil
		}
		return "", nil
	case "bool":
		pval, ok := ptr.(*sql.NullBool)
		if ok && pval.Valid {
			return pval.Bool, nil
		}
		return false, nil
	case "int":
		pval, ok := ptr.(*sql.NullInt32)
		if ok && pval.Valid {
			return int(pval.Int32), nil
		}
		return 0, nil
	case "int8":
		pval, ok := ptr.(*sql.NullInt32)
		if ok && pval.Valid {
			return int8(pval.Int32), nil
		}
		return int8(0), nil
	case "int16":
		pval, ok := ptr.(*sql.NullInt32)
		if ok && pval.Valid {
			return int16(pval.Int32), nil
		}
		return int16(0), nil
	case "int32":
		pval, ok := ptr.(*sql.NullInt32)
		if ok && pval.Valid {
			return pval.Int32, nil
		}
		return int32(0), nil
	case "uint":
		pval, ok := ptr.(*sql.NullInt32)
		if ok && pval.Valid {
			return uint(pval.Int32), nil
		}
		return uint(0), nil
	case "uint8":
		pval, ok := ptr.(*sql.NullInt32)
		if ok && pval.Valid {
			return uint8(pval.Int32), nil
		}
		return uint8(0), nil
	case "uint16":
		pval, ok := ptr.(*sql.NullInt32)
		if ok && pval.Valid {
			return uint16(pval.Int32), nil
		}
		return uint16(0), nil
	case "uint32":
		pval, ok := ptr.(*sql.NullInt32)
		if ok && pval.Valid {
			return uint32(pval.Int32), nil
		}
		return uint32(0), nil
	case "int64":
		pval, ok := ptr.(*sql.NullInt64)
		if ok && pval.Valid {
			return pval.Int64, nil
		}
		return int64(0), nil
	case "uint64":
		pval, ok := ptr.(*sql.NullInt64)
		if ok && pval.Valid {
			return uint64(pval.Int64), nil
		}
		return uint64(0), nil
	case "float32":
		pval, ok := ptr.(*sql.NullFloat64)
		if ok && pval.Valid {
			return float32(pval.Float64), nil
		}
		return float32(0.0), nil
	case "float64":
		pval, ok := ptr.(*sql.NullFloat64)
		if ok && pval.Valid {
			return pval.Float64, nil
		}
		return float64(0.0), nil
	case "time.Time":
		pval, ok := ptr.(*sql.NullTime)
		if ok && pval.Valid {
			return pval.Time, nil
		}
		return time.Time{}, nil
	}
	log.Warn("not support convert type: %v ,value: %v", typ, ptr)
	return nil, fmt.Errorf("not support convert type: %v ,value: %v", typ, ptr)
}
