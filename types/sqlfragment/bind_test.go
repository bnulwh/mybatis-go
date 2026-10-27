package sqlfragment

import (
	"strings"
	"testing"
)

// parseBody 解析语句体 XML（<x> ... </x>）为片段树。
func parseBody(t *testing.T, body string) []Node {
	t.Helper()
	root, err := ParseXmlContent([]byte("<x>" + body + "</x>"))
	if err != nil {
		t.Errorf("parse xml error: %v", err)
		return nil
	}
	nodes, err := ParseFragments(root.Elements, map[string]*SqlElement{})
	if err != nil {
		t.Errorf("ParseFragments error: %v", err)
	}
	return nodes
}

// Test_Bind_ConcatString <bind> 字符串拼接：'%' + keyword + '%'。
func Test_Bind_ConcatString(t *testing.T) {
	nodes := parseBody(t, `<bind name="pattern" value="'%' + keyword + '%'"/>where name like #{pattern}`)
	if !ContainsBind(nodes) {
		t.Error("ContainsBind should be true")
	}
	if got := CollectBoundNames(nodes); len(got) != 1 || got[0] != "pattern" {
		t.Errorf("CollectBoundNames = %v, want [pattern]", got)
	}
	if got := CollectBindRefs(nodes); len(got) != 1 || got[0] != "keyword" {
		t.Errorf("CollectBindRefs = %v, want [keyword]", got)
	}
	mp := map[string]interface{}{"keyword": "abc"}
	var sqlstr string
	for _, n := range nodes {
		sqlstr += n.GenerateSqlWithMap(mp, 0)
	}
	if want := "where name like '%abc%'"; strings.TrimSpace(sqlstr) != want {
		t.Errorf("generated sql = %q, want %q", strings.TrimSpace(sqlstr), want)
	}
	if got := mp["pattern"]; got != "%abc%" {
		t.Errorf("bound pattern = %v, want %%abc%%", got)
	}
	// 占位符路径同样应完成替换（不再把 #{pattern} 原样下发）
	var prepared string
	mp2 := map[string]interface{}{"keyword": "abc"}
	for _, n := range nodes {
		s, _ := n.PrepareSqlWithMap(mp2, 0)
		prepared += s
	}
	if strings.Contains(prepared, "#{pattern}") {
		t.Errorf("prepared sql should not keep #{pattern}: %q", prepared)
	}
}

// Test_Bind_LiteralOnly 纯字面量绑定（无参数引用）。
func Test_Bind_LiteralOnly(t *testing.T) {
	nodes := parseBody(t, `<bind name="status" value="'assigned'"/>where status = #{status}`)
	if refs := CollectBindRefs(nodes); len(refs) != 0 {
		t.Errorf("CollectBindRefs = %v, want empty", refs)
	}
	mp := map[string]interface{}{}
	for _, n := range nodes {
		n.PrepareSqlWithMap(mp, 0)
	}
	if mp["status"] != "assigned" {
		t.Errorf("bound status = %v, want assigned", mp["status"])
	}
}

// Test_Bind_NumberSum 数字求和与单值保留类型。
func Test_Bind_NumberSum(t *testing.T) {
	nodes := parseBody(t, `<bind name="off" value="100 + 20"/>x = #{off}`)
	mp := map[string]interface{}{}
	for _, n := range nodes {
		n.PrepareSqlWithMap(mp, 0)
	}
	if got, ok := mp["off"].(int64); !ok || got != 120 {
		t.Errorf("bound off = %v (%T), want int64(120)", mp["off"], mp["off"])
	}
}

// Test_Bind_Chained 前一个 bind 可被后续 bind 引用。
func Test_Bind_Chained(t *testing.T) {
	nodes := parseBody(t, `<bind name="a" value="'x'"/><bind name="b" value="a + 'y'"/>v = #{b}`)
	mp := map[string]interface{}{}
	for _, n := range nodes {
		n.PrepareSqlWithMap(mp, 0)
	}
	if mp["b"] != "xy" {
		t.Errorf("chained bind = %v, want xy", mp["b"])
	}
}

