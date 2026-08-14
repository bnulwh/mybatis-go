package orm

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func Test_parseMultiDatabaseConfig(t *testing.T) {
	cm := map[string]string{
		"spring.datasource.url":             "jdbc:sqlite:default.db",
		"mybatis.datasources":               "db2, db3, no_config",
		"spring.datasource.db2.url":         "jdbc:postgresql://localhost:5432/db2",
		"spring.datasource.db2.username":    "root",
		"spring.datasource.db2.password":    "123456",
		"spring.datasource.db3.url":         "jdbc:mysql://localhost:3306/db3",
		"spring.datasource.db3.username":    "root",
		"spring.datasource.db3.password":    "123456",
		"spring.datasource.db3.max-idle":    "5",
		"spring.datasource.db3.max-open":    "10",
		"spring.datasource.db3.max-timeout": "60",
	}
	configs := parseMultiDatabaseConfig(cm)
	if len(configs) != 3 {
		t.Errorf("expected 3 datasources (no_config skipped), got %d: %v", len(configs), configs)
	}
	// 默认 SQLite
	if got := configs[defaultDataSourceName].GenerateDSN(); got != "default.db?_loc=auto" {
		t.Errorf("default dsn wrong: %q", got)
	}
	// db2 PostgreSQL
	want2 := "host=localhost port=5432 user=root password=123456 dbname=db2 sslmode=disable"
	if got := configs["db2"].GenerateDSN(); got != want2 {
		t.Errorf("db2 dsn wrong: %q", got)
	}
	// db3 MySQL
	if got := configs["db3"].GenerateDSN(); got != "root:123456@tcp(localhost:3306)/db3" {
		t.Errorf("db3 dsn wrong: %q", got)
	}
	// 池参数独立
	if configs["db3"].MaxIdle != 5 || configs["db3"].MaxOpen != 10 || configs["db3"].MaxTimeout != 60 {
		t.Errorf("db3 pool settings wrong: %+v", configs["db3"])
	}
}

func initMultiSqlite(t *testing.T) (string, string) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatalf("create mapper dir failed: %v", err)
	}
	db1 := filepath.Join(dir, "db1.db")
	db2 := filepath.Join(dir, "db2.db")
	cm := map[string]string{
		"spring.datasource.url":     "jdbc:sqlite:" + db1,
		"mybatis.datasources":       "db2",
		"spring.datasource.db2.url": "jdbc:sqlite:" + db2,
		"mybatis.mapper-locations":  xmlDir,
	}
	if err := InitializeDataSourcesFromSettings(cm); err != nil {
		t.Fatalf("initialize multi datasource failed: %v", err)
	}
	return db1, db2
}

func Test_UseDataSource(t *testing.T) {
	initMultiSqlite(t)
	defer Close()

	if names := GetDataSourceNames(); len(names) != 2 {
		t.Errorf("expected 2 datasource names, got %v", names)
	}
	if _, err := GetDataSource("db2"); err != nil {
		t.Errorf("GetDataSource(db2) failed: %v", err)
	}

	// 默认数据源（default）
	if _, err := Execute(`CREATE TABLE t1 (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Errorf("create table on default failed: %v", err)
		return
	}
	if _, err := Execute(`INSERT INTO t1 (name) VALUES (?)`, "in_default"); err != nil {
		t.Errorf("insert on default failed: %v", err)
		return
	}

	// 切换到 db2
	if err := UseDataSource("db2"); err != nil {
		t.Errorf("UseDataSource(db2) failed: %v", err)
		return
	}
	if _, err := Execute(`CREATE TABLE t1 (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Errorf("create table on db2 failed: %v", err)
		return
	}
	// db2 应无数据（隔离性）
	res, err := Query(`SELECT count(*) as c FROM t1`)
	if err != nil {
		t.Errorf("query db2 failed: %v", err)
		return
	}
	var n int
	if _, err := fmt.Sscanf(fmt.Sprintf("%v", res[0]["c"]), "%d", &n); err != nil || n != 0 {
		t.Errorf("db2 should be empty, count=%v err=%v", res[0]["c"], err)
	}

	// 切回 default，数据应还在
	if err := UseDataSource(defaultDataSourceName); err != nil {
		t.Errorf("UseDataSource(default) failed: %v", err)
		return
	}
	res, err = Query(`SELECT count(*) as c FROM t1`)
	if err != nil {
		t.Errorf("query default failed: %v", err)
		return
	}
	if _, err := fmt.Sscanf(fmt.Sprintf("%v", res[0]["c"]), "%d", &n); err != nil || n != 1 {
		t.Errorf("default should have 1 row, count=%v err=%v", res[0]["c"], err)
	}

	// 不存在的数据源
	if err := UseDataSource("nope"); err == nil {
		t.Error("UseDataSource(nope) should fail")
	}
}

func Test_AddDataSource(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatalf("create mapper dir failed: %v", err)
	}
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + filepath.Join(dir, "default.db"),
		"mybatis.mapper-locations": xmlDir,
	}
	if err := InitializeDataSourcesFromSettings(cm); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	defer Close()

	// 编程方式添加命名数据源
	if err := AddDataSource("secondary", "sqlite", "", 0, "", "", filepath.Join(dir, "secondary.db")); err != nil {
		t.Fatalf("AddDataSource failed: %v", err)
	}
	// 重复添加报错
	if err := AddDataSource("secondary", "sqlite", "", 0, "", "", filepath.Join(dir, "x.db")); err == nil {
		t.Error("AddDataSource duplicate should fail")
	}
	// 非法名称
	if err := AddDataSource("default", "sqlite", "", 0, "", "", filepath.Join(dir, "x.db")); err == nil {
		t.Error("AddDataSource name 'default' should be rejected")
	}
	if err := AddDataSource("", "sqlite", "", 0, "", "", filepath.Join(dir, "x.db")); err == nil {
		t.Error("AddDataSource empty name should be rejected")
	}

	// 新数据源可用且与默认隔离
	if err := UseDataSource("secondary"); err != nil {
		t.Fatalf("UseDataSource(secondary) failed: %v", err)
	}
	if _, err := Execute(`CREATE TABLE t1 (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Errorf("create table on secondary failed: %v", err)
		return
	}
	if err := UseDataSource(defaultDataSourceName); err != nil {
		t.Fatalf("UseDataSource(default) failed: %v", err)
	}
	// default 上 t1 不存在（隔离）
	if _, err := Query(`SELECT count(*) as c FROM t1`); err == nil {
		t.Error("default should not have table t1")
	}

	// ReConnectDataSource
	if err := ReConnectDataSource("secondary"); err != nil {
		t.Errorf("ReConnectDataSource(secondary) failed: %v", err)
	}
	if err := ReConnectDataSource("nope"); err == nil {
		t.Error("ReConnectDataSource(nope) should fail")
	}
}
