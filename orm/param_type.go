package orm

import (
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
	"reflect"
)

type ParamType struct {
	TagArgs    []TagArg
	TagArgsLen int
	Args       []reflect.Type
	ArgsLen    int
}

func (in *ParamType) checkSql(f *types.SqlFunction, name string) error {
	if in.ArgsLen == 0 && f.Param.Need {
		return fmt.Errorf("%v check sql function %v failed, need func args: %v", name, f.Id, f.Param.Slots)
	}
	// 1.1（M-06）：有参函数 + 语句无显式 parameterType 不再失败。
	// Need 已由语句占位符推导（SqlParam.Slots）；语句无占位符但有参时按 Java 语义
	// 容忍「未使用参数」，仅 debug 提示。
	if in.ArgsLen > 0 && !f.Param.Need {
		log.Debugf("%v check sql function %v: statement has no param slots, %d func args ignored", name, f.Id, in.ArgsLen)
		return nil
	}
	return nil
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
	tagArgs := parseTagArgs(getTagArgNames(funcTag))
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
