package orm

import (
	"strings"
	"testing"
)

func Test_parseDatabaseType(t *testing.T) {
	r, err := parseDatabaseType("Mysql")
	if r != MySqlDb || err != nil {
		t.Error("test parseDatabaseType failed.")
	}
	r1, err := parseDatabaseType("POSTGRES")
	if r1 != PostgresDb || err != nil {
		t.Error("test parseDatabaseType failed.")
	}
	r2, err := parseDatabaseType("test")
	if r2 != "" || err == nil {
		t.Error("test parseDatabaseType failed.")
	}
	r3, err := parseDatabaseType("kingbase8")
	if r3 != KingbaseDb || err != nil {
		t.Error("test parseDatabaseType kingbase8 failed.")
	}
	r4, err := parseDatabaseType("Kingbase")
	if r4 != KingbaseDb || err != nil {
		t.Error("test parseDatabaseType kingbase failed.")
	}

}

func Test_parseAddr(t *testing.T) {
	mp := map[string]string{}
	tp, host, port, db, err := parseAddr(mp)
	if tp != "" || host != "" || port != 0 || db != "" || err == nil {
		t.Error("test parseAddr failed.")
	}
	mp["spring.datasource.url"] = "test"
	tp1, host1, port1, db1, err := parseAddr(mp)
	if tp1 != "" || host1 != "" || port1 != 0 || db1 != "" || err == nil {
		t.Error("test parseAddr failed.")
	}
	mp["spring.datasource.url"] = "jdbc:test://sss"
	tp2, host2, port2, db2, err := parseAddr(mp)
	if tp2 != "" || host2 != "" || port2 != 0 || db2 != "" || err == nil {
		t.Error("test parseAddr failed.")
	}
	mp["spring.datasource.url"] = "jdbc:mysql://a.bc.d.e:33/xxxx"
	tp3, host3, port3, db3, err := parseAddr(mp)
	if tp3 != "mysql" || host3 != "a.bc.d.e" || port3 != 33 || db3 != "xxxx" || err != nil {
		t.Error("test parseAddr failed.")
	}
	mp["spring.datasource.url"] = "jdbc:kingbase8://10.1.2.3:54321/testdb"
	tp4, host4, port4, db4, err := parseAddr(mp)
	if tp4 != "kingbase8" || host4 != "10.1.2.3" || port4 != 54321 || db4 != "testdb" || err != nil {
		t.Error("test parseAddr kingbase8 failed.")
	}
}

func Test_parseAddr_ipv6(t *testing.T) {
	mp := map[string]string{"spring.datasource.url": "jdbc:postgresql://[2001:db8::1]:5432/testdb"}
	tp, host, port, db, err := parseAddr(mp)
	if tp != "postgresql" || host != "2001:db8::1" || port != 5432 || db != "testdb" || err != nil {
		t.Errorf("test parseAddr ipv6 failed: type=%v host=%v port=%v db=%v err=%v", tp, host, port, db, err)
	}
	mp["spring.datasource.url"] = "jdbc:mysql://[fe80::1]:3306/my.db"
	tp2, host2, port2, db2, err2 := parseAddr(mp)
	if tp2 != "mysql" || host2 != "fe80::1" || port2 != 3306 || db2 != "my.db" || err2 != nil {
		t.Errorf("test parseAddr ipv6 mysql failed: type=%v host=%v port=%v db=%v err=%v", tp2, host2, port2, db2, err2)
	}
}

func Test_generateConn_ipv6(t *testing.T) {
	// MySQL：IPv6 地址必须带方括号
	want := "root:pwd@tcp([2001:db8::1]:3306)/mydb?parseTime=true&loc=Local"
	if got := newDatabaseConfig("mysql", "2001:db8::1", 3306, "root", "pwd", "mydb").GenerateDSN(); got != want {
		t.Errorf("mysql ipv6 dsn failed, got: %q want: %q", got, want)
	}
	// 已带方括号的 host 不重复加
	if got := newDatabaseConfig("mysql", "[2001:db8::1]", 3306, "root", "pwd", "mydb").GenerateDSN(); got != want {
		t.Errorf("mysql bracketed ipv6 dsn failed, got: %q want: %q", got, want)
	}
	// PostgreSQL：host= 直传即可，驱动内部 JoinHostPort 处理方括号
	want3 := "host=2001:db8::1 port=5432 user=root password=pwd dbname=testdb sslmode=disable"
	if got := newDatabaseConfig("postgres", "2001:db8::1", 5432, "root", "pwd", "testdb").GenerateDSN(); got != want3 {
		t.Errorf("postgres ipv6 dsn failed, got: %q want: %q", got, want3)
	}
	// 域名/IPv4 行为不变
	if got := newDatabaseConfig("mysql", "a.bc.d.e", 33, "root", "pwd", "mydb").GenerateDSN(); got != "root:pwd@tcp(a.bc.d.e:33)/mydb?parseTime=true&loc=Local" {
		t.Errorf("mysql ipv4 dsn failed, got: %q", got)
	}
}

