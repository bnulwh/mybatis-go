package orm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

// P1-6：namespace → 数据源静态绑定。

type NsRow struct {
	Id   int
	Name string
}

type NsUserMapper struct {
	BaseMapper
	SelectAll    func() ([]NsRow, error)
	InsertRow    func(name string) (int64, error)
	InsertRowCtx func(ctx context.Context, name string) (int64, error)
	SelectByName func(name string) ([]NsRow, error)
}

type NsOrderMapper struct {
	BaseMapper
	SelectAll    func() ([]NsRow, error)
	InsertRow    func(name string) (int64, error)
	InsertRowCtx func(ctx context.Context, name string) (int64, error)
}

const nsUserMapperXML = `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="com.demo.mapper.NsUserMapper">
  <resultMap id="nsRow" type="NsRow">
    <id column="id" jdbcType="INTEGER" property="id" />
    <result column="name" jdbcType="VARCHAR" property="name" />
  </resultMap>
  <select id="selectAll" resultMap="nsRow">select id, name from t_ns order by id</select>
  <select id="selectByName" resultMap="nsRow">select id, name from t_ns where name = #{name} order by id</select>
  <insert id="insertRow">
    insert into t_ns(name) values (#{name})
  </insert>
  <insert id="insertRowCtx">
    insert into t_ns(name) values (#{name})
  </insert>
</mapper>`

const nsOrderMapperXML = `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="com.demo.mapper.NsOrderMapper">
  <resultMap id="nsRow" type="NsRow">
    <id column="id" jdbcType="INTEGER" property="id" />
    <result column="name" jdbcType="VARCHAR" property="name" />
  </resultMap>
  <select id="selectAll" resultMap="nsRow">select id, name from t_ns order by id</select>
  <insert id="insertRow">
    insert into t_ns(name) values (#{name})
  </insert>
  <insert id="insertRowCtx">
    insert into t_ns(name) values (#{name})
  </insert>
</mapper>`

