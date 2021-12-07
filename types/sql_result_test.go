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
