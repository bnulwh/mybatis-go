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

// Test_parseIfConditionsFromText_Compare 数值比较解析（M-02）：
// userId != 0 / age > 18 / count >= 1 / x == 0 / y <= -1 / 集合长度 businessTypes.length > 0。
func Test_parseIfConditionsFromText_Compare(t *testing.T) {
	cases := []struct {
		test   string
		name   string
		op     string
		literal string
	}{
		{"userId != 0", "userId", "!=", "0"},
		{"age > 18", "age", ">", "18"},
		{"count >= 1", "count", ">=", "1"},
		{"status == 0", "status", "==", "0"},
		{"x <= 5", "x", "<=", "5"},
		{"y < -1", "y", "<", "-1"},
		{"score > 59.5", "score", ">", "59.5"},
		{"params.userId != 0", "params.userId", "!=", "0"},
		{"businessTypes.length > 0", "businessTypes.length", ">", "0"},
	}
	for _, c := range cases {
		r := parseIfConditionsFromText(c.test)
		if len(r) != 1 {
			t.Errorf("parseIfConditionsFromText(%q) conditions = %d, want 1", c.test, len(r))
			continue
		}
		if r[0].CheckType != compareCheckCond {
			t.Errorf("parseIfConditionsFromText(%q) type = %v, want compare", c.test, r[0].CheckType)
			continue
		}
		if r[0].CheckName != c.name || r[0].Operator != c.op || r[0].Literal != c.literal {
			t.Errorf("parseIfConditionsFromText(%q) = %+v, want name=%q op=%q literal=%q",
				c.test, r[0], c.name, c.op, c.literal)
		}
	}
	// 混合条件：deptId != null and deptId != 0
	r := parseIfConditionsFromText("deptId != null and deptId != 0")
	if len(r) != 2 {
		t.Error("parseIfConditionsFromText(deptId != null and deptId != 0) conditions =", len(r), "want 2")
		return
	}
	if r[0].CheckType != nullCheckCond || r[1].CheckType != compareCheckCond || r[1].Operator != "!=" || r[1].Literal != "0" {
		t.Error("mixed conditions parsed wrong:", r)
	}
}

// Test_IfCondition_CheckCompare 数值比较求值（M-02）。
func Test_IfCondition_CheckCompare(t *testing.T) {
	neq0 := ifCondition{CheckName: "userId", CheckType: compareCheckCond, Operator: "!=", Literal: "0"}
	eq0 := ifCondition{CheckName: "userId", CheckType: compareCheckCond, Operator: "==", Literal: "0"}
	gt := ifCondition{CheckName: "age", CheckType: compareCheckCond, Operator: ">", Literal: "18"}
	// 注意：内部 map 键按 buildKey 小写化（GenerateSQL 入口经 convert2Map 归一），这里直接用小写键
	if neq0.checkValue(map[string]interface{}{"userid": int64(0)}) {
		t.Error("userId != 0 with 0 should fail")
	}
	if !neq0.checkValue(map[string]interface{}{"userid": int64(5)}) {
		t.Error("userId != 0 with 5 should pass")
	}
	if neq0.checkValue(map[string]interface{}{"userid": 0.0}) {
		t.Error("userId != 0 with 0.0 float should fail")
	}
	if eq0.checkValue(map[string]interface{}{"userid": int64(5)}) {
		t.Error("userId == 0 with 5 should fail")
	}
	if !eq0.checkValue(map[string]interface{}{"userid": int64(0)}) {
		t.Error("userId == 0 with 0 should pass")
	}
	if gt.checkValue(map[string]interface{}{"age": int64(18)}) {
		t.Error("age > 18 with 18 should fail")
	}
	if !gt.checkValue(map[string]interface{}{"age": int64(20)}) {
		t.Error("age > 18 with 20 should pass")
	}
	// 缺失 / nil / 非数值
	if neq0.checkValue(map[string]interface{}{}) {
		t.Error("missing param should fail")
	}
	if neq0.checkValue(map[string]interface{}{"userid": nil}) {
		t.Error("nil param should fail")
	}
	if neq0.checkValue(map[string]interface{}{"userid": "abc"}) {
		t.Error("non-numeric param should fail")
	}
	// 字符串数字
	if !neq0.checkValue(map[string]interface{}{"userid": "5"}) {
		t.Error("numeric string \"5\" should pass userId != 0")
	}
	// 点号参数
	dot := ifCondition{CheckName: "params.userId", CheckType: compareCheckCond, Operator: "!=", Literal: "0"}
	if dot.checkValue(map[string]interface{}{"params": map[string]interface{}{"userid": int64(0)}}) {
		t.Error("params.userId != 0 with 0 should fail")
	}
	if !dot.checkValue(map[string]interface{}{"params": map[string]interface{}{"userid": int64(9)}}) {
		t.Error("params.userId != 0 with 9 should pass")
	}
	// 集合长度：businessTypes.length > 0
	lenCond := ifCondition{CheckName: "businessTypes.length", CheckType: compareCheckCond, Operator: ">", Literal: "0"}
	if lenCond.checkValue(map[string]interface{}{"businesstypes": []int{}}) {
		t.Error("empty slice length > 0 should fail")
	}
	if !lenCond.checkValue(map[string]interface{}{"businesstypes": []int{1, 2}}) {
		t.Error("non-empty slice length > 0 should pass")
	}
	if lenCond.checkValue(map[string]interface{}{}) {
		t.Error("missing collection should fail")
	}
}

// Test_IfCompare_Samples 使用 samples/（RuoYi Mapper）真实回归：
// selectUserList 的 <if test="userId != null and userId != 0"> 在 userId=0 时必须剔除 AND u.user_id 子句（M-02）。
func Test_IfCompare_Samples(t *testing.T) {
	mps := NewSqlMappers("../samples")
	if mps == nil || len(mps.Mappers) == 0 {
		t.Error("load samples failed")
		return
	}
	var fn *SqlFunction
	for _, m := range mps.Mappers {
		if f := m.NamedFunctions["selectUserList"]; f != nil {
			fn = f
			break
		}
	}
	if fn == nil {
		t.Error("selectUserList not found in samples")
		return
	}
	base := map[string]interface{}{"params": map[string]interface{}{"dataScope": ""}}
	// userId=0：必须剔除 AND u.user_id = 子句
	sqlZero, _, err := fn.GenerateSQL(map[string]interface{}{"userId": int64(0), "params": base["params"]})
	if err != nil {
		t.Error("selectUserList(userId=0) GenerateSQL failed:", err)
		return
	}
	sqlZero = collapseSpace(sqlZero)
	t.Log("userId=0 sql:", sqlZero)
	if strings.Contains(sqlZero, "u.user_id =") {
		t.Error("userId=0 should drop AND u.user_id clause, sql:", sqlZero)
	}
	// userId=5：必须保留 AND u.user_id = 子句
	sqlSet, _, err := fn.GenerateSQL(map[string]interface{}{"userId": int64(5), "params": base["params"]})
	if err != nil {
		t.Error("selectUserList(userId=5) GenerateSQL failed:", err)
		return
	}
	sqlSet = collapseSpace(sqlSet)
	t.Log("userId=5 sql:", sqlSet)
	if !strings.Contains(sqlSet, "u.user_id =") {
		t.Error("userId=5 should keep AND u.user_id clause, sql:", sqlSet)
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
