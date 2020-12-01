package orm

import (
	"fmt"
	log "github.com/astaxie/beego/logs"
	"github.com/bnulwh/mybatis-go/types"
	"reflect"
	"strings"
)

type funcInfo struct {
	Name string
	Type reflect.Type
	Tag  reflect.StructTag
}

type mapperInfo struct {
	Name           string
	Type           reflect.Type
	Functions      []*funcInfo
	NamedFunctions map[string]*funcInfo
}

type mapperCache struct {
	Mappers map[string]*mapperInfo
}

func getFunctions(typ reflect.Type) []*funcInfo {
	var infos []*funcInfo
	for i := 0; i < typ.NumField(); i++ {
		fieldName := typ.Field(i).Name
		fieldType := typ.Field(i).Type
		fieldTag := typ.Field(i).Tag
		if fieldType.Kind() != reflect.Func {
			continue
		}
		methodFieldCheck(&typ, &fieldType, true)
		infos = append(infos, &funcInfo{
			Name: fieldName,
			Type: fieldType,
			Tag:  fieldTag,
		})
	}
	return infos
}

func makeNamedFunctions(infos []*funcInfo) map[string]*funcInfo {
	mfs := map[string]*funcInfo{}
	for _, f := range infos {
		mfs[f.Name] = f
	}
	return mfs
}

func newMapperInfo(typ reflect.Type) *mapperInfo {
	fs := getFunctions(typ)
	mfs := makeNamedFunctions(fs)
	return &mapperInfo{
		Name:           typ.Name(),
		Type:           typ,
		Functions:      fs,
		NamedFunctions: mfs,
	}
}
func (in *mapperCache) registerMapper(inPtr interface{}) {
	val := reflect.ValueOf(inPtr)
	typ := reflect.Indirect(val).Type()
	fn := getFullName(typ)
	if val.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("<orm.RegisterMapper> cannot use non-ptr mapper struct `%s`", fn))
	}
	if typ.Kind() == reflect.Ptr {
		panic(fmt.Sprintf("<orm.RegisterMapper> only allow ptr mapper struct,it looks you use two reference to the struct `%s`", fn))
	}
	log.Debug("register  mapper struct `%s`", fn)
	_, ok := typ.FieldByName("BaseMapper")
	if !ok {
		panic(fmt.Sprintf("<orm.RegisterMapper> can only use mapper struct `%s` based on <orm.BaseMapper>", fn))
	}
	beanCheck(val)
	in.addMapper(typ)
}
func (in *mapperCache) addMapper(typ reflect.Type) {
	info := newMapperInfo(typ)
	name := typ.Name()
	sn := types.GetShortName(name)
	in.Mappers[name] = info
	in.Mappers[strings.ToLower(name)] = info
	in.Mappers[sn] = info
	in.Mappers[strings.ToLower(sn)] = info
	in.Mappers[getFullName(typ)] = info
}
func (in *mapperCache) createMapper(name string) (reflect.Value, error) {
	mp, ok := in.Mappers[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return reflect.ValueOf(-1), fmt.Errorf("mapper type %s not registered!!!", name)
	}
	return reflect.New(mp.Type), nil
}
