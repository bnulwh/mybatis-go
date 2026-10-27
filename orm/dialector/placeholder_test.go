package dialector

import "testing"

// Test_rewritePlaceholders_literals P0-1：字面量/标识符/注释里的 ? 不得改写，
// 代码区占位符序号不受影响（旧实现 strings.Split("?") 会整体错位）。
func Test_rewritePlaceholders_literals(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "string literal",
			in:   `SELECT * FROM t WHERE c = '?' AND id = ?`,
			want: `SELECT * FROM t WHERE c = '?' AND id = $1`,
		},
		{
			name: "escaped quote in literal",
			in:   `SELECT * FROM t WHERE c = 'it''s ?' AND id = ?`,
			want: `SELECT * FROM t WHERE c = 'it''s ?' AND id = $1`,
		},
		{
			name: "quoted identifier",
			in:   `SELECT "?" , ` + "`?`" + ` FROM t WHERE id = ?`,
			want: `SELECT "?" , ` + "`?`" + ` FROM t WHERE id = $1`,
		},
		{
			name: "bracket identifier",
			in:   `SELECT [?] FROM t WHERE id = ?`,
			want: `SELECT [?] FROM t WHERE id = $1`,
		},
		{
			name: "line comment",
			in:   "SELECT * FROM t WHERE a = ? -- is this ok?\nAND b = ?",
			want: "SELECT * FROM t WHERE a = $1 -- is this ok?\nAND b = $2",
		},
		{
			name: "block comment",
			in:   `SELECT * FROM t /* why? */ WHERE a = ?`,
			want: `SELECT * FROM t /* why? */ WHERE a = $1`,
		},
		{
			name: "oracle hint kept",
			in:   `SELECT /*+ INDEX(t idx?) */ * FROM t WHERE a = ?`,
			want: `SELECT /*+ INDEX(t idx?) */ * FROM t WHERE a = $1`,
		},
		{
			name: "dollar quoted",
			in:   `SELECT $$?$$, $tag$ ? $tag$ FROM t WHERE a = ?`,
			want: `SELECT $$?$$, $tag$ ? $tag$ FROM t WHERE a = $1`,
		},
		{
			name: "jsonb question operator",
			in:   `SELECT * FROM t WHERE payload ? 'k' AND id = ?`,
			want: `SELECT * FROM t WHERE payload ? 'k' AND id = $1`,
		},
		{
			name: "jsonb question or operator",
			in:   `SELECT * FROM t WHERE payload ?| array['a','b'] AND id = ?`,
			want: `SELECT * FROM t WHERE payload ?| array['a','b'] AND id = $1`,
		},
		{
			name: "jsonb question and operator",
			in:   `SELECT * FROM t WHERE payload ?& array['a'] AND id = ?`,
			want: `SELECT * FROM t WHERE payload ?& array['a'] AND id = $1`,
		},
		{
			name: "jsonb hash operator is not a comment",
			in:   `SELECT payload #> '{a}' FROM t WHERE id = ?`,
			want: `SELECT payload #> '{a}' FROM t WHERE id = $1`,
		},
		{
			name: "e-string backslash escape",
			in:   `SELECT E'a\'?b' FROM t WHERE id = ?`,
			want: `SELECT E'a\'?b' FROM t WHERE id = $1`,
		},
		{
			name: "placeholder with cast",
			in:   `SELECT ?::jsonb FROM t WHERE id = ?`,
			want: `SELECT $1::jsonb FROM t WHERE id = $2`,
		},
		{
			name: "multiline sql collapsed outside literals",
			in:   "SELECT *\n  FROM t\n WHERE a = ?\n   AND b = 'x\n?y'",
			want: "SELECT * FROM t WHERE a = $1 AND b = 'x\n?y'",
		},
	}
	for _, c := range cases {
		got := FormatPrepareSQLByStyle(c.in, PlaceholderDollar)
		if got != c.want {
			t.Errorf("%v: FormatPrepareSQLByStyle(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// Test_rewritePlaceholders_styles 三种非问号方言 + 问号方言（不改序号）。
func Test_rewritePlaceholders_styles(t *testing.T) {
	src := `SELECT * FROM t WHERE c = '?' AND a = ? AND b = ?`
	cases := []struct {
		style PlaceholderStyle
		want  string
	}{
		{PlaceholderDollar, `SELECT * FROM t WHERE c = '?' AND a = $1 AND b = $2`},
		{PlaceholderColon, `SELECT * FROM t WHERE c = '?' AND a = :1 AND b = :2`},
		{PlaceholderAtP, `SELECT * FROM t WHERE c = '?' AND a = @p1 AND b = @p2`},
		{PlaceholderQuestion, `SELECT * FROM t WHERE c = '?' AND a = ? AND b = ?`},
	}
	for _, c := range cases {
		got := FormatPrepareSQLByStyle(src, c.style)
		if got != c.want {
			t.Errorf("style %v: got %q, want %q", c.style, got, c.want)
		}
	}
}

// Test_rewritePlaceholders_mysql_hash_comment MySQL 族行注释 # 到行尾（问号方言不重编号）。
func Test_rewritePlaceholders_mysql_hash_comment(t *testing.T) {
	src := "SELECT * FROM t WHERE a = ? # trailing ? comment\nAND b = ?"
	got := FormatPrepareSQLByStyle(src, PlaceholderQuestion)
	want := "SELECT * FROM t WHERE a = ? # trailing ? comment\nAND b = ?"
	if got != want {
		t.Errorf("mysql hash comment: got %q, want %q", got, want)
	}
}
