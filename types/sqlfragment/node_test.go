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
	// 未注册标签：解析报错并跳过（ParseFragments log-and-continue）
	nodes2, _ := ParseFragments([]XmlElement{{ElementType: XmlNodeElem, Val: XmlNode{Name: "no_such_tag"}}}, nil)
	if len(nodes2) != 0 {
		t.Error("unregistered tag should be skipped, got", len(nodes2))
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
