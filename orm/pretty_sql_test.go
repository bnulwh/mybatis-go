package orm

import (
	"database/sql"
	"testing"
	"time"

	"github.com/bnulwh/mybatis-go/orm/dialector"
)

func Test_FormatArg(t *testing.T) {
	now := time.Date(2026, 9, 20, 10, 30, 0, 0, time.Local)
	tests := []struct {
		input  interface{}
		expect string
	}{
		{nil, "NULL"},
		{"hello", "'hello'"},
		{"it's", "'it''s'"},
		{42, "42"},
		{3.14, "3.14"},
		{true, "true"},
		{false, "false"},
		{now, "'2026-09-20 10:30:00'"},
		{&now, "'2026-09-20 10:30:00'"},
		{(*time.Time)(nil), "NULL"},
		{sql.NullString{String: "test", Valid: true}, "'test'"},
		{sql.NullString{Valid: false}, "NULL"},
		{sql.NullInt64{Int64: 99, Valid: true}, "99"},
		{sql.NullInt64{Valid: false}, "NULL"},
		{sql.NullBool{Bool: true, Valid: true}, "true"},
		{sql.NullBool{Valid: false}, "NULL"},
		{sql.NullTime{Time: now, Valid: true}, "'2026-09-20 10:30:00'"},
		{sql.NullTime{Valid: false}, "NULL"},
		{[]byte{0xAB, 0xCD}, "x'abcd'"},
	}
	for _, tt := range tests {
		got := formatArg(tt.input)
		if got != tt.expect {
			t.Errorf("formatArg(%v) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func Test_FormatSQLWithArgs_QuestionMark(t *testing.T) {
	sql := "SELECT * FROM user WHERE id = ? AND name = ?"
	args := []interface{}{1, "alice"}
	got := FormatSQLWithArgs(sql, args, dialector.PlaceholderQuestion)
	expect := "SELECT * FROM user WHERE id = 1 AND name = 'alice'"
	if got != expect {
		t.Errorf("got %q, want %q", got, expect)
	}
}

func Test_FormatSQLWithArgs_Dollar(t *testing.T) {
	sql := "SELECT * FROM user WHERE id = $1 AND name = $2"
	args := []interface{}{1, "alice"}
	got := FormatSQLWithArgs(sql, args, dialector.PlaceholderDollar)
	expect := "SELECT * FROM user WHERE id = 1 AND name = 'alice'"
	if got != expect {
		t.Errorf("got %q, want %q", got, expect)
	}
}

func Test_FormatSQLWithArgs_Colon(t *testing.T) {
	sql := "SELECT * FROM user WHERE id = :1 AND name = :2"
	args := []interface{}{1, "alice"}
	got := FormatSQLWithArgs(sql, args, dialector.PlaceholderColon)
	expect := "SELECT * FROM user WHERE id = 1 AND name = 'alice'"
	if got != expect {
		t.Errorf("got %q, want %q", got, expect)
	}
}

func Test_FormatSQLWithArgs_AtP(t *testing.T) {
	sql := "SELECT * FROM user WHERE id = @p1 AND name = @p2"
	args := []interface{}{1, "alice"}
	got := FormatSQLWithArgs(sql, args, dialector.PlaceholderAtP)
	expect := "SELECT * FROM user WHERE id = 1 AND name = 'alice'"
	if got != expect {
		t.Errorf("got %q, want %q", got, expect)
	}
}

func Test_FormatSQLWithArgs_NoArgs(t *testing.T) {
	sql := "SELECT 1"
	got := FormatSQLWithArgs(sql, nil, dialector.PlaceholderQuestion)
	if got != sql {
		t.Errorf("got %q, want %q", got, sql)
	}
}

func Test_FormatSQLWithArgs_NilArg(t *testing.T) {
	sql := "UPDATE user SET name = ? WHERE id = ?"
	args := []interface{}{nil, 1}
	got := FormatSQLWithArgs(sql, args, dialector.PlaceholderQuestion)
	expect := "UPDATE user SET name = NULL WHERE id = 1"
	if got != expect {
		t.Errorf("got %q, want %q", got, expect)
	}
}

func Test_PrettySQL_Select(t *testing.T) {
	input := "SELECT id, name FROM user WHERE id = 1 AND name = 'alice' ORDER BY id LIMIT 10"
	got := PrettySQL(input)
	lines := splitLines(got)
	if len(lines) < 5 {
		t.Errorf("expected >= 5 lines, got %d:\n%s", len(lines), got)
	}
	if lines[0] != "SELECT id, name " {
		t.Errorf("first line = %q, want %q", lines[0], "SELECT id, name ")
	}
	hasAND := false
	for _, l := range lines {
		if l == "  AND name = 'alice' " {
			hasAND = true
		}
	}
	if !hasAND {
		t.Errorf("AND not indented under WHERE:\n%s", got)
	}
}

func Test_PrettySQL_Insert(t *testing.T) {
	input := "INSERT INTO user (id, name) VALUES (1, 'alice')"
	got := PrettySQL(input)
	if !containsLine(got, "INSERT INTO user (id, name) ") {
		t.Errorf("missing INSERT INTO line:\n%s", got)
	}
	if !containsLine(got, "VALUES (1, 'alice')") {
		t.Errorf("missing VALUES line:\n%s", got)
	}
}

func Test_PrettySQL_Update(t *testing.T) {
	input := "UPDATE user SET name = 'bob' WHERE id = 1"
	got := PrettySQL(input)
	if !containsLine(got, "UPDATE user ") {
		t.Errorf("missing UPDATE line:\n%s", got)
	}
	if !containsLine(got, "SET name = 'bob' ") {
		t.Errorf("missing SET line:\n%s", got)
	}
}

func Test_PrettySQL_Delete(t *testing.T) {
	input := "DELETE FROM user WHERE id = 1"
	got := PrettySQL(input)
	if !containsLine(got, "DELETE FROM user ") {
		t.Errorf("missing DELETE FROM line:\n%s", got)
	}
}

func Test_PrettySQL_Join(t *testing.T) {
	input := "SELECT u.id, d.name FROM user u LEFT JOIN dept d ON u.dept_id = d.id WHERE u.id = 1"
	got := PrettySQL(input)
	if !containsLine(got, "LEFT JOIN dept d ") {
		t.Errorf("missing LEFT JOIN line:\n%s", got)
	}
	if !containsLine(got, "ON u.dept_id = d.id ") {
		t.Errorf("missing ON line:\n%s", got)
	}
}

func Test_PrettySQL_Empty(t *testing.T) {
	got := PrettySQL("")
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func Test_PrettySQL_Config(t *testing.T) {
	SetPrettySQL(true)
	if !PrettySQLEnabled() {
		t.Error("expected PrettySQLEnabled = true")
	}
	SetPrettySQL(false)
	if PrettySQLEnabled() {
		t.Error("expected PrettySQLEnabled = false")
	}
}

func Test_FormatSQLForLog_Disabled(t *testing.T) {
	SetPrettySQL(false)
	sqlStr := "SELECT * FROM user WHERE id = ?"
	args := []interface{}{1}
	got := formatSQLForLog(sqlStr, args)
	if got != sqlStr {
		t.Errorf("when disabled, got %q, want %q", got, sqlStr)
	}
}

func Test_FormatSQLForLog_Enabled(t *testing.T) {
	SetPrettySQL(true)
	defer SetPrettySQL(false)
	sqlStr := "SELECT id, name FROM user WHERE id = ? AND name = ? ORDER BY id"
	args := []interface{}{1, "alice"}
	got := formatSQLForLog(sqlStr, args)
	lines := splitLines(got)
	if len(lines) < 3 {
		t.Errorf("expected formatted multi-line output, got:\n%s", got)
	}
	if lines[0] != "SELECT id, name " {
		t.Errorf("first line = %q, want %q", lines[0], "SELECT id, name ")
	}
	hasAND := false
	for _, l := range lines {
		if l == "  AND name = 'alice' " {
			hasAND = true
		}
	}
	if !hasAND {
		t.Errorf("AND not indented under WHERE:\n%s", got)
	}
}

func splitLines(s string) []string {
	var lines []string
	cur := ""
	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, cur)
			cur = ""
		} else {
			cur += string(ch)
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

func containsLine(s, sub string) bool {
	for _, line := range splitLines(s) {
		if line == sub {
			return true
		}
	}
	return false
}
