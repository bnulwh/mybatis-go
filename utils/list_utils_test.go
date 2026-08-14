package utils

import "testing"

func Test_List2map(t *testing.T) {
	mp := List2map([]string{"a", "b", "c"})
	if len(mp) != 3 || mp["a"] != "a" || mp["b"] != "b" || mp["c"] != "c" {
		t.Errorf("List2map wrong: %v", mp)
	}
	if empty := List2map(nil); len(empty) != 0 {
		t.Errorf("List2map(nil) should be empty, got %v", empty)
	}
	if empty := List2map([]string{}); len(empty) != 0 {
		t.Errorf("List2map(empty) should be empty, got %v", empty)
	}
}
