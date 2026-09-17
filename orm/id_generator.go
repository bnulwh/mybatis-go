package orm

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bnulwh/mybatis-go/log"
)

type IdGenerator func(ctx context.Context) interface{}

var (
	idGenerators   = map[string]IdGenerator{}
	defaultIdType  string
	idGeneratorMu  sync.RWMutex
	snowflakeOnce  sync.Once
	snowflakeInst  *sfNode
)

func init() {
	RegisterIdGenerator("snowflake", snowflakeIdGenerator)
	RegisterIdGenerator("uuid", uuidIdGenerator)
}

func RegisterIdGenerator(name string, gen IdGenerator) {
	if name == "" || gen == nil {
		return
	}
	idGeneratorMu.Lock()
	idGenerators[strings.ToLower(name)] = gen
	idGeneratorMu.Unlock()
}

func SetDefaultIdType(idType string) {
	idGeneratorMu.Lock()
	defaultIdType = strings.TrimSpace(strings.ToLower(idType))
	idGeneratorMu.Unlock()
}

func generateId(ctx context.Context) (interface{}, bool) {
	idGeneratorMu.RLock()
	dt := defaultIdType
	idGeneratorMu.RUnlock()
	if dt == "" {
		return nil, false
	}
	idGeneratorMu.RLock()
	gen, ok := idGenerators[dt]
	idGeneratorMu.RUnlock()
	if !ok || gen == nil {
		return nil, false
	}
	val := gen(ctx)
	return val, val != nil
}

func fillIdIfZero(ctx context.Context, arg ProxyArg) {
	if arg.ArgsLen == 0 {
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
	if info == nil || info.PrimaryField == nil {
		return
	}
	field := iv.FieldByName(info.PrimaryField.Name)
	if !field.IsValid() || !field.CanSet() {
		return
	}
	if !reflect.ValueOf(field.Interface()).IsZero() {
		return
	}
	id, ok := generateId(ctx)
	if !ok || id == nil {
		return
	}
	rval, err := convertIdValue(id, field.Type())
	if err != nil {
		log.Warnf("id generator convert failed: %v", err)
		return
	}
	field.Set(reflect.ValueOf(rval))
}

func convertIdValue(id interface{}, targetType reflect.Type) (interface{}, error) {
	idVal := reflect.ValueOf(id)
	if idVal.Type().ConvertibleTo(targetType) {
		return idVal.Convert(targetType).Interface(), nil
	}
	return nil, fmt.Errorf("cannot convert id %T to %v", id, targetType)
}

// --- Snowflake ---

type sfNode struct {
	workerID int64
	sequence int64
	lastTime int64
	mu       sync.Mutex
}

var snowflakeEpoch int64 = 1704067200000

func snowflakeIdGenerator(ctx context.Context) interface{} {
	snowflakeOnce.Do(func() {
		workerID := int64(1)
		if gDbConn != nil && gDbConn.Config != nil && gDbConn.Setting.SnowflakeWorkerID > 0 {
			workerID = gDbConn.Setting.SnowflakeWorkerID
		}
		snowflakeInst = &sfNode{workerID: workerID % 1024}
	})
	n := snowflakeInst
	n.mu.Lock()
	defer n.mu.Unlock()
	now := time.Now().UnixMilli()
	if now <= n.lastTime {
		n.sequence = (n.sequence + 1) & 4095
		if n.sequence == 0 {
			for now <= n.lastTime {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		n.sequence = 0
	}
	n.lastTime = now
	id := ((now - snowflakeEpoch) << 22) | (n.workerID << 12) | n.sequence
	return id
}

// --- UUID ---

func uuidIdGenerator(ctx context.Context) interface{} {
	var buf [16]byte
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Read(buf[:])
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x%04x%04x%04x%012x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}

// --- Atomic id counter for simple assignment ---

var simpleIdCounter int64

func init() {
	RegisterIdGenerator("assign_id", func(ctx context.Context) interface{} {
		return atomic.AddInt64(&simpleIdCounter, 1)
	})
}
