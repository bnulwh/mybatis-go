package orm

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/bnulwh/mybatis-go/types"
	"reflect"
	"strings"
)

type modelCache struct {
	Models map[string]reflect.Type
}

func (in *modelCache) registerModel(inPtr interface{}) {
	val := reflect.ValueOf(inPtr)
	typ := reflect.Indirect(val).Type()
	fn := getFullName(typ)
	if val.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("<orm.RegisterModel> cannot use non-ptr model struct `%s`", fn))
	}
	if typ.Kind() == reflect.Ptr {
		panic(fmt.Sprintf("<orm.RegisterModel> only allow ptr model struct,it looks you use two reference to the struct `%s`", fn))
	}
	log.Debugf("register  model struct `%s`", fn)
	in.addModel(typ)
}

func (in *modelCache) addModel(typ reflect.Type) {
	name := typ.Name()
	log.Debugf("name: %v", name)
	sn := types.GetShortName(name)
	log.Debugf("short name: %v", sn)
	in.Models[name] = typ
	in.Models[strings.ToLower(name)] = typ
	in.Models[sn] = typ
	in.Models[strings.ToLower(sn)] = typ
	in.Models[getFullName(typ)] = typ
}

func (in *modelCache) createModel(name string) (reflect.Value, error) {
	typ, ok := in.Models[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return reflect.Value{}, fmt.Errorf("model type %s not registered!!!", name)
	}
	return reflect.New(typ), nil
}
