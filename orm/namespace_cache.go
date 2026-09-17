package orm

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/bnulwh/mybatis-go/log"
	lru "github.com/hashicorp/golang-lru/v2"
)

type cacheEntry struct {
	Value     reflect.Value
	ExpiresAt time.Time
}

func (e *cacheEntry) IsExpired() bool {
	return !e.ExpiresAt.IsZero() && time.Now().After(e.ExpiresAt)
}

type NamespaceCache struct {
	mu    sync.RWMutex
	lru   *lru.Cache[string, *cacheEntry]
	ttl   time.Duration
}

func NewNamespaceCache(size int, ttl time.Duration) *NamespaceCache {
	if size <= 0 {
		size = 1024
	}
	c, err := lru.New[string, *cacheEntry](size)
	if err != nil {
		log.Warnf("create namespace LRU cache failed: %v, fallback to size 256", err)
		c, _ = lru.New[string, *cacheEntry](256)
	}
	return &NamespaceCache{lru: c, ttl: ttl}
}

func cacheKey(namespace, sqlID string, args ...interface{}) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s.%s", namespace, sqlID)
	for _, a := range args {
		fmt.Fprintf(h, "%v", a)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (nc *NamespaceCache) Get(key string) (reflect.Value, bool) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	ent, ok := nc.lru.Get(key)
	if !ok || ent.IsExpired() {
		return reflect.Value{}, false
	}
	return ent.Value, true
}

func (nc *NamespaceCache) Set(key string, val reflect.Value) {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	var exp time.Time
	if nc.ttl > 0 {
		exp = time.Now().Add(nc.ttl)
	}
	nc.lru.Add(key, &cacheEntry{Value: val, ExpiresAt: exp})
}

func (nc *NamespaceCache) Flush() {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	nc.lru.Purge()
}

func (nc *NamespaceCache) Len() int {
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	return nc.lru.Len()
}

type secondLevelCache struct {
	mu       sync.RWMutex
	enabled  bool
	size     int
	ttl      time.Duration
	caches   map[string]*NamespaceCache
}

var gSecondCache secondLevelCache

func initSecondCache(enabled bool, size int, ttl time.Duration) {
	gSecondCache.mu.Lock()
	defer gSecondCache.mu.Unlock()
	gSecondCache.enabled = enabled
	gSecondCache.size = size
	gSecondCache.ttl = ttl
	if enabled {
		gSecondCache.caches = make(map[string]*NamespaceCache)
		log.Infof("second-level cache enabled: size=%d, ttl=%v", size, ttl)
	}
}

func SetCacheEnabled(on bool) {
	gSecondCache.mu.Lock()
	defer gSecondCache.mu.Unlock()
	if on && !gSecondCache.enabled {
		if gSecondCache.size <= 0 {
			gSecondCache.size = 1024
		}
		if gSecondCache.ttl <= 0 {
			gSecondCache.ttl = time.Hour
		}
		gSecondCache.caches = make(map[string]*NamespaceCache)
	}
	gSecondCache.enabled = on
}

func (sc *secondLevelCache) isEnabled() bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.enabled
}

func (sc *secondLevelCache) getNamespaceCache(namespace string) *NamespaceCache {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	nc, ok := sc.caches[namespace]
	if !ok {
		nc = NewNamespaceCache(sc.size, sc.ttl)
		sc.caches[namespace] = nc
	}
	return nc
}

func (sc *secondLevelCache) Get(namespace, key string) (reflect.Value, bool) {
	if !sc.isEnabled() {
		return reflect.Value{}, false
	}
	sc.mu.RLock()
	nc, ok := sc.caches[namespace]
	sc.mu.RUnlock()
	if !ok {
		return reflect.Value{}, false
	}
	return nc.Get(key)
}

func (sc *secondLevelCache) Set(namespace, key string, val reflect.Value) {
	if !sc.isEnabled() {
		return
	}
	nc := sc.getNamespaceCache(namespace)
	nc.Set(key, val)
}

func (sc *secondLevelCache) Flush(namespace string) {
	if !sc.isEnabled() {
		return
	}
	sc.mu.RLock()
	nc, ok := sc.caches[namespace]
	sc.mu.RUnlock()
	if ok {
		nc.Flush()
	}
}

func (sc *secondLevelCache) FlushAll() {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	for _, nc := range sc.caches {
		nc.Flush()
	}
}
