package orm

import (
	"testing"
)

// M-03：keyColumnToSnake 驼峰转下划线列名
func Test_keyColumnToSnake(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"id", "id"},
		{"jobId", "job_id"},
		{"createUserId", "create_user_id"},
		{"Name", "name"},
		{"CreateTime", "create_time"},
		{"", ""},
		{"A", "a"},
		{"userID", "user_i_d"},
	}
	for _, tt := range tests {
		if got := keyColumnToSnake(tt.in); got != tt.want {
			t.Errorf("keyColumnToSnake(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// M-03：toInt64 各种数值类型转 int64
func Test_toInt64(t *testing.T) {
	tests := []struct {
		in      interface{}
		want    int64
		wantOk  bool
	}{
		{int64(42), 42, true},
		{int(42), 42, true},
		{int32(42), 42, true},
		{int16(42), 42, true},
		{int8(42), 42, true},
		{uint(42), 42, true},
		{uint64(42), 42, true},
		{uint32(42), 42, true},
		{uint16(42), 42, true},
		{uint8(42), 42, true},
		{float64(42), 42, true},
		{float32(42), 42, true},
		{"not a number", 0, false},
	}
	for _, tt := range tests {
		got, ok := toInt64(tt.in)
		if ok != tt.wantOk || got != tt.want {
			t.Errorf("toInt64(%v) = (%v, %v), want (%v, %v)", tt.in, got, ok, tt.want, tt.wantOk)
		}
	}
}

// M-03：needsReturning 在未初始化时返回 false
func Test_needsReturning_noDb(t *testing.T) {
	old := gDbConn
	gDbConn = nil
	defer func() { gDbConn = old }()

	if needsReturning() {
		t.Error("needsReturning() should return false when gDbConn is nil")
	}
}
