package types

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

var (
	PrivateParseSqlFunctionType = parseSqlFunctionType
	PrivateGetFormatString      = getFormatString
)

func TestGetShortName(t *testing.T) {
	name := "test.abc.def"
	ret := GetShortName(name)
	if strings.Compare(ret, "def") != 0 {
		t.Error("GetShortName('test.abc.def') not equals 'def' ")
	}
	ret = GetShortName("test.abc")
	if strings.Compare(ret, "abc") != 0 {
		t.Error("GetShortName('test.abc') not equals abc ")
	}
	ret = GetShortName("test")
	if strings.Compare(ret, "test") != 0 {
		t.Error("GetShortName('test') not equals test ")
	}
	ret = GetShortName("test.")
	if len(ret) != 0 {
		t.Error("GetShortName('test.') not equals '' ")
	}
}

func TestToJson(t *testing.T) {
	ret1 := ToJson(map[string]string{"hello": "world"})
	if reflect.TypeOf(ret1).Kind() != reflect.String {
		t.Error("To Json failed ")
	}
	ret2 := ToJson([]string{"aa", "bb"})
	if reflect.TypeOf(ret2).Kind() != reflect.String {
		t.Error("To Json failed ")
	}
	//ret3 := ToJson(time.Now())
	//fmt.Println(ret3)
	//ret4 := ToJson(struct {
	//
	//}{})
	//fmt.Println(ret4)
}

func TestUpperFirst(t *testing.T) {
	ret := UpperFirst("")
	if len(ret) != 0 {
		t.Error("UpperFirst('') not equals '' ")
	}
	ret = UpperFirst("abc")
	if strings.Compare(ret, "Abc") != 0 {
		t.Error("UpperFirst('abc') not equals 'Abc' ")
	}
	ret = UpperFirst("Abc")
	if strings.Compare(ret, "Abc") != 0 {
		t.Error("UpperFirst('Abc') not equals 'Abc' ")
	}
}

func TestParseSqlFunctionType(t *testing.T) {
	if PrivateParseSqlFunctionType("update") != UpdateFunction {
		t.Error("ParseSqlFunctionType('update') not equals 'update' ")
	}
	if PrivateParseSqlFunctionType("Delete") != DeleteFunction {
		t.Error("ParseSqlFunctionType('Delete') not equals 'delete' ")
	}
	if PrivateParseSqlFunctionType("inserT") != InsertFunction {
		t.Error("ParseSqlFunctionType('inserT') not equals 'insert' ")
	}
	if PrivateParseSqlFunctionType("updates") != SelectFunction {
		t.Error("ParseSqlFunctionType('updates') not equals 'select' ")
	}
	if PrivateParseSqlFunctionType("select") != SelectFunction {
		t.Error("ParseSqlFunctionType('update') not equals 'select' ")
	}
}

func TestGetFormatString(t *testing.T) {
	if strings.Compare(PrivateGetFormatString(""), "''") != 0 {
		t.Error("GetFormatString('') not equals '''' ")
	}
	if strings.Compare(PrivateGetFormatString("'"), "'\"'") != 0 {
		t.Error("GetFormatString(''') not equals ''\"'' ")
	}
	if strings.Compare(PrivateGetFormatString("''"), "'\"\"'") != 0 {
		t.Error("GetFormatString('') not equals '''' ")
	}
	if strings.Compare(PrivateGetFormatString("'' AND TEST  ''"), "'\"\" AND TEST  \"\"'") != 0 {
		t.Error("GetFormatString('') not equals '''' ")
	}
	if strings.Compare(PrivateGetFormatString("'A B' CD 'DEF GH'"), "'\"A B\" CD \"DEF GH\"'") != 0 {
		t.Error("GetFormatString('') not equals '''' ")
	}
	if strings.Compare(PrivateGetFormatString("A B' CD 'DEF GH"), "'A B\" CD \"DEF GH'") != 0 {
		t.Error("GetFormatString('') not equals '''' ")
	}
}

func Test_parseSqlFunctionType(t *testing.T) {
	r1 := parseSqlFunctionType("Update")
	if r1 != UpdateFunction {
		t.Error("test parseSqlFunctionType failed.")
	}
	r2 := parseSqlFunctionType("Select")
	if r2 != SelectFunction {
		t.Error("test parseSqlFunctionType failed.")
	}
	r3 := parseSqlFunctionType("INsert")
	if r3 != InsertFunction {
		t.Error("test parseSqlFunctionType failed.")
	}
	r4 := parseSqlFunctionType("DELETE")
	if r4 != DeleteFunction {
		t.Error("test parseSqlFunctionType failed.")
	}
	r5 := parseSqlFunctionType("test")
	if r5 != SelectFunction {
		t.Error("test parseSqlFunctionType failed.")
	}
}

func Test_parseJdbcTypeFrom(t *testing.T) {
	mp := map[string]reflect.Type{
		"VARCHAR": reflect.TypeOf(""), "STRING": reflect.TypeOf(""), "LONGVARCHAR": reflect.TypeOf(""),
		"TIMESTAMP": reflect.TypeOf(time.Now()), "TIME": reflect.TypeOf(time.Now()),
		"INTEGER": reflect.TypeOf(1), "INT": reflect.TypeOf(1),
		"LONG": reflect.TypeOf(int64(1)), "BIGINT": reflect.TypeOf(int64(1)),
		"BOOLEAN": reflect.TypeOf(true), "BIT": reflect.TypeOf(true), "BOOL": reflect.TypeOf(true),
		"DOUBLE": reflect.TypeOf(0.0),
		"test":   reflect.TypeOf(""),
	}
	for k, v := range mp {
		r := ParseJdbcTypeFrom(k)
		if r != v {
			t.Error("test parseJdbcTypeFrom failed.")
		}
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

func Test_buildKey(t *testing.T) {
	if buildKey(" ABC ") != "abc" {
		t.Error("test buildKey failed.")
	}
}
func Test_parseResultTypeFrom(t *testing.T) {
	mp := map[string]reflect.Type{
		"VARCHAR": reflect.TypeOf(""), "STRING": reflect.TypeOf(""), "LONGVARCHAR": reflect.TypeOf(""),
		"TIMESTAMP": reflect.TypeOf(time.Now()), "TIME": reflect.TypeOf(time.Now()),
		"INTEGER": reflect.TypeOf(1), "INT": reflect.TypeOf(1),
		"LONG": reflect.TypeOf(int64(1)), "BIGINT": reflect.TypeOf(int64(1)),
		"BOOLEAN": reflect.TypeOf(true), "BIT": reflect.TypeOf(true), "BOOL": reflect.TypeOf(true),
		"DOUBLE": reflect.TypeOf(0.0),
		"test":   reflect.TypeOf(map[string]interface{}{}),
	}
	for k, v := range mp {
		r := parseResultTypeFrom(k)
		if r != v {
			t.Error("test parseResultTypeFrom failed.")
		}
	}
}
