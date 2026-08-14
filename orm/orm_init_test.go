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

func Test_getRealValue(t *testing.T) {
	em := map[string]string{"ENV_A": "a_value", "ENV_B": "b_value"}

	// 命中环境变量
	if got := getRealValue("${ENV_A}", em); got != "a_value" {
		t.Errorf("getRealValue(${ENV_A}) = %q, want a_value", got)
	}
	// 未命中且无默认值 → 空串
	if got := getRealValue("${NOT_EXIST}", em); got != "" {
		t.Errorf("getRealValue(${NOT_EXIST}) = %q, want empty", got)
	}
	// 未命中但有默认值 → 默认值
	if got := getRealValue("${NOT_EXIST:default}", em); got != "default" {
		t.Errorf("getRealValue(${NOT_EXIST:default}) = %q, want default", got)
	}
	// 环境变量优先于默认值
	if got := getRealValue("${ENV_B:default}", em); got != "b_value" {
		t.Errorf("getRealValue(${ENV_B:default}) = %q, want b_value", got)
	}
	// 默认值为空串
	if got := getRealValue("${NOT_EXIST:}", em); got != "" {
		t.Errorf("getRealValue(${NOT_EXIST:}) = %q, want empty", got)
	}
	// 修复 ④：非法输入不得 panic，且原样返回
	for _, bad := range []string{"${", "${:", "${:}", "${}", "${x", "$:{", "plain"} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("getRealValue(%q) panicked: %v", bad, r)
				}
			}()
			if got := getRealValue(bad, em); got != bad {
				t.Errorf("getRealValue(%q) = %q, want raw input %q", bad, got, bad)
			}
		}()
	}
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

func Test_LoadProperties(t *testing.T) {
	content := "" +
		"# 注释行\n" +
		"! 叹号注释行\n" +
		"\n" +
		"spring.datasource.url= jdbc:postgresql://localhost:5432/testdb\n" +
		"spring.datasource.username = root\n" +
		"key.with.colon: value:with:colons\n" +
		"no.separator\n" +
		"=no.key\n"
	fp := filepath.Join(t.TempDir(), "test.properties")
	if err := os.WriteFile(fp, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	m := LoadProperties(fp)

	if len(m) != 3 {
		t.Errorf("expected 3 entries, got %d: %v", len(m), m)
	}
	if m["spring.datasource.url"] != "jdbc:postgresql://localhost:5432/testdb" {
		t.Errorf("url parsed wrong: %q", m["spring.datasource.url"])
	}
	if m["spring.datasource.username"] != "root" {
		t.Errorf("username parsed wrong: %q", m["spring.datasource.username"])
	}
	if m["key.with.colon"] != "value:with:colons" {
		t.Errorf("colon-separated key parsed wrong: %q", m["key.with.colon"])
	}

	// 不存在的文件返回空 map，不报错
	if got := LoadProperties(filepath.Join(t.TempDir(), "no-such-file.properties")); len(got) != 0 {
		t.Errorf("expected empty map for missing file, got %v", got)
	}
}
