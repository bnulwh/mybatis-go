package orm

import (
	"context"
	"sync"
	"time"

	"github.com/bnulwh/mybatis-go/log"
)

// Hook 拦截链（P0-5）：乐观锁 / 多租户 / 自动填充 / SQL 审计等横切能力的基础设施。
//
// 注册期后 hooks 视为只读；钩子函数自身必须并发安全（可能被多 goroutine 并发调用）。
// Before 钩子在 SQL 生成后、执行前触发，可改写 ctx.SQL（执行器取改写后的值）；
// After 钩子在执行完毕后触发，可见 Error 与 Duration。

// HookPoint 钩子挂载点。
type HookPoint int

const (
	// HookBeforeExecute SQL 生成后、执行前；可改写 ctx.SQL。
	HookBeforeExecute HookPoint = iota
	// HookAfterExecute 执行后；可见 Error 与 Duration。
	HookAfterExecute
)

// HookContext 钩子上下文。
type HookContext struct {
	Ctx       context.Context
	Namespace string
	SqlId     string
	SqlType   string // "select" / "insert" / "update" / "delete"
	SQL       string // Before 钩子可修改，执行器取修改后的值
	Args      []interface{}
	Error     error         // After 专有
	Duration  time.Duration // After 专有
}

// Hook 钩子函数。
type Hook func(*HookContext)

var (
	hookMu      sync.RWMutex
	beforeHooks []Hook
	afterHooks  []Hook
)

// RegisterHook 注册钩子（按挂载点追加，注册期后视为只读）。
func RegisterHook(point HookPoint, hook Hook) {
	if hook == nil {
		return
	}
	hookMu.Lock()
	defer hookMu.Unlock()
	switch point {
	case HookBeforeExecute:
		beforeHooks = append(beforeHooks, hook)
	case HookAfterExecute:
		afterHooks = append(afterHooks, hook)
	}
}

// ClearHooks 清空全部钩子（测试用）。
func ClearHooks() {
	hookMu.Lock()
	beforeHooks = nil
	afterHooks = nil
	hookMu.Unlock()
}

// runBeforeHooks 依次执行 Before 钩子，返回（可能被改写的）SQL。
// 任一钩子 panic 由使用方负责；钩子内不应执行耗时操作（阻塞执行链）。
func runBeforeHooks(ctx context.Context, namespace, sqlId, sqlType, sqlStr string, args []interface{}) string {
	hookMu.RLock()
	hooks := beforeHooks
	hookMu.RUnlock()
	if len(hooks) == 0 {
		return sqlStr
	}
	hc := &HookContext{
		Ctx:       ctx,
		Namespace: namespace,
		SqlId:     sqlId,
		SqlType:   sqlType,
		SQL:       sqlStr,
		Args:      args,
	}
	for _, hook := range hooks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("before hook panic for %v.%v: %v", namespace, sqlId, r)
				}
			}()
			hook(hc)
		}()
	}
	return hc.SQL
}

// runAfterHooks 依次执行 After 钩子（执行完毕，含失败路径）。
func runAfterHooks(ctx context.Context, namespace, sqlId, sqlType, sqlStr string, args []interface{}, execErr error, d time.Duration) {
	hookMu.RLock()
	hooks := afterHooks
	hookMu.RUnlock()
	if len(hooks) == 0 {
		return
	}
	hc := &HookContext{
		Ctx:       ctx,
		Namespace: namespace,
		SqlId:     sqlId,
		SqlType:   sqlType,
		SQL:       sqlStr,
		Args:      args,
		Error:     execErr,
		Duration:  d,
	}
	for _, hook := range hooks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("after hook panic for %v.%v: %v", namespace, sqlId, r)
				}
			}()
			hook(hc)
		}()
	}
}
