package sqlfragment

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// 本文件用例自 types 包迁移（P0-3a）：validValue/formatValue/buildKey/parseJdbcTypeFrom
// 为 sqlfragment 私有副本，行为与 types 原版一致。

func TestGetFormatString(t *testing.T) {
	if strings.Compare(formatString(""), "''") != 0 {
		t.Error("GetFormatString('') not equals '''' ")
	}
	if strings.Compare(formatString("'"), "'\"'") != 0 {
		t.Error("GetFormatString(''') not equals ''\"'' ")
	}
	if strings.Compare(formatString("''"), "'\"\"'") != 0 {
		t.Error("GetFormatString('') not equals '''' ")
	}
	if strings.Compare(formatString("'' AND TEST  ''"), "'\"\" AND TEST  \"\"'") != 0 {
		t.Error("GetFormatString('') not equals '''' ")
	}
	if strings.Compare(formatString("'A B' CD 'DEF GH'"), "'\"A B\" CD \"DEF GH\"'") != 0 {
		t.Error("GetFormatString('') not equals '''' ")
	}
	if strings.Compare(formatString("A B' CD 'DEF GH"), "'A B\" CD \"DEF GH'") != 0 {
		t.Error("GetFormatString('') not equals '''' ")
	}
}

func Test_validValue(t *testing.T) {
	if validValue("") {
		t.Error("test validValue failed.")
	}
	if !validValue("s") {
		t.Error("test validValue failed.")
	}
	if !validValue(true) {
		t.Error("test validValue failed.")
	}
	if !validValue(false) {
		t.Error("test validValue failed.")
	}
	if !validValue(1) {
		t.Error("test validValue failed.")
	}
	if !validValue(int8(1)) {
		t.Error("test validValue failed.")
	}
	if !validValue(int16(1)) {
		t.Error("test validValue failed.")
	}
	if !validValue(int32(1)) {
		t.Error("test validValue failed.")
	}
	if !validValue(int64(1)) {
		t.Error("test validValue failed.")
	}
	if !validValue(uint(1)) {
		t.Error("test validValue failed.")
	}
	if !validValue(uint8(1)) {
		t.Error("test validValue failed.")
	}
	if !validValue(uint16(1)) {
		t.Error("test validValue failed.")
	}
	if !validValue(uint32(1)) {
		t.Error("test validValue failed.")
	}
	if !validValue(uint64(1)) {
		t.Error("test validValue failed.")
	}
	if !validValue(0.0) {
		t.Error("test validValue failed.")
	}
	if !validValue(float64(0.0)) {
		t.Error("test validValue failed.")
	}
	if !validValue(time.Now()) {
		t.Error("test validValue failed.")
	}
	if validValue(time.Time{}) {
		t.Error("test validValue failed.")
	}
	if !validValue([]string{"aaa"}) {
		t.Error("test validValue failed.")
	}
	if validValue([]string{}) {
		t.Error("test validValue failed.")
	}
	if !validValue(map[string]string{"aaa": "bbb"}) {
		t.Error("test validValue failed.")
	}
	if validValue(map[string]string{}) {
		t.Error("test validValue failed.")
	}

}

// Test_ValidValue_Nil S-09：nil 参数返回 false 而非反射零值 panic。
func Test_ValidValue_Nil(t *testing.T) {
	if validValue(nil) {
		t.Error("validValue(nil) should be false")
	}
}

func Test_buildKey(t *testing.T) {
	if buildKey(" ABC ") != "abc" {
		t.Error("test buildKey failed.")
	}
}

func Test_parseJdbcTypeFrom(t *testing.T) {
	mp := map[string]reflect.Type{
		"VARCHAR": reflect.TypeOf(""), "STRING": reflect.TypeOf(""), "LONGVARCHAR": reflect.TypeOf(""),
		"TIMESTAMP": reflect.TypeOf(time.Now()), "TIME": reflect.TypeOf(time.Now()), "DATETIME": reflect.TypeOf(time.Now()),
		"INTEGER": reflect.TypeOf(1), "INT": reflect.TypeOf(1), "TINYINT": reflect.TypeOf(1), "SMALLINT": reflect.TypeOf(1),
		"LONG": reflect.TypeOf(int64(1)), "BIGINT": reflect.TypeOf(int64(1)),
		"BOOLEAN": reflect.TypeOf(true), "BIT": reflect.TypeOf(true), "BOOL": reflect.TypeOf(true), "ENUM": reflect.TypeOf(true),
		"DOUBLE": reflect.TypeOf(0.0), "FLOAT": reflect.TypeOf(0.0), "NUMERIC": reflect.TypeOf(0.0),
		"TEXT": reflect.TypeOf(""), "CHAR": reflect.TypeOf(""),
		"test": reflect.TypeOf(""),
	}
	for k, v := range mp {
		r := parseJdbcTypeFrom(k)
		if r != v {
			t.Error("test parseJdbcTypeFrom failed.")
		}
	}
}
