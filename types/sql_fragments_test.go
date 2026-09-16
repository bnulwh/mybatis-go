package types

import (
	"strings"
	"testing"
)

// 本文件原片段引擎单测（if 条件解析/求值）已随引擎迁移至 types/sqlfragment 包（P0-3a），
// 此处仅保留基于 samples/（RuoYi Mapper）真实文件的端到端回归。

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
