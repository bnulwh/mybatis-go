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
	Variadic   bool // 变参函数（...T）：不校验 tag 与 NumIn 严格相等（1.3）
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

func makeParamType(funcName string, funcType reflect.Type, funcTag reflect.StructTag) (*ParamType, error) {
	if funcType.Kind() != reflect.Func {
		return nil, fmt.Errorf("[mybatis-go] %v is not a func kind, got %v", funcName, funcType.Kind())
	}
	if funcType.NumIn() == 0 {
		return &ParamType{
			TagArgs:    []TagArg{},
			TagArgsLen: 0,
			Args:       []reflect.Type{},
			ArgsLen:    0,
		}, nil
	}
	tagArgs := parseTagArgs(getTagArgNames(funcTag))
	variadic := funcType.IsVariadic()
	if !variadic { // 变参不校验：NumIn 只反映 []T 一个槽位（1.3）
		if len(tagArgs) > funcType.NumIn() {
			return nil, fmt.Errorf(`[mybatis-go] method fail! the tag "args" length can not > arg length ! filed=%s`, funcName)
		}
		if len(tagArgs) > 0 && funcType.NumIn() != len(tagArgs) {
			return nil, fmt.Errorf(`[mybatis-go] method fail! the tag "args" length  != args length ! filed=%s`, funcName)
		}
	}
	var args []reflect.Type
	for i := 0; i < funcType.NumIn(); i++ {
		args = append(args, funcType.In(i))
	}
	return &ParamType{
		TagArgs:    tagArgs,
		TagArgsLen: len(tagArgs),
		Args:       args,
		ArgsLen:    len(args),
		Variadic:   variadic,
	}, nil
}
