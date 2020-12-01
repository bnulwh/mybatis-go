package orm

import (
	"fmt"
	log "github.com/astaxie/beego/logs"
	"github.com/bnulwh/mybatis-go/types"
	"reflect"
	"strings"
)

type ormCache struct {
	models  modelCache
	mappers mapperCache
}

var (
	gCache ormCache
)

func init() {
	gCache = ormCache{
		models:  modelCache{Models: map[string]reflect.Type{}},
		mappers: mapperCache{Mappers: map[string]*mapperInfo{}},
	}
}


func (in *ormCache) createModel(name string) (reflect.Value, error) {
	return in.models.createModel(name)
}

func (in *ormCache) createMapper(name string) (reflect.Value, error) {
	return in.mappers.createMapper(name)
}

func RegisterModel(inPtr interface{}) {
	gCache.models.registerModel(inPtr)
}
func RegisterMapper(inPtr interface{}) {
	gCache.mappers.registerMapper(inPtr)
}

func NewMapper(name string) interface{} {
	mp, err := gCache.createMapper(name)
	if err != nil {
		log.Warn("cannot find mapper struct `%s`", name)
		panic(err)
	}
	bindMapper(name, mp)
	return reflect.Indirect(mp).Interface()
}

func NewMapperPtr(name string) interface{} {
	mp, err := gCache.createMapper(name)
	if err != nil {
		log.Warn("cannot find mapper struct `%s`", name)
		panic(err)
	}
	bindMapper(name, mp)
	return mp.Interface()
}

func bindMapper(name string, value reflect.Value) {
	sn := types.GetShortName(name)
	mp, ok := gMappers.NamedMappers[strings.ToLower(sn)]
	if !ok {
		panic(fmt.Sprintf("bind mapper struct `%s` failed,not found in xml files", name))
	}
	outVal := value.Elem()
	outTyp := outVal.Type()
	bmf := outVal.FieldByName("BaseMapper")
	bmf.Set(reflect.ValueOf(BaseMapper{mapper: mp}))
	bm := bmf.Interface().(BaseMapper)
	returnTypeMap := makeReturnTypeMap(outTyp)
	proxyValue(value, func(funcField reflect.StructField, field reflect.Value) func(arg ProxyArg) []reflect.Value {
		//构建期
		var funcName = funcField.Name
		var returnType = returnTypeMap[funcName]
		if returnType == nil {
			panic("[mybatis-go] struct have no return values!")
		}
		//mapper
		sqlFunc, err := bm.fetchSqlFunction(funcName)
		if err != nil {
			panic(err)
		}
		// methodFieldCheck(&outTyp, &funcField, true)
		//执行期
		var proxyFunc = func(arg ProxyArg) []reflect.Value {
			//var returnValue *reflect.Value = nil
			////build return Type
			//if returnType.ReturnOutType != nil {
			//	var returnV = reflect.New(*returnType.ReturnOutType)
			//	switch (*returnType.ReturnOutType).Kind() {
			//	case reflect.Map:
			//		returnV.Elem().Set(reflect.MakeMap(*returnType.ReturnOutType))
			//	case reflect.Slice:
			//		returnV.Elem().Set(reflect.MakeSlice(*returnType.ReturnOutType, 0, 0))
			//	}
			//	returnValue = &returnV
			//}
			//exe sql
			rv, e := bm.executeMethod(sqlFunc, arg)
			return buildReturnValues(returnType, rv, e)
		}
		return proxyFunc

	})
}

func getFullName(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name()
}