// initNamespaceSqlite 初始化两个 SQLite 数据源（default / orderdb），并把
// com.demo.mapper.NsOrderMapper 绑定到 orderdb；两张库各自建表并写入不同数据。
func initNamespaceSqlite(t *testing.T, bindOrder bool) {
	t.Helper()
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatalf("create mapper dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "UserMapper.xml"), []byte(nsUserMapperXML), 0644); err != nil {
		t.Fatalf("write user mapper failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "NsOrderMapper.xml"), []byte(nsOrderMapperXML), 0644); err != nil {
		t.Fatalf("write order mapper failed: %v", err)
	}
	cm := map[string]string{
		"spring.datasource.url":           "jdbc:sqlite:" + filepath.Join(dir, "default.db"),
		"mybatis.datasources":             "orderdb",
		"spring.datasource.orderdb.url":   "jdbc:sqlite:" + filepath.Join(dir, "order.db"),
		"mybatis.mapper-locations":        xmlDir,
		"mybatis.datasource.bind.orderdb": "com.demo.mapper.NsOrderMapper",
	}
	if !bindOrder {
		delete(cm, "mybatis.datasource.bind.orderdb")
	}
	if err := InitializeDataSourcesFromSettings(cm); err != nil {
		t.Fatalf("initialize datasources failed: %v", err)
	}
	RegisterModel(new(NsRow))
	for _, name := range []string{"default", "orderdb"} {
		db, err := GetDataSource(name)
		if err != nil {
			t.Fatalf("get datasource %v failed: %v", name, err)
		}
		if _, err := db.ExecContext(context.Background(),
			`CREATE TABLE t_ns (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
			t.Fatalf("create table on %v failed: %v", name, err)
		}
		if _, err := db.ExecContext(context.Background(),
			`INSERT INTO t_ns(name) VALUES (?)`, name+"-row"); err != nil {
			t.Fatalf("seed %v failed: %v", name, err)
		}
	}
	if err := RegisterMapper(new(NsUserMapper)); err != nil {
		t.Fatalf("register user mapper failed: %v", err)
	}
	if err := RegisterMapper(new(NsOrderMapper)); err != nil {
		t.Fatalf("register order mapper failed: %v", err)
	}
}

// Test_NamespaceDataSource_Routing 不同 namespace 的语句各自命中绑定的数据源，
// 无需 UseDataSource 全局切换，且 gDbConn 保持默认。
func Test_NamespaceDataSource_Routing(t *testing.T) {
	initNamespaceSqlite(t, true)
	defer Close()

	user := NewMapper("NsUserMapper").(NsUserMapper)
	order := NewMapper("NsOrderMapper").(NsOrderMapper)

	urs, err := user.SelectAll()
	if err != nil {
		t.Fatalf("user select failed: %v", err)
	}
	if len(urs) != 1 || urs[0].Name != "default-row" {
		t.Errorf("user namespace should hit default datasource, got %v", urs)
	}
	ors, err := order.SelectAll()
	if err != nil {
		t.Fatalf("order select failed: %v", err)
	}
	if len(ors) != 1 || ors[0].Name != "orderdb-row" {
		t.Errorf("order namespace should hit orderdb, got %v", ors)
	}
	// 全局活跃数据源未被切换
	if GetActiveDataSource() != mustDataSource(t, "default") {
		t.Error("active datasource should stay default (no global switching)")
	}

	// 写入也按 namespace 落库
	if _, err := order.InsertRow("order-new"); err != nil {
		t.Fatalf("order insert failed: %v", err)
	}
	if _, err := user.InsertRow("user-new"); err != nil {
		t.Fatalf("user insert failed: %v", err)
	}
	if n := countRows(t, "default"); n != 2 {
		t.Errorf("default datasource row count = %d, want 2", n)
	}
	if n := countRows(t, "orderdb"); n != 2 {
		t.Errorf("orderdb row count = %d, want 2", n)
	}
}

// Test_NamespaceDataSource_Concurrent 并发访问两个 namespace 不会互相串库
// （旧方案 UseDataSource 全局切换在并发下会互相劫持）。
func Test_NamespaceDataSource_Concurrent(t *testing.T) {
	initNamespaceSqlite(t, true)
	defer Close()

	user := NewMapper("NsUserMapper").(NsUserMapper)
	order := NewMapper("NsOrderMapper").(NsOrderMapper)

	var wg sync.WaitGroup
	errs := make(chan string, 64)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if urs, err := user.SelectAll(); err != nil || len(urs) != 1 || urs[0].Name != "default-row" {
					errs <- "user namespace cross-routed"
					return
				}
				if ors, err := order.SelectAll(); err != nil || len(ors) != 1 || ors[0].Name != "orderdb-row" {
					errs <- "order namespace cross-routed"
					return
				}
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for msg := range errs {
		t.Error(msg)
	}
}

// Test_NamespaceDataSource_Bindings 绑定匹配（精确优先 / 最长前缀）、解绑与配置校验。
func Test_NamespaceDataSource_Bindings(t *testing.T) {
	initNamespaceSqlite(t, true)
	defer Close()

	bindings := NamespaceDataSourceBindings()
	if bindings["com.demo.mapper.nsordermapper"] != "orderdb" {
		t.Errorf("config binding not applied, got %v", bindings)
	}
	if err := BindNamespaceDataSource("com.demo.mapper", "default"); err != nil {
		t.Errorf("bind prefix failed: %v", err)
	}
	if b := matchNamespaceBinding("com.demo.mapper.NsOrderMapper"); b == nil || b.dsName != "orderdb" {
		t.Errorf("exact/longest match should pick orderdb, got %v", b)
	}
	if b := matchNamespaceBinding("com.demo.mapper.NsDeptMapper"); b == nil || b.dsName != "default" {
		t.Errorf("prefix match should pick default, got %v", b)
	}
	if b := matchNamespaceBinding("other.namespace.Mapper"); b != nil {
		t.Errorf("unbound namespace should not match, got %v", b)
	}
	if UnbindNamespaceDataSource("com.demo/mapper") {
		t.Error("unbind of non-existent pattern should return false")
	}
	if !UnbindNamespaceDataSource("com.demo.mapper") {
		t.Error("unbind of existing prefix should return true")
	}
	if UnbindNamespaceDataSource("com.demo.mapper") {
		t.Error("second unbind of the same prefix should return false")
	}
	if err := BindNamespaceDataSource("com.demo.mapper", "no_such_ds"); err == nil {
		t.Error("binding to unknown datasource should fail")
	}
	if err := BindNamespaceDataSource("", "default"); err == nil {
		t.Error("empty namespace pattern should fail")
	}

	// 配置里绑定不存在的数据源 → 初始化期报错
	cm := map[string]string{
		"spring.datasource.url":          "jdbc:sqlite:" + filepath.Join(t.TempDir(), "x.db"),
		"mybatis.datasource.bind.nosuch": "com.demo.mapper.Foo",
	}
	err := InitializeDataSourcesFromSettings(cm)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("init with unknown bound datasource should fail, got %v", err)
	}
}

// Test_NamespaceDataSource_WithTxForNamespace 事务按 namespace 绑定数据源开启，
// 回滚只影响该数据源。
func Test_NamespaceDataSource_WithTxForNamespace(t *testing.T) {
	initNamespaceSqlite(t, true)
	defer Close()

	order := NewMapper("NsOrderMapper").(NsOrderMapper)
	wantErr := context.Canceled
	err := WithTxForNamespace(context.Background(), "com.demo.mapper.NsOrderMapper", func(ctx context.Context) error {
		if _, e := order.InsertRowCtx(ctx, "tx-row"); e != nil {
			t.Errorf("insert in tx failed: %v", e)
			return e
		}
		return wantErr
	})
	if err != wantErr {
		t.Errorf("WithTxForNamespace should return fn error, got %v", err)
	}
	if n := countRows(t, "orderdb"); n != 1 {
		t.Errorf("rollback should keep orderdb at 1 row, got %d", n)
	}
	if n := countRows(t, "default"); n != 1 {
		t.Errorf("default datasource must be untouched, got %d", n)
	}

	// 提交路径
	err = WithTxForNamespace(context.Background(), "com.demo.mapper.NsOrderMapper", func(ctx context.Context) error {
		_, e := order.InsertRowCtx(ctx, "tx-row-2")
		return e
	})
	if err != nil {
		t.Errorf("commit path failed: %v", err)
	}
	if n := countRows(t, "orderdb"); n != 2 {
		t.Errorf("commit should make orderdb 2 rows, got %d", n)
	}
}

// Test_NamespaceDataSource_ReadWriteSplitBound 绑定的数据源不参与默认数据源的副本路由。
func Test_NamespaceDataSource_ReadWriteSplitBound(t *testing.T) {
	initNamespaceSqlite(t, true)
	defer Close()
	defer SetReadWriteSplitting(false)

	SetReadWriteSplitting(true)
	SetReplicaNames([]string{"default"})
	ctx := routeNamespaceDB(context.Background(), "com.demo.mapper.NsOrderMapper")
	bound := routedDB(ctx)
	routed := routedDB(routeReadDB(ctx))
	if bound == nil || routed != bound {
		t.Errorf("bound datasource should not be replaced by replica routing, bound=%v routed=%v", bound, routed)
	}
}

func mustDataSource(t *testing.T, name string) *DB {
	t.Helper()
	db, err := GetDataSource(name)
	if err != nil {
		t.Fatalf("get datasource %v failed: %v", name, err)
	}
	return db
}

func countRows(t *testing.T, dsName string) int {
	t.Helper()
	db := mustDataSource(t, dsName)
	rows, err := db.QueryContext(context.Background(), `SELECT count(*) FROM t_ns`)
	if err != nil {
		t.Fatalf("count on %v failed: %v", dsName, err)
	}
	defer rows.Close()
	if !rows.Next() {
		return 0
	}
	var n int
	if err := rows.Scan(&n); err != nil {
		t.Fatalf("scan count failed: %v", err)
	}
	return n
}
