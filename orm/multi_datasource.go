package orm

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bnulwh/mybatis-go/log"
)

// 多数据源注册表：维护 名称 -> *DB 的映射，并通过 gDbConn 指向当前活跃数据源。
// Mapper 代理方法、orm.Execute / orm.Query / Begin 均在活跃数据源上执行。
type dataSources struct {
	mu      sync.RWMutex
	sources map[string]*DB
}

var gDataSources = dataSources{sources: map[string]*DB{}}

const defaultDataSourceName = "default"

func (r *dataSources) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources = map[string]*DB{}
}

func (r *dataSources) add(name string, db *DB) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources[name] = db
}

func (r *dataSources) get(name string) (*DB, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	db, ok := r.sources[name]
	return db, ok
}

func (r *dataSources) names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.sources))
	for name := range r.sources {
		names = append(names, name)
	}
	return names
}

func (r *dataSources) closeAll() {
	r.mu.RLock()
	dbs := make([]*DB, 0, len(r.sources))
	for _, db := range r.sources {
		dbs = append(dbs, db)
	}
	r.mu.RUnlock()
	for _, db := range dbs {
		db.close()
	}
}

// InitializeDataSources 从 properties 文件初始化多数据源。
// 默认数据源使用 spring.datasource.* 配置，附加数据源需在
// mybatis.datasources 中列出名称并使用 spring.datasource.<name>.* 配置。
func InitializeDataSources(filename string) error {
	cm := LoadSettings(filename)
	return InitializeDataSourcesFromSettings(cm)
}

// InitializeDataSourcesFromSettings 同 InitializeDataSources，直接传入配置 map。
func InitializeDataSourcesFromSettings(cm map[string]string) error {
	configs := parseMultiDatabaseConfig(cm)
	err1 := gCache.initSqls(cm["mybatis.mapper-locations"])
	gDataSources.closeAll()
	gDataSources.reset()
	var errs []error
	for name, cfg := range configs {
		db, err := Open(cfg)
		if err != nil {
			errs = append(errs, fmt.Errorf("open datasource %s failed: %v", name, err))
			continue
		}
		gDataSources.add(name, db)
	}
	// 默认数据源设为活跃
	if db, ok := gDataSources.get(defaultDataSourceName); ok {
		gDbConn = db
	}
	return combineErrors(err1, combineErrors(errs...))
}

// UseDataSource 切换当前活跃数据源，影响后续 Mapper 方法 / orm.Execute / orm.Query / Begin。
// 注意：切换发生在全局层，请避免在并发 goroutine 中交错切换。
func UseDataSource(name string) error {
	db, ok := gDataSources.get(name)
	if !ok {
		return fmt.Errorf("datasource %s not found, available: %v", name, gDataSources.names())
	}
	gDbConn = db
	return nil
}

// GetDataSource 获取指定名称的数据源。
func GetDataSource(name string) (*DB, error) {
	db, ok := gDataSources.get(name)
	if !ok {
		return nil, fmt.Errorf("datasource %s not found, available: %v", name, gDataSources.names())
	}
	return db, nil
}

// GetDataSourceNames 返回全部已注册数据源名称。
func GetDataSourceNames() []string {
	return gDataSources.names()
}

// AddDataSource 以编程方式注册命名数据源（不切换活跃数据源，需配合 UseDataSource）。
func AddDataSource(name, dbType string, host string, port int, user, pwd, dbName string) error {
	if name == "" || name == defaultDataSourceName {
		return fmt.Errorf("invalid datasource name %q", name)
	}
	if _, ok := gDataSources.get(name); ok {
		return fmt.Errorf("datasource %s already exists", name)
	}
	cfg := newDatabaseConfig(dbType, host, port, user, pwd, dbName)
	db, err := Open(cfg)
	if err != nil {
		return err
	}
	gDataSources.add(name, db)
	return nil
}

// ReConnectDataSource 重连指定数据源。
func ReConnectDataSource(name string) error {
	db, ok := gDataSources.get(name)
	if !ok {
		return fmt.Errorf("datasource %s not found", name)
	}
	return db.Reconnect()
}

// parseMultiDatabaseConfig 解析多数据源配置：
// 默认数据源使用 spring.datasource.url / username / password 等无名称前缀的键；
// 附加数据源在 mybatis.datasources 中列出，配置键为 spring.datasource.<name>.<key>。
func parseMultiDatabaseConfig(cm map[string]string) map[string]*Config {
	configs := map[string]*Config{}
	configs[defaultDataSourceName] = parseDatabaseConfig(cm)
	names := strings.Split(cm["mybatis.datasources"], ",")
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || name == defaultDataSourceName {
			continue
		}
		prefix := "spring.datasource." + name + "."
		sub := map[string]string{}
		found := false
		for k, v := range cm {
			if strings.HasPrefix(k, prefix) {
				// 去掉名称前缀，复用现有单数据源解析逻辑
				sub["spring.datasource."+strings.TrimPrefix(k, prefix)] = v
				found = true
			}
		}
		if !found {
			log.Warnf("datasource %s has no config, skipped", name)
			continue
		}
		configs[name] = parseDatabaseConfig(sub)
	}
	return configs
}
