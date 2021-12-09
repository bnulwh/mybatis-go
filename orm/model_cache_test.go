package orm

import (
	"fmt"
	"reflect"
	"testing"
)

func Test_modelCache_registerModel(t *testing.T) {
	mc := modelCache{
		Models: map[string]reflect.Type{},
	}
	type ts struct {
		Name string
	}
	mc.registerModel(new(ts))
	fmt.Println(mc)
	fmt.Println(len(mc.Models))
	if len(mc.Models) == 0 {
		t.Error("test modelCache registerModel failed.")
	}
	l0 := len(mc.Models)
	pts := &ts{Name: "test"}
	mc.registerModel(pts)
	l1 := len(mc.Models)
	if l0 != l1 {
		t.Error("test modelCache registerModel failed.")
	}
	fmt.Println(mc.Models["ts"])
	if mc.Models["ts"] == reflect.TypeOf(pts) {
		t.Error("test modelCache registerModel failed.")
	}
	if mc.Models["ts"].Kind() == reflect.Ptr {
		t.Error("test modelCache registerModel failed.")
	}
	if mc.Models["ts"] == reflect.TypeOf(pts) {
		t.Error("test modelCache registerModel failed.")
	}
	if mc.Models["ts"] != reflect.TypeOf(ts{}) {
		t.Error("test modelCache registerModel failed.")
	}
}
func Test_modelCache_addModel(t *testing.T) {
	mc := modelCache{
		Models: map[string]reflect.Type{},
	}
	type ts struct {
		Name string
	}
	mc.addModel(reflect.TypeOf(ts{}))
	if mc.Models["ts"] != reflect.TypeOf(ts{}) {
		t.Error("test modelCache addModel failed.")
	}
	l0 := len(mc.Models)
	mc.addModel(reflect.TypeOf(&ts{}))
	l1 := len(mc.Models)
	if l0 != l1 {
		t.Error("test modelCache addModel failed.")
	}
}
func Test_modelCache_createModel(t *testing.T) {
	mc := modelCache{
		Models: map[string]reflect.Type{},
	}
	type ts struct {
		Name string
	}
	mc.addModel(reflect.TypeOf(ts{}))
	r, err := mc.createModel("ts")
	if err != nil {
		t.Errorf("test modelCache createModel failed. %v", err)
	}
	if r.Kind() != reflect.Ptr || r.Elem().Type() != reflect.TypeOf(ts{}) {
		t.Error("test modelCache createModel failed.")
	}
	_, err = mc.createModel("tst")
	if err == nil {
		t.Error("test modelCache createModel failed.")
	}

}
