package sqlfragment

import (
	"testing"
)

// Test_ContainsForEach 递归检测片段树中的 <foreach>（含 if/include/where 嵌套）。
func Test_ContainsForEach(t *testing.T) {
	if ContainsForEach(nil) {
		t.Error("nil items should not contain foreach")
	}
	if ContainsForEach([]Node{&simpleSql{}}) {
		t.Error("plain sql should not contain foreach")
	}
	// 直接 foreach
	if !ContainsForEach([]Node{&sqlForLoop{}}) {
		t.Error("direct foreach not detected")
	}
	// if 嵌套 foreach
	if !ContainsForEach([]Node{&sqlIfTest{Sql: []Node{&sqlForLoop{}}}}) {
		t.Error("foreach inside if not detected")
	}
	// include 嵌套 foreach
	if !ContainsForEach([]Node{&sqlInclude{Fragments: []Node{&sqlForLoop{}}}}) {
		t.Error("foreach inside include not detected")
	}
	// where 嵌套 foreach
	if !ContainsForEach([]Node{&sqlWhere{Sql: []Node{&sqlForLoop{}}}}) {
		t.Error("foreach inside where not detected")
	}
	// 无 foreach
	if ContainsForEach([]Node{&sqlIfTest{Sql: []Node{&simpleSql{Sql: "x"}}}}) {
		t.Error("nested plain sql should not contain foreach")
	}
}
