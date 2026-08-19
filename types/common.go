package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"reflect"
	"strings"
	"time"
)

type SqlFunctionType string
type QueryType string

const (
	SelectFunction SqlFunctionType = "select"
	UpdateFunction SqlFunctionType = "update"
	DeleteFunction SqlFunctionType = "delete"
	InsertFunction SqlFunctionType = "insert"
	//Query
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
		log.Warnf("unsupport sql function type: %v", tps)
	}
	return SelectFunction
}

func getFormatString(ms string) string {
	var buf bytes.Buffer
	if len(ms) == 0 {
		return "''"
	}
	buf.WriteString("'")
	buf.WriteString(strings.ReplaceAll(ms, "'", "\""))
	buf.WriteString("'")
	return buf.String()
}

func getFormatValue(m interface{}) string {
	if m == nil {
		// S-09：nil 参数反射零值 panic 防御，按 SQL NULL 渲染
		return "null"
	}
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
		return fmt.Sprintf("'%v'", m.(time.Time).Format("2006-01-02 15:04:05.000000000"))
	default:
		log.Warnf("not support convert type %v", typ)
	}
	return ""
}

func buildKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

func parseResultTypeFrom(tps string) reflect.Type {
	switch strings.ToUpper(GetShortName(tps)) {
	case "VARCHAR", "STRING", "LONGVARCHAR":
		return reflect.TypeOf("")
	case "TIMESTAMP", "TIME":
		return reflect.TypeOf(time.Now())
	case "INTEGER", "INT":
		return reflect.TypeOf(0)
	case "LONG", "BIGINT":
		return reflect.TypeOf(int64(0))
	case "BOOLEAN", "BIT", "BOOL":
		return reflect.TypeOf(true)
	case "DOUBLE", "FLOAT":
		return reflect.TypeOf(0.0)
	default:
		log.Warnf("unsupport type to parse: %v", tps)
	}
	return reflect.TypeOf(map[string]interface{}{})
}
func GetJdbcTypePart(tps string) string {
	arr := strings.Split(tps, " ")
	ret := arr[0]
	idx := strings.Index(ret, "(")
	if idx > 0 {
		ret = tps[0:idx]
	}
	return strings.TrimSpace(ret)
}
func ParseJdbcTypeFrom(tps string) reflect.Type {
	tps = GetJdbcTypePart(tps)
	switch strings.ToUpper(GetShortName(tps)) {
	case "VARCHAR", "STRING", "LONGVARCHAR", "TEXT", "TINYTEXT", "CHAR", "MEDIUMTEXT",
		"BLOB", "LONGBLOB", "CHARACTER":
		return reflect.TypeOf("")
	case "TIMESTAMP", "TIME", "DATETIME":
		return reflect.TypeOf(time.Now())
	case "INTEGER", "INT", "TINYINT", "SMALLINT":
		return reflect.TypeOf(0)
	case "LONG", "BIGINT":
		return reflect.TypeOf(int64(0))
	case "BOOLEAN", "BIT", "BOOL", "ENUM":
		return reflect.TypeOf(true)
	case "DOUBLE", "FLOAT", "NUMERIC":
		return reflect.TypeOf(0.0)
	default:
		log.Warnf("unsupport jdbc type to parse: %v", tps)
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
	if !val.IsValid() {
		// S-09：nil 参数反射零值（reflect.Indirect(reflect.ValueOf(nil))）直接返回空 map，避免 Type() panic
		return nmp
	}
	typ := val.Type()
	switch typ.Kind() {
	case reflect.Map:
		for _, key := range val.MapKeys() {
			nmp[buildKey(fmt.Sprintf("%v", reflect.Indirect(key).Interface()))] = safeIndirectInterface(val.MapIndex(key))
		}
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			fval := val.Field(i)
			// M-01：nil 指针字段（如 *time.Time 零值）解引用后是零 Value，直接调 Interface() 会 panic；
			// 统一走 safeIndirectInterface，nil 指针/接口输出 nil（getFormatValue(nil) 渲染 SQL null）
			nmp[buildKey(typ.Field(i).Name)] = safeIndirectInterface(fval)
		}
	}
	return nmp
}

// safeIndirectInterface 安全解引用取 Interface：
// nil 指针 / nil 接口 / 解引用后仍为零 Value 时返回 nil，
// 避免对零 Value 调 Interface() panic（M-01）。
func safeIndirectInterface(v reflect.Value) interface{} {
	if !v.IsValid() {
		return nil
	}
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
		if !v.IsValid() {
			return nil
		}
	}
	return v.Interface()
}
func convert2Slice(val reflect.Value) []interface{} {
	var ns []interface{}
	for i := 0; i < val.Len(); i++ {
		item := val.Index(i)
		ns = append(ns, reflect.Indirect(item).Interface())
	}
	return ns
}

// sliceArgsFrom 提取 SliceSqlParam 的参数切片：
// 优先取 args[0]（代理调用传入单个切片参数，如 mapper.UpdateDeptChildren(depts)），
// 否则回退整个 args（调用方展开传入多个元素）。
func sliceArgsFrom(args []interface{}) []interface{} {
	if len(args) == 0 {
		return nil
	}
	v := reflect.ValueOf(args[0])
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		return convert2Slice(reflect.Indirect(v))
	}
	return args
}

func validValue(m interface{}) bool {
	if m == nil {
		// S-09：nil 参数 reflect.TypeOf 返回 nil，String() 会 panic
		return false
	}
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
	case reflect.Map:
		val := reflect.ValueOf(m)
		return val.Len() > 0

	}
	log.Warnf("not support valid value: %v ,type: %v, kind: %v", m, typ, typ.Kind())
	return true
}

func toGolangType(tn string) string {
	sname := GetShortName(tn)
	switch strings.ToUpper(sname) {
	case "STRING", "VARCHAR":
		return "string"
	case "BOOLEAN", "BOOL":
		return "bool"
	case "INT", "INTEGER", "INT8", "INT16", "INT32":
		return "int32"
	case "INT64", "LONG", "BIGINT":
		return "int64"
	case "UINT", "UINT8", "UINT16", "UINT32":
		return "uint32"
	case "UINT64":
		return "uint64"
	case "FLOAT", "FLOAT32":
		return "float32"
	case "FLOAT64", "DOUBLE":
		return "float64"
	case "TIME", "TIMESTAMP":
		return "time.Time"
	case "LIST", "ARRAY", "ARRAYLIST", "SLICE":
		return "[]interface{}"
	case "MAP", "HASHMAP", "TREEMAP":
		return "map[string]interface{}"
	}
	return "models." + sname
}

func ToJavaType(typ reflect.Type) string {
	switch typ.String() {
	case "string":
		return "java.lang.String"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return "java.lang.Integer"

	}
	log.Warnf("unsupport type %v", typ)
	return "java.lang.String"
}
