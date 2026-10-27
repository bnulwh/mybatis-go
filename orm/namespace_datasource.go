package orm

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/bnulwh/mybatis-go/log"
)

// 按 namespace 绑定数据源（P1-6）。
//
// 背景：多数据源此前的唯一用法是 orm.UseDataSource(name) 切换**全局**活跃数据源
// （gDbConn），并发场景下两个请求交错切换会互相劫持（A 请求的 SQL 打到 B 的数据源）。
// 本机制把「数据源」与「Mapper namespace」静态绑定，执行期按语句所属 namespace
// 解析数据源，不再依赖全局切换：
//
//	mybatis.datasources=orderdb,userdb
//	mybatis.datasource.bind.orderdb=com.demo.mapper.Order,com.demo.order
//	mybatis.datasource.bind.userdb=com.demo.mapper.User
//
// 或编程式：
//
//	orm.BindNamespaceDataSource("com.demo.mapper.Order", "orderdb")
//
// 匹配规则：namespace 先精确匹配，再按**最长前缀**匹配（`*` 结尾的绑定等价于去星前缀）。
// 未命中任何绑定的语句沿用全局活跃数据源（gDbConn），存量行为不变。
//
// 事务：ctx 携带 WithTx 事务时以事务所属数据源为准（同一事务内 SQL 不会跨库），
// 绑定与事务数据源不一致时按 namespace 告警一次；跨库事务请用
// orm.WithTxForNamespace(ctx, namespace, fn)。

type namespaceBinding struct {
	pattern string // 归一化后的匹配前缀（小写、去空白、去尾部 *）
	dsName  string
}

var (
	nsBindMu    sync.RWMutex
	nsBindings  []namespaceBinding
	nsWarned    sync.Map // 冲突告警去重（namespace → struct{}）
	nsWarnedKey sync.Map // 绑定失效告警去重（dsName → struct{}）
)

// normalizeNamespacePattern 归一化绑定模式：去空白、小写、去尾部 '*' 与多余空白。
func normalizeNamespacePattern(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimSuffix(s, "*")
	return strings.TrimSpace(s)
}

// BindNamespaceDataSource 把 Mapper namespace（或其前缀）绑定到指定数据源（P1-6）。
// dsName 必须是已注册的数据源名称（default 或 mybatis.datasources 中列出的名称）；
// 同一 pattern 重复绑定以最后一次为准。
func BindNamespaceDataSource(namespacePrefix, dsName string) error {
	pattern := normalizeNamespacePattern(namespacePrefix)
	if pattern == "" {
		return fmt.Errorf("bind datasource: namespace prefix must not be empty")
	}
	dsName = strings.TrimSpace(dsName)
	if dsName == "" {
		return fmt.Errorf("bind datasource: datasource name must not be empty")
	}
	if _, ok := gDataSources.get(dsName); !ok {
		return fmt.Errorf("bind datasource: datasource %q not found, available: %v", dsName, gDataSources.names())
	}
	nsBindMu.Lock()
	defer nsBindMu.Unlock()
	for i := range nsBindings {
		if nsBindings[i].pattern == pattern {
			nsBindings[i].dsName = dsName
			log.Infof("[namespace-ds] rebind %q -> %s", pattern, dsName)
			return nil
		}
	}
	nsBindings = append(nsBindings, namespaceBinding{pattern: pattern, dsName: dsName})
	log.Infof("[namespace-ds] bind %q -> %s", pattern, dsName)
	return nil
}

// UnbindNamespaceDataSource 取消某个 namespace 前缀的绑定；返回是否存在该绑定。
func UnbindNamespaceDataSource(namespacePrefix string) bool {
	pattern := normalizeNamespacePattern(namespacePrefix)
	nsBindMu.Lock()
	defer nsBindMu.Unlock()
	for i := range nsBindings {
		if nsBindings[i].pattern == pattern {
			nsBindings = append(nsBindings[:i], nsBindings[i+1:]...)
			log.Infof("[namespace-ds] unbind %q", pattern)
			return true
		}
	}
	return false
}

// ResetNamespaceDataSourceBindings 清空全部 namespace 绑定（初始化/测试用）。
func ResetNamespaceDataSourceBindings() {
	nsBindMu.Lock()
	nsBindings = nil
	nsBindMu.Unlock()
	nsWarned = sync.Map{}
}

// NamespaceDataSourceBindings 返回当前 namespace → 数据源绑定（pattern 归一化后）。
func NamespaceDataSourceBindings() map[string]string {
	nsBindMu.RLock()
	defer nsBindMu.RUnlock()
	out := make(map[string]string, len(nsBindings))
	for _, b := range nsBindings {
		out[b.pattern] = b.dsName
	}
	return out
}

// matchNamespaceBinding 返回命中的绑定（精确优先，其次最长前缀）；未命中返回 nil。
func matchNamespaceBinding(namespace string) *namespaceBinding {
	ns := strings.ToLower(strings.TrimSpace(namespace))
	if ns == "" {
		return nil
	}
	nsBindMu.RLock()
	defer nsBindMu.RUnlock()
	if len(nsBindings) == 0 {
		return nil
	}
	var best *namespaceBinding
	for i := range nsBindings {
		b := &nsBindings[i]
		if b.pattern == "" || !strings.HasPrefix(ns, b.pattern) {
			continue
		}
		if best == nil || len(b.pattern) > len(best.pattern) {
			best = b
		}
	}
	if best == nil {
		return nil
	}
	// 拷贝一份，避免调用方持有切片元素指针（Unbind 会重排切片）
	return &namespaceBinding{pattern: best.pattern, dsName: best.dsName}
}

