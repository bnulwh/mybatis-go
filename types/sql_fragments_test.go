package types

import (
	"fmt"
	"strings"
	"testing"
)

func Test_parseIfConditionsFromText(t *testing.T) {
	r1 := parseIfConditionsFromText("name != null")
	fmt.Println(r1)
	if len(r1) == 0 {
		t.Error("test parseIfConditionsFromText failed.")
	}
	if r1[0].CheckName != "name" {
		t.Error("test parseIfConditionsFromText failed.")
	}
	if r1[0].CheckType != nullCheckCond {
		t.Error("test parseIfConditionsFromText failed.")
	}
	r2 := parseIfConditionsFromText("name != null and name != '' ")
	fmt.Println(r1)
	if len(r2) != 2 {
		t.Error("test parseIfConditionsFromText failed.")
	}
	if r2[1].CheckName != "name" {
		t.Error("test parseIfConditionsFromText failed.")
	}
	if r2[1].CheckType != emptyCheckCond {
		t.Error("test parseIfConditionsFromText failed.")
	}
}

// Test_parseIfConditionsFromText_Bool 裸标识符按布尔条件解析（S-07）。
func Test_parseIfConditionsFromText_Bool(t *testing.T) {
	r1 := parseIfConditionsFromText("deptCheckStrictly")
	if len(r1) != 1 {
		t.Error("parseIfConditionsFromText(deptCheckStrictly) conditions =", len(r1), "want 1")
		return
	}
	if r1[0].CheckName != "deptCheckStrictly" {
		t.Error("bool condition name =", r1[0].CheckName, "want deptCheckStrictly")
	}
	if r1[0].CheckType != boolCheckCond {
		t.Error("bool condition type =", r1[0].CheckType, "want", boolCheckCond)
	}
	r2 := parseIfConditionsFromText("menuCheckStrictly")
	if len(r2) != 1 || r2[0].CheckType != boolCheckCond {
		t.Error("parseIfConditionsFromText(menuCheckStrictly) =", r2, "want 1 bool condition")
	}
	// 普通比较不受影响
	r3 := parseIfConditionsFromText("deptName != null and deptName != ''")
	if len(r3) != 2 {
		t.Error("parseIfConditionsFromText(deptName != null ...) conditions =", len(r3), "want 2")
	}
	for _, c := range r3 {
		if c.CheckType == boolCheckCond {
			t.Error("comparison expression parsed as bool condition:", c)
		}
	}
}

// Test_IfCondition_CheckBool 布尔裸标识符求值：true 通过、false/缺失/非布尔不通过（S-07）。
func Test_IfCondition_CheckBool(t *testing.T) {
	cond := ifCondition{CheckName: "deptCheckStrictly", CheckType: boolCheckCond}
	if !cond.checkValue(map[string]interface{}{"deptcheckstrictly": true}) {
		t.Error("bool true should pass")
	}
	if cond.checkValue(map[string]interface{}{"deptcheckstrictly": false}) {
		t.Error("bool false should fail")
	}
	if cond.checkValue(map[string]interface{}{}) {
		t.Error("missing param should fail")
	}
	if cond.checkValue(map[string]interface{}{"deptcheckstrictly": nil}) {
		t.Error("nil param should fail")
	}
	if cond.checkValue(map[string]interface{}{"deptcheckstrictly": "true"}) {
		t.Error("non-bool param should fail")
	}
}

// Test_IfBool_Samples 使用 samples/（RuoYi Mapper）真实文件回归：
// selectDeptListByRoleId 的 <if test="deptCheckStrictly"> 在 false 时必须剔除（S-07）。
func Test_IfBool_Samples(t *testing.T) {
	mps := NewSqlMappers("../samples")
	if mps == nil || len(mps.Mappers) == 0 {
		t.Error("load samples failed")
		return
	}
	for _, m := range mps.Mappers {
		fn := m.NamedFunctions["selectDeptListByRoleId"]
		if fn == nil {
			continue
		}
		// 开启严格校验：SQL 必须包含 not in 子句
		sqlOn, _, err := fn.GenerateSQL(map[string]interface{}{
			"roleId":            int64(2),
			"deptCheckStrictly": true,
		})
		if err != nil {
			t.Error("selectDeptListByRoleId(true) GenerateSQL failed:", err)
			return
		}
		sqlOn = collapseSpace(sqlOn)
		t.Log("deptCheckStrictly=true sql:", sqlOn)
		if !strings.Contains(sqlOn, "not in") {
			t.Error("deptCheckStrictly=true should include not-in clause, sql:", sqlOn)
		}
		// 关闭严格校验：SQL 必须剔除 not in 子句
		sqlOff, _, err := fn.GenerateSQL(map[string]interface{}{
			"roleId":            int64(2),
			"deptCheckStrictly": false,
		})
		if err != nil {
			t.Error("selectDeptListByRoleId(false) GenerateSQL failed:", err)
			return
		}
		sqlOff = collapseSpace(sqlOff)
		t.Log("deptCheckStrictly=false sql:", sqlOff)
		if strings.Contains(sqlOff, "not in") {
			t.Error("deptCheckStrictly=false should drop not-in clause, sql:", sqlOff)
		}
		// 缺失参数同样剔除
		sqlMissing, _, err := fn.GenerateSQL(map[string]interface{}{"roleId": int64(2)})
		if err != nil {
			t.Error("selectDeptListByRoleId(missing) GenerateSQL failed:", err)
			return
		}
		sqlMissing = collapseSpace(sqlMissing)
		if strings.Contains(sqlMissing, "not in") {
			t.Error("deptCheckStrictly missing should drop not-in clause, sql:", sqlMissing)
		}
	}
}
