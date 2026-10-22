package orm

import (
	"context"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/bnulwh/mybatis-go/log"
)

// C2：WithTx 作用域诊断 —— 检测「WithTx 内调用了未加入事务的 Mapper 方法」并告警。
//
// 背景：WithTx 只对携带事务的 context 生效（声明 context.Context 参数并传入
// WithTx ctx 的 Mapper 方法、ExecuteContext / QueryContext 等显式 ctx API）。
// 方法签名未声明 ctx 时框架拿到的是 context.Background()，在 WithTx 内调用会
// 静默走非事务连接——语义上等于事务失效，但此前无任何提示。
//
// 机制：WithTx 进入/退出时在当前 goroutine 上打作用域标记（嵌套计数）；Mapper
// 代理执行时若语句未携带事务且处于 WithTx 作用域内，log.Warnf 告警一次
// （每 namespace.id 一次；可用 SetTxScopeWarn(false) 或配置
// mybatis.configuration.tx-scope-warn=false 关闭）。

// txScopeWarnEnabled C2 告警开关（默认开）。
var txScopeWarnEnabled atomic.Bool

// activeTxScopes 进程内活动 WithTx 作用域计数（含嵌套）；
// 为 0 时 inTxScope 走快路径，不解析 goroutine id。
var activeTxScopes atomic.Int64

// txScopeDepths WithTx 所在 goroutine id → 嵌套深度。
// 同一 goroutine 内 enter/leave 单写者，sync.Map 按键操作无竞态。
var txScopeDepths sync.Map // int64 → int64

// nonCtxTxWarned 已告警的「WithTx 内未加入事务」语句集合（namespace.id，C2）。
var nonCtxTxWarned sync.Map

func init() {
	txScopeWarnEnabled.Store(true)
}

// SetTxScopeWarn 开关 C2 告警：WithTx 作用域内执行未加入事务的 Mapper 方法时
// Warn（默认开）。
func SetTxScopeWarn(on bool) {
	txScopeWarnEnabled.Store(on)
	log.Infof("set tx scope warn: %v", on)
}

// TxScopeWarn 返回 C2 告警开关当前值。
func TxScopeWarn() bool {
	return txScopeWarnEnabled.Load()
}

// currentGoID 解析当前 goroutine id（runtime.Stack 首行 "goroutine N [running]:"）。
// 解析失败返回 0（视为未知，检测跳过）。
func currentGoID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	fields := strings.Fields(string(buf[:n]))
	if len(fields) < 2 {
		return 0
	}
	id, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// enterTxScope / leaveTxScope 由 WithTx 在进入/退出作用域时调用，标记当前
// goroutine；同一 goroutine 的嵌套 WithTx 计数累加。leave 必须与 enter 配对
// （含 panic 路径，WithTx 内以 defer 保证）。
func enterTxScope() {
	activeTxScopes.Add(1)
	id := currentGoID()
	if id == 0 {
		return
	}
	if d, ok := txScopeDepths.Load(id); ok {
		txScopeDepths.Store(id, d.(int64)+1)
		return
	}
	txScopeDepths.Store(id, int64(1))
}

func leaveTxScope() {
	activeTxScopes.Add(-1)
	id := currentGoID()
	if id == 0 {
		return
	}
	if d, ok := txScopeDepths.Load(id); ok {
		if d.(int64) <= 1 {
			txScopeDepths.Delete(id)
			return
		}
		txScopeDepths.Store(id, d.(int64)-1)
	}
}

// inTxScope 判断当前 goroutine 是否处于 WithTx 作用域内（C2 检测）。
// 快路径：进程内无活动 WithTx 时仅一次原子读，不解析 goroutine id。
func inTxScope() bool {
	if activeTxScopes.Load() <= 0 {
		return false
	}
	id := currentGoID()
	if id == 0 {
		return false
	}
	d, ok := txScopeDepths.Load(id)
	return ok && d.(int64) > 0
}

// warnNonCtxInTxScope C2：Mapper 方法在 WithTx 作用域内执行但未加入事务时告警
// 一次——方法未声明 context.Context 参数，或调用方传入了未携带事务的 ctx。
// 该语句会走独立连接，不参与 WithTx 的提交/回滚。
func warnNonCtxInTxScope(ctx context.Context, namespace, id string) {
	if TxFromContext(ctx) != nil {
		return
	}
	key := namespace + "." + id
	if _, warned := nonCtxTxWarned.Load(key); warned {
		return
	}
	if !txScopeWarnEnabled.Load() || !inTxScope() {
		return
	}
	nonCtxTxWarned.Store(key, struct{}{})
	log.Warnf("statement %s executed inside WithTx without joining the transaction "+
		"(mapper method has no context.Context parameter, or caller passed a non-tx ctx); "+
		"it runs on a separate connection and is NOT committed/rolled back with WithTx — "+
		"add ctx to the method signature and pass the WithTx ctx", key)
}
