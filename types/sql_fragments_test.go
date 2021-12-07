package types

import (
	"fmt"
	"testing"
)

func Test_parseIfConditionsFromText(t *testing.T) {
	r1 := parseIfConditionsFromText("name != null")
	fmt.Println(r1)
	if len(r1) == 0 {
		t.Error("test parseIfConditionsFromText failed.")
	}
	if r1[0].CheckName != "name" {
		t.Error("test parseIfConditionsFromText failed.")
	}
	if r1[0].CheckType != nullCheckCond {
		t.Error("test parseIfConditionsFromText failed.")
	}
	r2 := parseIfConditionsFromText("name != null and name != '' ")
	fmt.Println(r1)
	if len(r2) != 2 {
		t.Error("test parseIfConditionsFromText failed.")
	}
	if r2[1].CheckName != "name" {
		t.Error("test parseIfConditionsFromText failed.")
	}
	if r2[1].CheckType != emptyCheckCond {
		t.Error("test parseIfConditionsFromText failed.")
	}
}
