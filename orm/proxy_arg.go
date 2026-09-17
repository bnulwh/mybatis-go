package orm

import (
	"context"
	"reflect"
)

// contextType context.Context 接口的反射类型（G0：Mapper 方法 ctx 参数自动提取）。
var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()

type ProxyArg struct {
	TagArgs    []TagArg
	TagArgsLen int
	Args       []reflect.Value
	ArgsLen    int
	PageParam  *PageParam
	Wrapper    *QueryWrapper // P0-1：QueryWrapper 参数（自动提取，照搬 PageParam 模式）
	Ctx        context.Context // G0：context.Context 参数（自动提取，参与 WithTx 事务/超时控制）
}

func NewProxyArg(tagArgs []TagArg, args []reflect.Value) ProxyArg {
	var pp *PageParam
	var qw *QueryWrapper
	var ctx context.Context
	var filtered []reflect.Value
	for _, arg := range args {
		if arg.Type() == contextType {
			if c, ok := arg.Interface().(context.Context); ok && c != nil {
				ctx = c
			}
			continue
		}
		if arg.Type() == pageParamType {
			pp = arg.Interface().(*PageParam)
			continue
		}
		if arg.Type() == queryWrapperType {
			if !arg.IsNil() {
				qw = arg.Interface().(*QueryWrapper)
			}
			continue
		}
		filtered = append(filtered, arg)
	}
	if filtered == nil {
		filtered = []reflect.Value{}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return ProxyArg{
		TagArgs:    tagArgs,
		Args:       filtered,
		ArgsLen:    len(filtered),
		TagArgsLen: len(tagArgs),
		PageParam:  pp,
		Wrapper:    qw,
		Ctx:        ctx,
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
