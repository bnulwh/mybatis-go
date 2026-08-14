package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_PathExists(t *testing.T) {
	dir := t.TempDir()
	ok, err := PathExists(dir)
	if err != nil || !ok {
		t.Errorf("PathExists(existing dir) = %v/%v, want true/nil", ok, err)
	}
	missing := filepath.Join(dir, "not-exist")
	ok, err = PathExists(missing)
	if err != nil || ok {
		t.Errorf("PathExists(missing) = %v/%v, want false/nil", ok, err)
	}
}

func Test_MakeDirAll(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "a", "b")
	if err := MakeDirAll(target); err != nil {
		t.Errorf("MakeDirAll failed: %v", err)
		return
	}
	st, err := os.Stat(target)
	if err != nil || !st.IsDir() {
		t.Errorf("target dir not created: %v %v", st, err)
		return
	}
	// 已存在时幂等
	if err := MakeDirAll(target); err != nil {
		t.Errorf("MakeDirAll idempotent call failed: %v", err)
	}
}
