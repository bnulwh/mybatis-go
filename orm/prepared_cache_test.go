package orm

import (
	"fmt"
	"testing"
)

// p0-0：预编译语句缓存接通后的回归测试。
// 默认 PreparedStmt=true 时，参数化 SQL 必须真正走 PreparedStmtDB 缓存，
// 无参数 SQL（DDL/静态查询）直接执行且不进入缓存。

func stmtCacheKeys(stmts map[string]*Stmt) []string {
	keys := make([]string, 0, len(stmts))
	for k := range stmts {
		keys = append(keys, k)
	}
	return keys
}

func Test_PreparedStmtCacheWiredUp(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	if _, err := Execute(`CREATE TABLE t_sqlite (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	// 参数化 SQL 应进入预编译缓存
	if _, err := Execute(`INSERT INTO t_sqlite (name) VALUES (?)`, "cache_user"); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	// 相同 SQL 第二次执行应命中缓存，不新增缓存条目
	if _, err := Execute(`INSERT INTO t_sqlite (name) VALUES (?)`, "cache_user2"); err != nil {
		t.Errorf("insert 2 failed: %v", err)
		return
	}
	if gDbConn == nil {
		t.Error("gDbConn is nil")
		return
	}
	v, ok := gDbConn.cacheStore.Load(preparedStmtDBKey)
	if !ok {
		t.Fatal("preparedStmtDB not found in cacheStore")
	}
	ps := v.(*PreparedStmtDB)
	key := "INSERT INTO t_sqlite (name) VALUES (?)"
	if _, ok := ps.Stmts[key]; !ok {
		t.Errorf("expected cached stmt for %q, got keys: %v", key, stmtCacheKeys(ps.Stmts))
	}
	// 无参数 SQL（DDL）不得进入预编译缓存
	if len(ps.Stmts) != 1 {
		t.Errorf("expected only parameterized stmts cached, got %d: %v", len(ps.Stmts), stmtCacheKeys(ps.Stmts))
	}
	// 数据正确性
	res, err := Query(`SELECT count(*) as c FROM t_sqlite`)
	if err != nil {
		t.Errorf("query failed: %v", err)
		return
	}
	if len(res) != 1 || fmt.Sprintf("%v", res[0]["c"]) != "2" {
		t.Errorf("expected 2 rows, got %v", res)
	}
}

// 事务内参数化 SQL 不应使用连接池上的预编译语句（必须走 tx 连接）
func Test_PreparedStmtBypassedInTx(t *testing.T) {
	if !initTxTest(t) {
		return
	}
	defer Close()

	tx, err := Begin()
	if err != nil {
		t.Errorf("Begin failed: %v", err)
		return
	}
	if _, err := Execute(`INSERT INTO t_sqlite (name) VALUES (?)`, "tx_cache"); err != nil {
		t.Errorf("insert in tx failed: %v", err)
		return
	}
	if n := rowCount(t); n != 1 {
		t.Errorf("row count inside tx should be 1, got %d", n)
	}
	if err := tx.Commit(); err != nil {
		t.Errorf("Commit failed: %v", err)
		return
	}
	if n := rowCount(t); n != 1 {
		t.Errorf("row count after commit should be 1, got %d", n)
	}
}

// formatSQL 按方言转换占位符：PG/Kingbase ? -> $n，MySQL/SQLite 保留 ?
func Test_DBFormatSQLDialect(t *testing.T) {
	pg := &DB{Config: &Config{Dialector: NewPostgresDialector(&Config{})}}
	if got := pg.formatSQL("select * from t where a = ? and b = ?", []interface{}{1, 2}); got != "select * from t where a = $1 and b = $2" {
		t.Errorf("postgres formatSQL failed, got: %q", got)
	}
	// 无参数 SQL 原样返回，避免误伤字面量 '?'
	if got := pg.formatSQL("select '?' as q", nil); got != "select '?' as q" {
		t.Errorf("no-arg formatSQL should be unchanged, got: %q", got)
	}
	my := &DB{Config: &Config{Dialector: NewMySqlDialector(&Config{})}}
	if got := my.formatSQL("select * from t where a = ?", []interface{}{1}); got != "select * from t where a = ?" {
		t.Errorf("mysql formatSQL should keep ?, got: %q", got)
	}
	sq := &DB{Config: &Config{Dialector: NewSqliteDialector(&Config{})}}
	if got := sq.formatSQL("select * from t where a = ?", []interface{}{1}); got != "select * from t where a = ?" {
		t.Errorf("sqlite formatSQL should keep ?, got: %q", got)
	}
}

// spring.datasource.prepared-stmt 属性解析（默认开启，可关闭）
func Test_PreparedStmtPropertyParsing(t *testing.T) {
	cm := map[string]string{"spring.datasource.url": "jdbc:sqlite:test.db"}
	cfg := parseDatabaseConfig(cm)
	if !cfg.PreparedStmt {
		t.Error("prepared-stmt should default to true")
	}
	cm["spring.datasource.prepared-stmt"] = "false"
	cfg = parseDatabaseConfig(cm)
	if cfg.PreparedStmt {
		t.Error("prepared-stmt=false should disable prepared stmt cache")
	}
	cm["spring.datasource.prepared-stmt"] = "yes"
	cfg = parseDatabaseConfig(cm)
	if !cfg.PreparedStmt {
		t.Error("prepared-stmt=yes should enable prepared stmt cache")
	}
	cm["spring.datasource.prepared-stmt"] = "bogus"
	cfg = parseDatabaseConfig(cm)
	if !cfg.PreparedStmt {
		t.Error("invalid prepared-stmt value should fall back to default true")
	}
}
