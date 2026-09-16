package orm

import (
	"strings"
	"testing"

	"github.com/bnulwh/mybatis-go/orm/dialector"
)

func Test_parseDatabaseType(t *testing.T) {
	r, err := dialector.ParseDatabaseType("Mysql")
	if r != MySqlDb || err != nil {
		t.Error("test parseDatabaseType failed.")
	}
	r1, err := dialector.ParseDatabaseType("POSTGRES")
	if r1 != PostgresDb || err != nil {
		t.Error("test parseDatabaseType failed.")
	}
	r2, err := dialector.ParseDatabaseType("test")
	if r2 != "" || err == nil {
		t.Error("test parseDatabaseType failed.")
	}
	r3, err := dialector.ParseDatabaseType("kingbase8")
	if r3 != KingbaseDb || err != nil {
		t.Error("test parseDatabaseType kingbase8 failed.")
	}
	r4, err := dialector.ParseDatabaseType("Kingbase")
	if r4 != KingbaseDb || err != nil {
		t.Error("test parseDatabaseType kingbase failed.")
	}
	r5, err := dialector.ParseDatabaseType("tidb")
	if r5 != TiDBDb || err != nil {
		t.Error("test parseDatabaseType tidb failed.")
	}
	r6, err := dialector.ParseDatabaseType("tdsql")
	if r6 != TDSQLDb || err != nil {
		t.Error("test parseDatabaseType tdsql failed.")
	}
	r7, err := dialector.ParseDatabaseType("polardb")
	if r7 != PolarDBMyDb || err != nil {
		t.Error("test parseDatabaseType polardb failed.")
	}
	r8, err := dialector.ParseDatabaseType("opengauss")
	if r8 != OpenGaussDb || err != nil {
		t.Error("test parseDatabaseType opengauss failed.")
	}
	r9, err := dialector.ParseDatabaseType("gaussdb")
	if r9 != GaussDBDb || err != nil {
		t.Error("test parseDatabaseType gaussdb failed.")
	}
	r10, err := dialector.ParseDatabaseType("highgo")
	if r10 != HighGoDb || err != nil {
		t.Error("test parseDatabaseType highgo failed.")
	}
	r11, err := dialector.ParseDatabaseType("vastbase")
	if r11 != VastbaseDb || err != nil {
		t.Error("test parseDatabaseType vastbase failed.")
	}
	r12, err := dialector.ParseDatabaseType("oceanbase")
	if r12 != OceanBaseDb || err != nil {
		t.Error("test parseDatabaseType oceanbase failed.")
	}
	r13, err := dialector.ParseDatabaseType("oceanbase-oracle")
	if r13 != OceanBaseOracleDb || err != nil {
		t.Error("test parseDatabaseType oceanbase-oracle failed.")
	}
	r14, err := dialector.ParseDatabaseType("dameng")
	if r14 != DamengDb || err != nil {
		t.Error("test parseDatabaseType dameng failed.")
	}
	r15, err := dialector.ParseDatabaseType("dm8")
	if r15 != DamengDb || err != nil {
		t.Error("test parseDatabaseType dm8 failed.")
	}
	r16, err := dialector.ParseDatabaseType("gbase8s")
	if r16 != GBase8sDb || err != nil {
		t.Error("test parseDatabaseType gbase8s failed.")
	}
	r17, err := dialector.ParseDatabaseType("gbase")
	if r17 != GBase8sDb || err != nil {
		t.Error("test parseDatabaseType gbase failed.")
	}
	r18, err := dialector.ParseDatabaseType("sqlserver")
	if r18 != MssqlDb || err != nil {
		t.Error("test parseDatabaseType sqlserver failed.")
	}
	r19, err := dialector.ParseDatabaseType("mssql")
	if r19 != MssqlDb || err != nil {
		t.Error("test parseDatabaseType mssql failed.")
	}
	r20, err := dialector.ParseDatabaseType("oracle")
	if r20 != OracleDb || err != nil {
		t.Error("test parseDatabaseType oracle failed.")
	}
	r21, err := dialector.ParseDatabaseType("db2")
	if r21 != Db2Db || err != nil {
		t.Error("test parseDatabaseType db2 failed.")
	}
	r22, err := dialector.ParseDatabaseType("ibmdb2")
	if r22 != Db2Db || err != nil {
		t.Error("test parseDatabaseType ibmdb2 failed.")
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
	mp["spring.datasource.url"] = "jdbc:tidb://10.1.2.3:4000/testdb"
	tp5, host5, port5, db5, err := parseAddr(mp)
	if tp5 != "tidb" || host5 != "10.1.2.3" || port5 != 4000 || db5 != "testdb" || err != nil {
		t.Error("test parseAddr tidb failed.")
	}
	mp["spring.datasource.url"] = "jdbc:tdsql://10.1.2.3:3306/mydb"
	tp6, host6, port6, db6, err := parseAddr(mp)
	if tp6 != "tdsql" || host6 != "10.1.2.3" || port6 != 3306 || db6 != "mydb" || err != nil {
		t.Error("test parseAddr tdsql failed.")
	}
	mp["spring.datasource.url"] = "jdbc:polardb://10.1.2.3:3306/polardb_test"
	tp7, host7, port7, db7, err := parseAddr(mp)
	if tp7 != "polardb" || host7 != "10.1.2.3" || port7 != 3306 || db7 != "polardb_test" || err != nil {
		t.Error("test parseAddr polardb failed.")
	}
	mp["spring.datasource.url"] = "jdbc:opengauss://10.1.2.3:5432/testdb"
	tp8, host8, port8, db8, err := parseAddr(mp)
	if tp8 != "opengauss" || host8 != "10.1.2.3" || port8 != 5432 || db8 != "testdb" || err != nil {
		t.Error("test parseAddr opengauss failed.")
	}
	mp["spring.datasource.url"] = "jdbc:gaussdb://10.1.2.3:5432/gaussdb_test"
	tp9, host9, port9, db9, err := parseAddr(mp)
	if tp9 != "gaussdb" || host9 != "10.1.2.3" || port9 != 5432 || db9 != "gaussdb_test" || err != nil {
		t.Error("test parseAddr gaussdb failed.")
	}
	mp["spring.datasource.url"] = "jdbc:highgo://10.1.2.3:5432/highgo_test"
	tp10, host10, port10, db10, err := parseAddr(mp)
	if tp10 != "highgo" || host10 != "10.1.2.3" || port10 != 5432 || db10 != "highgo_test" || err != nil {
		t.Error("test parseAddr highgo failed.")
	}
	mp["spring.datasource.url"] = "jdbc:vastbase://10.1.2.3:5432/vastbase_test"
	tp11, host11, port11, db11, err := parseAddr(mp)
	if tp11 != "vastbase" || host11 != "10.1.2.3" || port11 != 5432 || db11 != "vastbase_test" || err != nil {
		t.Error("test parseAddr vastbase failed.")
	}
	mp["spring.datasource.url"] = "jdbc:oceanbase://10.1.2.3:2881/oceanbase_test"
	tp12, host12, port12, db12, err := parseAddr(mp)
	if tp12 != "oceanbase" || host12 != "10.1.2.3" || port12 != 2881 || db12 != "oceanbase_test" || err != nil {
		t.Error("test parseAddr oceanbase failed.")
	}
	mp["spring.datasource.url"] = "jdbc:oceanbase-oracle://10.1.2.3:2881/ob_test"
	tp13, host13, port13, db13, err := parseAddr(mp)
	if tp13 != "oceanbase-oracle" || host13 != "10.1.2.3" || port13 != 2881 || db13 != "ob_test" || err != nil {
		t.Error("test parseAddr oceanbase-oracle failed.")
	}
	mp["spring.datasource.url"] = "jdbc:dameng://10.1.2.3:5236/dameng_test"
	tp14, host14, port14, db14, err := parseAddr(mp)
	if tp14 != "dameng" || host14 != "10.1.2.3" || port14 != 5236 || db14 != "dameng_test" || err != nil {
		t.Error("test parseAddr dameng failed.")
	}
	mp["spring.datasource.url"] = "jdbc:gbase8s://10.1.2.3:9088/gbase_test"
	tp15, host15, port15, db15, err := parseAddr(mp)
	if tp15 != "gbase8s" || host15 != "10.1.2.3" || port15 != 9088 || db15 != "gbase_test" || err != nil {
		t.Error("test parseAddr gbase8s failed.")
	}
	mp["spring.datasource.url"] = "jdbc:sqlserver://10.1.2.3:1433/mssql_test"
	tp16, host16, port16, db16, err := parseAddr(mp)
	if tp16 != "sqlserver" || host16 != "10.1.2.3" || port16 != 1433 || db16 != "mssql_test" || err != nil {
		t.Error("test parseAddr sqlserver failed.")
	}
	mp["spring.datasource.url"] = "jdbc:oracle://10.1.2.3:1521/orcl"
	tp17, host17, port17, db17, err := parseAddr(mp)
	if tp17 != "oracle" || host17 != "10.1.2.3" || port17 != 1521 || db17 != "orcl" || err != nil {
		t.Error("test parseAddr oracle failed.")
	}
	mp["spring.datasource.url"] = "jdbc:db2://10.1.2.3:50000/db2test"
	tp18, host18, port18, db18, err := parseAddr(mp)
	if tp18 != "db2" || host18 != "10.1.2.3" || port18 != 50000 || db18 != "db2test" || err != nil {
		t.Error("test parseAddr db2 failed.")
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
	d := dialector.NewMySqlDialector(cfg)
	if d.DSN() != cfg.DSN {
		t.Errorf("mysql custom DSN not honored, got: %q", d.DSN())
	}
	cfg2 := newDatabaseConfig("postgres", "localhost", 5432, "root", "123456", "testdb")
	cfg2.DSN = "host=10.0.0.2 port=5433 user=root password=pwd dbname=custom sslmode=disable"
	if d2 := dialector.NewPostgresDialector(cfg2); d2.DSN() != cfg2.DSN {
		t.Errorf("postgres custom DSN not honored, got: %q", d2.DSN())
	}
	cfg3 := newDatabaseConfig("sqlite", "", 0, "", "", "test.db")
	cfg3.DSN = "/abs/custom.db?_loc=auto&_pragma=busy_timeout(5000)"
	if d3 := dialector.NewSqliteDialector(cfg3); d3.DSN() != cfg3.DSN {
		t.Errorf("sqlite custom DSN not honored, got: %q", d3.DSN())
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
	pg := newDatabaseConfig("postgres", "h", 1, "u", "p", "db")
	if got := pg.EffectiveSchema(); got != "public" {
		t.Errorf("pg effectiveSchema default failed, got: %q", got)
	}
	pg.Setting.Schema = "scm"
	if got := pg.EffectiveSchema(); got != "scm" {
		t.Errorf("pg effectiveSchema configured failed, got: %q", got)
	}
	my := newDatabaseConfig("mysql", "h", 1, "u", "p", "mydb")
	if got := my.EffectiveSchema(); got != "mydb" {
		t.Errorf("mysql effectiveSchema fallback failed, got: %q", got)
	}
	my.Setting.Schema = "otherdb"
	if got := my.EffectiveSchema(); got != "otherdb" {
		t.Errorf("mysql effectiveSchema override failed, got: %q", got)
	}
	sq := newDatabaseConfig("sqlite", "", 0, "", "", "t.db")
	if got := sq.EffectiveSchema(); got != "" {
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
