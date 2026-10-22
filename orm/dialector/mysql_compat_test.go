package dialector

import (
	"testing"
)

// F1：MySQL→PG 族函数/语法翻译引擎。

func Test_RewriteMySqlCompat(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		// ifnull → coalesce
		{"ifnull simple", "SELECT ifnull(a, b) FROM t", "SELECT coalesce(a, b) FROM t"},
		{"IFNULL case-insensitive", "SELECT IFNULL(a, 0) FROM t", "SELECT coalesce(a, 0) FROM t"},
		{"ifnull nested args", "SELECT ifnull(ifnull(a, b), c) FROM t", "SELECT coalesce(coalesce(a, b), c) FROM t"},
		{"ifnull no paren untouched", "SELECT ifnull_col FROM t", "SELECT ifnull_col FROM t"},
		{"ifnull space before paren", "SELECT ifnull (a, b) FROM t", "SELECT coalesce(a, b) FROM t"},
		// find_in_set → position
		{"find_in_set simple", "SELECT * FROM t WHERE find_in_set(x, csv) > 0",
			"SELECT * FROM t WHERE position(','||x||',' in ','||csv||',') > 0"},
		{"find_in_set nested paren in arg", "SELECT * FROM t WHERE find_in_set(concat(a, b), tags) > 0",
			"SELECT * FROM t WHERE position(','||concat(a, b)||',' in ','||tags||',') > 0"},
		{"find_in_set one arg verbatim", "SELECT find_in_set(x) FROM t", "SELECT find_in_set(x) FROM t"},
		{"find_in_set nested call in arg", "SELECT find_in_set(ifnull(a, b), csv) FROM t",
			"SELECT position(','||coalesce(a, b)||',' in ','||csv||',') FROM t"},
		// date_format → to_char
		{"date_format simple", "SELECT date_format(d, '%Y-%m-%d') FROM t",
			"SELECT to_char(d, 'YYYY-MM-DD') FROM t"},
		{"date_format datetime", "SELECT date_format(d, '%Y-%m-%d %H:%i:%s') FROM t",
			"SELECT to_char(d, 'YYYY-MM-DD HH24:MI:SS') FROM t"},
		{"date_format percent T", "SELECT DATE_FORMAT(d, '%T') FROM t",
			"SELECT to_char(d, 'HH24:MI:SS') FROM t"},
		{"date_format percent r", "SELECT DATE_FORMAT(d, '%r') FROM t",
			"SELECT to_char(d, 'HH12:MI:SS AM') FROM t"},
		{"date_format month names", "SELECT DATE_FORMAT(d, '%M %b %W %a') FROM t",
			"SELECT to_char(d, 'Month Mon Day Dy') FROM t"},
		{"date_format misc", "SELECT DATE_FORMAT(d, '%j %p %f %y %I %l %%') FROM t",
			"SELECT to_char(d, 'DDD AM US YY HH12 HH12 %') FROM t"},
		{"date_format unmapped spec", "SELECT DATE_FORMAT(d, '%D %q') FROM t",
			"SELECT to_char(d, '%D %q') FROM t"},
		{"date_format non literal skipped", "SELECT date_format(d, fmt_col) FROM t",
			"SELECT date_format(d, fmt_col) FROM t"},
		// 反引号 → 双引号
		{"backtick ident", "SELECT `col` FROM `tab`", `SELECT "col" FROM "tab"`},
		{"backtick with escaped backtick", "SELECT `a``b` FROM t", "SELECT \"a`b\" FROM t"},
		{"unterminated backtick verbatim", "SELECT `col FROM t", "SELECT `col FROM t"},
		// 引号/注释保护
		{"string literal untouched", "SELECT 'ifnull(a,b)' , d FROM t", "SELECT 'ifnull(a,b)' , d FROM t"},
		{"escaped quote in literal", "SELECT 'it''s ifnull' FROM t", "SELECT 'it''s ifnull' FROM t"},
		{"line comment untouched", "SELECT a -- ifnull(x,y)\n FROM t", "SELECT a -- ifnull(x,y)\n FROM t"},
		{"block comment untouched", "SELECT a /* date_format(d,'%Y') */ FROM t", "SELECT a /* date_format(d,'%Y') */ FROM t"},
		// 词边界
		{"word boundary", "SELECT xifnull(a,b), find_in_set_x FROM t", "SELECT xifnull(a,b), find_in_set_x FROM t"},
		// 组合
		{"combined", "SELECT ifnull(`a`, 0), date_format(d, '%Y-%m-%d') FROM `t` WHERE find_in_set(`id`, `ids`) > 0",
			`SELECT coalesce("a", 0), to_char(d, 'YYYY-MM-DD') FROM "t" WHERE position(','||"id"||',' in ','||"ids"||',') > 0`},
		// 占位符不动
		{"placeholders kept", "SELECT ifnull(a, ?) FROM t WHERE b = ?", "SELECT coalesce(a, ?) FROM t WHERE b = ?"},
	}
	for _, c := range cases {
		got := RewriteMySqlCompat(c.in)
		if got != c.want {
			t.Errorf("%s: RewriteMySqlCompat(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// Test_RewriteMySqlCompat_Impotent 改写可重入：二次改写结果不变。
func Test_RewriteMySqlCompat_Impotent(t *testing.T) {
	in := "SELECT ifnull(`a`, 0), date_format(d, '%Y-%m-%d %H:%i:%s') FROM `t` WHERE find_in_set(`id`, ids) > 0"
	once := RewriteMySqlCompat(in)
	twice := RewriteMySqlCompat(once)
	if once != twice {
		t.Errorf("rewrite not impotent:\nonce  = %q\ntwice = %q", once, twice)
	}
}

// Test_RewriteMySqlCompat_NoMatch 不含目标语法时原样返回（快速路径）。
func Test_RewriteMySqlCompat_NoMatch(t *testing.T) {
	in := "SELECT a, b FROM tab WHERE c = 1"
	if got := RewriteMySqlCompat(in); got != in {
		t.Errorf("no-match input should be returned as-is, got %q", got)
	}
}
