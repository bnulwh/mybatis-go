package orm

import (
	"reflect"
	"testing"
	"time"
)

func Test_isCustomStruct(t *testing.T) {
	if isCustomStruct(reflect.TypeOf(1)) {
		t.Error("test isCustomStruct failed.")
	}
	if isCustomStruct(reflect.TypeOf(time.Now())) {
		t.Error("test isCustomStruct failed.")
	}
	tm := &time.Time{}
	if isCustomStruct(reflect.TypeOf(tm)) {
		t.Error("test isCustomStruct failed.")
	}
	if !isCustomStruct(reflect.TypeOf(BaseMapper{})) {
		t.Error("test isCustomStruct failed.")
	}

}
