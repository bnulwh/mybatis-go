package orm

import (
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/bnulwh/mybatis-go/utils"
)

// fakeConnPool 内嵌 *sql.DB 满足 ConnPool 接口，重写 Ping/GetDBConn 以便 PreparedStmt 包装下可 Ping。
type fakeConnPool struct {
	*sql.DB
	pingCalled bool
}

func (f *fakeConnPool) GetDBConn() (*sql.DB, error) {
	return f.DB, nil
}

func (f *fakeConnPool) Ping() error {
	f.pingCalled = true
	return nil
}

// Test_Open_CustomConnPool 注入自定义连接池（*sql.DB / ConnPool 实现）被 Open 使用，
// 不再按 DSN 新建连接（自定义 DB 注入）。
func Test_Open_CustomConnPool(t *testing.T) {
	cfg := newDatabaseConfig("mysql", "localhost", 3306, "root", "123456", "testdb")
	cfg.PreparedStmt = false
	fake := &fakeConnPool{}
	cfg.ConnPool = fake
	db, err := Open(cfg)
	if err != nil {
		t.Error("Open with custom conn pool failed:", err)
		return
	}
	if db.ConnPool != fake {
		t.Errorf("custom conn pool not used, got %T", db.ConnPool)
	}
	if !fake.pingCalled {
		t.Error("custom conn pool Ping not invoked")
	}
}

// Test_Open_CustomConnPool_Prepared 自定义连接池 + 预编译缓存包装共存
func Test_Open_CustomConnPool_Prepared(t *testing.T) {
	sqldb, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Error("open sqlite failed:", err)
		return
	}
	defer sqldb.Close()
	cfg := newDatabaseConfig("postgres", "localhost", 5432, "root", "123456", "testdb")
	fake := &fakeConnPool{DB: sqldb}
	cfg.ConnPool = fake
	db, err := Open(cfg)
	if err != nil {
		t.Error("Open with custom conn pool + prepared failed:", err)
		return
	}
	ps, ok := db.ConnPool.(*PreparedStmtDB)
	if !ok {
		t.Errorf("prepared wrap missing, got %T", db.ConnPool)
		return
	}
	if ps.ConnPool != fake {
		t.Error("PreparedStmtDB should wrap the custom conn pool")
	}
	// PreparedStmtDB.Ping 经 GetDBConn 委托到底层连接池，验证取回的是注入的 *sql.DB
	if got, err := ps.GetDBConn(); err != nil || got != sqldb {
		t.Errorf("GetDBConn = %v, %v; want injected *sql.DB", got, err)
	}
}

// Test_Uint8ColumnFallback []uint8 列（MySQL 未开 parseTime 时 DATETIME/文本列）的兜底转换：
// newInstance 建 *sql.RawBytes，convertRawBytes2String 转字符串，change2Time 可再解析为时间。
func Test_Uint8ColumnFallback(t *testing.T) {
	// newInstance([]uint8) → *sql.RawBytes
	inst := newInstance(reflect.TypeOf([]byte(nil)))
	if _, ok := inst.(*sql.RawBytes); !ok {
		t.Errorf("newInstance([]uint8) = %T, want *sql.RawBytes", inst)
	}
	// convertRawBytes2String
	raw := sql.RawBytes("2026-08-19 10:00:00")
	s, err := convertRawBytes2String(&raw)
	if err != nil || s != "2026-08-19 10:00:00" {
		t.Errorf("convertRawBytes2String = %q, %v", s, err)
	}
	// 时间串可经 ChangeType 解析回 time.Time（DATETIME 不再被丢弃）
	tm, err := utils.ChangeType("2026-08-19 10:00:00", reflect.TypeOf(time.Time{}))
	if err != nil {
		t.Error("change raw datetime string failed:", err)
	} else if tm.(time.Time).Year() != 2026 || tm.(time.Time).Hour() != 10 {
		t.Errorf("parsed time mismatch: %v", tm)
	}
}
