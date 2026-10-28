package sqlfragment

import (
	"strings"
	"testing"
)

// P0-2：<if test> 表达式 AST（解析 + 求值 + 字段收集 + legacy 回退）。

// evalIfExpr 编译并求值（测试辅助）。
func evalIfExpr(t *testing.T, test string, mp map[string]interface{}) bool {
	t.Helper()
	expr, err := parseIfExpr(test)
	if err != nil {
		t.Fatalf("parseIfExpr(%q) failed: %v", test, err)
	}
	v, err := expr.eval(mp)
	if err != nil {
		t.Fatalf("eval(%q) failed: %v", test, err)
	}
	return ifTruthy(v)
}

// Test_IfExpr_LogicalOrRegression P0-2 核心回归：`a != null or b != null`
// 旧实现只识别 `a != null`（or 分支被静默丢弃，生成错 SQL）。
func Test_IfExpr_LogicalOrRegression(t *testing.T) {
	cases := []struct {
		test string
		mp   map[string]interface{}
		want bool
	}{
		{"a != null or b != null", map[string]interface{}{}, false},
		{"a != null or b != null", map[string]interface{}{"a": "x"}, true},
		{"a != null or b != null", map[string]interface{}{"b": "y"}, true},
		{"a != null or b != null", map[string]interface{}{"a": "", "b": ""}, false}, // 空串按 legacy 语义不算有效值
		{"a != null and b != null", map[string]interface{}{"a": "x"}, false},
		{"a != null and b != null", map[string]interface{}{"a": "x", "b": "y"}, true},
	}
	for _, c := range cases {
		if got := evalIfExpr(t, c.test, c.mp); got != c.want {
			t.Errorf("eval(%q, %v) = %v, want %v", c.test, c.mp, got, c.want)
		}
	}
}

// Test_IfExpr_NullAndEmpty 判空与空串比较：相等形式（== null / == 空串）为 P0-2 新增，
// 不等形式（!= null / != 空串）保持 legacy「有效值」语义（空串/空集合/nil 为假）。
func Test_IfExpr_NullAndEmpty(t *testing.T) {
	cases := []struct {
		test string
		mp   map[string]interface{}
		want bool
	}{
		{"name == null", map[string]interface{}{}, true},
		{"name == null", map[string]interface{}{"name": nil}, true},
		{"name == null", map[string]interface{}{"name": ""}, true},
		{"name == null", map[string]interface{}{"name": "x"}, false},
		{"null == name", map[string]interface{}{}, true},
		{"name != null", map[string]interface{}{"name": ""}, false},
		{"name != null", map[string]interface{}{"name": "x"}, true},
		{"name == ''", map[string]interface{}{"name": ""}, true},
		{"name == ''", map[string]interface{}{"name": "x"}, false},
		{"name != ''", map[string]interface{}{"name": ""}, false},
		{"name != ''", map[string]interface{}{"name": "x"}, true},
		{"ids == null", map[string]interface{}{"ids": []int{}}, true},
		{"ids != null", map[string]interface{}{"ids": []int{1}}, true},
		{"ids != null", map[string]interface{}{"ids": []int{}}, false},
	}
	for _, c := range cases {
		if got := evalIfExpr(t, c.test, c.mp); got != c.want {
			t.Errorf("eval(%q, %v) = %v, want %v", c.test, c.mp, got, c.want)
		}
	}
}

// Test_IfExpr_Compare 比较：数值、字符串、引用、布尔、算术。
func Test_IfExpr_Compare(t *testing.T) {
	cases := []struct {
		test string
		mp   map[string]interface{}
		want bool
	}{
		{"status == 5", map[string]interface{}{"status": 5}, true},
		{"status == 5", map[string]interface{}{"status": 6}, false},
		{"status > 1", map[string]interface{}{"status": 2}, true},
		{"status >= 2", map[string]interface{}{"status": 2}, true},
		{"status < 2", map[string]interface{}{"status": "1"}, true}, // 数字字符串按数值比较（与 legacy 一致）
		{"status != 0", map[string]interface{}{"status": 0}, false},
		{"status == 'active'", map[string]interface{}{"status": "active"}, true},
		{"status != 'active'", map[string]interface{}{"status": "draft"}, true},
		{"status > 'b'", map[string]interface{}{"status": "c"}, true},
		{"a == b", map[string]interface{}{"a": 1, "b": 1}, true},
		{"a == b", map[string]interface{}{"a": 1, "b": 2}, false},
		{"a == b", map[string]interface{}{}, true}, // 两侧均缺失 → 均视为 null
		{"flag == true", map[string]interface{}{"flag": true}, true},
		{"flag != false", map[string]interface{}{"flag": false}, false},
		{"count + 1 > 2", map[string]interface{}{"count": 2}, true},
		{"count - 1 == 0", map[string]interface{}{"count": 1}, true},
		{"count * 2 >= 4", map[string]interface{}{"count": 2}, true},
		{"count / 2 == 1.5", map[string]interface{}{"count": 3}, true},
		{"-1 < count", map[string]interface{}{"count": 0}, true},
		{"name + '!' == 'abc!'", map[string]interface{}{"name": "abc"}, true},
		{"user.age > 18", map[string]interface{}{"user": map[string]interface{}{"age": 20}}, true},
		{"user.age > 18", map[string]interface{}{"user": map[string]interface{}{"age": 10}}, false},
	}
	for _, c := range cases {
		if got := evalIfExpr(t, c.test, c.mp); got != c.want {
			t.Errorf("eval(%q, %v) = %v, want %v", c.test, c.mp, got, c.want)
		}
	}
}

