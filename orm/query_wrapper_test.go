package orm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bnulwh/mybatis-go/types/sqlfragment"
	_ "modernc.org/sqlite"
)

// Test_QueryWrapperOperators 全部条件操作符渲染（值内联转义、In 多值、空集合）。
func Test_QueryWrapperOperators(t *testing.T) {
	w := NewQueryWrapper().
		Eq("name", "tom").
		Ne("dept", "dev").
		Gt("age", 18).
		Ge("score", 60).
		Lt("level", 9).
		Le("rank", 100)
	want := "name = 'tom' AND dept <> 'dev' AND age > 18 AND score >= 60 AND level < 9 AND rank <= 100"
	if got := w.GetSQLSegment(); got != want {
		t.Errorf("operators segment =\n  %q\nwant\n  %q", got, want)
	}
	// 模糊
	w2 := NewQueryWrapper().Like("name", "a").NotLike("name", "b").LikeLeft("name", "c").LikeRight("name", "d")
	want2 := "name LIKE '%a%' AND name NOT LIKE '%b%' AND name LIKE '%c' AND name LIKE 'd%'"
	if got := w2.GetSQLSegment(); got != want2 {
		t.Errorf("like segment = %q, want %q", got, want2)
	}
	// 区间 / 空判断
	w3 := NewQueryWrapper().Between("age", 18, 30).IsNull("remark").IsNotNull("name")
	want3 := "age BETWEEN 18 AND 30 AND remark IS NULL AND name IS NOT NULL"
	if got := w3.GetSQLSegment(); got != want3 {
		t.Errorf("between segment = %q, want %q", got, want3)
	}
	// In 多值 / 空集合
	w4 := NewQueryWrapper().In("id", 1, 2, 3).NotIn("dept", "a", "b")
	want4 := "id IN (1, 2, 3) AND dept NOT IN ('a', 'b')"
	if got := w4.GetSQLSegment(); got != want4 {
		t.Errorf("in segment = %q, want %q", got, want4)
	}
	if got := NewQueryWrapper().In("id").GetSQLSegment(); got != "1=0" {
		t.Errorf("empty in = %q, want 1=0", got)
	}
	if got := NewQueryWrapper().NotIn("id").GetSQLSegment(); got != "1=1" {
		t.Errorf("empty not-in = %q, want 1=1", got)
	}
}

// Test_QueryWrapperEscaping 值转义：单引号替换为双引号（与框架值内联约定一致）、
// 字符串经 FormatValue 与 SQLSegmentProvider 接口路径渲染。
func Test_QueryWrapperEscaping(t *testing.T) {
	w := NewQueryWrapper().Eq("name", "o'neil")
	if got := w.GetSQLSegment(); got != `name = 'o"neil'` {
		t.Errorf("quote escape = %q", got)
	}
	// QueryWrapper 实现 sqlfragment.SQLSegmentProvider（FormatValue 原样输出片段）
	var sp sqlfragment.SQLSegmentProvider = *NewQueryWrapper().Eq("id", 1)
	if got := sqlfragment.FormatValue(sp); got != "id = 1" {
		t.Errorf("FormatValue(wrapper) = %q, want %q", got, "id = 1")
	}
}

// Test_QueryWrapperOrNested Or / Nested 括号嵌套 / Apply / 子句拼接 / Where 前缀。
func Test_QueryWrapperOrNested(t *testing.T) {
	w := NewQueryWrapper().
		Eq("a", 1).
		Or().
		Nested(func(sub *QueryWrapper) {
			sub.Gt("b", 2).Or().IsNull("c")
		}).
		Apply("d = raw_func()")
	want := "a = 1 OR (b > 2 OR c IS NULL) AND d = raw_func()"
	if got := w.GetSQLSegment(); got != want {
		t.Errorf("or/nested segment =\n  %q\nwant\n  %q", got, want)
	}
	// 子句 + last + where 前缀
	w2 := NewQueryWrapper().Eq("a", 1).GroupBy("g").Having("count(*) > 1").OrderByAsc("x", "y").OrderByDesc("z").Last("limit 10")
	want2 := "a = 1 GROUP BY g HAVING count(*) > 1 ORDER BY x ASC, y ASC, z DESC limit 10"
	if got := w2.GetSQLSegment(); got != want2 {
		t.Errorf("clause segment =\n  %q\nwant\n  %q", got, want2)
	}
	wantWhere := "WHERE " + want2
	if got := w2.GetSQLWhereSegment(); got != wantWhere {
		t.Errorf("where segment = %q, want %q", got, wantWhere)
	}
	// 空 Wrapper：segment / where 均为空
	if got := NewQueryWrapper().GetSQLSegment(); got != "" {
		t.Errorf("empty segment = %q, want empty", got)
	}
	if got := NewQueryWrapper().GetSQLWhereSegment(); got != "" {
		t.Errorf("empty where segment = %q, want empty", got)
	}
	// Select 列
	if got := NewQueryWrapper().Select("id", "name").GetSQLSelectSegment(); got != "id, name" {
		t.Errorf("select segment = %q", got)
	}
}

type WrapUserMapper struct {
	BaseMapper
	SelectByWrapper     func(w *QueryWrapper) ([]map[string]interface{}, error)
	SelectPageByWrapper func(pp *PageParam, w *QueryWrapper) (*Page, error)
	SelectByWrapperEw   func(w *QueryWrapper) ([]map[string]interface{}, error)
	UpdateAll           func(w *QueryWrapper) (int64, error)
}

