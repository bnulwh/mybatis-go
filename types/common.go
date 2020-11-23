package types

import(
	"bytes"
	"fmt"
	log "github.com/astaxie/beego/logs"
	"reflect"
	"strings"
	"time"
)

type SqlFunctionType string
const (
	SelectSQL SqlFunctionType="select"
	UpdateSQL SqlFunctionType="update"
	DeleteSQL SqlFunctionType="delete"
	InsertSQL SqlFunctionType="insert"
)

func parseSqlFunctionType(tps string)SqlFunctionType{
	switch strings.ToLower(strings.TrimSpace(tps)){
	case "update":
		return UpdateSQL
	case "delete":
		return DeleteSQL
	case "insert":
		return InsertSQL
	case "select":
	default:
	}
	return SelectSQL
}

func getFormatValue(m interface{}) string{
	typ := reflect.TypeOf(m)
	switch typ.String(){
	case "string":
		ms :=m.(string)
		var buf bytes.Buffer
		if strings.Compare(ms[0:1],"'") !=0{
			buf.WriteString("'")
		}
		buf.WriteString(ms)
		if strings.Compare(ms[len(ms)-1:],"'") !=0{
			buf.WriteString("'")
		}
		return buf.String()
	case "bool",
	 "int","int8","int16","int32",
	  "uint","uint8","uint16","uint32",
	   "int64","uint64",
	    "float32","float64":
		return fmt.Sprintf("%v",m)
	case "time.Time":
		return m.(time.Time).Format("2006-01-02 15:04:05")
	}
	log.Info("not support convert type %v",typ)
	return ""
}

func buildKey(key string) string{
	return strings.ToLower(strings.TrimSpace(key))
}

func GetShortName(name string) string{
	pos := strings.LastIndex(name,".")
	if pos >0{
		return name[pos+1:]
	}
	return name
}

func parseResultTypeFrom(tps string) reflect.Type {
	switch strings.ToUpper(GetShortName(tps)){
	case "VARCHAR","STRING":
		return reflect.TypeOf("")
	case "TIMESTAMP","TIME":
		return reflect.TypeOf(time.Now())
	case "INTEGER","INT","LONG","BIGINT":
		return reflect.TypeOf(0)
	case "BOOLEAN","BIT","BOOL":
		return reflect.TypeOf(true)
	case "DOUBLE":
		return reflect.TypeOf(0.0)
	default:
		log.Warn("unsupport type to parse: %v",tps)
	}
	return reflect.TypeOf(map[string]interface{}{})
}
func parseJdbcTypeFrom(tps string) reflect.Type {
	switch strings.ToUpper(GetShortName(tps)){
	case "VARCHAR","STRING":
		return reflect.TypeOf("")
	case "TIMESTAMP","TIME":
		return reflect.TypeOf(time.Now())
	case "INTEGER","INT","LONG","BIGINT":
		return reflect.TypeOf(0)
	case "BOOLEAN","BIT","BOOL":
		return reflect.TypeOf(true)
	case "DOUBLE":
		return reflect.TypeOf(0.0)
	default:
		log.Warn("unsupport type to parse: %v",tps)
	}
	return reflect.TypeOf("")
}
func UpperFirst(s string) string{
	if len(s) ==0{
		return s
	}
	return fmt.Sprintf("%s%s",strings.ToUpper(s[0:1]),s[1:])
}

func convert2Map(val reflect.Value) map[string]interface{}{
	nmp := map[string]interface{}{}
	typ := val.Type()
	switch typ.Kind(){
	case reflect.Map:
		for _,key := range val.MapKeys(){
			nmp[fmt.Sprintf("%v",reflect.Indirect(key).Interface())] = reflect.Indirect(val.MapIndex(key)).Interface()
		}
		break
	case reflect.Struct:
		for i:=0; i<val.NumField();i++{
			fval := val.Field(i)
			nmp[buildKey(typ.Field(i).Name)] = reflect.Indirect(fval).Interface()
		}
		break
	}
	return nmp
}
func convert2Slice(val reflect.Value) []interface{}{
	ns := []interface{}{}
	for i:=0; i<val.Len();i++{
		item := val.Index(i)
		ns = append(ns,reflect.Indirect(item).Interface())
	}
	return ns
}

func validParams(inPtr interface{}) error{
	val = reflect.ValueOf(inPtr)
	typ := reflect.Indirect(val).Type()

	if typ.Kind() == reflect.Ptr{
		return fmt.Errorf("use two references to the struce %v",typ)
	}
	return nil
}

func validValue(m interface{}) bool{
	typ := reflect.TypeOf(m)
	switch typ.String(){
	case "string":
		ms :=m.(string)
		return len(ms)>0
	case "bool",
	 "int","int8","int16","int32",
	  "uint","uint8","uint16","uint32",
	   "int64","uint64",
	    "float32","float64":
		return true
	case "time.Time":
		return !m.(time.Time).IsZero()
	}
	switch typ.Kind(){
	case reflect.Slice:
		val := reflect.ValueOf(m)
		return val.Len()>0
	}
	log.Info("not support convert type %v",typ)
	return true
}