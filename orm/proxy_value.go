package orm

import (
	log "github.com/bnulwh/logrus"
	"reflect"
)

// AopProxy 可写入每个函数代理方法.proxyPtr:代理对象指针，buildFunc:构建代理函数
func proxy(proxyPtr interface{}, buildFunc func(funcField reflect.StructField, field reflect.Value) func(arg ProxyArg) []reflect.Value) {
	v := reflect.ValueOf(proxyPtr)
	if v.Kind() != reflect.Ptr {
		panic("AopProxy: AopProxy arg must be a pointer")
	}
	buildProxy(v, buildFunc)
}

// AopProxy 可写入每个函数代理方法
func proxyValue(mapperValue reflect.Value, buildFunc func(funcField reflect.StructField, field reflect.Value) func(arg ProxyArg) []reflect.Value) {
	buildProxy(mapperValue, buildFunc)
}

func buildProxy(mapperValue reflect.Value, buildFunc func(funcField reflect.StructField, field reflect.Value) func(arg ProxyArg) []reflect.Value) {
	for {
		if mapperValue.Kind() == reflect.Ptr {
			mapperValue = mapperValue.Elem()
		} else {
			break
		}
	}
	t := mapperValue.Type()
	et := t
	if et.Kind() == reflect.Ptr {
		et = et.Elem()
	}
	ptr := mapperValue
	var obj reflect.Value
	if ptr.Kind() == reflect.Ptr {
		obj = ptr.Elem()
	} else {
		obj = ptr
	}
	count := obj.NumField()
	for i := 0; i < count; i++ {
		fieldVal := obj.Field(i)
		fieldTyp := fieldVal.Type()
		sructField := et.Field(i)
		if fieldTyp.Kind() == reflect.Ptr {
			fieldTyp = fieldTyp.Elem()
		}
		if fieldVal.CanSet() {
			switch fieldTyp.Kind() {
			case reflect.Struct:
				if buildFunc != nil {
					buildProxy(fieldVal, buildFunc) //循环扫描
				}
			case reflect.Func:
				if buildFunc != nil {
					buildRemoteMethod(mapperValue, fieldVal, fieldTyp, sructField, buildFunc(sructField, fieldVal))
				}
			}
		}
	}
	if t.Kind() == reflect.Ptr {
		mapperValue.Set(ptr)
	} else {
		mapperValue.Set(obj)
	}
}

func buildRemoteMethod(source reflect.Value, fieldVal reflect.Value, fieldTyp reflect.Type, sructField reflect.StructField, proxyFunc func(arg ProxyArg) []reflect.Value) {
	var tagArgs = parseTagArgs(sructField.Tag.Get(`args`))
	if len(tagArgs) > fieldTyp.NumIn() {
		panic(`[mybatis-go] method fail! the tag "args" length can not > arg length ! filed=` + sructField.Name)
	}
	var tagArgsLen = len(tagArgs)
	if tagArgsLen > 0 && fieldTyp.NumIn() != tagArgsLen {
		panic(`[mybatis-go] method fail! the tag "args" length  != args length ! filed = ` + sructField.Name)
	}
	var fn = func(args []reflect.Value) (results []reflect.Value) {
		proxyResults := proxyFunc(NewProxyArg(tagArgs, args))
		for _, returnV := range proxyResults {
			results = append(results, returnV)
		}
		return results
	}
	if fieldVal.Kind() == reflect.Ptr {
		fp := reflect.New(fieldTyp)
		fp.Elem().Set(reflect.MakeFunc(fieldTyp, fn))
		fieldVal.Set(fp)
	} else {
		fieldVal.Set(reflect.MakeFunc(fieldTyp, fn))
	}
	log.Debug("[mybatis-go] write method success:" + source.Type().Name() + " > " + sructField.Name + " " + fieldVal.Type().String())
	//tagParams = nil
}
