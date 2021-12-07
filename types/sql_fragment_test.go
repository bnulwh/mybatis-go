package types

import (
	"fmt"
	"reflect"
	"testing"
)

func Test_parseSimpleSqlFromText(t *testing.T) {
	r := parseSimpleSqlFromText("from user_info")
	fmt.Println(r)
	if len(r.Params) != 0 {
		t.Error("test parseSimpleSqlFromText failed.")
	}
	r1 := parseSimpleSqlFromText("from user_info where id = #{id,jdbcType=INTEGER}")
	fmt.Println(r1)
	if len(r1.Params) != 1 {
		t.Error("test parseSimpleSqlFromText failed.")
	}
	if r1.Params[0].Name != "id" {
		t.Error("test parseSimpleSqlFromText failed.")
	}
	if r1.Params[0].Type != reflect.TypeOf(1) {
		t.Error("test parseSimpleSqlFromText failed.")
	}
}
