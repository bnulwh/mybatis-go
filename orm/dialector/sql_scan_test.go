package dialector

import "testing"

// 顶层子句扫描器：定位 / 剥离（B1/B2 修复的基石）。
func Test_TopLevelClausePos(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		keywords []string
		want     int
	}{
		{"top_level_limit", "SELECT * FROM t LIMIT 5", []string{"limit"}, 16},
		{"case_insensitive", "select * from t limit 5", []string{"limit"}, 16},
		{"mixed_case", "SELECT * FROM t Limit 5", []string{"limit"}, 16},
		{"subquery_limit_ignored", "SELECT a, (SELECT b FROM c LIMIT 1) AS x FROM t", []string{"limit"}, -1},
		{"window_order_by_ignored", "SELECT ROW_NUMBER() OVER (ORDER BY id) AS rn FROM t", []string{"order by"}, -1},
		{"string_literal_ignored", "SELECT * FROM t WHERE name = 'LIMIT'", []string{"limit"}, -1},
		{"quoted_ident_ignored", `SELECT "limit" FROM t`, []string{"limit"}, -1},
		{"identifier_with_suffix_ignored", "SELECT limit_x FROM t", []string{"limit"}, -1},
		{"identifier_with_prefix_ignored", "SELECT xlimit FROM t", []string{"limit"}, -1},
		{"line_comment_ignored", "SELECT * FROM t -- LIMIT 5\n", []string{"limit"}, -1},
		{"block_comment_ignored", "SELECT /* LIMIT 99 */ * FROM t LIMIT 5", []string{"limit"}, 31},
		{"order_by_top_level", "SELECT id FROM t WHERE id > 0 ORDER BY id", []string{"order by"}, 30},
		{"order_by_multispace", "SELECT id FROM t ORDER\t\n BY id", []string{"order by"}, 17},
		{"leftmost_wins", "SELECT id FROM t ORDER BY id LIMIT 5", []string{"order by", "limit"}, 17},
		{"for_update", "SELECT id FROM t FOR UPDATE", []string{"for update"}, 17},
		{"for_no_key_update", "SELECT id FROM t FOR NO KEY UPDATE", []string{"for update", "for no key update"}, 17},
		{"for_update_multiline", "SELECT id FROM t WHERE x = 1 FOR\n  UPDATE", []string{"for update"}, 29},
		{"offset_only", "SELECT * FROM t OFFSET 10", []string{"limit", "offset"}, 16},
		{"union_paren_limit_ignored", "(SELECT a FROM x LIMIT 1) UNION (SELECT b FROM y LIMIT 2) LIMIT 10", []string{"limit"}, 58},
		{"empty_sql", "", []string{"limit"}, -1},
		{"empty_keywords", "SELECT 1", nil, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TopLevelClausePos(tt.sql, tt.keywords...)
			if got != tt.want {
				t.Errorf("TopLevelClausePos(%q, %v) = %d, want %d", tt.sql, tt.keywords, got, tt.want)
			}
		})
	}
}

// 剥离顶层尾部子句：截断点正确、尾部空白/分号清理。
func Test_StripTopLevelTailClauses(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		keywords []string
		want     string
	}{
		{"strip_limit", "SELECT * FROM t LIMIT 5", []string{"limit"}, "SELECT * FROM t"},
		{"strip_limit_offset", "SELECT * FROM t LIMIT 5 OFFSET 3", []string{"limit", "offset"}, "SELECT * FROM t"},
		{"strip_chain_from_order", "SELECT id FROM t WHERE x = 1 ORDER BY a, b LIMIT 5", []string{"order by", "limit", "offset"}, "SELECT id FROM t WHERE x = 1"},
		{"keep_subquery", "SELECT a FROM (SELECT b FROM c ORDER BY b LIMIT 1) t", []string{"order by", "limit"}, "SELECT a FROM (SELECT b FROM c ORDER BY b LIMIT 1) t"},
		{"strip_returning", "INSERT INTO t (a) VALUES (1) RETURNING id", []string{"returning"}, "INSERT INTO t (a) VALUES (1)"},
		{"no_match_unchanged", "SELECT * FROM t", []string{"limit"}, "SELECT * FROM t"},
		{"trailing_semicolon", "SELECT * FROM t LIMIT 5;", []string{"limit"}, "SELECT * FROM t"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripTopLevelTailClauses(tt.sql, tt.keywords...)
			if got != tt.want {
				t.Errorf("StripTopLevelTailClauses(%q, %v) = %q, want %q", tt.sql, tt.keywords, got, tt.want)
			}
		})
	}
}

func Test_HasTopLevelClause(t *testing.T) {
	if !HasTopLevelClause("SELECT * FROM t LIMIT 5", "limit") {
		t.Error("top-level LIMIT should be detected")
	}
	if HasTopLevelClause("SELECT (SELECT b FROM c LIMIT 1) AS x FROM t", "limit") {
		t.Error("subquery LIMIT should not be detected")
	}
}

func Test_TopLevelKeywordEnd(t *testing.T) {
	got := TopLevelKeywordEnd("SELECT  id FROM t", "select")
	if got != 8 {
		t.Errorf("TopLevelKeywordEnd = %d, want 8 (keyword + trailing whitespace)", got)
	}
	if got := TopLevelKeywordEnd("INSERT INTO t VALUES (1)", "select"); got != -1 {
		t.Errorf("missing keyword should return -1, got %d", got)
	}
}

func Test_ContainsConcatOp(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"plain concat", "SELECT a || b FROM t", true},
		{"inside single quote", "SELECT 'a || b' FROM t", false},
		{"escaped quote then concat", "SELECT 'it''s' || a FROM t", true},
		{"line comment", "SELECT a FROM t -- x || y", false},
		{"block comment", "SELECT /* x || y */ a FROM t", false},
		{"after comment concat", "SELECT a -- c\n || b FROM t", true},
		{"single pipe", "SELECT a | b FROM t", false},
		{"no pipe", "SELECT a FROM t", false},
		{"double quote ident", `SELECT "a||b" FROM t`, false},
	}
	for _, c := range cases {
		if got := ContainsConcatOp(c.in); got != c.want {
			t.Errorf("%s: ContainsConcatOp(%q) = %v, want %v", c.name, c.in, got, c.want)
		}
	}
}
