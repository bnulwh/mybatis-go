package orm

import (
	"reflect"
)

// 代理数据
type ProxyArg struct {
	TagArgs    []TagArg
	TagArgsLen int
	Args       []reflect.Value
	ArgsLen    int
}

func NewProxyArg(tagArgs []TagArg, args []reflect.Value) ProxyArg {
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
		var i = 0
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
