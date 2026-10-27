package orm

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/bnulwh/mybatis-go/log"
)

type rwRouteKey struct{}

var (
	rwSplittingEnabled  int32
	replicaNames        []string
	replicaRoundRobin   uint64
	rwMu                sync.RWMutex
)

func SetReadWriteSplitting(on bool) {
	if on {
		atomic.StoreInt32(&rwSplittingEnabled, 1)
	} else {
		atomic.StoreInt32(&rwSplittingEnabled, 0)
	}
}

func ReadWriteSplittingEnabled() bool {
	return atomic.LoadInt32(&rwSplittingEnabled) == 1
}

func SetReplicaNames(names []string) {
	rwMu.Lock()
	defer rwMu.Unlock()
	replicaNames = names
	log.Infof("read-write splitting replicas: %v", names)
}

func GetReplicaNames() []string {
	rwMu.RLock()
	defer rwMu.RUnlock()
	result := make([]string, len(replicaNames))
	copy(result, replicaNames)
	return result
}

func routeReplica() *DB {
	if !ReadWriteSplittingEnabled() {
		return nil
	}
	if inTransaction() {
		return nil
	}
	rwMu.RLock()
	names := replicaNames
	rwMu.RUnlock()
	if len(names) == 0 {
		return nil
	}
	idx := atomic.AddUint64(&replicaRoundRobin, 1) % uint64(len(names))
	db, ok := gDataSources.get(names[idx])
	if !ok {
		log.Warnf("replica datasource %q not found, fallback to master", names[idx])
		return nil
	}
	return db
}

// inTransaction 判断默认活跃数据源上是否存在 TCC 全局事务槽（兼容既有调用）。
func inTransaction() bool {
	if gDbConn == nil {
		return false
	}
	if gDbConn.currentTx() != nil {
		return true
	}
	return false
}

// inTransactionWith 判断 ctx 是否处于事务：ctx 事务优先，其次检查当前路由数据源
// （namespace 绑定的数据源）与默认活跃数据源上的 TCC 全局事务槽（P1-6）。
func inTransactionWith(ctx context.Context) bool {
	if ctx != nil && TxFromContext(ctx) != nil {
		return true
	}
	if db := routedDB(ctx); db != nil && db.currentTx() != nil {
		return true
	}
	return false
}

func contextWithRoute(ctx context.Context, db *DB) context.Context {
	if db == nil {
		return ctx
	}
	return context.WithValue(ctx, rwRouteKey{}, db)
}

func routedDB(ctx context.Context) *DB {
	if ctx != nil {
		if db, ok := ctx.Value(rwRouteKey{}).(*DB); ok && db != nil {
			return db
		}
	}
	return gDbConn
}

func routeReadDB(ctx context.Context) context.Context {
	if !ReadWriteSplittingEnabled() {
		return ctx
	}
	// P1-6：namespace 已绑定到非默认数据源时不参与全局副本路由——副本集合
	// （mybatis.replicas）属于默认数据源，跨数据源路由会打到错误的库。
	if base := routedDB(ctx); base != nil && base != gDbConn {
		return ctx
	}
	if inTransactionWith(ctx) {
		return ctx
	}
	replica := routeReplica()
	if replica == nil {
		return ctx
	}
	return contextWithRoute(ctx, replica)
}

func routeWriteDB(ctx context.Context) context.Context {
	return ctx
}

func initReplicaDatasources(cm map[string]string) {
	replicaCfg := strings.TrimSpace(cm["mybatis.replicas"])
	if replicaCfg == "" {
		return
	}
	names := strings.Split(replicaCfg, ",")
	var valid []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || name == defaultDataSourceName {
			continue
		}
		db, ok := gDataSources.get(name)
		if ok {
			valid = append(valid, name)
			log.Infof("replica datasource %q registered (type=%v)", name, db.Setting.Type)
		} else {
			log.Warnf("replica datasource %q not found in registered datasources, skipped", name)
		}
	}
	if len(valid) > 0 {
		SetReplicaNames(valid)
	}
}

// RegisterReplica 编程方式注册副本数据源（不切换活跃数据源）。
func RegisterReplica(name, dbType string, host string, port int, user, pwd, dbName string) error {
	if name == "" || name == defaultDataSourceName {
		return fmt.Errorf("invalid replica name %q", name)
	}
	if err := AddDataSource(name, dbType, host, port, user, pwd, dbName); err != nil {
		return err
	}
	rwMu.Lock()
	defer rwMu.Unlock()
	replicaNames = append(replicaNames, name)
	return nil
}

// PickReplica 随机选取一个副本数据源（测试/调试用）。
func PickReplica() (*DB, error) {
	rwMu.RLock()
	names := replicaNames
	rwMu.RUnlock()
	if len(names) == 0 {
		return nil, fmt.Errorf("no replica datasources registered")
	}
	idx := rand.Intn(len(names))
	db, ok := gDataSources.get(names[idx])
	if !ok {
		return nil, fmt.Errorf("replica datasource %q not found", names[idx])
	}
	return db, nil
}
