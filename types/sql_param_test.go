package types

import (
	"strings"
	"testing"
)

// Test_collectSqlSlots_NoParameterType：samples（RuoYi 共享 Java XML，无 parameterType）
// 语句应推导出 Need=true 与占位符 slots（1.1 核心场景）
func Test_collectSqlSlots_NoParameterType(t *testing.T) {
	mps := NewSqlMappers("../samples")
	mp, ok := mps.NamedMappers[buildKey("MdmOrgRawMapper")]
	if !ok {
		t.Error("MdmOrgRawMapper not found in samples")
		return
	}
	sf, ok := mp.NamedFunctions["selectbyorgcode"]
	if !ok {
		t.Error("selectByOrgCode not found")
		return
	}
	if !sf.Param.Need {
		t.Error("selectByOrgCode contains #{orgCode}, Need should be true")
	}
	if len(sf.Param.Slots) != 1 || buildKey(sf.Param.Slots[0]) != "orgcode" {
		t.Errorf("slots = %v, want [orgCode]", sf.Param.Slots)
	}
	if !sf.Param.AutoDerive {
		t.Error("AutoDerive should be true when parameterType is absent")
	}
}

// Test_collectSqlSlots_Static：纯静态 SQL（无占位符、无动态标签）→ 推导 Need=false
func Test_collectSqlSlots_Static(t *testing.T) {
	n := xmlNode{
		Id: "selectOne", Name: "select",
		Elements: []xmlElement{{ElementType: xmlTextElem, Val: "select 1 as one"}},
	}
	f := parseSqlFunctionFromXmlNode(n, nil, nil, "test")
	if f.Param.Need {
		t.Error("static sql should not need params")
	}
	if !f.Param.AutoDerive {
		t.Error("AutoDerive should be true when parameterType is absent")
	}
}

// Test_collectSqlSlots_IfNoPlaceholder：动态 <if> 但体无占位符 → 不视为必参（1.1 风险项）
func Test_collectSqlSlots_IfNoPlaceholder(t *testing.T) {
	n := xmlNode{
		Id: "list", Name: "select",
		Elements: []xmlElement{
			{ElementType: xmlTextElem, Val: "select * from t where 1=1 "},
			{ElementType: xmlNodeElem, Val: xmlNode{
				Name:  "if",
				Attrs: map[string]string{"test": "deptCheckStrictly"},
				Elements: []xmlElement{
					{ElementType: xmlTextElem, Val: " and t.dept_id = 1"},
				},
			}},
		},
	}
	f := parseSqlFunctionFromXmlNode(n, nil, nil, "test")
	if f.Param.Need {
		t.Error("if without placeholder should not force need params")
	}
}

// Test_collectSqlSlots_MultiPlaceholder：多占位符去重、按首现序（1.2 按位绑定依赖）
func Test_collectSqlSlots_MultiPlaceholder(t *testing.T) {
	n := xmlNode{
		Id: "cols", Name: "select",
		Elements: []xmlElement{{ElementType: xmlTextElem, Val: "select * from information_schema.COLUMNS where TABLE_SCHEMA = #{schema} and TABLE_NAME = #{tableName} and TABLE_SCHEMA = #{schema}"}},
	}
	f := parseSqlFunctionFromXmlNode(n, nil, nil, "test")
	if len(f.Param.Slots) != 2 {
		t.Errorf("slots = %v, want 2 distinct [schema, tableName]", f.Param.Slots)
	}
	if buildKey(f.Param.Slots[0]) != "schema" || buildKey(f.Param.Slots[1]) != "tablename" {
		t.Errorf("slot order = %v, want [schema tableName]", f.Param.Slots)
	}
	if !f.Param.Need {
		t.Error("multi placeholder statement should need params")
	}
}

// Test_collectSqlSlots_DeclaredParameterType：显式 parameterType 时 AutoDerive=false、Need=true
func Test_collectSqlSlots_DeclaredParameterType(t *testing.T) {
	n := xmlNode{
		Id: "one", Name: "select",
		Attrs: map[string]string{"parameterType": "String"},
		Elements: []xmlElement{{ElementType: xmlTextElem, Val: "select * from t where id = #{id}"}},
	}
	f := parseSqlFunctionFromXmlNode(n, nil, nil, "test")
	if !f.Param.Need {
		t.Error("declared parameterType should set Need=true")
	}
	if f.Param.AutoDerive {
		t.Error("declared parameterType should not set AutoDerive")
	}
	if len(f.Param.Slots) != 1 {
		t.Error("slots should still be collected for declared parameterType")
	}
}

// Test_collectSqlSlots_IfBodyPlaceholderExcluded：占位符仅在 <if> 体内 → 不推导为必参，
// 保持 selectAll 类「无参调用 → where 省略」的既有契约（1.1 风险项）
func Test_collectSqlSlots_IfBodyPlaceholderExcluded(t *testing.T) {
	n := xmlNode{
		Id: "selectAll", Name: "select",
		Elements: []xmlElement{
			{ElementType: xmlTextElem, Val: "select * from sys_config "},
			{ElementType: xmlNodeElem, Val: xmlNode{
				Name: "where",
				Elements: []xmlElement{
					{ElementType: xmlNodeElem, Val: xmlNode{
						Name:  "if",
						Attrs: map[string]string{"test": "configName != null"},
						Elements: []xmlElement{
							{ElementType: xmlTextElem, Val: " AND config_name = #{configName}"},
						},
					}},
				},
			}},
		},
	}
	f := parseSqlFunctionFromXmlNode(n, nil, nil, "test")
	if f.Param.Need {
		t.Error("placeholder only inside <if> should not force need params")
	}
	if len(f.Param.Slots) != 0 {
		t.Errorf("slots should be empty for if-body placeholders, got %v", f.Param.Slots)
	}
}

// 防止 strings 未使用（后续断言可能用到）
var _ = strings.ToLower

func Test_toGolangType(t *testing.T) {
	mp := map[string]string{
		"STRING": "string", "VARCHAR": "string",
		"BOOLEAN": "bool", "BOOL": "bool",
		"INT": "int32", "INTEGER": "int32", "INT8": "int32", "INT16": "int32", "INT32": "int32",
		"INT64": "int64",
		"UINT":  "uint32", "UINT8": "uint32", "UINT16": "uint32", "UINT32": "uint32",
		"UINT64": "uint64",
		"FLOAT":  "float32", "FLOAT32": "float32",
		"FLOAT64": "float64", "DOUBLE": "float64",
		"TIME": "time.Time", "TIMESTAMP": "time.Time",
		"LIST": "[]interface{}", "ARRAY": "[]interface{}", "ARRAYLIST": "[]interface{}", "SLICE": "[]interface{}",
		"MAP": "map[string]interface{}", "HASHMAP": "map[string]interface{}", "TREEMAP": "map[string]interface{}",
	}
	for k, v := range mp {
		r := toGolangType(k)
		if r != v {
			t.Errorf("test toGolangType failed.k= %v v=%v r=%v", k, v, r)
		}
	}
}
