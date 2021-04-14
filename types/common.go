package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/astaxie/beego/logs"
	"reflect"
	"strings"
	"time"
)

type SqlFunctionType string

const (
	SelectFunction SqlFunctionType = "select"
	UpdateFunction SqlFunctionType = "update"
	DeleteFunction SqlFunctionType = "delete"
	InsertFunction SqlFunctionType = "insert"
)

func GetShortName(name string) string {
	pos := strings.LastIndex(name, ".")
	if pos > 0 {
		return name[pos+1:]
	}
	return name
}
func ToJson(v interface{}) string {
	dt, err := json.Marshal(v)
	if err != nil {
		fmt.Printf("to json failed: %v", err)
		return ""
	}
	return string(dt)
}

func parseSqlFunctionType(tps string) SqlFunctionType {
	switch strings.ToLower(strings.TrimSpace(tps)) {
	case "update":
		return UpdateFunction
	case "delete":
		return DeleteFunction
	case "insert":
		return InsertFunction
	case "select":
		return SelectFunction
	default:
		log.Warn("unsupport sql function type: %v", tps)
	}
	return SelectFunction
}

func getFormatString(ms string) string {
	var buf bytes.Buffer
	if strings.Compare(ms[0:1], "'") != 0 {
		buf.WriteString("'")
	}
	buf.WriteString(ms)
	if strings.Compare(ms[len(ms)-1:], "'") != 0 {
		buf.WriteString("'")
	}
	return buf.String()
}

func getFormatValue(m interface{}) string {
	typ := reflect.TypeOf(m)
	switch typ.String() {
	case "string":
		return getFormatString(m.(string))
	case "bool",
		"int", "int8", "int16", "int32",
		"uint", "uint8", "uint16", "uint32",
		"int64", "uint64",
		"float32", "float64":
		return fmt.Sprintf("%v", m)
	case "time.Time":
		return fmt.Sprintf("'%v'", m.(time.Time).Format("2006-01-02 15:04:05"))
	default:
		log.Warn("not support convert type %v", typ)
	}
	return ""
}

func buildKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

func parseResultTypeFrom(tps string) reflect.Type {
	switch strings.ToUpper(GetShortName(tps)) {
	case "VARCHAR", "STRING":
		return reflect.TypeOf("")
	case "TIMESTAMP", "TIME":
		return reflect.TypeOf(time.Now())
	case "INTEGER", "INT", "LONG", "BIGINT":
		return reflect.TypeOf(0)
	case "BOOLEAN", "BIT", "BOOL":
		return reflect.TypeOf(true)
	case "DOUBLE":
		return reflect.TypeOf(0.0)
	default:
		log.Warn("unsupport type to parse: %v", tps)
	}
	return reflect.TypeOf(map[string]interface{}{})
}
func parseJdbcTypeFrom(tps string) reflect.Type {
	switch strings.ToUpper(GetShortName(tps)) {
	case "VARCHAR", "STRING":
		return reflect.TypeOf("")
	case "TIMESTAMP", "TIME":
		return reflect.TypeOf(time.Now())
	case "INTEGER", "INT", "LONG", "BIGINT":
		return reflect.TypeOf(0)
	case "BOOLEAN", "BIT", "BOOL":
		return reflect.TypeOf(true)
	case "DOUBLE":
		return reflect.TypeOf(0.0)
	default:
		log.Warn("unsupport jdbc type to parse: %v", tps)
	}
	return reflect.TypeOf("")
}
func UpperFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return fmt.Sprintf("%s%s", strings.ToUpper(s[0:1]), s[1:])
}

func convert2Map(val reflect.Value) map[string]interface{} {
	nmp := map[string]interface{}{}
	typ := val.Type()
	switch typ.Kind() {
	case reflect.Map:
		for _, key := range val.MapKeys() {
			nmp[buildKey(fmt.Sprintf("%v", reflect.Indirect(key).Interface()))] = reflect.Indirect(val.MapIndex(key)).Interface()
		}
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			fval := val.Field(i)
			nmp[buildKey(typ.Field(i).Name)] = reflect.Indirect(fval).Interface()
		}
	}
	return nmp
}
func convert2Slice(val reflect.Value) []interface{} {
	var ns []interface{}
	for i := 0; i < val.Len(); i++ {
		item := val.Index(i)
		ns = append(ns, reflect.Indirect(item).Interface())
	}
	return ns
}

func validValue(m interface{}) bool {
	typ := reflect.TypeOf(m)
	switch typ.String() {
	case "string":
		ms := m.(string)
		return len(ms) > 0
	case "bool",
		"int", "int8", "int16", "int32",
		"uint", "uint8", "uint16", "uint32",
		"int64", "uint64",
		"float32", "float64":
		return true
	case "time.Time":
		return !m.(time.Time).IsZero()
	}
	switch typ.Kind() {
	case reflect.Slice:
		val := reflect.ValueOf(m)
		return val.Len() > 0
	}
	log.Warn("not support valid value: %v ,type: %v", m, typ)
	return true
}
