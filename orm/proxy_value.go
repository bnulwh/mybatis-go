package orm

import (
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
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

func buildRemoteMethod(source reflect.Value, fieldVal reflect.Value, fieldTyp reflect.Type, structField reflect.StructField, proxyFunc func(arg ProxyArg) []reflect.Value) {
	var tagArgs = parseTagArgs(structField.Tag.Get(`args`))
	if len(tagArgs) > fieldTyp.NumIn() {
		panic(`[mybatis-go] method fail! the tag "args" length can not > arg length ! filed=` + structField.Name)
	}
	var tagArgsLen = len(tagArgs)
	if tagArgsLen > 0 && fieldTyp.NumIn() != tagArgsLen {
		panic(`[mybatis-go] method fail! the tag "args" length  != args length ! filed = ` + structField.Name)
	}
	var fn = func(args []reflect.Value) (results []reflect.Value) {
		proxyResults := proxyFunc(NewProxyArg(tagArgs, args))
		for _, returnV := range proxyResults {
			results = append(results, returnV)
		}
		return results
	}
	vfn := reflect.ValueOf(fn)
	if fieldVal.Kind() == reflect.Ptr {
		fp := reflect.New(fieldTyp)
		fp.Elem().Set(reflect.MakeFunc(fieldTyp, fn))
		err := checkResults(vfn, fp.Elem())
		if err != nil {
			panic(fmt.Sprintf("%v %v %v %v", err, source.Type().Name(), structField.Name, fieldVal.Type().String()))
		}
		fieldVal.Set(fp)
	} else {
		err := checkResults(vfn, fieldVal)
		if err != nil {
			panic(fmt.Sprintf("%v %v %v %v", err, source.Type().Name(), structField.Name, fieldVal.Type().String()))
		}
		fieldVal.Set(reflect.MakeFunc(fieldTyp, fn))
	}
	log.Debugf("[mybatis-go] write method success:" + source.Type().Name() + " > " + structField.Name + " " + fieldVal.Type().String())
	//tagParams = nil
}

func checkResults(vfn, fval reflect.Value) error {
	if vfn.Kind() != reflect.Func || fval.Kind() != reflect.Func {
		return fmt.Errorf(`[mybatis-go] method fail! wrong kind check`)
	}
	if vfn.Type().NumOut() != fval.Type().NumOut() {
		return fmt.Errorf(`[mybatis-go] method fail! wrong num out check %v %v`, vfn.Type().NumOut(), fval.Type().NumOut())
	}
	if vfn.Type().NumIn() != fval.Type().NumIn() {
		return fmt.Errorf(`[mybatis-go] method fail! wrong num in check %v %v`, vfn.Type().NumIn(), fval.Type().NumIn())
	}
	for i := 0; i < vfn.Type().NumIn(); i++ {
		if vfn.Type().In(i) != fval.Type().In(i) {
			return fmt.Errorf(`[mybatis-go] method fail! wrong num in check params pos %v, %v, %v`, i, vfn.Type().In(i), fval.Type().In(i))
		}
	}
	for i := 0; i < vfn.Type().NumOut(); i++ {
		if vfn.Type().Out(i) != fval.Type().Out(i) {
			return fmt.Errorf(`[mybatis-go] method fail! wrong num out check params pos %v, %v, %v`, i, vfn.Type().Out(i), fval.Type().Out(i))
		}
	}
	return nil
}
