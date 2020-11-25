package orm

import (
	"fmt"
	log "github.com/astaxie/beego/logs"
	"github.com/bnulwh/mybatis-go/types"
	"reflect"
	"strings"
)

type ormCache struct {
	modelCache  map[string]reflect.Type
	mapperCache map[string]reflect.Type
}

var (
	gCache ormCache
)

func init() {
	gCache = ormCache{
		modelCache:  map[string]reflect.Type{},
		mapperCache: map[string]reflect.Type{},
	}
}

func (in *ormCache) addModel(typ reflect.Type) {
	name := typ.Name()
	sn := types.GetShortName(name)
	in.modelCache[name] = typ
	in.modelCache[strings.ToLower(name)] = typ
	in.modelCache[sn] = typ
	in.modelCache[strings.ToLower(sn)] = typ
	in.modelCache[getFullName(typ)] = typ
}
func (in *ormCache) addMapper(typ reflect.Type) {
	name := typ.Name()
	sn := types.GetShortName(name)
	in.mapperCache[name] = typ
	in.mapperCache[strings.ToLower(name)] = typ
	in.mapperCache[sn] = typ
	in.mapperCache[strings.ToLower(sn)] = typ
	in.mapperCache[getFullName(typ)] = typ
}

func (in *ormCache) createModel(name string) (reflect.Value, error) {
	typ, ok := in.modelCache[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return reflect.ValueOf(-1), fmt.Errorf("model type %s not registered!!!", name)
	}
	return reflect.New(typ), nil
}

func (in *ormCache) createMapper(name string) (reflect.Value, error) {
	typ, ok := in.mapperCache[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return reflect.ValueOf(-1), fmt.Errorf("mapper type %s not registered!!!", name)
	}
	return reflect.New(typ), nil
}

func RegisterModel(inPtr interface{}) {
	val := reflect.ValueOf(inPtr)
	typ := reflect.Indirect(val).Type()
	fn := getFullName(typ)
	if val.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("<orm.RegisterModel> cannot use non-ptr model struct `%s`", fn))
	}
	if typ.Kind() == reflect.Ptr {
		panic(fmt.Sprintf("<orm.RegisterModel> only allow ptr model struct,it looks you use two reference to the struct `%s`", fn))
	}
	log.Info("register  model struct `%s`", fn)
	gCache.addModel(typ)
}
func RegisterMapper(inPtr interface{}) {
	val := reflect.ValueOf(inPtr)
	typ := reflect.Indirect(val).Type()
	fn := getFullName(typ)
	if val.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("<orm.RegisterMapper> cannot use non-ptr mapper struct `%s`", fn))
	}
	if typ.Kind() == reflect.Ptr {
		panic(fmt.Sprintf("<orm.RegisterMapper> only allow ptr mapper struct,it looks you use two reference to the struct `%s`", fn))
	}
	log.Info("register  mapper struct `%s`", fn)
	_, ok := typ.FieldByName("BaseMapper")
	if !ok {
		panic(fmt.Sprintf("<orm.RegisterMapper> can only use mapper struct `%s` based on <orm.BaseMapper>", fn))
	}
	beanCheck(val)
	gCache.addMapper(typ)
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
		methodFieldCheck(&outTyp, &funcField, true)
		//执行期
		var proxyFunc = func(arg ProxyArg) []reflect.Value {
			var returnValue *reflect.Value = nil
			//build return Type
			if returnType.ReturnOutType != nil {
				var returnV = reflect.New(*returnType.ReturnOutType)
				switch (*returnType.ReturnOutType).Kind() {
				case reflect.Map:
					returnV.Elem().Set(reflect.MakeMap(*returnType.ReturnOutType))
				case reflect.Slice:
					returnV.Elem().Set(reflect.MakeSlice(*returnType.ReturnOutType, 0, 0))
				}
				returnValue = &returnV
			}
			//exe sql
			var e = bm.executeMethod(sqlFunc, arg, returnValue)
			return buildReturnValues(returnType, returnValue, e)
		}
		return proxyFunc

	})
}

func getFullName(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name()
}
