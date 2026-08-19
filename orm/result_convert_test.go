package orm

import (
	"reflect"
	"strings"
	"testing"

	"github.com/bnulwh/mybatis-go/types"
)

// P1-4：resultMap 的 property -> 字段索引 预编译与字段赋值回归测试

type resultConvertTestModel struct {
	Id   int
	Name string
}

// M-05：convert2Results 不再静默丢弃转换失败的行。
// ResultT 基本类型路径：某行首列类型无法转换 -> 该行计入 Skipped，错误含行号/列名；正常行保留。
func Test_Convert2Results_Report(t *testing.T) {
	resInfo := types.SqlResult{ResultT: reflect.TypeOf(int(0))}
	rows := []map[string]interface{}{
		{"id": int64(1)},
		{"id": "not-a-number"},
		{"id": int64(3)},
	}
	results, report := convert2Results(rows, resInfo)
	if report.Total != 3 {
		t.Errorf("expected total 3, got %d", report.Total)
	}
	if report.Converted != 2 {
		t.Errorf("expected converted 2, got %d", report.Converted)
	}
	if report.Skipped != 1 {
		t.Errorf("expected skipped 1, got %d", report.Skipped)
	}
	if len(report.Errors) != 1 {
		t.Errorf("expected 1 error, got %d: %v", len(report.Errors), report.Errors)
	} else {
		// 行号 1-based：坏行是第 2 行；错误信息应包含列名提示
		if report.Errors[0].Row != 2 {
			t.Errorf("expected error row 2, got %d", report.Errors[0].Row)
		}
		if !strings.Contains(report.Errors[0].Message, "id") {
			t.Errorf("expected column hint in error message, got %q", report.Errors[0].Message)
		}
	}
	if results.Len() != 2 {
		t.Errorf("expected 2 results, got %d", results.Len())
	}
}

// M-05：resultMap 路径下列级类型不匹配聚合到报告（行不丢弃、字段保持零值，按列名提示）。
func Test_Convert2Results_ResultMapColumnErr(t *testing.T) {
	RegisterModel(new(resultConvertTestModel))
	rmp := &types.ResultMap{
		TypeName: "resultConvertTestModel",
		ColumnMap: map[string]*types.ResultItem{
			"id":   {Column: "id", Property: "Id"},
			"name": {Column: "name", Property: "Name"},
		},
	}
	resInfo := types.SqlResult{ResultM: rmp}
	rows := []map[string]interface{}{
		{"id": int64(1), "name": "ok"},
		{"id": "bad-value", "name": "partial"}, // Id 转换失败 -> 列级错误，行保留
	}
	results, report := convert2Results(rows, resInfo)
	if report.Total != 2 {
		t.Errorf("expected total 2, got %d", report.Total)
	}
	if report.Converted != 2 {
		t.Errorf("expected 2 converted rows, got %d", report.Converted)
	}
	if report.Skipped != 0 {
		t.Errorf("expected no skipped rows, got %d", report.Skipped)
	}
	if len(report.Errors) != 1 {
		t.Errorf("expected 1 column error, got %d: %v", len(report.Errors), report.Errors)
	} else if report.Errors[0].Column != "id" {
		t.Errorf("expected error column id, got %q", report.Errors[0].Column)
	}
	if results.Len() != 2 {
		t.Errorf("expected 2 results, got %d", results.Len())
	}
	// 坏行仍保留：Id 保持零值，Name 正常填充
	r2 := results.Index(1).Interface().(resultConvertTestModel)
	if r2.Id != 0 {
		t.Errorf("row2 Id should stay zero, got %d", r2.Id)
	}
	if r2.Name != "partial" {
		t.Errorf("row2 Name should be partial, got %q", r2.Name)
	}
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
