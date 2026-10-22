package orm

import (
	"fmt"
	"reflect"
)

// D1：NewMapper 的类型安全 / error / 泛型版本。
//
// 背景：NewMapper(name) 返回 interface{}，未注册时 panic，调用方 120 处裸类型断言
// （`.（XMapper)`，无 ok 检查）——类型漂移即 panic，且 panic 混在业务调用栈里难以定位。
// 本组 API 把「构建失败 / 类型不匹配」变成显式 error；bindMapper 内部的绑定 panic
// 同样转为 error 返回（不再击穿调用栈）。

// buildMapper 按名称构建并绑定 Mapper 实例（D1 内部共用）；bindMapper 的绑定
// panic 转为 error 返回。
func buildMapper(name string) (mp reflect.Value, val interface{}, err error) {
	defer func() {
		if p := recover(); p != nil {
			mp, val = reflect.Value{}, nil
			err = fmt.Errorf("create mapper %q failed: %v", name, p)
		}
	}()
	mp, err = gCache.createMapper(name)
	if err != nil {
		return reflect.Value{}, nil, err
	}
	bindMapper(name, mp)
	return mp, reflect.Indirect(mp).Interface(), nil
}

// Mapper 按名称取已注册的 Mapper（D1：类型安全 + error）。
//
//	T 为 Mapper 结构体值类型：mp, err := orm.Mapper[mapper.SysUserMapper]("SysUserMapper")
//
// T 也可为指针类型（*XMapper）或该结构体实现的接口类型；未注册 / XML 绑定失败 /
// 类型不匹配返回 error 而非 panic。
func Mapper[T any](name string) (T, error) {
	var zero T
	mp, val, err := buildMapper(name)
	if err != nil {
		return zero, err
	}
	if v, ok := val.(T); ok {
		return v, nil
	}
	// 指针形态 *XMapper
	if v, ok := mp.Interface().(T); ok {
		return v, nil
	}
	return zero, fmt.Errorf("mapper %q is %T, not assignable to %T", name, val, zero)
}

// MustMapper 与 Mapper 相同但失败时 panic（D1：显式命名，适合 init 期取用，
// 失败即快速失败且错误信息含 mapper 名）。
func MustMapper[T any](name string) T {
	v, err := Mapper[T](name)
	if err != nil {
		panic(err)
	}
	return v
}

// NewMapperE NewMapper 的 error 版本（D1）：未注册 / XML 绑定失败返回 error
// 而非 panic；返回 interface{}，类型断言仍由调用方负责（需要类型安全请用 Mapper[T]）。
func NewMapperE(name string) (interface{}, error) {
	return Mapper[interface{}](name)
}