// initWrapUserTest 初始化 SQLite 数据源（QueryWrapper 专用 mapper：
// 无参 select（自动注入）、分页、${ew} 显式占位、update（select 限定校验用））。
func initWrapUserTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="WrapUserMapper">
  <select id="selectByWrapper" resultType="map">
    select id, name from wrap_user
  </select>
  <select id="selectPageByWrapper" resultType="map">
    select id, name from wrap_user
  </select>
  <select id="selectByWrapperEw" resultType="map">
    select id, name from wrap_user where ${ew}
  </select>
  <update id="updateAll">
    update wrap_user set name = 'x'
  </update>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "WrapUserMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	dbPath := filepath.Join(dir, "test.db")
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return ""
	}
	return dir
}

func initWrapUserData(t *testing.T) {
	t.Helper()
	if _, err := Execute(`CREATE TABLE wrap_user (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	names := []string{"a", "ab", "b", "c"}
	for i, n := range names {
		if _, err := Execute(`INSERT INTO wrap_user (id, name) VALUES (?, ?)`, i+1, n); err != nil {
			t.Errorf("insert failed: %v", err)
			return
		}
	}
}

// Test_SqliteQueryWrapper SQLite 端到端（自动注入）：Eq/Like/In/OrderBy 组合验证结果集；
// update 语句携带 Wrapper 返回错误（P0 限定 select）。
func Test_SqliteQueryWrapper(t *testing.T) {
	dir := initWrapUserTest(t)
	if dir == "" {
		return
	}
	defer Close()
	initWrapUserData(t)
	if err := RegisterMapper(new(WrapUserMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("WrapUserMapper").(WrapUserMapper)

	// Like + OrderByAsc：name LIKE 'a%' → a, ab
	w := NewQueryWrapper().LikeRight("name", "a").OrderByAsc("id")
	rows, err := mp.SelectByWrapper(w)
	if err != nil {
		t.Error("select by wrapper failed:", err)
		return
	}
	if len(rows) != 2 || rows[0]["name"] != "a" || rows[1]["name"] != "ab" {
		t.Errorf("like-right rows = %v, want [a ab]", rows)
		return
	}
	// In + OrderByDesc：id IN (1, 3) → 3, 1
	w2 := NewQueryWrapper().In("id", 1, 3).OrderByDesc("id")
	rows2, err := mp.SelectByWrapper(w2)
	if err != nil {
		t.Error("select by wrapper (in) failed:", err)
		return
	}
	if len(rows2) != 2 || rows2[0]["id"].(int64) != 3 || rows2[1]["id"].(int64) != 1 {
		t.Errorf("in rows = %v, want id [3 1]", rows2)
		return
	}
	// Or / Nested：(name = 'a' OR id >= 4) → a, c
	w3 := NewQueryWrapper().Nested(func(sub *QueryWrapper) {
		sub.Eq("name", "a").Or().Ge("id", 4)
	})
	rows3, err := mp.SelectByWrapper(w3)
	if err != nil {
		t.Error("select by wrapper (nested) failed:", err)
		return
	}
	if len(rows3) != 2 {
		t.Errorf("nested rows = %v, want 2 rows", rows3)
		return
	}
	// update 携带 Wrapper → 错误（P0 限定 select）
	if _, err := mp.UpdateAll(NewQueryWrapper().Eq("id", 1)); err == nil {
		t.Error("update with wrapper should return error")
	} else if !strings.Contains(err.Error(), "only support select") {
		t.Error("update with wrapper error =", err)
	}
}

// Test_SqliteQueryWrapperPage SQLite 端到端：PageParam + QueryWrapper 组合，
// Total 与 Records 均受 wrapper 条件影响（count/page 一致）。
func Test_SqliteQueryWrapperPage(t *testing.T) {
	dir := initWrapUserTest(t)
	if dir == "" {
		return
	}
	defer Close()
	initWrapUserData(t)
	if err := RegisterMapper(new(WrapUserMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("WrapUserMapper").(WrapUserMapper)

	// 基线：无 wrapper 全量分页
	page, err := mp.SelectPageByWrapper(&PageParam{PageNum: 1, PageSize: 2}, NewQueryWrapper())
	if err != nil {
		t.Error("page without wrapper failed:", err)
		return
	}
	if page.Total != 4 || len(page.Records) != 2 {
		t.Errorf("baseline page: total=%d records=%d, want 4/2", page.Total, len(page.Records))
		return
	}
	// wrapper 条件：name LIKE 'a%' → total 2、records 2
	w := NewQueryWrapper().LikeRight("name", "a")
	page2, err := mp.SelectPageByWrapper(&PageParam{PageNum: 1, PageSize: 10}, w)
	if err != nil {
		t.Error("page with wrapper failed:", err)
		return
	}
	if page2.Total != 2 {
		t.Errorf("filtered total = %d, want 2", page2.Total)
	}
	if len(page2.Records) != 2 {
		t.Errorf("filtered records = %d, want 2", len(page2.Records))
	}
}

// Test_SqliteQueryWrapperExplicit SQLite 端到端（显式 ${ew} 占位）：
// XML 中 where ${ew}，Wrapper 片段按占位符渲染（含 ORDER BY 子句）。
func Test_SqliteQueryWrapperExplicit(t *testing.T) {
	dir := initWrapUserTest(t)
	if dir == "" {
		return
	}
	defer Close()
	initWrapUserData(t)
	if err := RegisterMapper(new(WrapUserMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("WrapUserMapper").(WrapUserMapper)

	w := NewQueryWrapper().Ge("id", 3).OrderByDesc("id")
	rows, err := mp.SelectByWrapperEw(w)
	if err != nil {
		t.Error("select by wrapper ew failed:", err)
		return
	}
	if len(rows) != 2 {
		t.Errorf("ew rows = %d, want 2", len(rows))
		return
	}
	if rows[0]["id"].(int64) != 4 || rows[1]["id"].(int64) != 3 {
		t.Errorf("ew rows order = %v, want id [4 3]", rows)
	}
}
