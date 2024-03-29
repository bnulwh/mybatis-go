package orm

import (
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
	"reflect"
	"strings"
)

type funcInfo struct {
	Name       string
	Type       reflect.Type
	Tag        reflect.StructTag
	ParamType  *ParamType
	ReturnType *ReturnType
	SqlFunc    *types.SqlFunction
}

type mapperInfo struct {
	Name           string
	Type           reflect.Type
	Functions      []*funcInfo
	NamedFunctions map[string]*funcInfo
	SqlMapper      *types.SqlMapper
}

type mapperCache struct {
	Mappers map[string]*mapperInfo
}

func (in *funcInfo) bindSql(f *types.SqlFunction) error {
	if in.Type.Kind() == reflect.Func {
		err := in.ParamType.checkSql(f, in.Name)
		if err != nil {
			return err
		}
		err2 := in.ReturnType.checkSql(f, in.Name)
		if err2 != nil {
			return err2
		}
		in.SqlFunc = f
		return nil
	}
	return fmt.Errorf("function %s's kind is %v, cannot bind sql", in.Name, in.Type.Kind())
}

func (in *mapperInfo) bindSql(smp *types.SqlMapper) error {
	var errs []error
	for i, fi := range in.Functions {
		sname := strings.ToLower(fi.Name)
		sf, ok := smp.NamedFunctions[sname]
		if !ok {
			log.Errorf("%v.%v has no sql function to map in %v", in.Name, fi.Name, smp.Filename)
			errs = append(errs, fmt.Errorf("%v.%v has no sql function to map in %v", in.Name, fi.Name, smp.Filename))
			continue
			//panic(fmt.Sprintf("%v.%v has no sql function to map in %v", in.Name, fi.Name, smp.Filename))
		}
		err := in.Functions[i].bindSql(sf)
		if err != nil {
			errs = append(errs, err)
		}
	}
	log.Debugf("%v bind sql mapper %v finished", in.Name, smp.Filename)
	return combineErrors(errs...)
}
func getFunctions(typ reflect.Type) []*funcInfo {
	var infos []*funcInfo
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldName := typ.Field(i).Name
		fieldType := typ.Field(i).Type
		fieldTag := typ.Field(i).Tag
		if fieldType.Kind() != reflect.Func {
			continue
		}
		methodFieldCheck(&typ, &field, true)
		infos = append(infos, &funcInfo{
			Name:       fieldName,
			Type:       fieldType,
			Tag:        fieldTag,
			ParamType:  makeParamType(fieldName, fieldType, fieldTag),
			ReturnType: makeReturnType(fieldName, fieldType),
			SqlFunc:    nil,
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
	log.Debugf("register  mapper struct `%s`", fn)
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
		return reflect.ValueOf(-1), fmt.Errorf("mapper type %s not registered", name)
	}
	return reflect.New(mp.Type), nil
}
