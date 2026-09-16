package sqlfragment

import (
	"testing"
)

// parseTrimNodeFromText XML 文本 → trim 节点（走 XML 解析 → 标签注册表 → 节点构造全链路）。
func parseTrimNodeFromText(t *testing.T, content string) Node {
	t.Helper()
	root, err := ParseXmlContent([]byte(content))
	if err != nil {
		t.Error("parse xml error:", err)
		return nil
	}
	nodes, err := ParseFragments([]XmlElement{{ElementType: XmlNodeElem, Val: *root}}, nil)
	if err != nil {
		t.Error("parse fragments error:", err)
		return nil
	}
	if len(nodes) != 1 {
		t.Error("expected 1 node, got", len(nodes))
		return nil
	}
	return nodes[0]
}

// Test_SqlTrimParse 属性解析：prefix/suffix 字段、overrides 正则编译、无属性时正则为 nil。
func Test_SqlTrimParse(t *testing.T) {
	node := parseTrimNodeFromText(t,
		`<trim prefix="where" suffix="order by id" prefixOverrides="AND |OR " suffixOverrides=",">id = #{id}</trim>`)
	if node == nil {
		return
	}
	trim, ok := node.(*sqlTrim)
	if !ok {
		t.Errorf("node type = %T, want *sqlTrim", node)
		return
	}
	if trim.Prefix != "where" {
		t.Error("prefix =", trim.Prefix, "want where")
	}
	if trim.Suffix != "order by id" {
		t.Error("suffix =", trim.Suffix, "want order by id")
	}
	if trim.rePrefix == nil {
		t.Error("prefixOverrides regex should be compiled")
	}
	if trim.reSuffix == nil {
		t.Error("suffixOverrides regex should be compiled")
	}
	if len(trim.Sql) != 1 {
		t.Error("children =", len(trim.Sql), "want 1")
	}
	plain := parseTrimNodeFromText(t, `<trim>a = #{a}</trim>`)
	if plain == nil {
		return
	}
	pt, ok := plain.(*sqlTrim)
	if !ok {
		t.Errorf("node type = %T, want *sqlTrim", plain)
		return
	}
	if pt.Prefix != "" || pt.Suffix != "" || pt.rePrefix != nil || pt.reSuffix != nil {
		t.Error("attrs absent should yield empty fields and nil regexes")
	}
}