// dbForNamespace 解析 namespace 绑定的数据源；未绑定或绑定失效返回 nil（回退 gDbConn）。
func dbForNamespace(namespace string) *DB {
	b := matchNamespaceBinding(namespace)
	if b == nil {
		return nil
	}
	db, ok := gDataSources.get(b.dsName)
	if !ok || db == nil {
		if _, loaded := nsWarnedKey.LoadOrStore(b.dsName, struct{}{}); !loaded {
			log.Warnf("[namespace-ds] bound datasource %q for namespace pattern %q not found, fallback to active datasource", b.dsName, b.pattern)
		}
		return nil
	}
	return db
}

// routeNamespaceDB 按 namespace 把数据源写入 ctx（routedDB 优先读取）：
//   - ctx 已携带事务 → 以事务所属数据源为准（事务内 SQL 不跨库）；
//   - 命中 namespace 绑定 → 该数据源；
//   - 否则原样返回（执行期回退 gDbConn）。
func routeNamespaceDB(ctx context.Context, namespace string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if tx := TxFromContext(ctx); tx != nil {
		if tx.db != nil {
			if b := matchNamespaceBinding(namespace); b != nil && dbForNamespace(namespace) != tx.db {
				warnNamespaceTxConflict(namespace, b.dsName, dataSourceName(tx.db))
			}
			return contextWithRoute(ctx, tx.db)
		}
		return ctx
	}
	if db := dbForNamespace(namespace); db != nil {
		return contextWithRoute(ctx, db)
	}
	return ctx
}

// warnNamespaceTxConflict 事务数据源与 namespace 绑定不一致时告警一次（不阻断执行，
// 事务内的语句始终跟随事务连接，避免半提交/跨库写入）。
func warnNamespaceTxConflict(namespace, boundName, txName string) {
	if _, loaded := nsWarned.LoadOrStore(namespace, struct{}{}); loaded {
		return
	}
	log.Warnf("[namespace-ds] %v is bound to datasource %q but executes inside a transaction of %q; "+
		"statement follows the transaction (open it with orm.WithTxForNamespace to target the bound datasource)",
		namespace, boundName, txName)
}

// dataSourceName 返回数据源注册别名（找不到时回退 Setting.Name），仅用于日志。
func dataSourceName(db *DB) string {
	if db == nil {
		return "<nil>"
	}
	for name, item := range gDataSources.snapshot() {
		if item == db {
			return name
		}
	}
	if db.Config != nil {
		return db.Setting.Name
	}
	return "<unknown>"
}

// WithTxForNamespace 在 namespace 绑定的数据源上开启回调事务（P1-6）：
// fn 内声明了 context.Context 参数且传入本 ctx 的 Mapper 方法自动加入该事务。
// 未绑定 namespace 时等价于 orm.WithTx(ctx, fn)。
func WithTxForNamespace(ctx context.Context, namespace string, fn func(ctx context.Context) error) error {
	if fn == nil {
		return fmt.Errorf("WithTxForNamespace: fn cannot be nil")
	}
	db := dbForNamespace(namespace)
	if db == nil {
		db = routedDB(ctx)
	}
	if db == nil {
		return fmt.Errorf("connection not init.")
	}
	return db.WithTx(ctx, fn)
}

// applyNamespaceBindingConfig 解析配置项 mybatis.datasource.bind.<dsName>=<prefix,...>。
// 数据源不存在时报错（启动期暴露配置错误，而不是运行期静默回退）。
func applyNamespaceBindingConfig(cm map[string]string) error {
	const (
		keyPrefix     = "mybatis.datasource.bind."
		legacyKeyName = "mybatis.namespace.datasource.bind."
	)
	keys := make([]string, 0, len(cm))
	for k := range cm {
		lk := strings.ToLower(strings.TrimSpace(k))
		if strings.HasPrefix(lk, keyPrefix) || strings.HasPrefix(lk, legacyKeyName) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys) // 保证解析顺序稳定（redis 风格配置存在重复 pattern 时结果可预期）
	var errs []error
	for _, k := range keys {
		lk := strings.ToLower(strings.TrimSpace(k))
		var dsName string
		if strings.HasPrefix(lk, keyPrefix) {
			dsName = strings.TrimSpace(k[len(keyPrefix):])
		} else {
			dsName = strings.TrimSpace(k[len(legacyKeyName):])
		}
		if dsName == "" {
			continue
		}
		for _, raw := range strings.Split(cm[k], ",") {
			pattern := normalizeNamespacePattern(raw)
			if pattern == "" {
				continue
			}
			if err := BindNamespaceDataSource(pattern, dsName); err != nil {
				errs = append(errs, fmt.Errorf("%v: %v", k, err))
			}
		}
	}
	return combineErrors(errs...)
}
