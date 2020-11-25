package orm

import (
	"reflect"
)

type ReturnType struct {
	ErrorType     *reflect.Type
	ReturnOutType *reflect.Type
	ReturnIndex   int //返回数据位置索引
	NumOut        int //返回总数
}

func makeReturnTypeMap(value reflect.Type) map[string]*ReturnType {
	returnMap := make(map[string]*ReturnType)
	var proxyType = value
	for i := 0; i < proxyType.NumField(); i++ {
		var funcType = proxyType.Field(i).Type
		var funcName = proxyType.Field(i).Name

		if funcType.Kind() != reflect.Func {
			if funcType.Kind() == reflect.Struct {
				var childMap = makeReturnTypeMap(funcType)
				for k, v := range childMap {
					returnMap[k] = v
				}
			}
			continue
		}

		var numOut = funcType.NumOut()
		if numOut > 2 || numOut == 0 {
			panic("[mybatis-go] func '" + funcName + "()' return num out must = 1 or = 2!")
		}
		for f := 0; f < numOut; f++ {
			var outType = funcType.Out(f)
			//过滤NewSession方法
			if outType.Kind() == reflect.Ptr || (outType.Kind() == reflect.Interface && outType.String() != "error") {
				panic("[mybatis-go] func '" + funcName + "()' return '" + outType.String() + "' can not be a 'ptr' or 'interface'!")
			}

			var returnType = returnMap[funcName]
			if returnType == nil {
				returnMap[funcName] = &ReturnType{
					ReturnIndex: -1,
					NumOut:      numOut,
				}
			}
			if outType.String() != "error" {
				returnMap[funcName].ReturnIndex = f
				returnMap[funcName].ReturnOutType = &outType
			} else {
				//error
				returnMap[funcName].ErrorType = &outType
			}
		}
		if returnMap[funcName].ErrorType == nil {
			panic("[mybatis-go] func '" + funcName + "()' must return an 'error'!")
		}
	}
	return returnMap
}
