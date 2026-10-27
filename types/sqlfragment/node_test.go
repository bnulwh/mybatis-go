package sqlfragment

import (
	"strings"
	"testing"
)

// Test_ContainsForEach 递归检测片段树中的 <foreach>（含 if/include/where 嵌套）。
func Test_ContainsForEach(t *testing.T) {
	if ContainsForEach(nil) {
		t.Error("nil items should not contain foreach")
	}
	if ContainsForEach([]Node{&simpleSql{}}) {
		t.Error("plain sql should not contain foreach")
	}
	// 直接 foreach
	if !ContainsForEach([]Node{&sqlForLoop{}}) {
		t.Error("direct foreach not detected")
	}
	// if 嵌套 foreach
	if !ContainsForEach([]Node{&sqlIfTest{Sql: []Node{&sqlForLoop{}}}}) {
		t.Error("foreach inside if not detected")
	}
	// include 嵌套 foreach
	if !ContainsForEach([]Node{&sqlInclude{Fragments: []Node{&sqlForLoop{}}}}) {
		t.Error("foreach inside include not detected")
	}
	// where 嵌套 foreach
	if !ContainsForEach([]Node{&sqlWhere{Sql: []Node{&sqlForLoop{}}}}) {
		t.Error("foreach inside where not detected")
	}
	// 无 foreach
	if ContainsForEach([]Node{&sqlIfTest{Sql: []Node{&simpleSql{Sql: "x"}}}}) {
		t.Error("nested plain sql should not contain foreach")
	}
}

// Test_CollectAllSqlSlots 全量占位符收集（E1）：含 <if>/<choose>/<foreach>/
// <where>/<set>/<include> 体内，去重、按首现序；与 CollectSqlSlots（仅无条件位置）互补。
func Test_CollectAllSqlSlots(t *testing.T) {
	slotsEq := func(got []string, want ...string) bool {
		if len(got) != len(want) {
			return false
		}
		for i := range got {
			if buildKey(got[i]) != buildKey(want[i]) {
				return false
			}
		}
		return true
	}
	if got := CollectAllSqlSlots(nil); len(got) != 0 {
		t.Error("nil items should collect nothing, got", got)
	}
	// ① 纯文本：与 CollectSqlSlots 一致
	plain := []Node{parseSimpleSqlFromText("a = #{a}")}
	if !slotsEq(CollectSqlSlots(plain), "a") || !slotsEq(CollectAllSqlSlots(plain), "a") {
		t.Error("plain slot mismatch:", CollectSqlSlots(plain), CollectAllSqlSlots(plain))
	}
	// ② if 体内：CollectSqlSlots 不统计，CollectAllSqlSlots 统计
	ifBody := []Node{&sqlIfTest{Sql: []Node{parseSimpleSqlFromText("b = #{b}")}}}
	if !slotsEq(CollectSqlSlots(ifBody)) || !slotsEq(CollectAllSqlSlots(ifBody), "b") {
		t.Error("if body slot mismatch:", CollectSqlSlots(ifBody), CollectAllSqlSlots(ifBody))
	}
	// ③ 无条件 + if 体内同名/异名：去重、按首现序
	mixed := []Node{
		parseSimpleSqlFromText("a = #{a}"),
		&sqlIfTest{Sql: []Node{parseSimpleSqlFromText("a = #{a} and b = #{b}")}},
	}
	if !slotsEq(CollectSqlSlots(mixed), "a") || !slotsEq(CollectAllSqlSlots(mixed), "a", "b") {
		t.Error("mixed slot mismatch:", CollectSqlSlots(mixed), CollectAllSqlSlots(mixed))
	}
	// ④ foreach 体内
	loop := []Node{&sqlForLoop{Sql: []Node{parseSimpleSqlFromText("#{item.id}")}}}
	if !slotsEq(CollectSqlSlots(loop)) || !slotsEq(CollectAllSqlSlots(loop), "item.id") {
		t.Error("foreach slot mismatch:", CollectSqlSlots(loop), CollectAllSqlSlots(loop))
	}
	// ⑤ choose when/otherwise 体内
	ch := []Node{&sqlChoose{
		When:      []*sqlIfTest{{Sql: []Node{parseSimpleSqlFromText("x = #{x}")}}},
		Otherwise: []Node{parseSimpleSqlFromText("y = #{y}")},
	}}
	if !slotsEq(CollectSqlSlots(ch)) || !slotsEq(CollectAllSqlSlots(ch), "x", "y") {
		t.Error("choose slot mismatch:", CollectSqlSlots(ch), CollectAllSqlSlots(ch))
	}
	// ⑥ where 嵌套 if
	wh := []Node{&sqlWhere{Sql: []Node{&sqlIfTest{Sql: []Node{parseSimpleSqlFromText("c = #{c}")}}}}}
	if !slotsEq(CollectSqlSlots(wh)) || !slotsEq(CollectAllSqlSlots(wh), "c") {
		t.Error("where slot mismatch:", CollectSqlSlots(wh), CollectAllSqlSlots(wh))
	}
}

