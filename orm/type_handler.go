package orm

import (
	"reflect"
	"sync"

	"github.com/bnulwh/mybatis-go/utils"
)

// TypeHandler 自定义类型扫描处理器：DB 值 → Go 目标类型。
// 返回值须可赋值给 targetType 字段；错误沿用 ResultConvertError 聚合机制（M-05）。
type TypeHandler func(source interface{}, targetType reflect.Type) (interface{}, error)

// typeHandlerCache Go 类型 → 处理器注册表（RWMutex 保护）。
var typeHandlerCache = struct {
	mu       sync.RWMutex
	handlers map[reflect.Type]TypeHandler
}{handlers: map[reflect.Type]TypeHandler{}}

// RegisterTypeHandler 按 Go 类型注册自定义扫描处理器（优先于内置 utils.ChangeType 转换）。
func RegisterTypeHandler(typ reflect.Type, handler TypeHandler) {
	typeHandlerCache.mu.Lock()
	defer typeHandlerCache.mu.Unlock()
	typeHandlerCache.handlers[typ] = handler
}

// RegisterTypeHandlerFor 便捷泛型注册：按 T 的类型注册处理器。
func RegisterTypeHandlerFor[T any](fn func(source interface{}) (T, error)) {
	typ := reflect.TypeOf((*T)(nil)).Elem()
	RegisterTypeHandler(typ, func(source interface{}, _ reflect.Type) (interface{}, error) {
		return fn(source)
	})
}

// getTypeHandler 查询已注册处理器；未注册返回 nil。
func getTypeHandler(typ reflect.Type) TypeHandler {
	typeHandlerCache.mu.RLock()
	defer typeHandlerCache.mu.RUnlock()
	return typeHandlerCache.handlers[typ]
}

// convertFieldValue 列值 → 字段类型统一入口：
// 优先已注册的 TypeHandler，未注册回退内置 utils.ChangeType（零行为变化）。
func convertFieldValue(val interface{}, targetType reflect.Type) (interface{}, error) {
	if h := getTypeHandler(targetType); h != nil {
		return h(val, targetType)
	}
	return utils.ChangeType(val, targetType)
}
