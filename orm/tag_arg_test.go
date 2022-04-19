package orm

import "testing"

func Test_parseTagArgs(t *testing.T) {
	r0 := parseTagArgs("")
	if len(r0) != 0 {
		t.Error("test parseTagArgs failed")
	}
	r1 := parseTagArgs(`json:",inline"`)
	if len(r1) == 0 {
		t.Error("test parseTagArgs failed")
	}
}
