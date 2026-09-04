package types

import (
	"reflect"
	"testing"
)

func Test_parseSqlResult1(t *testing.T) {
	r := parseSqlResult1("int")
	if r.ResultT != reflect.TypeOf(1) {
		t.Error("test parseSqlResult1 failed")
	}
}

// Test_ParseResultTypeFrom_Map：map 识别为合法通用类型，不再走 default warn 路径（1.4）
func Test_ParseResultTypeFrom_Map(t *testing.T) {
	typ := parseResultTypeFrom("map")
	if typ.Kind() != reflect.Map {
		t.Errorf("parseResultTypeFrom(map) = %v, want map kind", typ)
	}
	if parseResultTypeFrom("MAP").Kind() != reflect.Map {
		t.Error("MAP uppercase should also be map")
	}
	if parseResultTypeFrom("HASHMAP").Kind() != reflect.Map {
		t.Error("HASHMAP should also be map")
	}
}
