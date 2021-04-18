package types

import (
	"reflect"
	"strings"
	"testing"
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

func TestGetFormatString(t *testing.T)  {
	if strings.Compare(PrivateGetFormatString(""),"''")!=0{
		t.Error("GetFormatString('') not equals '''' ")
	}
	if strings.Compare(PrivateGetFormatString("'"),"'\"'")!=0{
		t.Error("GetFormatString(''') not equals ''\"'' ")
	}
	if strings.Compare(PrivateGetFormatString("''"),"'\"\"'")!=0{
		t.Error("GetFormatString('') not equals '''' ")
	}
	if strings.Compare(PrivateGetFormatString("'' AND TEST  ''"),"'\"\" AND TEST  \"\"'")!=0{
		t.Error("GetFormatString('') not equals '''' ")
	}
	if strings.Compare(PrivateGetFormatString("'A B' CD 'DEF GH'"),"'\"A B\" CD \"DEF GH\"'")!=0{
		t.Error("GetFormatString('') not equals '''' ")
	}
	if strings.Compare(PrivateGetFormatString("A B' CD 'DEF GH"),"'A B\" CD \"DEF GH'")!=0{
		t.Error("GetFormatString('') not equals '''' ")
	}
}