// Test_SqlTrimGenerate 渲染行为：空体、prefix/suffix、首尾剥离、嵌套节点、
// 与 <where>/<set> 等价、遍历器聚合。
func Test_SqlTrimGenerate(t *testing.T) {
	// ① 子片段全空 → 空串
	n := parseTrimNodeFromText(t,
		`<trim prefix="where"><if test="name != null">and name = #{name}</if></trim>`)
	if n == nil {
		return
	}
	if got := n.GenerateSqlWithMap(map[string]interface{}{}, 0); got != "" {
		t.Error("empty body should render empty, got", got)
	}
	if sqlstr, args := n.PrepareSqlWithMap(map[string]interface{}{}, 0); sqlstr != "" || len(args) != 0 {
		t.Error("empty body prepare should be empty, got", sqlstr, args)
	}

	// ② prefix / suffix 生效
	n = parseTrimNodeFromText(t,
		`<trim prefix="where" suffix="and del_flag = 0">id = #{id}</trim>`)
	if n == nil {
		return
	}
	if got := n.GenerateSqlWithoutParam(); got != "where id = #{id} and del_flag = 0" {
		t.Error("generate without param =", got, "want where id = #{id} and del_flag = 0")
	}
	if got := n.GenerateSqlWithMap(map[string]interface{}{"id": 1}, 0); got != "where id = 1 and del_flag = 0" {
		t.Error("generate with map =", got, "want where id = 1 and del_flag = 0")
	}

	// ③ prefixOverrides 多值剥头（仅首个，片段之间的 and/or 不受影响）
	n = parseTrimNodeFromText(t,
		`<trim prefix="where" prefixOverrides="AND |OR "><if test="a != null">and a = #{a}</if><if test="b != null">or b = #{b}</if></trim>`)
	if n == nil {
		return
	}
	got, args := n.PrepareSqlWithMap(map[string]interface{}{"a": 1, "b": 2}, 0)
	// 片段间双空格：if 子片段自带前导空格 + 容器连接空格（与 <where> 行为一致）
	if got != "where a = ?  or b = ?" {
		t.Error("prepare with map =", got, "want where a = ?  or b = ?")
	}
	if len(args) != 2 {
		t.Error("prepare args =", args, "want 2")
	}
	got, _ = n.PrepareSqlWithMap(map[string]interface{}{"b": 2}, 0)
	if got != "where b = ?" {
		t.Error("only b: prepare =", got, "want where b = ?")
	}

	// ④ suffixOverrides 尾部逗号剥离
	n = parseTrimNodeFromText(t, `<trim suffixOverrides=",">id, name,</trim>`)
	if n == nil {
		return
	}
	if got := n.GenerateSqlWithoutParam(); got != "id, name" {
		t.Error("trailing comma stripped =", got, "want id, name")
	}

	// ⑤ 嵌套 if/foreach、where
	n = parseTrimNodeFromText(t,
		`<trim prefix="where" prefixOverrides="AND |OR "><if test="ids != null">and id in <foreach collection="ids" item="id" open="(" separator="," close=")">#{id}</foreach></if></trim>`)
	if n == nil {
		return
	}
	got, args = n.PrepareSqlWithMap(map[string]interface{}{"ids": []int{1, 2}}, 0)
	if got != "where id in  ( ?, ?)" {
		t.Error("nested foreach prepare =", got, "want where id in  ( ?, ?)")
	}
	if len(args) != 2 {
		t.Error("foreach args =", args, "want 2")
	}
	n = parseTrimNodeFromText(t,
		`<trim suffix="limit 10"><where><if test="name != null">and name = #{name}</if></where></trim>`)
	if n == nil {
		return
	}
	if got := n.GenerateSqlWithMap(map[string]interface{}{"name": "x"}, 0); got != "where name = 'x' limit 10" {
		t.Error("nested where generate =", got, "want where name = 'x' limit 10")
	}

	// ⑥ <trim prefix="where" prefixOverrides="AND |OR "> 等价 <where>
	trimWhere := parseTrimNodeFromText(t,
		`<trim prefix="where" prefixOverrides="AND |OR "><if test="a != null">and a = #{a}</if><if test="b != null">and b = #{b}</if></trim>`)
	whereNode := parseTrimNodeFromText(t,
		`<where><if test="a != null">and a = #{a}</if><if test="b != null">and b = #{b}</if></where>`)
	if trimWhere == nil || whereNode == nil {
		return
	}
	for _, mp := range []map[string]interface{}{
		{"a": 1, "b": 2},
		{"a": 1},
		{},
	} {
		tw := trimWhere.GenerateSqlWithMap(mp, 0)
		ww := whereNode.GenerateSqlWithMap(mp, 0)
		if tw != ww {
			t.Errorf("trim(%q) != where(%q) with %v", tw, ww, mp)
		}
	}

	// ⑦ <trim prefix="set" suffixOverrides=","> 等价 <set>
	trimSet := parseTrimNodeFromText(t,
		`<trim prefix="set" suffixOverrides=","><if test="a != null">a = #{a},</if><if test="b != null">b = #{b},</if></trim>`)
	setNode := parseTrimNodeFromText(t,
		`<set><if test="a != null">a = #{a},</if><if test="b != null">b = #{b},</if></set>`)
	if trimSet == nil || setNode == nil {
		return
	}
	for _, mp := range []map[string]interface{}{
		{"a": 1, "b": 2},
		{"b": 2},
		{},
	} {
		ts := trimSet.GenerateSqlWithMap(mp, 0)
		ss := setNode.GenerateSqlWithMap(mp, 0)
		if ts != ss {
			t.Errorf("trim(%q) != set(%q) with %v", ts, ss, mp)
		}
	}

	// ⑧ 遍历器聚合：CollectSlots（直接文本）/ CollectIfTestFields / ContainsForEach
	n = parseTrimNodeFromText(t,
		`<trim prefix="where">id = #{id}<if test="name != null">and name = #{name}</if><if test="ids != null">and id in <foreach collection="ids" item="id" open="(" separator="," close=")">#{id}</foreach></if></trim>`)
	if n == nil {
		return
	}
	slots := n.CollectSlots()
	if len(slots) != 1 || slots[0] != "id" {
		t.Error("CollectSlots =", slots, "want [id]")
	}
	fields := n.CollectIfTestFields()
	if len(fields) != 2 || fields[0] != "name" || fields[1] != "ids" {
		t.Error("CollectIfTestFields =", fields, "want [name ids]")
	}
	if !n.ContainsForEach() {
		t.Error("ContainsForEach should be true with nested foreach")
	}
}
