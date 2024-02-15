package orm

import (
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
	"reflect"
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
	panic(fmt.Sprintf("convert val %v to type %v failed", val.Interface(), typ))
}