// Test_generateConn_MySQLParseTime MySQL DSN 必须带 parseTime=true（DATETIME 列才能 Scan 到 time.Time）
func Test_generateConn_MySQLParseTime(t *testing.T) {
	dsn := newDatabaseConfig("mysql", "localhost", 3306, "root", "123456", "testdb").GenerateDSN()
	want := "root:123456@tcp(localhost:3306)/testdb?parseTime=true&loc=Local"
	if dsn != want {
		t.Errorf("mysql dsn failed, got: %q want: %q", dsn, want)
	}
	// Name 已带查询参数时用 & 拼接
	dsn2 := newDatabaseConfig("mysql", "localhost", 3306, "root", "123456", "testdb?charset=utf8mb4").GenerateDSN()
	if !strings.Contains(dsn2, "/testdb?charset=utf8mb4&parseTime=true&loc=Local") {
		t.Errorf("mysql dsn with existing params failed, got: %q", dsn2)
	}
	// 其他方言 DSN 不受影响
	pg := newDatabaseConfig("postgres", "localhost", 5432, "root", "123456", "testdb").GenerateDSN()
	if strings.Contains(pg, "parseTime") {
		t.Errorf("postgres dsn should not contain parseTime, got: %q", pg)
	}
	sqlite := newDatabaseConfig("sqlite", "", 0, "", "", "test.db").GenerateDSN()
	if sqlite != "test.db?_loc=auto" {
		t.Errorf("sqlite dsn failed, got: %q", sqlite)
	}
}

// Test_Config_CustomDSN 自定义 DSN（cfg.DSN）优先于自动生成，各方言 dialector 均支持
func Test_Config_CustomDSN(t *testing.T) {
	cfg := newDatabaseConfig("mysql", "localhost", 3306, "root", "123456", "testdb")
	cfg.DSN = "root:pwd@tcp(10.0.0.1:3307)/custom?parseTime=true&charset=utf8mb4&loc=Local"
	d := NewMySqlDialector(cfg)
	if d.DSN != cfg.DSN {
		t.Errorf("mysql custom DSN not honored, got: %q", d.DSN)
	}
	cfg2 := newDatabaseConfig("postgres", "localhost", 5432, "root", "123456", "testdb")
	cfg2.DSN = "host=10.0.0.2 port=5433 user=root password=pwd dbname=custom sslmode=disable"
	if d2 := NewPostgresDialector(cfg2); d2.DSN != cfg2.DSN {
		t.Errorf("postgres custom DSN not honored, got: %q", d2.DSN)
	}
	cfg3 := newDatabaseConfig("sqlite", "", 0, "", "", "test.db")
	cfg3.DSN = "/abs/custom.db?_loc=auto&_pragma=busy_timeout(5000)"
	if d3 := NewSqliteDialector(cfg3); d3.DSN != cfg3.DSN {
		t.Errorf("sqlite custom DSN not honored, got: %q", d3.DSN)
	}
}

// Test_schemaFromURL 从 JDBC URL query 参数提取 schema（currentSchema / search_path / schema）
func Test_schemaFromURL(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"jdbc:postgresql://localhost:5432/testdb?currentSchema=myschema", "myschema"},
		{"jdbc:postgresql://localhost:5432/testdb?search_path=myschema", "myschema"},
		{"jdbc:postgresql://localhost:5432/testdb?schema=myschema", "myschema"},
		// search_path 支持逗号多值
		{"jdbc:postgresql://localhost:5432/testdb?search_path=a,b", "a,b"},
		// URL 编码值
		{"jdbc:postgresql://localhost:5432/testdb?currentSchema=my%20schema", "my schema"},
		// 大小写不敏感
		{"jdbc:postgresql://localhost:5432/testdb?currentSchema=MySchema", "MySchema"},
		// 无关 query 参数不影响
		{"jdbc:postgresql://localhost:5432/testdb?useUnicode=true&characterEncoding=utf-8", ""},
		{"jdbc:postgresql://localhost:5432/testdb?useSSL=false&currentSchema=scm", "scm"},
		// 无 query 部分 / 无 URL
		{"jdbc:postgresql://localhost:5432/testdb", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := schemaFromURL(c.url); got != c.want {
			t.Errorf("schemaFromURL(%q) = %q, want %q", c.url, got, c.want)
		}
	}
}

// Test_parseSchema 配置优先级：spring.datasource.schema 键 > URL query 参数
func Test_parseSchema(t *testing.T) {
	// 键优先于 URL 参数
	mp := map[string]string{
		"spring.datasource.schema": "from_key",
		"spring.datasource.url":    "jdbc:postgresql://localhost:5432/testdb?currentSchema=from_url",
	}
	if got := parseSchema(mp); got != "from_key" {
		t.Errorf("parseSchema key precedence failed, got: %q", got)
	}
	// 无键时从 URL 提取
	mp2 := map[string]string{"spring.datasource.url": "jdbc:postgresql://localhost:5432/testdb?search_path=scm2"}
	if got := parseSchema(mp2); got != "scm2" {
		t.Errorf("parseSchema from url failed, got: %q", got)
	}
	// 均未配置返回空
	if got := parseSchema(map[string]string{}); got != "" {
		t.Errorf("parseSchema default should be empty, got: %q", got)
	}
	// 键为空串时回退 URL
	mp3 := map[string]string{
		"spring.datasource.schema": "  ",
		"spring.datasource.url":    "jdbc:postgresql://localhost:5432/testdb?currentSchema=scm3",
	}
	if got := parseSchema(mp3); got != "scm3" {
		t.Errorf("parseSchema blank key fallback failed, got: %q", got)
	}
}

