package orm

import (
	"fmt"
	"github.com/bnulwh/mybatis-go/types"
	"github.com/bnulwh/mybatis-go/utils"
	"reflect"
	"strings"
)

type ReturnType struct {
	ErrorType     *reflect.Type
	ReturnOutType *reflect.Type
	ReturnIndex   int //返回数据位置索引
	NumOut        int //返回总数
}

func (in *ReturnType) checkSql(f *types.SqlFunction, name string) {
	typ := *in.ReturnOutType
	if typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}
	if f.Result.ResultM != nil && typ.Kind() == reflect.Struct {
		rname := types.GetShortName(f.Result.ResultM.TypeName)
		sname := types.GetShortName(typ.Name())
		if strings.Compare(strings.ToLower(rname), strings.ToLower(sname)) != 0 {
			panic(fmt.Sprintf("%v check sql function %v failed, return type valid failed `%v` != `%v` ",
				name, f.Id, f.Result.ResultM.TypeName, typ.String()))
		}
	} else {
		if !utils.SameTypeCheck(f.Result.ResultT, typ) {
			panic(fmt.Sprintf("%v check sql function %v failed, return type valid failed `%v` != `%v`",
				name, f.Id, f.Result.ResultT.String(), typ.String()))
		}
	}
}

func makeReturnType(funcName string, funcType reflect.Type) *ReturnType {
	if funcType.Kind() != reflect.Func {
		return nil
	}
	var numOut = funcType.NumOut()
	if numOut > 2 || numOut == 0 {
		panic("[mybatis-go] func '" + funcName + "()' return num out must = 1 or = 2!")
	}
	returnType := &ReturnType{
		ReturnIndex: -1,
		NumOut:      numOut,
	}
	for f := 0; f < numOut; f++ {
		var outType = funcType.Out(f)
		//过滤NewSession方法
		if outType.Kind() == reflect.Ptr || (outType.Kind() == reflect.Interface && outType.String() != "error") {
			panic("[mybatis-go] func '" + funcName + "()' return '" + outType.String() + "' can not be a 'ptr' or 'interface'!")
		}
		if outType.String() != "error" {
			returnType.ReturnIndex = f
			returnType.ReturnOutType = &outType
		} else {
			//error
			returnType.ErrorType = &outType
		}
	}
	if returnType.ErrorType == nil {
		panic("[mybatis-go] func '" + funcName + "()' must return an 'error'!")
	}
	return returnType
}

func makeReturnTypeMap(value reflect.Type) map[string]*ReturnType {
	returnMap := make(map[string]*ReturnType)
	var proxyType = value
	for i := 0; i < proxyType.NumField(); i++ {
		var funcType = proxyType.Field(i).Type
		var funcName = proxyType.Field(i).Name

		if funcType.Kind() != reflect.Func {
			if funcType.Kind() == reflect.Struct {
				var childMap = makeReturnTypeMap(funcType)
				for k, v := range childMap {
					returnMap[k] = v
				}
			}
			continue
		}
		returnMap[funcName] = makeReturnType(funcName, funcType)
	}
	return returnMap
}
