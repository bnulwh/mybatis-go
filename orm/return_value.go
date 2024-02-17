package orm

import (
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
	"reflect"
	"strconv"
)

func buildReturnValues(returnType *ReturnType, returnValue reflect.Value, e error) []reflect.Value {
	var returnValues = make([]reflect.Value, returnType.NumOut)
	for index, _ := range returnValues {
		if index == returnType.ReturnIndex {
			if e != nil {
				returnValues[index] = reflect.Zero(*returnType.ReturnOutType)
			} else {

				returnValues[index] = ensureReturnType(*returnType.ReturnOutType, returnValue)
				log.Debugf("results: %v", types.ToJson(reflect.Indirect(returnValue).Interface()))
			}
		} else {
			if e != nil {
				returnValues[index] = reflect.New(*returnType.ErrorType)
				returnValues[index].Elem().Set(reflect.ValueOf(e))
				returnValues[index] = returnValues[index].Elem()
			} else {
				returnValues[index] = reflect.Zero(*returnType.ErrorType)
			}
		}
	}
	return returnValues
}

func ensureReturnType(typ reflect.Type, val reflect.Value) reflect.Value {
	if val.Type() == typ {
		return val
	}
	switch typ.Kind() {
	case reflect.Array:
	case reflect.Slice:
	case reflect.Map:
	case reflect.Struct:

	case reflect.Int:
		nv, err := strconv.ParseInt(fmt.Sprint(val.Interface()), 10, 0)
		if err != nil {
			panic(fmt.Sprintf("convert val %v %v to type %v failed: %v", val.Interface(), val.Type(), typ, err))
		}
		return reflect.ValueOf(int(nv))
	case reflect.Int8:
		nv, err := strconv.ParseInt(fmt.Sprint(val.Interface()), 10, 0)
		if err != nil {
			panic(fmt.Sprintf("convert val %v %v to type %v failed: %v", val.Interface(), val.Type(), typ, err))
		}
		return reflect.ValueOf(int8(nv))
	case reflect.Int16:
		nv, err := strconv.ParseInt(fmt.Sprint(val.Interface()), 10, 0)
		if err != nil {
			panic(fmt.Sprintf("convert val %v %v to type %v failed: %v", val.Interface(), val.Type(), typ, err))
		}
		return reflect.ValueOf(int16(nv))
	case reflect.Int32:
		nv, err := strconv.ParseInt(fmt.Sprint(val.Interface()), 10, 0)
		if err != nil {
			panic(fmt.Sprintf("convert val %v %v to type %v failed: %v", val.Interface(), val.Type(), typ, err))
		}
		return reflect.ValueOf(int32(nv))
	case reflect.Int64:
		nv, err := strconv.ParseInt(fmt.Sprint(val.Interface()), 10, 0)
		if err != nil {
			panic(fmt.Sprintf("convert val %v %v to type %v failed: %v", val.Interface(), val.Type(), typ, err))
		}
		return reflect.ValueOf(int64(nv))
	case reflect.Uint:
		uv, err := strconv.ParseUint(fmt.Sprint(val.Interface()), 10, 0)
		if err != nil {
			panic(fmt.Sprintf("convert val %v %v to type %v failed: %v", val.Interface(), val.Type(), typ, err))
		}
		return reflect.ValueOf(uint(uv))
	case reflect.Uint8:
		uv, err := strconv.ParseUint(fmt.Sprint(val.Interface()), 10, 0)
		if err != nil {
			panic(fmt.Sprintf("convert val %v %v to type %v failed: %v", val.Interface(), val.Type(), typ, err))
		}
		return reflect.ValueOf(uint8(uv))
	case reflect.Uint16:
		uv, err := strconv.ParseUint(fmt.Sprint(val.Interface()), 10, 0)
		if err != nil {
			panic(fmt.Sprintf("convert val %v %v to type %v failed: %v", val.Interface(), val.Type(), typ, err))
		}
		return reflect.ValueOf(uint16(uv))
	case reflect.Uint32:
		uv, err := strconv.ParseUint(fmt.Sprint(val.Interface()), 10, 0)
		if err != nil {
			panic(fmt.Sprintf("convert val %v %v to type %v failed: %v", val.Interface(), val.Type(), typ, err))
		}
		return reflect.ValueOf(uint32(uv))
	case reflect.Uint64:
		uv, err := strconv.ParseUint(fmt.Sprint(val.Interface()), 10, 0)
		if err != nil {
			panic(fmt.Sprintf("convert val %v %v to type %v failed: %v", val.Interface(), val.Type(), typ, err))
		}
		return reflect.ValueOf(uint64(uv))
	case reflect.Float32:
		fv, err := strconv.ParseFloat(fmt.Sprint(val.Interface()), 32)
		if err != nil {
			panic(fmt.Sprintf("convert val %v %v to type %v failed: %v", val.Interface(), val.Type(), typ, err))
		}
		return reflect.ValueOf(float32(fv))
	case reflect.Float64:
		fv, err := strconv.ParseFloat(fmt.Sprint(val.Interface()), 0)
		if err != nil {
			panic(fmt.Sprintf("convert val %v %v to type %v failed: %v", val.Interface(), val.Type(), typ, err))
		}
		return reflect.ValueOf(fv)
	case reflect.String:
		return reflect.ValueOf(fmt.Sprint(val.Interface()))
	}
	panic(fmt.Sprintf("convert val %v %v to type %v failed", val.Interface(), val.Type(), typ))
}
