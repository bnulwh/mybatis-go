package orm

import (
	"reflect"
	"strings"
)

type TagArg struct {
	Name  string
	Index int
}

//代理数据
type ProxyArg struct {
	TagArgs    []TagArg
	TagArgsLen int
	Args       []reflect.Value
	ArgsLen    int
}

func parseTagArgs(tagstr string) []TagArg {
	var tagArgs = make([]TagArg, 0)
	if len(tagstr) == 0 {
		return tagArgs
	}
	tagParams := strings.Split(tagstr, `,`)
	if len(tagParams) != 0 {
		for index, v := range tagParams {
			var tagArg = TagArg{
				Index: index,
				Name:  v,
			}
			tagArgs = append(tagArgs, tagArg)
		}
	}
	return tagArgs
}

func (it ProxyArg) New(tagArgs []TagArg, args []reflect.Value) ProxyArg {
	return ProxyArg{
		TagArgs:    tagArgs,
		Args:       args,
		TagArgsLen: len(tagArgs),
		ArgsLen:    len(args),
	}
}
func (in *ProxyArg) buildArgs() []interface{} {
	var args []interface{}
	if in.TagArgsLen == 0 {
		for _, arg := range in.Args {
			args = append(args, arg.Interface())
		}
	} else {
		mp := make(map[string]interface{})
		var i = 0;
		for ; i < in.TagArgsLen; i++ {
			if i < in.ArgsLen {
				mp[in.TagArgs[i].Name] = in.Args[i].Interface()
			}
		}
		args = append(args, mp)
		for ; i < in.ArgsLen; i++ {
			args = append(args, in.Args[i].Interface())
		}
	}
	return args
}

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

func buildProxy(v reflect.Value, buildFunc func(funcField reflect.StructField, field reflect.Value) func(arg ProxyArg) []reflect.Value) {
	for {
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		} else {
			break
		}
	}
	t := v.Type()
	et := t
	if et.Kind() == reflect.Ptr {
		et = et.Elem()
	}
	ptr := v
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
					buildRemoteMethod(v, fieldVal, fieldTyp, sructField, buildFunc(sructField, fieldVal))
				}
			}
		}
	}
	if t.Kind() == reflect.Ptr {
		v.Set(ptr)
	} else {
		v.Set(obj)
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
		proxyResults := proxyFunc(ProxyArg{}.New(tagArgs, args))
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
	//println("[mybatis-go] write method success:" + source.Type().Name() + " > " + sf.Name + " " + f.Type().String())
	//tagParams = nil
}
