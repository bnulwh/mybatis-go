package orm

import (
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
	"reflect"
	"strings"
	"sync"
)

type modelCache struct {
	mu     sync.RWMutex
	Models map[string]reflect.Type
	Infos  map[string]*ModelInfo // db tag 元数据（P0-2），无 tag 的模型存 nil
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
	if typ.Kind() == reflect.Ptr {
		return
	}
	name := typ.Name()
	log.Debugf("name: %v", name)
	sn := types.GetShortName(name)
	log.Debugf("short name: %v", sn)
	info, err := parseModelInfoFromType(typ)
	if err != nil {
		log.Warnf("parse model info for %s failed: %v", getFullName(typ), err)
		info = nil
	}
	in.mu.Lock()
	if in.Infos == nil {
		in.Infos = map[string]*ModelInfo{}
	}
	in.Models[name] = typ
	in.Models[strings.ToLower(name)] = typ
	in.Models[sn] = typ
	in.Models[strings.ToLower(sn)] = typ
	in.Models[getFullName(typ)] = typ
	in.Infos[name] = info
	in.Infos[strings.ToLower(name)] = info
	in.Infos[sn] = info
	in.Infos[strings.ToLower(sn)] = info
	in.Infos[getFullName(typ)] = info
	in.mu.Unlock()
}

func (in *modelCache) createModel(name string) (reflect.Value, error) {
	in.mu.RLock()
	typ, ok := in.Models[strings.ToLower(strings.TrimSpace(name))]
	in.mu.RUnlock()
	if !ok {
		return reflect.Value{}, fmt.Errorf("model type %s not registered", name)
	}
	return reflect.New(typ), nil
}
