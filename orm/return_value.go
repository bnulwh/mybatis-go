package orm

import (
	log "github.com/bnulwh/logrus"
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
				returnValues[index] = returnValue
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
