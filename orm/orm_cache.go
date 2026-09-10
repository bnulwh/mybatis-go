package orm

import (
	"fmt"
	"io/fs"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
)

type ormCache struct {
	models  modelCache
	mappers mapperCache
	sqls    *types.SqlMappers
}

var (
	gCache       ormCache
	strictReg    atomic.Bool // 5.1：注册模式，默认严格（任一函数绑定失败即整体失败）
)

func init() {
	gCache = ormCache{
		models:  modelCache{Models: map[string]reflect.Type{}},
		mappers: mapperCache{Mappers: map[string]*mapperInfo{}},
		sqls:    nil,
	}
	strictReg.Store(true)
}

// SetStrictRegister 设置注册模式：true=严格（默认），任一函数绑定失败即整体失败；
// false=宽松：失败函数记 error 日志后跳过（该函数运行时返回明确错误），其余正常注册（5.1）。
func SetStrictRegister(strict bool) { strictReg.Store(strict) }

// IsStrictRegister 返回当前注册模式。
func IsStrictRegister() bool { return strictReg.Load() }


func (in *ormCache) createModel(name string) (reflect.Value, error) {
	return in.models.createModel(name)
}

func (in *ormCache) createMapper(name string) (reflect.Value, error) {
	return in.mappers.createMapper(name)
}

func (in *ormCache) initSqls(dir string) error {
	in.sqls = types.NewSqlMappers(dir)
	return in.bindSqls()
}

// initSqlsFromFS 从任意文件系统（如 go:embed 的 embed.FS）加载 Mapper XML。
// patterns 为目录（递归）或单个 .xml 文件路径，语义与 mybatis.mapper-locations 一致。
func (in *ormCache) initSqlsFromFS(fsys fs.FS, patterns ...string) error {
	in.sqls = types.NewSqlMappersFrom(fsys, patterns...)
	return in.bindSqls()
}

// initSqlsFromSources 合并磁盘目录与 embed.FS 等多来源加载（同 namespace 后者覆盖前者）。
func (in *ormCache) initSqlsFromSources(sources []MapperSource) error {
	in.sqls = types.NewSqlMappersFromSources(sources...)
	return in.bindSqls()
}

func (in *ormCache) bindSqls() error {
	// bindSql 会修改 funcInfo.SqlFunc，持写锁防止与并发 createMapper 读冲突
	in.mappers.mu.Lock()
	defer in.mappers.mu.Unlock()
	var errs []error
	for name := range in.mappers.Mappers {
		log.Debugf("bind mapper %s", name)
		sn := types.GetShortName(name)
		smp, ok := gCache.sqls.NamedMappers[strings.ToLower(sn)]
		if !ok {
			continue
		}
		mi := in.mappers.Mappers[name]
		errs = append(errs, mi.FuncErrs...)
		err := mi.bindSql(smp)
		if err != nil {
			if !IsStrictRegister() {
				log.Errorf("lax register: skip mapper %s: %v", name, err)
				continue
			}
			errs = append(errs, err)
		}
	}
	return combineErrors(errs...)
}

func RegisterModel(inPtr interface{}) {
	gCache.models.registerModel(inPtr)
}
func RegisterMapper(inPtr interface{}) error {
	gCache.mappers.registerMapper(inPtr)
	if gCache.sqls != nil {
		return gCache.bindSqls()
	}
	return nil
}

func NewMapper(name string) interface{} {
	mp, err := gCache.createMapper(name)
	if err != nil {
		log.Warnf("cannot find mapper struct `%s`", name)
		panic(err)
	}
	bindMapper(name, mp)
	return reflect.Indirect(mp).Interface()
}

func NewMapperPtr(name string) interface{} {
	mp, err := gCache.createMapper(name)
	if err != nil {
		log.Warnf("cannot find mapper struct `%s`", name)
		panic(err)
	}
	bindMapper(name, mp)
	return mp.Interface()
}

func bindMapper(name string, mapper reflect.Value) {
	sn := types.GetShortName(name)
	mp, ok := gCache.sqls.NamedMappers[strings.ToLower(sn)]
	if !ok {
		panic(fmt.Sprintf("bind mapper struct `%s` failed,not found in xml files", name))
	}
	outVal := mapper.Elem()
	outTyp := outVal.Type()
	bmf := outVal.FieldByName("BaseMapper")
	bmf.Set(reflect.ValueOf(BaseMapper{SqlMapper: mp}))
	bm := bmf.Interface().(BaseMapper)
	returnTypeMap := makeReturnTypeMap(outTyp)
	proxyValue(mapper, func(funcField reflect.StructField, field reflect.Value) func(arg ProxyArg) []reflect.Value {
		//构建期
		var funcName = funcField.Name
		var returnType = returnTypeMap[funcName]
		if returnType == nil {
			panic("[mybatis-go] struct have no return values!")
		}
		//mapper
		sqlFunc, err := bm.fetchSqlFunction(funcName)
		if err != nil {
			// 5.1 宽松注册：函数因注册期失败被跳过（SqlFunc 未绑定），
			// 运行时返回明确错误而非 panic（该错误已在注册期记录日志）。
			rerr := err
			methodFieldCheck(&outTyp, &funcField, true)
			return func(arg ProxyArg) []reflect.Value {
				return buildReturnValues(returnType, reflect.Value{}, rerr)
			}
		}
		methodFieldCheck(&outTyp, &funcField, true)
		//执行期
		var proxyFunc = func(arg ProxyArg) []reflect.Value {
			// P4-2：流式 select —— 方法返回 *RowStream，逐行消费，不整表进内存
			if returnType.ReturnOutType != nil && *returnType.ReturnOutType == rowStreamType {
				rv, e := bm.executeStream(sqlFunc, arg)
				return buildReturnValues(returnType, rv, e)
			}
			//exe sql
			rv, e := bm.executeMethod(sqlFunc, arg)
			switch sqlFunc.Type {
			case types.InsertFunction, types.DeleteFunction, types.UpdateFunction:
				return buildReturnValues(returnType, rv, e)
			default:
				if returnType.ReturnOutType != nil {
					switch (*returnType.ReturnOutType).Kind() {
					case reflect.Slice:
						return buildReturnValues(returnType, rv, e)
					}
					switch rv.Kind() {
					case reflect.Slice:
						item := rv.Index(0)
						return buildReturnValues(returnType, item, e)
					}
					return buildReturnValues(returnType, rv, e)
				}
				return buildReturnValues(returnType, rv, e)
			}
		}
		return proxyFunc
	})
}

func getFullName(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name()
}
