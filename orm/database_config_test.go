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
