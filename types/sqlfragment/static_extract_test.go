package sqlfragment

import (
	"strings"
	"testing"
)

// buildStaticItems 从 XML 元素列表解析片段（测试辅助）。
func buildStaticItems(t *testing.T, elems []XmlElement, sns map[string]*SqlElement) []Node {
	t.Helper()
	nodes, err := ParseFragments(elems, sns)
	if err != nil {
		t.Error("ParseFragments error:", err)
	}
	return nodes
}

// Test_ExtractStaticSQL_StaticText 纯静态文本：#{} 按出现顺序替换为 ?。
func Test_ExtractStaticSQL_StaticText(t *testing.T) {
	items := []Node{parseSimpleSqlFromText("select * from t_user where id = #{id,jdbcType=INTEGER} and name = #{name}")}
	sqlstr, params, ok := ExtractStaticSQL(items)
	if !ok {
		t.Error("static text should be extracted")
		return
	}
	if sqlstr != "select * from t_user where id = ? and name = ?" {
		t.Error("unexpected sql:", sqlstr)
	}
	if len(params) != 2 {
		t.Error("want 2 params, got", len(params))
		return
	}
	if params[0].Name != "id" || params[0].JdbcType != "INTEGER" {
		t.Error("params[0] unexpected:", params[0])
	}
	if params[1].Name != "name" || params[1].JdbcType != "" {
		t.Error("params[1] unexpected:", params[1])
	}
}

// Test_ExtractStaticSQL_RepeatPlaceholder 同名占位符重复出现：逐个替换、参数按位推进。
func Test_ExtractStaticSQL_RepeatPlaceholder(t *testing.T) {
	items := []Node{parseSimpleSqlFromText("select * from t where a = #{a} or a = #{a}")}
	sqlstr, params, ok := ExtractStaticSQL(items)
	if !ok {
		t.Error("repeat placeholder should be static")
		return
	}
	if sqlstr != "select * from t where a = ? or a = ?" {
		t.Error("unexpected sql:", sqlstr)
	}
	if len(params) != 2 || params[0].Name != "a" || params[1].Name != "a" {
		t.Error("params unexpected:", params)
	}
}

// Test_ExtractStaticSQL_Include 纯文本 <include>：内联展开后仍静态。
func Test_ExtractStaticSQL_Include(t *testing.T) {
	sns := map[string]*SqlElement{
		"cols": {Id: "cols", Sql: "id, user_name, group_id"},
	}
	items := buildStaticItems(t, []XmlElement{
		{ElementType: XmlTextElem, Val: "select "},
		{ElementType: XmlNodeElem, Val: XmlNode{Name: "include", Attrs: map[string]string{"refid": "cols"}}},
		{ElementType: XmlTextElem, Val: " from user_info where id = #{id,jdbcType=INTEGER}"},
	}, sns)
	sqlstr, params, ok := ExtractStaticSQL(items)
	if !ok {
		t.Error("include with plain text sql element should be static")
		return
	}
	got := collapseExtractSpace(sqlstr)
	if got != "select id, user_name, group_id from user_info where id = ?" {
		t.Error("unexpected sql:", sqlstr)
	}
	if len(params) != 1 || params[0].Name != "id" {
		t.Error("params unexpected:", params)
	}
}

// Test_ExtractStaticSQL_IncludeDynamic include 引用的 <sql> 内含动态标签：非静态。
func Test_ExtractStaticSQL_IncludeDynamic(t *testing.T) {
	sns := map[string]*SqlElement{
		"dyn": {Id: "dyn", Sql: "", Fragments: []Node{&sqlIfTest{Sql: []Node{&simpleSql{Sql: "and x = 1"}}}}},
	}
	items := buildStaticItems(t, []XmlElement{
		{ElementType: XmlTextElem, Val: "select * from t "},
		{ElementType: XmlNodeElem, Val: XmlNode{Name: "include", Attrs: map[string]string{"refid": "dyn"}}},
	}, sns)
	if _, _, ok := ExtractStaticSQL(items); ok {
		t.Error("include with dynamic fragment should not be static")
	}
}

// Test_ExtractStaticSQL_DynamicTags if/where/foreach 等动态节点一律非静态。
func Test_ExtractStaticSQL_DynamicTags(t *testing.T) {
	cases := map[string][]Node{
		"if":      {&sqlIfTest{Sql: []Node{&simpleSql{Sql: "and x = 1"}}}},
		"foreach": {&sqlForLoop{}},
		"where":   {&sqlWhere{Sql: []Node{&simpleSql{Sql: "x = 1"}}}},
		"trim":    {&sqlTrim{containerBase: containerBase{Sql: []Node{&simpleSql{Sql: "x = 1"}}}}},
		"choose":  {&sqlChoose{}},
		"set":     {&sqlSet{Sql: []Node{&simpleSql{Sql: "x = 1"}}}},
	}
	for name, items := range cases {
		if _, _, ok := ExtractStaticSQL(append([]Node{&simpleSql{Sql: "select * from t"}}, items...)); ok {
			t.Errorf("%v node should not be static", name)
		}
	}
}

// Test_ExtractStaticSQL_RawInjection ${} 原始注入：非静态。
func Test_ExtractStaticSQL_RawInjection(t *testing.T) {
	items := []Node{parseSimpleSqlFromText("select * from t order by ${sortCol}")}
	if _, _, ok := ExtractStaticSQL(items); ok {
		t.Error("${} raw injection should not be static")
	}
}

// Test_ExtractStaticSQL_Empty 空片段树 / 纯空白文本：无可提取 SQL。
func Test_ExtractStaticSQL_Empty(t *testing.T) {
	if _, _, ok := ExtractStaticSQL(nil); ok {
		t.Error("nil items should not be static")
	}
	if _, _, ok := ExtractStaticSQL([]Node{}); ok {
		t.Error("empty items should not be static")
	}
	if _, _, ok := ExtractStaticSQL([]Node{&simpleSql{Sql: "  "}}); ok {
		t.Error("whitespace-only items should not be static")
	}
}

// Test_ExtractStaticSQL_MultiFragments 多文本片段拼接：片段间以空格连接（与 Prepare 语义一致）。
func Test_ExtractStaticSQL_MultiFragments(t *testing.T) {
	items := []Node{
		parseSimpleSqlFromText("select *"),
		parseSimpleSqlFromText("from t"),
		parseSimpleSqlFromText("where id = #{id}"),
	}
	sqlstr, params, ok := ExtractStaticSQL(items)
	if !ok {
		t.Error("multi fragments should be static")
		return
	}
	if collapseExtractSpace(sqlstr) != "select * from t where id = ?" {
		t.Error("unexpected sql:", sqlstr)
	}
	if len(params) != 1 {
		t.Error("params unexpected:", params)
	}
}

// collapseExtractSpace 压缩空白便于断言（提取器保留原文空白）。
func collapseExtractSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