// Test_generateConn_schema PG/Kingbase 配置 schema 时 DSN 追加 search_path，其余方言不受影响
func Test_generateConn_schema(t *testing.T) {
	pg := newDatabaseConfig("postgres", "localhost", 5432, "root", "123456", "testdb")
	pg.Setting.Schema = "myschema"
	want := "host=localhost port=5432 user=root password=123456 dbname=testdb sslmode=disable search_path=myschema"
	if got := pg.GenerateDSN(); got != want {
		t.Errorf("postgres dsn with schema failed, got: %q want: %q", got, want)
	}
	// 未配置 schema 时 DSN 与历史一致，不含 search_path
	pg2 := newDatabaseConfig("postgres", "localhost", 5432, "root", "123456", "testdb")
	if got := pg2.GenerateDSN(); got != "host=localhost port=5432 user=root password=123456 dbname=testdb sslmode=disable" {
		t.Errorf("postgres dsn without schema failed, got: %q", got)
	}
	// Kingbase 同 PG 格式
	kb := newDatabaseConfig("kingbase", "10.0.0.9", 54321, "system", "pwd", "testdb")
	kb.Setting.Schema = "scm"
	if got := kb.GenerateDSN(); !strings.Contains(got, "search_path=scm") {
		t.Errorf("kingbase dsn with schema failed, got: %q", got)
	}
	// MySQL DSN 不受 schema 影响（schema 即库名，由表格查询侧处理）
	my := newDatabaseConfig("mysql", "localhost", 3306, "root", "123456", "testdb")
	my.Setting.Schema = "otherdb"
	if got := my.GenerateDSN(); got != "root:123456@tcp(localhost:3306)/testdb?parseTime=true&loc=Local" {
		t.Errorf("mysql dsn should ignore schema, got: %q", got)
	}
	// SQLite 不受影响
	sq := newDatabaseConfig("sqlite", "", 0, "", "", "test.db")
	sq.Setting.Schema = "s"
	if got := sq.GenerateDSN(); got != "test.db?_loc=auto" {
		t.Errorf("sqlite dsn should ignore schema, got: %q", got)
	}
}

// Test_effectiveSchema 表结构查询的 schema 取值规则
func Test_effectiveSchema(t *testing.T) {
	// PG/Kingbase：默认 public
	pg := newDatabaseConfig("postgres", "h", 1, "u", "p", "db")
	if got := pg.Setting.effectiveSchema("db"); got != "public" {
		t.Errorf("pg effectiveSchema default failed, got: %q", got)
	}
	pg.Setting.Schema = "scm"
	if got := pg.Setting.effectiveSchema("db"); got != "scm" {
		t.Errorf("pg effectiveSchema configured failed, got: %q", got)
	}
	// MySQL：显式 schema 优先，否则回退库名
	my := newDatabaseConfig("mysql", "h", 1, "u", "p", "mydb")
	if got := my.Setting.effectiveSchema("mydb"); got != "mydb" {
		t.Errorf("mysql effectiveSchema fallback failed, got: %q", got)
	}
	my.Setting.Schema = "otherdb"
	if got := my.Setting.effectiveSchema("mydb"); got != "otherdb" {
		t.Errorf("mysql effectiveSchema override failed, got: %q", got)
	}
	// SQLite：无 schema
	sq := newDatabaseConfig("sqlite", "", 0, "", "", "t.db")
	if got := sq.Setting.effectiveSchema("t.db"); got != "" {
		t.Errorf("sqlite effectiveSchema failed, got: %q", got)
	}
}

// Test_parseDatabaseConfig_schema 整体配置解析：schema 写入 Setting.Schema
func Test_parseDatabaseConfig_schema(t *testing.T) {
	mp := map[string]string{
		"spring.datasource.url":      "jdbc:postgresql://10.1.2.3:5432/testdb?currentSchema=myschema",
		"spring.datasource.username": "root",
		"spring.datasource.password": "123456",
	}
	cfg := parseDatabaseConfig(mp)
	if cfg.Setting.Schema != "myschema" {
		t.Errorf("parseDatabaseConfig schema from url failed, got: %q", cfg.Setting.Schema)
	}
	if cfg.DriverName() != "postgres" {
		t.Errorf("parseDatabaseConfig type failed, got: %q", cfg.DriverName())
	}
	// 显式键覆盖 URL 参数
	mp["spring.datasource.schema"] = "explicit"
	cfg2 := parseDatabaseConfig(mp)
	if cfg2.Setting.Schema != "explicit" {
		t.Errorf("parseDatabaseConfig schema key precedence failed, got: %q", cfg2.Setting.Schema)
	}
}
