package orm

import (
	"reflect"
	"testing"
)

// P3-1：核心代理机制单测（不依赖数据库）

type proxyUnitMapper struct {
	Add  func(a, b int) (int, error)
	Echo func(s string) (string, error) `args:"name"`
}

// sumProxy 构造代理函数：Add 直接求和，Echo 验证 tag args 映射
func sumProxy() func(field reflect.StructField, fieldVal reflect.Value) func(arg ProxyArg) []reflect.Value {
	return func(field reflect.StructField, fieldVal reflect.Value) func(arg ProxyArg) []reflect.Value {
		switch field.Name {
		case "Add":
			return func(arg ProxyArg) []reflect.Value {
				args := arg.buildArgs()
				a := args[0].(int)
				b := args[1].(int)
				return []reflect.Value{reflect.ValueOf(a + b), reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())}
			}
		case "Echo":
			return func(arg ProxyArg) []reflect.Value {
				args := arg.buildArgs()
				// tag `args:name` → 参数以 map 形式传入
				mp := args[0].(map[string]interface{})
				s := mp["name"].(string)
				return []reflect.Value{reflect.ValueOf(s), reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())}
			}
		}
		return nil
	}
}

func Test_proxyValueInjectsAndCalls(t *testing.T) {
	m := &proxyUnitMapper{}
	proxyValue(reflect.ValueOf(m), sumProxy())
	if m.Add == nil || m.Echo == nil {
		t.Fatal("proxy funcs were not injected")
	}
	got, err := m.Add(2, 3)
	if err != nil || got != 5 {
		t.Errorf("Add(2,3) = %d/%v, want 5/nil", got, err)
	}
	echo, err := m.Echo("hello")
	if err != nil || echo != "hello" {
		t.Errorf("Echo(hello) = %q/%v, want hello/nil", echo, err)
	}
}

// 兼容 tag 解析：标准 args:"name" 与文档 legacy args:name 两种写法
func Test_getTagArgNames(t *testing.T) {
	if got := getTagArgNames(reflect.StructTag(`args:"name,other"`)); got != "name,other" {
		t.Errorf("quoted tag = %q", got)
	}
	if got := getTagArgNames(reflect.StructTag(`args:name`)); got != "name" {
		t.Errorf("legacy unquoted tag = %q", got)
	}
	if got := getTagArgNames(reflect.StructTag(`json:",inline"`)); got != "" {
		t.Errorf("unrelated tag should be empty, got %q", got)
	}
}

func Test_proxyPanicsOnNonPointer(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("proxy() should panic on non-pointer")
		}
	}()
	proxy(proxyUnitMapper{}, sumProxy())
}

type badTagMapper struct {
	Bad func(a, b int) (int, error) `args:"x"`
}

func Test_proxyValuePanicsOnTagMismatch(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("proxyValue should panic when args tag length != params")
		}
	}()
	proxyValue(reflect.ValueOf(&badTagMapper{}), sumProxy())
}

func Test_makeReturnType(t *testing.T) {
	// func() (int, error)
	rt := makeReturnType("f", reflect.TypeOf(func() (int, error) { return 0, nil }))
	if rt.NumOut != 2 || rt.ReturnIndex != 0 || rt.ErrorType == nil {
		t.Errorf("two-out return type wrong: %+v", rt)
	}
	// func() error
	rt2 := makeReturnType("f", reflect.TypeOf(func() error { return nil }))
	if rt2.NumOut != 1 || rt2.ReturnIndex != -1 || rt2.ErrorType == nil {
		t.Errorf("error-only return type wrong: %+v", rt2)
	}
}

func Test_makeReturnTypePanics(t *testing.T) {
	bad := []reflect.Type{
		reflect.TypeOf(func() {}),                                   // 0 返回
		reflect.TypeOf(func() (int, int, error) { return 0, 0, nil }), // 3 返回
		reflect.TypeOf(func() int { return 0 }),                     // 缺 error
		reflect.TypeOf(func() (*int, error) { return nil, nil }),    // ptr 返回
	}
	for _, bt := range bad {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("makeReturnType should panic for %v", bt)
				}
			}()
			makeReturnType("f", bt)
		}()
	}
}

func Test_makeParamType(t *testing.T) {
	pt := makeParamType("f", reflect.TypeOf(func() {}), reflect.StructTag(""))
	if pt.ArgsLen != 0 || pt.TagArgsLen != 0 {
		t.Errorf("no-arg param type wrong: %+v", pt)
	}
	pt2 := makeParamType("f", reflect.TypeOf(func(a int) {}), reflect.StructTag(`args:a`))
	if pt2.ArgsLen != 1 || pt2.TagArgsLen != 1 {
		t.Errorf("tagged param type wrong: %+v", pt2)
	}
	// tag 长度与参数个数不一致 → panic
	func() {
		defer func() {
			if recover() == nil {
				t.Error("makeParamType should panic when tag length != params")
			}
		}()
		makeParamType("f", reflect.TypeOf(func(a, b int) {}), reflect.StructTag(`args:x`))
	}()
	func() {
		defer func() {
			if recover() == nil {
				t.Error("makeParamType should panic when tag length > params")
			}
		}()
		makeParamType("f", reflect.TypeOf(func(a int) {}), reflect.StructTag(`args:x,y`))
	}()
}

func Test_methodFieldCheck(t *testing.T) {
	typ := reflect.TypeOf(func() (int, error) { return 0, nil })
	sf := reflect.StructField{Name: "F", Type: typ}
	methodFieldCheck(&typ, &sf, false) // 合法，不 panic
	// 缺 error 返回 → panic
	badTyp := reflect.TypeOf(func() int { return 0 })
	badSf := reflect.StructField{Name: "F", Type: badTyp}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("methodFieldCheck should panic without error return")
			}
		}()
		methodFieldCheck(&badTyp, &badSf, false)
	}()
}