// Test_IfExpr_ParenAndNot 括号优先级与取反。
func Test_IfExpr_ParenAndNot(t *testing.T) {
	cases := []struct {
		test string
		mp   map[string]interface{}
		want bool
	}{
		{"(a == null or b == null) and c != null", map[string]interface{}{"c": "x"}, true},
		{"(a == null or b == null) and c != null", map[string]interface{}{"a": 1, "b": 2, "c": "x"}, false},
		{"(a != null or b != null) and c != null", map[string]interface{}{"b": 1}, false},
		{"(a != null or b != null) and c != null", map[string]interface{}{"b": 1, "c": 2}, true},
		{"not a != null", map[string]interface{}{}, true},
		{"not a != null", map[string]interface{}{"a": "x"}, false},
		{"!a", map[string]interface{}{"a": false}, true},
		{"!(a != null)", map[string]interface{}{}, true},
		// and 优先于 or
		{"a != null or b != null and c != null", map[string]interface{}{"a": "x"}, true},
		{"a != null or b != null and c != null", map[string]interface{}{"b": "x"}, false},
	}
	for _, c := range cases {
		if got := evalIfExpr(t, c.test, c.mp); got != c.want {
			t.Errorf("eval(%q, %v) = %v, want %v", c.test, c.mp, got, c.want)
		}
	}
}

// Test_IfExpr_Length 集合/字符串长度（OGNL .length 兼容）。
func Test_IfExpr_Length(t *testing.T) {
	cases := []struct {
		test string
		mp   map[string]interface{}
		want bool
	}{
		{"ids.length > 0", map[string]interface{}{"ids": []int{1}}, true},
		{"ids.length > 0", map[string]interface{}{"ids": []int{}}, false},
		{"ids.length > 0", map[string]interface{}{}, false},
		{"ids.size > 1", map[string]interface{}{"ids": []int{1, 2}}, true},
		{"name.length > 2", map[string]interface{}{"name": "abc"}, true},
		{"user.roles.length > 0", map[string]interface{}{"user": map[string]interface{}{"roles": []string{"a"}}}, true},
	}
	for _, c := range cases {
		if got := evalIfExpr(t, c.test, c.mp); got != c.want {
			t.Errorf("eval(%q, %v) = %v, want %v", c.test, c.mp, got, c.want)
		}
	}
}

// Test_IfExpr_BareIdent S-07 兼容：裸标识符只认 bool true。
func Test_IfExpr_BareIdent(t *testing.T) {
	cases := []struct {
		test string
		mp   map[string]interface{}
		want bool
	}{
		{"deptCheckStrictly", map[string]interface{}{"deptCheckStrictly": true}, true},
		{"deptCheckStrictly", map[string]interface{}{"deptCheckStrictly": false}, false},
		{"deptCheckStrictly", map[string]interface{}{}, false},
		{"deptCheckStrictly", map[string]interface{}{"deptCheckStrictly": "true"}, false},
	}
	for _, c := range cases {
		nodes := parseBody(t, `<if test="`+c.test+`">X</if>`)
		got := strings.TrimSpace(renderNodes(nodes, c.mp))
		want := ""
		if c.want {
			want = "X"
		}
		if got != want {
			t.Errorf("render(<if test=%q>, %v) = %q, want %q", c.test, c.mp, got, want)
		}
	}
}

// Test_IfExpr_FallbackUnsupported 编译失败回退 legacy 解析，A3 未识别清单照旧。
func Test_IfExpr_FallbackUnsupported(t *testing.T) {
	cases := []string{
		"a.toUpperCase() == 'X'",
		"cond ? 1 : 0",
		"a = 1",
		"'x'.toString() == 'y'",
	}
	for _, test := range cases {
		nodes := parseBody(t, `<if test="`+test+`">X</if>`)
		if len(nodes) != 1 {
			t.Errorf("%q: if node missing", test)
			continue
		}
		it, ok := nodes[0].(*sqlIfTest)
		if !ok {
			t.Errorf("%q: node is %T, want *sqlIfTest", test, nodes[0])
			continue
		}
		if it.Expr != nil {
			t.Errorf("%q: should fall back to legacy parser, got AST", test)
		}
		if got := CollectUnsupportedIfTests(nodes); len(got) != 1 || got[0] != test {
			t.Errorf("%q: Unsupported = %v, want [%s]", test, got, test)
		}
	}
}

// Test_IfExpr_Fields 字段收集：AST 取根名、跳过 params.。
func Test_IfExpr_Fields(t *testing.T) {
	nodes := parseBody(t, `<if test="user.deptId != null or params.dataScope == null">X</if>`)
	fields := CollectIfTestFieldNames(nodes)
	if len(fields) != 1 || fields[0] != "user" {
		t.Errorf("CollectIfTestFieldNames = %v, want [user]", fields)
	}
}

// renderNodes 用 map 参数渲染片段树（测试辅助）。
func renderNodes(nodes []Node, mp map[string]interface{}) string {
	var buf strings.Builder
	for _, n := range nodes {
		buf.WriteString(n.GenerateSqlWithMap(mp, 0))
	}
	return buf.String()
}
