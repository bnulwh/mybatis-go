package orm

import (
	"strings"
	"testing"

	"github.com/bnulwh/mybatis-go/types"
)

// Test_ApplyQueryWrapper 自动注入算法：无 WHERE / 已有 WHERE / 尾部子句边界 /
// 空 Wrapper / nil Wrapper / 附加子句合并 / 与分页组合。
func Test_ApplyQueryWrapper(t *testing.T) {
	// ① 无 WHERE：追加 WHERE (seg)
	w := NewQueryWrapper().Eq("name", "tom")
	got := applyQueryWrapper("select * from t_user", w)
	if got != "select * from t_user WHERE (name = 'tom')" {
		t.Errorf("no-where case = %q", got)
	}
	// ② 已有 WHERE：追加 AND (seg)
	w2 := NewQueryWrapper().Ge("age", 18)
	got = applyQueryWrapper("select * from t_user where id = 1", w2)
	if got != "select * from t_user where id = 1 AND (age >= 18)" {
		t.Errorf("has-where case = %q", got)
	}
	// ③ 尾部 ORDER BY/LIMIT：条件插入在 tail 之前，不落到 LIMIT 之后
	w3 := NewQueryWrapper().Le("id", 10)
	got = applyQueryWrapper("select * from t_user order by id limit 5", w3)
	if got != "select * from t_user WHERE (id <= 10) order by id limit 5" {
		t.Errorf("tail-clause case = %q", got)
	}
	// ④ 空 Wrapper：原样返回
	src := "select * from t_user order by id"
	if got = applyQueryWrapper(src, NewQueryWrapper()); got != src {
		t.Errorf("empty wrapper should keep sql, got %q", got)
	}
	// ⑤ nil Wrapper：原样返回
	if got = applyQueryWrapper(src, nil); got != src {
		t.Errorf("nil wrapper should keep sql, got %q", got)
	}
	// ⑥ Wrapper 附加子句：追加在原 tail 之后，last 恒置末尾
	w6 := NewQueryWrapper().OrderByDesc("id").Last("limit 3")
	got = applyQueryWrapper("select * from t_user", w6)
	if got != "select * from t_user ORDER BY id DESC limit 3" {
		t.Errorf("clause-merge case = %q", got)
	}
	w7 := NewQueryWrapper().Eq("status", 1)
	got = applyQueryWrapper("select * from t_user where deleted = 0 order by id", w7)
	if got != "select * from t_user where deleted = 0 AND (status = 1) order by id" {
		t.Errorf("where+tail case = %q", got)
	}
	// ⑦ 与 applyPagination 组合：条件注入后再追加 LIMIT
	w8 := NewQueryWrapper().Eq("status", 1)
	pageSQL := applyQueryWrapper("select * from t_user order by id", w8)
	pageSQL = applyPagination(pageSQL, 10, 20)
	if !strings.Contains(pageSQL, "WHERE (status = 1)") || !strings.Contains(pageSQL, "LIMIT") {
		t.Errorf("pagination combo = %q", pageSQL)
	}
	// ⑧ GroupBy/Having 合并
	w9 := NewQueryWrapper().GroupBy("dept").Having("count(*) > 1")
	got = applyQueryWrapper("select dept from t_user", w9)
	if got != "select dept from t_user GROUP BY dept HAVING count(*) > 1" {
		t.Errorf("group-having case = %q", got)
	}
}

// Test_HasNamedSlot Slots 含名检测（大小写/空白容忍）。
func Test_HasNamedSlot(t *testing.T) {
	f := &types.SqlFunction{Id: "x", Param: types.SqlParam{Slots: []string{"name", "EW"}}}
	if !hasNamedSlot(f, "ew") {
		t.Error("slot ew should match case-insensitively")
	}
	if hasNamedSlot(f, "other") {
		t.Error("nonexistent slot should not match")
	}
}

// Test_InjectWrapperArg 显式 ${ew} 参数注入：无参 → 单 map；map 参数 → 并入；位置参数 → 追加。
func Test_InjectWrapperArg(t *testing.T) {
	w := NewQueryWrapper().Eq("id", 1)
	// 无参
	args := injectWrapperArg(nil, w)
	if len(args) != 1 {
		t.Fatalf("no-arg inject len = %d, want 1", len(args))
	}
	m, ok := args[0].(map[string]interface{})
	if !ok || m["ew"] != w {
		t.Errorf("no-arg inject = %#v, want map{ew: wrapper}", args[0])
	}
	// 末参为 map → 并入
	mp := map[string]interface{}{"name": "n"}
	args = injectWrapperArg([]interface{}{mp}, w)
	if len(args) != 1 || mp["ew"] != w {
		t.Errorf("map merge inject = %#v", args)
	}
	// 位置参数 → 追加
	args = injectWrapperArg([]interface{}{"n"}, w)
	if len(args) != 2 || args[1] != w {
		t.Errorf("positional inject = %#v", args)
	}
}
