package orm

import (
	"fmt"
	"github.com/bnulwh/mybatis-go/types"
	"reflect"
)

type ParamType struct {
	TagArgs    []TagArg
	TagArgsLen int
	Args       []reflect.Type
	ArgsLen    int
}

func (in *ParamType) checkSql(f *types.SqlFunction, name string) {
	if in.ArgsLen == 0 && f.Param.Need {
		panic(fmt.Sprintf("%v check sql function %v failed, need func args", name, f.Id))
	}
	if in.ArgsLen > 0 && !f.Param.Need {
		panic(fmt.Sprintf("%v check sql function %v failed, not need func args", name, f.Id))
	}
}

func makeParamType(funcName string, funcType reflect.Type, funcTag reflect.StructTag) *ParamType {
	if funcType.Kind() != reflect.Func {
		return nil
	}
	if funcType.NumIn() == 0 {
		return &ParamType{
			TagArgs:    []TagArg{},
			TagArgsLen: 0,
			Args:       []reflect.Type{},
			ArgsLen:    0,
		}
	}
	tagArgs := parseTagArgs(funcTag.Get(`args`))
	if len(tagArgs) > funcType.NumIn() {
		panic(`[mybatis-go] method fail! the tag "args" length can not > arg length ! filed=` + funcName)
	}
	var tagArgsLen = len(tagArgs)
	if tagArgsLen > 0 && funcType.NumIn() != tagArgsLen {
		panic(`[mybatis-go] method fail! the tag "args" length  != args length ! filed = ` + funcName)
	}
	var args []reflect.Type
	for i := 0; i < funcType.NumIn(); i++ {
		args = append(args, funcType.In(i))
	}
	return &ParamType{
		TagArgs:    tagArgs,
		TagArgsLen: tagArgsLen,
		Args:       args,
		ArgsLen:    len(args),
	}
}
