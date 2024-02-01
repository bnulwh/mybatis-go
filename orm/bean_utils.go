package orm

import (
	"bytes"
	"github.com/bnulwh/mybatis-go/log"
	"reflect"
)

//check beans
func beanCheck(value reflect.Value) {
	var t = value.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		var fieldItem = t.Field(i)
		if fieldItem.Type.Kind() != reflect.Func {
			continue
		}
		var argsLen = fieldItem.Type.NumIn() //参数长度，除session参数外。
		var customLen = 0
		for argIndex := 0; argIndex < fieldItem.Type.NumIn(); argIndex++ {
			var inType = fieldItem.Type.In(argIndex)
			if isCustomStruct(inType) {
				customLen++
			}
		}
		if argsLen > 1 && customLen > 1 {
			panic(`[mybatis-go] ` + fieldItem.Name + ` must add tag "args:"*,*..."`)
		}
	}
}

func isCustomStruct(value reflect.Type) bool {
	if value.Kind() == reflect.Struct && value.String() != "time.Time" && value.String() != "*time.Time" {
		return true
	} else {
		return false
	}
}

//方法基本规则检查
func methodFieldCheck(beanType *reflect.Type, methodType *reflect.StructField, warning bool) {
	if methodType.Type.NumOut() < 1 {
		var buffer bytes.Buffer
		buffer.WriteString("[mybatis-go] bean ")
		buffer.WriteString((*beanType).Name())
		buffer.WriteString(".")
		buffer.WriteString(methodType.Name)
		buffer.WriteString("() must be return a 'error' type!")
		panic(buffer.String())
	}
	var errorTypeNum = 0
	for i := 0; i < methodType.Type.NumOut(); i++ {
		var outType = methodType.Type.Out(i)
		if outType.Kind() == reflect.Interface && outType.String() == "error" {
			errorTypeNum++
		}
	}
	if errorTypeNum != 1 {
		var buffer bytes.Buffer
		buffer.WriteString("[mybatis-go] bean ")
		buffer.WriteString((*beanType).Name())
		buffer.WriteString(".")
		buffer.WriteString(methodType.Name)
		buffer.WriteString("() must be return a 'error' type!")
		panic(buffer.String())
	}

	var args = methodType.Tag.Get("args")
	if methodType.Type.NumIn() > 1 && args == "" {
		if warning {
			log.Warnf("[mybatis-go] warning ======================== " + (*beanType).Name() + "." + methodType.Name + "() have not define tag args:\"\",maybe can not get param value!")
		}
	}
}
