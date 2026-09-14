package orm

import (
	"testing"
)

func Test_applyPagination(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		limit  int
		offset int
		want   string
	}{
		{"simple", "SELECT * FROM t", 10, 0, "SELECT * FROM t LIMIT 10"},
		{"with_offset", "SELECT * FROM t", 10, 20, "SELECT * FROM t LIMIT 10 OFFSET 20"},
		{"replace_limit", "SELECT * FROM t LIMIT 5", 10, 0, "SELECT * FROM t LIMIT 10"},
		{"replace_limit_offset", "SELECT * FROM t LIMIT 5 OFFSET 3", 10, 20, "SELECT * FROM t LIMIT 10 OFFSET 20"},
		{"zero_limit", "SELECT * FROM t", 0, 0, "SELECT * FROM t"},
		{"negative_limit", "SELECT * FROM t", -1, 0, "SELECT * FROM t"},
		{"with_where", "SELECT * FROM t WHERE id = ?", 10, 0, "SELECT * FROM t WHERE id = ? LIMIT 10"},
		{"with_order", "SELECT * FROM t ORDER BY id", 10, 5, "SELECT * FROM t ORDER BY id LIMIT 10 OFFSET 5"},
		{"trailing_semicolon", "SELECT * FROM t;", 10, 0, "SELECT * FROM t LIMIT 10"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyPagination(tt.query, tt.limit, tt.offset)
			if got != tt.want {
				t.Errorf("applyPagination(%q, %d, %d) = %q, want %q", tt.query, tt.limit, tt.offset, got, tt.want)
			}
		})
	}
}

func Test_buildCountSQL(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{"simple", "SELECT id, name FROM t", "SELECT COUNT(*) FROM t"},
		{"with_where", "SELECT id FROM t WHERE id > 5", "SELECT COUNT(*) FROM t WHERE id > 5"},
		{"with_order", "SELECT id FROM t ORDER BY id", "SELECT COUNT(*) FROM t"},
		{"with_limit", "SELECT id FROM t LIMIT 10", "SELECT COUNT(*) FROM t"},
		{"with_limit_offset", "SELECT id FROM t LIMIT 10 OFFSET 5", "SELECT COUNT(*) FROM t"},
		{"with_returning", "SELECT id FROM t RETURNING id", "SELECT COUNT(*) FROM t"},
		{"with_for_update", "SELECT id FROM t FOR UPDATE", "SELECT COUNT(*) FROM t"},
		{"distinct", "SELECT DISTINCT name FROM t", "SELECT COUNT(*) FROM t"},
		{"star", "SELECT * FROM t WHERE status = 1", "SELECT COUNT(*) FROM t WHERE status = 1"},
		{"trailing_semicolon", "SELECT id FROM t;", "SELECT COUNT(*) FROM t"},
		{"with_join", "SELECT a.id, b.name FROM a JOIN b ON a.id = b.id WHERE a.status = 1",
			"SELECT COUNT(*) FROM a JOIN b ON a.id = b.id WHERE a.status = 1"},
		{"multiline", "SELECT id\nFROM t\nWHERE id > 0", "SELECT COUNT(*)\nFROM t\nWHERE id > 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCountSQL(tt.query)
			if got != tt.want {
				t.Errorf("buildCountSQL(%q) = %q, want %q", tt.query, got, tt.want)
			}
		})
	}
}
