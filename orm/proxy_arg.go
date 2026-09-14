package orm

import (
	"reflect"
)

type ProxyArg struct {
	TagArgs    []TagArg
	TagArgsLen int
	Args       []reflect.Value
	ArgsLen    int
	PageParam  *PageParam
}

func NewProxyArg(tagArgs []TagArg, args []reflect.Value) ProxyArg {
	var pp *PageParam
	var filtered []reflect.Value
	for _, arg := range args {
		if arg.Type() == pageParamType {
			pp = arg.Interface().(*PageParam)
			continue
		}
		filtered = append(filtered, arg)
	}
	if filtered == nil {
		filtered = []reflect.Value{}
	}
	return ProxyArg{
		TagArgs:    tagArgs,
		Args:       filtered,
		ArgsLen:    len(filtered),
		TagArgsLen: len(tagArgs),
		PageParam:  pp,
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
