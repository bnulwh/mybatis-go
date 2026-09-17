package orm

import (
	"context"
	"reflect"
	"sync"

	"github.com/bnulwh/mybatis-go/log"
)

type FillHandler func(ctx context.Context, field *FieldInfo) interface{}

var (
	fillHandlersMu sync.RWMutex
	fillHandlers   = map[string]map[string]FillHandler{}
)

func RegisterFillHandler(column string, when string, handler FillHandler) {
	if column == "" || when == "" || handler == nil {
		return
	}
	fillHandlersMu.Lock()
	defer fillHandlersMu.Unlock()
	if fillHandlers[column] == nil {
		fillHandlers[column] = map[string]FillHandler{}
	}
	fillHandlers[column][when] = handler
}

func runFillHandlers(ctx context.Context, sqlType string, arg ProxyArg) {
	if arg.ArgsLen == 0 {
		return
	}
	fillHandlersMu.RLock()
	handlers := fillHandlers
	fillHandlersMu.RUnlock()
	if len(handlers) == 0 {
		return
	}
	val := arg.Args[0]
	if !val.IsValid() {
		return
	}
	iv := reflect.Indirect(val)
	if iv.Kind() != reflect.Struct {
		return
	}
	info := GetModelInfo(iv.Interface())
	if info == nil {
		return
	}
	when := ""
	switch sqlType {
	case "insert":
		when = FillInsert
	case "update":
		when = FillUpdate
	default:
		return
	}
	for _, fi := range info.FillColumns(when) {
		colHandlers, ok := handlers[fi.Column]
		if !ok {
			continue
		}
		var handler FillHandler
		if h, ok := colHandlers[when]; ok {
			handler = h
		} else if h, ok := colHandlers[FillInsertUpdate]; ok {
			handler = h
		}
		if handler == nil {
			continue
		}
		field := iv.FieldByName(fi.Name)
		if !field.IsValid() || !field.CanSet() {
			continue
		}
		result := handler(ctx, fi)
		if result == nil {
			continue
		}
		rval := reflect.ValueOf(result)
		if rval.Type().ConvertibleTo(field.Type()) {
			field.Set(rval.Convert(field.Type()))
		} else {
			log.Warnf("fill handler for %s: cannot convert %T to %v", fi.Column, result, field.Type())
		}
	}
}
