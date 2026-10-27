package orm

import (
	"testing"

	"github.com/bnulwh/mybatis-go/orm/dialector"
)

// Test_DBFormatSQL_LiteralSafe P0-1：执行路径上的占位符改写必须跳过字面量/注释
// （旧实现按 ? 顺序编号，字面量里的 ? 会让后续参数整体错位）。
func Test_DBFormatSQL_LiteralSafe(t *testing.T) {
	cfg := newDatabaseConfig("postgres", "localhost", 5432, "u", "p", "db")
	d, err := dialector.NewForType(dialector.PostgresDb, cfg)
	if err != nil {
		t.Errorf("NewForType failed: %v", err)
		return
	}
	db := &DB{Config: &Config{Dialector: d}}

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
			name: "jsonb operator",
			in:   `SELECT * FROM t WHERE payload ? 'k' AND id = ?`,
			want: `SELECT * FROM t WHERE payload ? 'k' AND id = $1`,
		},
		{
			name: "line comment keeps newline",
			in:   "SELECT a FROM t -- what?\nWHERE id = ?",
			want: "SELECT a FROM t -- what?\nWHERE id = $1",
		},
		{
			name: "block comment",
			in:   `SELECT /* why? */ a FROM t WHERE id = ?`,
			want: `SELECT /* why? */ a FROM t WHERE id = $1`,
		},
	}
	for _, c := range cases {
		got := db.formatSQL(c.in, []interface{}{1})
		if got != c.want {
			t.Errorf("%v: formatSQL(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}
