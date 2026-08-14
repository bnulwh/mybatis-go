package orm

import (
	"reflect"
	"testing"

	"github.com/bnulwh/mybatis-go/types"
)

// P1-4：resultMap 的 property -> 字段索引 预编译与字段赋值回归测试

type resultConvertTestModel struct {
	Id   int
	Name string
}

func Test_buildFieldIndexMap(t *testing.T) {
	rmp := &types.ResultMap{
		TypeName: "resultConvertTestModel",
		ColumnMap: map[string]*types.ResultItem{
			"id":   {Column: "id", Property: "Id"},
			"name": {Column: "name", Property: "name"}, // 小写，需 UpperFirst 回退
			"miss": {Column: "miss", Property: "NotExist"},
		},
	}
	idx := buildFieldIndexMap(reflect.TypeOf(resultConvertTestModel{}), rmp)
	if len(idx) != 2 {
		t.Errorf("expected 2 resolvable properties, got %d: %v", len(idx), idx)
	}
	if len(idx["Id"]) != 1 || idx["Id"][0] != 0 {
		t.Errorf("property Id should map to field index 0, got %v", idx["Id"])
	}
	if len(idx["name"]) != 1 || idx["name"][0] != 1 {
		t.Errorf("property name (UpperFirst fallback) should map to field index 1, got %v", idx["name"])
	}
	if _, ok := idx["NotExist"]; ok {
		t.Error("missing property should not be in index map")
	}
}

func Test_setColumnValuesPrepared(t *testing.T) {
	rmp := &types.ResultMap{
		TypeName: "resultConvertTestModel",
		ColumnMap: map[string]*types.ResultItem{
			"id":   {Column: "id", Property: "Id"},
			"name": {Column: "name", Property: "name"},
		},
	}
	idx := buildFieldIndexMap(reflect.TypeOf(resultConvertTestModel{}), rmp)
	mp := map[string]interface{}{
		"id":   int64(7),
		"name": "hello",
		"junk": "ignored", // 不在 resultMap 中 -> 跳过
	}
	inst := new(resultConvertTestModel)
	setColumnValuesPrepared(reflect.ValueOf(inst), rmp, mp, idx)
	if inst.Id != 7 {
		t.Errorf("Id not set, got %d", inst.Id)
	}
	if inst.Name != "hello" {
		t.Errorf("Name not set, got %q", inst.Name)
	}
	// 与旧的 setColumnValues 包装路径结果一致
	inst2 := new(resultConvertTestModel)
	setColumnValues(reflect.ValueOf(inst2), rmp, mp)
	if inst2.Id != 7 || inst2.Name != "hello" {
		t.Errorf("setColumnValues wrapper mismatch: %+v", *inst2)
	}
}
