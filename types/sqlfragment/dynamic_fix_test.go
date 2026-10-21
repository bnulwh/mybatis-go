package sqlfragment

import (
	"strings"
	"testing"
)

// mustParseNamedChild 解析 mapper 形态 XML 并取指定名称的首个子节点（测试辅助）。
func mustParseNamedChild(t *testing.T, content, name string) XmlNode {
	t.Helper()
	root, err := ParseXmlContent([]byte(content))
	if err != nil {
		t.Error("ParseXmlContent error:", err)
		return XmlNode{}
	}
	for _, e := range root.Elements {
		if e.ElementType != XmlNodeElem {
			continue
		}
		xn := e.Val.(XmlNode)
		if xn.Name == name {
			return xn
		}
	}
	t.Errorf("child node %q not found", name)
	return XmlNode{}
}

// mustParseFragment 将 XmlNode 包装为 XmlElement 后按注册表解析为片段（测试辅助）。
func mustParseFragment(t *testing.T, xn XmlNode, sns map[string]*SqlElement) Node {
	t.Helper()
	node, err := parseFragmentFromXmlElement(XmlElement{ElementType: XmlNodeElem, Val: xn}, sns)
	if err != nil {
		t.Error("parseFragmentFromXmlElement error:", err)
		return nil
	}
	return node
}