// Test_RegisterNodeParser 注册表：buildKey 归一化（大小写/空白不敏感）注册与查询、
// 未注册标签经 ParseFragments 报错跳过。
func Test_RegisterNodeParser(t *testing.T) {
	RegisterNodeParser(" testtag ", func(node XmlNode, sns map[string]*SqlElement) (Node, error) {
		return &simpleSql{Sql: "ok"}, nil
	})
	if _, ok := lookupNodeParser("TestTag"); !ok {
		t.Error("lookupNodeParser(TestTag) should hit testtag (case/space normalized)")
	}
	nodes, err := ParseFragments([]XmlElement{{ElementType: XmlNodeElem, Val: XmlNode{Name: "TESTTAG"}}}, nil)
	if err != nil {
		t.Error("ParseFragments error:", err)
	}
	if len(nodes) != 1 {
		t.Error("registered tag should parse to 1 node, got", len(nodes))
		return
	}
	if got := nodes[0].GenerateSqlWithoutParam(); got != "ok" {
		t.Error("custom node render =", got, "want ok")
	}
	// 未注册标签：P0-3 起不再丢弃，保留占位节点（不产出 SQL），
	// 供注册期 CollectUnsupportedTags 定位汇总（默认 Warn / strict-unknown-tag 报错）。
	nodes2, _ := ParseFragments([]XmlElement{{ElementType: XmlNodeElem, Val: XmlNode{Name: "no_such_tag"}}}, nil)
	if len(nodes2) != 1 {
		t.Error("unregistered tag should be kept as placeholder node, got", len(nodes2))
		return
	}
	if got := nodes2[0].GenerateSqlWithoutParam(); got != "" {
		t.Error("unregistered tag should render empty sql, got", got)
	}
	if tags := CollectUnsupportedTags(nodes2); len(tags) != 1 || tags[0] != "no_such_tag" {
		t.Error("CollectUnsupportedTags =", tags, "want [no_such_tag]")
	}
}

// Test_NodeOpenClosed 开放封闭验证：测试文件内注册自定义 <upper> 节点
// （body 转大写，复用 containerBase 骨架），走 XML 文本 → ParseFragments → 渲染全链路，
// 证明新节点接入零改动（不动包内任何既有代码）。
func Test_NodeOpenClosed(t *testing.T) {
	RegisterNodeParser("upper", func(node XmlNode, sns map[string]*SqlElement) (Node, error) {
		sts, _ := ParseFragments(node.Elements, sns)
		return &containerBase{
			Sql: sts,
			Wrap: func(body string) string {
				return strings.ToUpper(strings.TrimSpace(body))
			},
		}, nil
	})
	root, err := ParseXmlContent([]byte(`<upper>name = #{name}</upper>`))
	if err != nil {
		t.Error("parse xml error:", err)
		return
	}
	nodes, err := ParseFragments([]XmlElement{{ElementType: XmlNodeElem, Val: *root}}, nil)
	if err != nil {
		t.Error("ParseFragments error:", err)
		return
	}
	if len(nodes) != 1 {
		t.Error("expected 1 node, got", len(nodes))
		return
	}
	mp := map[string]interface{}{"name": "bob"}
	if got := nodes[0].GenerateSqlWithMap(mp, 0); got != "NAME = 'BOB'" {
		t.Error("generate with map =", got, "want NAME = 'BOB'")
	}
	sqlstr, args := nodes[0].PrepareSqlWithMap(mp, 0)
	if sqlstr != "NAME = ?" {
		t.Error("prepare with map =", sqlstr, "want NAME = ?")
	}
	if len(args) != 1 {
		t.Error("prepare args =", args, "want 1")
	}
	if got := nodes[0].Children(); len(got) != 1 {
		t.Error("children =", len(got), "want 1")
	}
}
