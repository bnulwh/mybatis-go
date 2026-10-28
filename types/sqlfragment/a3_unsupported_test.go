package sqlfragment

import (
	"testing"
)

// A3：未识别 <if test> 表达式的解析期记录、树级聚合与严格模式求值。

// Test_parseIfTestConditions_UnsupportedCollected 未命中已知模式的子表达式
// 被原样收集到 unsupported 列表，已知模式不受影响。
func Test_parseIfTestConditions_UnsupportedCollected(t *testing.T) {
	cases := []struct {
		test      string
		wantConds int
		wantUnsup int
	}{
		{"name != null", 1, 0},
		{"name != null and name != ''", 2, 0},
		{"deptId != null and deptId != 0", 2, 0},
		{"status == 'active'", 1, 0},
		{"deptCheckStrictly", 1, 0},
		{"a == b", 0, 1},
		{"flag ? 1 : 0", 0, 1},
		{"'x'.toString() == 'y'", 0, 1},
		{"name != null and a == b", 1, 1},
	}
	for _, c := range cases {
		conds, unsup := parseIfTestConditions(c.test)
		if len(conds) != c.wantConds {
			t.Errorf("parseIfTestConditions(%q) conditions = %d, want %d", c.test, len(conds), c.wantConds)
		}
		if len(unsup) != c.wantUnsup {
			t.Errorf("parseIfTestConditions(%q) unsupported = %v, want len %d", c.test, unsup, c.wantUnsup)
		}
	}
}

// Test_sqlIfTest_UnsupportedIfTests 解析含方法调用的 test 后可经访问器取回未识别表达式
// （P0-2 起 `a == b` 已由表达式 AST 支持，未识别判定改用真正超出子集的写法）。
func Test_sqlIfTest_UnsupportedIfTests(t *testing.T) {
	xn := mustParseNamedChild(t,
		`<mapper><if test="a.toUpperCase() == 'X'">X</if></mapper>`, "if")
	node := mustParseFragment(t, xn, nil)
	if node == nil {
		return
	}
	p, ok := node.(interface{ UnsupportedIfTests() []string })
	if !ok {
		t.Error("sqlIfTest should implement UnsupportedIfTests()")
		return
	}
	got := p.UnsupportedIfTests()
	if len(got) != 1 || got[0] != "a.toUpperCase() == 'X'" {
		t.Errorf("UnsupportedIfTests = %v, want [a.toUpperCase() == 'X']", got)
	}
}

// Test_CollectUnsupportedIfTests_Nested 树级聚合：嵌套 <if>、<choose>/<when>
// 的未识别表达式经 Children() 遍历收集并去重。
func Test_CollectUnsupportedIfTests_Nested(t *testing.T) {
	xn := mustParseNamedChild(t,
		`<mapper><if test="flag != null">OUTER <if test="a.toUpperCase() == 'X'">MID <if test="a.toUpperCase() == 'X'">INNER</if></if> TAIL</if></mapper>`,
		"if")
	outer := mustParseFragment(t, xn, nil)
	if outer == nil {
		return
	}
	got := CollectUnsupportedIfTests([]Node{outer})
	if len(got) != 1 || got[0] != "a.toUpperCase() == 'X'" {
		t.Errorf("CollectUnsupportedIfTests nested if = %v, want [a.toUpperCase() == 'X'] (deduped)", got)
	}

	cxn := mustParseNamedChild(t,
		`<mapper><choose><when test="cond ? 1 : 0">W</when><otherwise>D</otherwise></choose></mapper>`,
		"choose")
	chooseNode := mustParseFragment(t, cxn, nil)
	if chooseNode == nil {
		return
	}
	got = CollectUnsupportedIfTests([]Node{chooseNode})
	if len(got) != 1 || got[0] != "cond ? 1 : 0" {
		t.Errorf("CollectUnsupportedIfTests choose/when = %v, want [cond ? 1 : 0]", got)
	}

	// 全识别语句返回 nil
	okXn := mustParseNamedChild(t,
		`<mapper><if test="name != null and name != ''">X</if></mapper>`, "if")
	okNode := mustParseFragment(t, okXn, nil)
	if okNode == nil {
		return
	}
	if got := CollectUnsupportedIfTests([]Node{okNode}); got != nil {
		t.Errorf("fully recognized test should collect nothing, got %v", got)
	}
}

// Test_StrictIfTest_EvalFalse 严格模式：整个 test 未识别时求值按 false（不渲染）；
// 默认（非严格）保持恒真渲染，兼容存量。
func Test_StrictIfTest_EvalFalse(t *testing.T) {
	defer SetStrictIfTest(StrictIfTest()) // 恢复原值

	xn := mustParseNamedChild(t,
		`<mapper><if test="a.toUpperCase() == 'X'">X</if></mapper>`, "if")
	node := mustParseFragment(t, xn, nil)
	if node == nil {
		return
	}
	// 默认：恒真渲染（A3 现状语义保持）
	if got := collapseWs(node.GenerateSqlWithMap(map[string]interface{}{}, 0)); got != "X" {
		t.Errorf("default mode should keep always-true render, got %q", got)
	}
	// 严格模式：未识别 test 按 false 处理
	SetStrictIfTest(true)
	if got := collapseWs(node.GenerateSqlWithMap(map[string]interface{}{}, 0)); got != "" {
		t.Errorf("strict mode should not render unrecognized test, got %q", got)
	}
}

// Test_StrictIfTest_PartialUnrecognizedStillRenders 严格模式仅当整个 test 未识别
// 才按 false；部分识别（如 name != null and a == b）仍按已识别条件求值。
func Test_StrictIfTest_PartialUnrecognizedStillRenders(t *testing.T) {
	defer SetStrictIfTest(StrictIfTest())
	SetStrictIfTest(true)

	xn := mustParseNamedChild(t,
		`<mapper><if test="name != null and a == b">X</if></mapper>`, "if")
	node := mustParseFragment(t, xn, nil)
	if node == nil {
		return
	}
	if got := collapseWs(node.GenerateSqlWithMap(map[string]interface{}{"name": "n"}, 0)); got != "X" {
		t.Errorf("partial recognized test should render when known cond passes, got %q", got)
	}
	if got := collapseWs(node.GenerateSqlWithMap(map[string]interface{}{}, 0)); got != "" {
		t.Errorf("partial recognized test should not render when known cond fails, got %q", got)
	}
}

// Test_StrictIfTest_Flag 开关读写往返。
func Test_StrictIfTest_Flag(t *testing.T) {
	defer SetStrictIfTest(StrictIfTest())
	if StrictIfTest() {
		t.Error("StrictIfTest should default to off")
	}
	SetStrictIfTest(true)
	if !StrictIfTest() {
		t.Error("StrictIfTest should be on after SetStrictIfTest(true)")
	}
	SetStrictIfTest(false)
	if StrictIfTest() {
		t.Error("StrictIfTest should be off after SetStrictIfTest(false)")
	}
}