func collapseWs(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// A2-1 注释切割静态文本：中间空格不得丢失（"SELECT aFROM t" 缺陷回归）。
func Test_SqlElement_CommentInsideTextKeepsSpace(t *testing.T) {
	xn := mustParseNamedChild(t,
		`<mapper><sql id="frag">SELECT a <!-- inner comment --> FROM t</sql></mapper>`, "sql")
	se := ParseSqlElementFromXmlNode(xn)
	if got := collapseWs(se.Sql); got != "SELECT a FROM t" {
		t.Error("comment inside static text lost inner space, got:", se.Sql)
	}
	if len(se.Fragments) != 0 {
		t.Error("pure text sql element should have no fragments, got:", se.Fragments)
	}
}

// A2-2 静态文本 + 动态标签混排：include 渲染不得丢静态文本，且保持文档序。
func Test_Include_MixedFragmentKeepsStaticText(t *testing.T) {
	xn := mustParseNamedChild(t,
		`<mapper><sql id="frag">SELECT a, b FROM t <if test="filter != null">WHERE b = #{filter}</if> ORDER BY id</sql></mapper>`,
		"sql")
	se := ParseSqlElementFromXmlNode(xn)
	sns := map[string]*SqlElement{"frag": se}
	incXn := mustParseNamedChild(t, `<mapper><include refid="frag"/></mapper>`, "include")
	inc := mustParseFragment(t, incXn, sns)
	if inc == nil {
		return
	}
	sqlstr, args := inc.PrepareSqlWithMap(map[string]interface{}{"filter": "x"}, 0)
	if got := collapseWs(sqlstr); got != "SELECT a, b FROM t WHERE b = ? ORDER BY id" {
		t.Errorf("include mixed render with filter:\n got  %q\n want %q", got, "SELECT a, b FROM t WHERE b = ? ORDER BY id")
	}
	if len(args) != 1 {
		t.Error("args want 1, got:", args)
	}
	sqlstr, args = inc.PrepareSqlWithMap(map[string]interface{}{}, 0)
	if got := collapseWs(sqlstr); got != "SELECT a, b FROM t ORDER BY id" {
		t.Errorf("include mixed render without filter:\n got  %q", got)
	}
	if len(args) != 0 {
		t.Error("args should be empty, got:", args)
	}
}

// A2-3 纯文本片段行为保持不变（回归保护）。
func Test_Include_PureTextFragmentUnchanged(t *testing.T) {
	xn := mustParseNamedChild(t,
		`<mapper><sql id="cols">id, user_name, group_id</sql></mapper>`, "sql")
	se := ParseSqlElementFromXmlNode(xn)
	sns := map[string]*SqlElement{"cols": se}
	incXn := mustParseNamedChild(t, `<mapper><include refid="cols"/></mapper>`, "include")
	inc := mustParseFragment(t, incXn, sns)
	if inc == nil {
		return
	}
	sqlstr, args := inc.PrepareSqlWithMap(map[string]interface{}{}, 0)
	if got := collapseWs(sqlstr); got != "id, user_name, group_id" {
		t.Error("pure text include render changed:", sqlstr)
	}
	if len(args) != 0 {
		t.Error("args should be empty, got:", args)
	}
}

// A1-1 foreach 循环体嵌套 <if>：不得 panic，按元素逐项求值渲染。
func Test_Foreach_NestedIfRenders(t *testing.T) {
	xn := mustParseNamedChild(t,
		`<mapper><foreach collection="items" item="it" open="(" close=")" separator=","><if test="it != null">#{it}</if></foreach></mapper>`,
		"foreach")
	node := mustParseFragment(t, xn, nil)
	if node == nil {
		return
	}
	sqlstr, args := node.PrepareSqlWithMap(map[string]interface{}{
		"items": []interface{}{1, "x"},
	}, 0)
	if got := collapseWs(sqlstr); got != "( ?, ?)" {
		t.Error("foreach nested if render unexpected:", sqlstr)
	}
	if len(args) != 2 {
		t.Error("args want 2, got:", args)
	}
	// nil 项渲染空但分隔符仍拼接（MyBatis foreach 语义：分隔符位于所有迭代之间）
	sqlstr, args = node.PrepareSqlWithMap(map[string]interface{}{
		"items": []interface{}{1, nil, "x"},
	}, 0)
	if got := collapseWs(sqlstr); got != "( ?, , ?)" {
		t.Error("foreach nested if render with nil item unexpected:", sqlstr)
	}
	if len(args) != 2 {
		t.Error("args want 2, got:", args)
	}
}

// A1-2 foreach 纯文本循环体：行为保持不变（回归保护）。
func Test_Foreach_PlainTextUnchanged(t *testing.T) {
	xn := mustParseNamedChild(t,
		`<mapper><foreach collection="ids" item="id" open="(" close=")" separator=",">#{id}</foreach></mapper>`,
		"foreach")
	node := mustParseFragment(t, xn, nil)
	if node == nil {
		return
	}
	sqlstr, args := node.PrepareSqlWithMap(map[string]interface{}{
		"ids": []interface{}{1, 2, 3},
	}, 0)
	if got := collapseWs(sqlstr); got != "( ?, ?, ?)" {
		t.Error("foreach plain text render changed:", sqlstr)
	}
	if len(args) != 3 {
		t.Error("args want 3, got:", args)
	}
}

// A1-3 choose 的 otherwise 含嵌套标签：不得 panic，正常渲染。
func Test_Choose_OtherwiseNestedTags(t *testing.T) {
	xn := mustParseNamedChild(t,
		`<mapper><choose><when test="a != null">WHEN_A</when><otherwise><if test="b != null">OTH_B</if> OTH_STATIC</otherwise></choose></mapper>`,
		"choose")
	node := mustParseFragment(t, xn, nil)
	if node == nil {
		return
	}
	sqlstr, _ := node.PrepareSqlWithMap(map[string]interface{}{"b": 1}, 0)
	if got := collapseWs(sqlstr); got != "OTH_B OTH_STATIC" {
		t.Error("choose otherwise nested render unexpected:", sqlstr)
	}
	sqlstr, _ = node.PrepareSqlWithMap(map[string]interface{}{"a": 1}, 0)
	if got := collapseWs(sqlstr); got != "WHEN_A" {
		t.Error("choose when render unexpected:", sqlstr)
	}
}

// A1-4 choose 纯文本 otherwise：行为保持不变（回归保护）。
func Test_Choose_PlainOtherwiseUnchanged(t *testing.T) {
	xn := mustParseNamedChild(t,
		`<mapper><choose><when test="a != null">A</when><otherwise>DEFAULT</otherwise></choose></mapper>`,
		"choose")
	node := mustParseFragment(t, xn, nil)
	if node == nil {
		return
	}
	sqlstr, _ := node.PrepareSqlWithMap(map[string]interface{}{}, 0)
	if got := collapseWs(sqlstr); got != "DEFAULT" {
		t.Error("choose plain otherwise render changed:", sqlstr)
	}
	if got := node.GenerateSqlWithoutParam(); collapseWs(got) != "DEFAULT" {
		t.Error("choose without param render changed:", got)
	}
}

// A3 现状语义锁定：未识别表达式产生空条件（恒真渲染），解析不报错。
func Test_ParseIfConditions_UnrecognizedKeepsAlwaysTrue(t *testing.T) {
	conds := parseIfConditionsFromText("a == b")
	if len(conds) != 0 {
		t.Error("unrecognized expression should produce no conditions, got:", conds)
	}
	inf := &sqlIfTest{Test: "a == b", Conditions: conds, Sql: []Node{parseSimpleSqlFromText("X")}}
	if !inf.checkConditions(map[string]interface{}{}) {
		t.Error("empty conditions keep always-true semantics (documented A3)")
	}
}

// A2-4 注释预处理：剥离注释保文本、CDATA 内注释保留、未闭合注释原样保留。
func Test_stripXmlComments(t *testing.T) {
	out := string(stripXmlComments([]byte(`<a>SELECT a <!-- c --> FROM t</a>`)))
	if strings.Contains(out, "<!--") || strings.Contains(out, "-->") {
		t.Error("comment not stripped:", out)
	}
	if got := collapseWs(out); got != "<a>SELECT a FROM t</a>" {
		t.Error("text around comment corrupted:", out)
	}
	out = string(stripXmlComments([]byte(`<a><![CDATA[x <!-- y --> z]]></a>`)))
	if !strings.Contains(out, "x <!-- y --> z") {
		t.Error("comment inside CDATA should be preserved:", out)
	}
	out = string(stripXmlComments([]byte(`<a><!-- unclosed`)))
	if !strings.Contains(out, "<!-- unclosed") {
		t.Error("unterminated comment should be kept as-is:", out)
	}
	out = string(stripXmlComments([]byte(`<a>no comment</a>`)))
	if got := collapseWs(out); got != "<a>no comment</a>" {
		t.Error("content without comment changed:", out)
	}
}
