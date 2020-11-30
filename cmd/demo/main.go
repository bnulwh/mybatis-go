package main

import (
	"fmt"
	"github.com/bnulwh/mybatis-go/types"
	"reflect"
	"strings"
)

type modelCache struct {
	Models map[string]reflect.Type
}

func (in *modelCache)addModel(typ reflect.Type) {
	if in.Models == nil{
		in.Models = map[string]reflect.Type{}
	}
	name := typ.Name()
	sn := types.GetShortName(name)
	in.Models[name] = typ
	in.Models[strings.ToLower(name)] = typ
	in.Models[sn] = typ
	in.Models[strings.ToLower(sn)] = typ
}

func (in *modelCache) createModel(name string) (reflect.Value, error) {
	typ, ok := in.Models[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return reflect.Value{}, fmt.Errorf("model type %s not registered!!!", name)
	}
	return reflect.New(typ), nil
}

type model struct {

}

func main() {
	var c modelCache
	c.addModel(reflect.Indirect(reflect.ValueOf(new(model))).Type())
}
