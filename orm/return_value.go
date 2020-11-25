package orm

import "reflect"

func buildReturnValues(returnType *ReturnType, returnValue *reflect.Value, e error) []reflect.Value {
	var returnValues = make([]reflect.Value, returnType.NumOut)
	for index, _ := range returnValues {
		if index == returnType.ReturnIndex {
			if returnValue != nil {
				returnValues[index] = (*returnValue).Elem()
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
