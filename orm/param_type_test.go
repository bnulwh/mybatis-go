package orm

import (
	"reflect"
	"testing"

	"github.com/bnulwh/mybatis-go/types"
)

type NoParamTypeMapper struct {
	BaseMapper
	SelectByOrgCode func(params map[string]interface{}) ([]map[string]interface{}, error) `args:"params"`
}

// Test_checkSql_NoParameterType：1.1 放宽 —— 有参函数 + 语句无显式 parameterType 不再失败
func Test_checkSql_NoParameterType(t *testing.T) {
	// 有参函数 + 语句推导为无参（AutoDerive）→ 允许（Java 语义：容忍未使用参数）
	pt := &ParamType{ArgsLen: 1}
	f := &types.SqlFunction{Id: "x", Owner: "o", Param: types.SqlParam{Need: false, AutoDerive: true}}
	if err := pt.checkSql(f, "m"); err != nil {
		t.Errorf("with-args func + no-parameterType statement should pass: %v", err)
	}
	// 无参函数 + 语句有占位符（Need 由语句推导）→ 仍失败
	pt2 := &ParamType{ArgsLen: 0}
	f2 := &types.SqlFunction{Id: "y", Owner: "o", Param: types.SqlParam{Need: true, Slots: []string{"id"}}}
	if err := pt2.checkSql(f2, "m"); err == nil {
		t.Error("no-arg func + param statement should fail")
	}
}

// Test_bindSql_NoParameterType：samples 无 parameterType 语句 + 有参函数绑定成功（1.1 核心场景）
func Test_bindSql_NoParameterType(t *testing.T) {
	info := newMapperInfo(reflect.TypeOf(NoParamTypeMapper{}))
	if len(info.Functions) != 1 {
		t.Fatalf("expect 1 func, got %d", len(info.Functions))
	}
	mps := types.NewSqlMappers("../samples")
	mp := mps.NamedMappers["mdmorgrawmapper"]
	if mp == nil {
		t.Error("MdmOrgRawMapper not found in samples")
		return
	}
	sf := mp.NamedFunctions["selectbyorgcode"]
	if sf == nil {
		t.Error("selectByOrgCode not found")
		return
	}
	if !sf.Param.Need {
		t.Error("selectByOrgCode should derive Need=true (has #{orgCode})")
	}
	if err := info.Functions[0].bindSql(sf); err != nil {
		t.Errorf("bind with-params func to no-parameterType statement failed: %v", err)
	}
	if info.Functions[0].SqlFunc == nil {
		t.Error("SqlFunc should be bound after successful bindSql")
	}
}