package orm

import (
	"fmt"
	log "github.com/astaxie/beego/logs"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func change2String(val interface{}) (string, error) {
	switch val.(type) {
	case string:
		return val.(string), nil
	default:
		return fmt.Sprintf("%v", val), nil
	}
}
func change2Bool(val interface{}) (bool, error) {
	switch val.(type) {
	case bool:
		return val.(bool), nil
	default:
		return strconv.ParseBool(fmt.Sprintf("%v", val))
	}
}
func change2Int(val interface{}) (int, error) {
	switch val.(type) {
	case int:
		return val.(int), nil
	case int8:
		return int(val.(int8)), nil
	case int16:
		return int(val.(int16)), nil
	case int32:
		return int(val.(int32)), nil
	case uint:
		return int(val.(uint)), nil
	case uint8:
		return int(val.(uint8)), nil
	case uint16:
		return int(val.(uint16)), nil
	case uint32:
		return int(val.(uint32)), nil
	case int64:
		return int(val.(int64)), nil
	case uint64:
		return int(val.(uint64)), nil
	default:
		return strconv.Atoi(fmt.Sprintf("%v", val))
	}
}
func change2Int8(val interface{}) (int8, error) {
	switch val.(type) {
	case int:
		return int8(val.(int)), nil
	case int8:
		return val.(int8), nil
	case int16:
		return int8(val.(int16)), nil
	case int32:
		return int8(val.(int32)), nil
	case uint:
		return int8(val.(uint)), nil
	case uint8:
		return int8(val.(uint8)), nil
	case uint16:
		return int8(val.(uint16)), nil
	case uint32:
		return int8(val.(uint32)), nil
	case int64:
		return int8(val.(int64)), nil
	case uint64:
		return int8(val.(uint64)), nil
	default:
		nv, err := strconv.ParseInt(fmt.Sprintf("%v", val), 10, 0)
		if err != nil {
			return int8(0), err
		}
		return int8(nv), nil
	}
}
func change2Int16(val interface{}) (int16, error) {
	switch val.(type) {
	case int:
		return int16(val.(int)), nil
	case int8:
		return int16(val.(int8)), nil
	case int16:
		return val.(int16), nil
	case int32:
		return int16(val.(int32)), nil
	case uint:
		return int16(val.(uint)), nil
	case uint8:
		return int16(val.(uint8)), nil
	case uint16:
		return int16(val.(uint16)), nil
	case uint32:
		return int16(val.(uint32)), nil
	case int64:
		return int16(val.(int64)), nil
	case uint64:
		return int16(val.(uint64)), nil
	default:
		nv, err := strconv.ParseInt(fmt.Sprintf("%v", val), 10, 0)
		if err != nil {
			return int16(0), err
		}
		return int16(nv), nil
	}
}
func change2Int32(val interface{}) (int32, error) {
	switch val.(type) {
	case int:
		return int32(val.(int)), nil
	case int8:
		return int32(val.(int8)), nil
	case int16:
		return int32(val.(int16)), nil
	case int32:
		return val.(int32), nil
	case uint:
		return int32(val.(uint)), nil
	case uint8:
		return int32(val.(uint8)), nil
	case uint16:
		return int32(val.(uint16)), nil
	case uint32:
		return int32(val.(uint32)), nil
	case int64:
		return int32(val.(int64)), nil
	case uint64:
		return int32(val.(uint64)), nil
	default:
		nv, err := strconv.ParseInt(fmt.Sprintf("%v", val), 10, 0)
		if err != nil {
			return int32(0), err
		}
		return int32(nv), nil
	}
}
func change2Int64(val interface{}) (int64, error) {
	switch val.(type) {
	case int:
		return int64(val.(int)), nil
	case int8:
		return int64(val.(int8)), nil
	case int16:
		return int64(val.(int16)), nil
	case int32:
		return int64(val.(int32)), nil
	case uint:
		return int64(val.(uint)), nil
	case uint8:
		return int64(val.(uint8)), nil
	case uint16:
		return int64(val.(uint16)), nil
	case uint32:
		return int64(val.(uint32)), nil
	case int64:
		return int64(val.(int64)), nil
	case uint64:
		return int64(val.(uint64)), nil
	default:
		return strconv.ParseInt(fmt.Sprintf("%v", val), 10, 0)
	}
}
func change2UInt(val interface{}) (uint, error) {
	switch val.(type) {
	case int:
		return uint(val.(int)), nil
	case int8:
		return uint(val.(int8)), nil
	case int16:
		return uint(val.(int16)), nil
	case int32:
		return uint(val.(int32)), nil
	case uint:
		return val.(uint), nil
	case uint8:
		return uint(val.(uint8)), nil
	case uint16:
		return uint(val.(uint16)), nil
	case uint32:
		return uint(val.(uint32)), nil
	case int64:
		return uint(val.(int64)), nil
	case uint64:
		return uint(val.(uint64)), nil
	default:
		nv, err := strconv.ParseUint(fmt.Sprintf("%v", val), 10, 0)
		if err != nil {
			return uint(0), err
		}
		return uint(nv), nil
	}
}
func change2UInt8(val interface{}) (uint8, error) {
	switch val.(type) {
	case int:
		return uint8(val.(int)), nil
	case int8:
		return uint8(val.(int8)), nil
	case int16:
		return uint8(val.(int16)), nil
	case int32:
		return uint8(val.(int32)), nil
	case uint:
		return uint8(val.(uint)), nil
	case uint8:
		return val.(uint8), nil
	case uint16:
		return uint8(val.(uint16)), nil
	case uint32:
		return uint8(val.(uint32)), nil
	case int64:
		return uint8(val.(int64)), nil
	case uint64:
		return uint8(val.(uint64)), nil
	default:
		nv, err := strconv.ParseUint(fmt.Sprintf("%v", val), 10, 0)
		if err != nil {
			return uint8(0), err
		}
		return uint8(nv), nil
	}
}
func change2UInt16(val interface{}) (uint16, error) {
	switch val.(type) {
	case int:
		return uint16(val.(int)), nil
	case int8:
		return uint16(val.(int8)), nil
	case int16:
		return uint16(val.(int16)), nil
	case int32:
		return uint16(val.(int32)), nil
	case uint:
		return uint16(val.(uint)), nil
	case uint8:
		return uint16(val.(uint8)), nil
	case uint16:
		return val.(uint16), nil
	case uint32:
		return uint16(val.(uint32)), nil
	case int64:
		return uint16(val.(int64)), nil
	case uint64:
		return uint16(val.(uint64)), nil
	default:
		nv, err := strconv.ParseUint(fmt.Sprintf("%v", val), 10, 0)
		if err != nil {
			return uint16(0), err
		}
		return uint16(nv), nil
	}
}
func change2UInt32(val interface{}) (uint32, error) {
	switch val.(type) {
	case int:
		return uint32(val.(int)), nil
	case int8:
		return uint32(val.(int8)), nil
	case int16:
		return uint32(val.(int16)), nil
	case int32:
		return uint32(val.(int32)), nil
	case uint:
		return uint32(val.(uint)), nil
	case uint8:
		return uint32(val.(uint8)), nil
	case uint16:
		return uint32(val.(uint16)), nil
	case uint32:
		return val.(uint32), nil
	case int64:
		return uint32(val.(int64)), nil
	case uint64:
		return uint32(val.(uint64)), nil
	default:
		nv, err := strconv.ParseUint(fmt.Sprintf("%v", val), 10, 0)
		if err != nil {
			return uint32(0), err
		}
		return uint32(nv), nil
	}
}
func change2UInt64(val interface{}) (uint64, error) {
	switch val.(type) {
	case int:
		return uint64(val.(int)), nil
	case int8:
		return uint64(val.(int8)), nil
	case int16:
		return uint64(val.(int16)), nil
	case int32:
		return uint64(val.(int32)), nil
	case uint:
		return uint64(val.(uint)), nil
	case uint8:
		return uint64(val.(uint8)), nil
	case uint16:
		return uint64(val.(uint16)), nil
	case uint32:
		return uint64(val.(uint32)), nil
	case int64:
		return uint64(val.(int64)), nil
	case uint64:
		return val.(uint64), nil
	default:
		return strconv.ParseUint(fmt.Sprintf("%v", val), 10, 0)
	}
}
func change2Float32(val interface{}) (float32, error) {
	switch val.(type) {
	case int:
		return float32(val.(int)), nil
	case int8:
		return float32(val.(int8)), nil
	case int16:
		return float32(val.(int16)), nil
	case int32:
		return float32(val.(int32)), nil
	case uint:
		return float32(val.(uint)), nil
	case uint8:
		return float32(val.(uint8)), nil
	case uint16:
		return float32(val.(uint16)), nil
	case uint32:
		return float32(val.(uint32)), nil
	case int64:
		return float32(val.(int64)), nil
	case uint64:
		return float32(val.(uint64)), nil
	case float32:
		return val.(float32), nil
	case float64:
		return float32(val.(float64)), nil
	default:
		fv, err := strconv.ParseFloat(fmt.Sprintf("%v", val), 0)
		if err != nil {
			return float32(0.0), err
		}
		return float32(fv), nil
	}
}
func change2Float64(val interface{}) (float64, error) {
	switch val.(type) {
	case int:
		return float64(val.(int)), nil
	case int8:
		return float64(val.(int8)), nil
	case int16:
		return float64(val.(int16)), nil
	case int32:
		return float64(val.(int32)), nil
	case uint:
		return float64(val.(uint)), nil
	case uint8:
		return float64(val.(uint8)), nil
	case uint16:
		return float64(val.(uint16)), nil
	case uint32:
		return float64(val.(uint32)), nil
	case int64:
		return float64(val.(int64)), nil
	case uint64:
		return float64(val.(uint64)), nil
	case float32:
		return float64(val.(float32)), nil
	case float64:
		return val.(float64), nil
	default:
		return strconv.ParseFloat(fmt.Sprintf("%v", val), 0)
	}
}
func change2Time(val interface{}) (time.Time, error) {
	switch val.(type) {
	case time.Time:
		return val.(time.Time), nil
	default:
		return time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%v", val))
	}
}
func changeType(val interface{}, typ reflect.Type) (interface{}, error) {
	switch typ.String() {
	case "string":
		return change2String(val)
	case "bool":
		return change2Bool(val)
	case "int":
		return change2Int(val)
	case "int8":
		return change2Int8(val)
	case "int16":
		return change2Int16(val)
	case "int32":
		return change2Int32(val)
	case "uint":
		return change2UInt(val)
	case "uint8":
		return change2UInt8(val)
	case "uint16":
		return change2UInt16(val)
	case "uint32":
		return change2UInt32(val)
	case "int64":
		return change2Int64(val)
	case "uint64":
		return change2UInt64(val)
	case "float32":
		return change2Float32(val)
	case "float64":
		return change2Float64(val)
	case "time.Time":
		return change2Time(val)
	}
	log.Warn("not support convert type: %v ,value: %v", typ, val)
	return nil, fmt.Errorf("not support convert type: %v ,value: %v", typ, val)
}
func isNumberType(typ reflect.Type) bool{
	switch strings.ToLower(typ.String()){
	case "int8":
		return true
	case "int16":
		return true
	case "int32":
		return true
	case "int64":
		return true
	case "uint":
		return true
	case "uint8":
		return true
	case "uint16":
		return true
	case "uint32":
		return true
	case "uint64":
		return true
	case "float32":
		return true
	case "float64":
		return true
	}
	return false
}

func isStringType(typ reflect.Type) bool{
	return strings.Compare(strings.ToLower(typ.String()),"string")==0
}

func sameTypeCheck(typA,typB reflect.Type) bool{
	if strings.Compare(strings.ToLower(typA.String()),strings.ToLower(typB.String()))==0{
		return true
	}
	if isNumberType(typA) && isNumberType(typB){
		return true
	}
	if isNumberType(typA) && isStringType(typB){
		return true
	}
	if isStringType(typA) && isNumberType(typB){
		return true
	}
	return false
}