// Test_Bind_UnsupportedExpr 超出子集的表达式加载期报错（不再静默失效）。
func Test_Bind_UnsupportedExpr(t *testing.T) {
	cases := []string{
		`<bind name="x" value="a.toUpperCase()"/>`,
		`<bind name="x" value="a * 2"/>`,
		`<bind name="x" value="a > 1 ? 'y' : 'n'"/>`,
	}
	for _, c := range cases {
		root, err := ParseXmlContent([]byte("<x>" + c + "</x>"))
		if err != nil {
			t.Errorf("parse xml error: %v", err)
			continue
		}
		nodes, _ := ParseFragments(root.Elements, map[string]*SqlElement{})
		if len(nodes) != 0 {
			t.Errorf("%v: unsupported expression should be dropped, got %d nodes", c, len(nodes))
		}
	}
	// 缺 name/value 同样报错
	for _, c := range []string{`<bind value="'x'"/>`, `<bind name="x"/>`} {
		root, err := ParseXmlContent([]byte("<x>" + c + "</x>"))
		if err != nil {
			t.Errorf("parse xml error: %v", err)
			continue
		}
		if nodes, _ := ParseFragments(root.Elements, map[string]*SqlElement{}); len(nodes) != 0 {
			t.Errorf("%v: invalid <bind> should be dropped, got %d nodes", c, len(nodes))
		}
	}
}

// Test_UnsupportedTag_Collected 未注册标签保留可观测信息（P0-3）。
func Test_UnsupportedTag_Collected(t *testing.T) {
	nodes := parseBody(t, `insert into t(a, b) values(1, 2)<selectKey keyProperty="id">select 1</selectKey>`)
	tags := CollectUnsupportedTags(nodes)
	if len(tags) != 1 || tags[0] != "selectKey" {
		t.Errorf("CollectUnsupportedTags = %v, want [selectKey]", tags)
	}
	var sqlstr string
	for _, n := range nodes {
		s, _ := n.PrepareSqlWithMap(map[string]interface{}{}, 0)
		sqlstr += s
	}
	if !strings.Contains(sqlstr, "values(1, 2)") {
		t.Errorf("text around unsupported tag should keep rendering, got %q", sqlstr)
	}
}

// Test_UnsupportedTag_Nested 嵌套在 <where>/<if> 内的未注册标签也要被收集。
func Test_UnsupportedTag_Nested(t *testing.T) {
	nodes := parseBody(t, `<where><if test="a != null"><foo/></if>id = 1</where>`)
	tags := CollectUnsupportedTags(nodes)
	if len(tags) != 1 || tags[0] != "foo" {
		t.Errorf("CollectUnsupportedTags = %v, want [foo]", tags)
	}
}

// Test_Bind_InWhereAndIf <bind> 在 <where>/<if> 体内按文档序求值：
// 条件命中时先绑定再使用；条件不命中时绑定不执行、占位符所在文本也不渲染。
func Test_Bind_InWhereAndIf(t *testing.T) {
	nodes := parseBody(t, `<where>
  <if test="keyword != null"><bind name="pattern" value="'%' + keyword + '%'"/>name like #{pattern}</if>
</where>`)
	mp := map[string]interface{}{"keyword": "ab"}
	var sqlstr string
	for _, n := range nodes {
		sqlstr += n.GenerateSqlWithMap(mp, 0)
	}
	if !strings.Contains(sqlstr, "name like '%ab%'") {
		t.Errorf("bind inside <if> not applied, got %q", sqlstr)
	}

	mp2 := map[string]interface{}{}
	sqlstr = ""
	for _, n := range nodes {
		sqlstr += n.GenerateSqlWithMap(mp2, 0)
	}
	if strings.TrimSpace(sqlstr) != "" {
		t.Errorf("condition not met should render nothing, got %q", sqlstr)
	}
}
