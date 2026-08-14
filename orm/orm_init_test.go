package orm

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func Test_InitializeDatabaseReturnsError(t *testing.T) {
	// 目标目录不存在，SQLite 无法打开数据库文件，应返回错误而非 nil
	badPath := filepath.Join(os.TempDir(), "mybatis-go-no-such-dir", "test.db")
	err := InitializeDatabase("sqlite", "", 0, "", "", badPath)
	if err == nil {
		t.Error("InitializeDatabase should return error when database cannot be opened")
	}
}

func Test_InitializeDatabaseSuccess(t *testing.T) {
	// 正常路径应成功并设置全局连接
	dbPath := filepath.Join(t.TempDir(), "init_test.db")
	err := InitializeDatabase("sqlite", "", 0, "", "", dbPath)
	if err != nil {
		t.Errorf("InitializeDatabase failed: %v", err)
		return
	}
	if gDbConn == nil {
		t.Error("gDbConn should be set after successful InitializeDatabase")
		return
	}
	// 清理，避免影响后续测试
	Close()
	gDbConn = nil
}

func Test_ReConnectResetsPreparedStmt(t *testing.T) {
	// 修复 ③：ReConnect 后预编译缓存必须清空重建并指向新连接池
	dbPath := filepath.Join(t.TempDir(), "reconnect_test.db")
	cfg := newDatabaseConfig("sqlite", "", 0, "", "", dbPath)
	cfg.PreparedStmt = true
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() {
		gDbConn = nil
		db.close()
	}()
	gDbConn = db

	v, _ := db.cacheStore.Load(preparedStmtDBKey)
	ps := v.(*PreparedStmtDB)

	// 重连前先缓存一个 statement，模拟旧连接上的预编译缓存
	if _, err := ps.prepare(context.Background(), db.ConnPool, "select 1"); err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	if len(ps.Stmts) != 1 {
		t.Fatalf("expected 1 cached stmt before reconnect, got %d", len(ps.Stmts))
	}

	if err := ReConnect(); err != nil {
		t.Fatalf("ReConnect failed: %v", err)
	}

	// 预编译缓存必须清空
	if len(ps.Stmts) != 0 || len(ps.PreparedSQL) != 0 {
		t.Errorf("prepared stmt cache not reset after reconnect: stmts=%d preparedSql=%d",
			len(ps.Stmts), len(ps.PreparedSQL))
	}
	// 包装器必须指向新连接池
	newSQLDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB failed: %v", err)
	}
	if ps.ConnPool != newSQLDB {
		t.Error("PreparedStmtDB.ConnPool should point to new connection after reconnect")
	}
	// PreparedStmt 模式下 db.ConnPool 应恢复为包装器
	if db.ConnPool != ps {
		t.Error("db.ConnPool should be re-wrapped with PreparedStmtDB after reconnect")
	}
	// 重连后新连接必须可用
	if err := ps.Ping(); err != nil {
		t.Errorf("Ping on new connection failed: %v", err)
	}
}

func Test_OpenWithPreparedStmtStillPings(t *testing.T) {
	// 修复 ②：PreparedStmt=true 时 ConnPool 被替换为 PreparedStmtDB，
	// 其缺少 Ping() 导致连接健康检查被静默跳过，数据库不可用时 Open 仍返回 nil。
	badPath := filepath.Join(os.TempDir(), "mybatis-go-no-such-dir", "test.db")
	cfg := newDatabaseConfig("sqlite", "", 0, "", "", badPath)
	cfg.PreparedStmt = true
	db, err := Open(cfg)
	if err == nil {
		t.Error("Open with PreparedStmt=true should still ping and return error for unopenable db")
	}
	if db != nil {
		db.close()
	}
}

func Test_PreparedStmtDBPing(t *testing.T) {
	// PreparedStmtDB.Ping 应委托到底层连接池
	dbPath := filepath.Join(t.TempDir(), "ping_test.db")
	cfg := newDatabaseConfig("sqlite", "", 0, "", "", dbPath)
	cfg.PreparedStmt = true
	db, err := Open(cfg)
	if err != nil {
		t.Errorf("Open failed: %v", err)
		return
	}
	defer db.close()

	v, ok := db.cacheStore.Load(preparedStmtDBKey)
	if !ok {
		t.Fatal("preparedStmtDB not stored in cacheStore")
	}
	ps := v.(*PreparedStmtDB)
	if err := ps.Ping(); err != nil {
		t.Errorf("Ping on healthy PreparedStmtDB failed: %v", err)
	}
}
