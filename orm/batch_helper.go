package orm

import (
	"context"
	"fmt"
	"reflect"
)

func UpdateBatch(ctx context.Context, updateFn func(ctx context.Context, entity interface{}) (int64, error), entities []interface{}, batchSize int) (int64, error) {
	if len(entities) == 0 {
		return 0, nil
	}
	if batchSize <= 0 {
		batchSize = 50
	}
	var total int64
	err := WithTx(ctx, func(txCtx context.Context) error {
		for i := 0; i < len(entities); i += batchSize {
			end := i + batchSize
			if end > len(entities) {
				end = len(entities)
			}
			for j := i; j < end; j++ {
				n, err := updateFn(txCtx, entities[j])
				if err != nil {
					return err
				}
				total += n
			}
		}
		return nil
	})
	return total, err
}

func InsertBatch(ctx context.Context, insertFn interface{}, entities interface{}) (int64, error) {
	fnVal := reflect.ValueOf(insertFn)
	if fnVal.Kind() != reflect.Func {
		return 0, fmt.Errorf("insertFn must be a function")
	}
	sliceVal := reflect.ValueOf(entities)
	if sliceVal.Kind() != reflect.Slice {
		return 0, fmt.Errorf("entities must be a slice")
	}
	length := sliceVal.Len()
	if length == 0 {
		return 0, nil
	}
	args := make([]reflect.Value, 1)
	args[0] = sliceVal
	results := fnVal.Call(args)
	if len(results) >= 2 {
		if !results[1].IsNil() {
			return 0, results[1].Interface().(error)
		}
		return results[0].Int(), nil
	}
	if len(results) == 1 {
		if !results[0].IsNil() {
			return 0, results[0].Interface().(error)
		}
	}
	return 0, nil
}
