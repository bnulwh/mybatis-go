package dialector

import "testing"

func Test_KingbaseDialectorFormatPrepareSQL(t *testing.T) {
	d := NewKingbaseDialector(&testConfig{dbType: KingbaseDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	want := "select * from t where a = $1 and b = $2"
	if got != want {
		t.Errorf("format prepare sql failed, got: %q want: %q", got, want)
	}
}

func Test_KingbaseDialectorName(t *testing.T) {
	d := NewKingbaseDialector(&testConfig{dbType: KingbaseDb})
	if d.Name() != "kingbase" {
		t.Errorf("dialector name failed, got: %q", d.Name())
	}
}

func Test_KingbaseDriverRegistered(t *testing.T) {
	if !isDriverRegistered("kingbase") {
		t.Error("kingbase driver should be registered by init")
	}
}

func Test_PostgresFormatPrepareSQL(t *testing.T) {
	d := NewPostgresDialector(&testConfig{dbType: PostgresDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	want := "select * from t where a = $1 and b = $2"
	if got != want {
		t.Errorf("postgres format prepare sql failed, got: %q want: %q", got, want)
	}
}

func Test_MySqlFormatPrepareSQL(t *testing.T) {
	d := NewMySqlDialector(&testConfig{dbType: MySqlDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	if got != src {
		t.Errorf("mysql format prepare sql should keep ?, got: %q", got)
	}
}

func Test_SqliteFormatPrepareSQL(t *testing.T) {
	d := NewSqliteDialector(&testConfig{dbType: SqliteDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	if got != src {
		t.Errorf("sqlite format prepare sql should keep ?, got: %q", got)
	}
}

func Test_PostgresNeedsReturning(t *testing.T) {
	d := NewPostgresDialector(&testConfig{dbType: PostgresDb})
	if !d.NeedsReturning() {
		t.Error("postgres should need RETURNING")
	}
}

func Test_MySqlNeedsReturning(t *testing.T) {
	d := NewMySqlDialector(&testConfig{dbType: MySqlDb})
	if d.NeedsReturning() {
		t.Error("mysql should not need RETURNING")
	}
}

func Test_SqliteDefaultMaxIdle(t *testing.T) {
	d := NewSqliteDialector(&testConfig{dbType: SqliteDb})
	if d.DefaultMaxIdle() != 1 {
		t.Errorf("sqlite default max idle should be 1, got: %d", d.DefaultMaxIdle())
	}
}

func Test_ApplyPagination(t *testing.T) {
	tests := []struct {
		query  string
		limit  int
		offset int
		want   string
	}{
		{"SELECT * FROM t", 10, 0, "SELECT * FROM t LIMIT 10"},
		{"SELECT * FROM t", 10, 5, "SELECT * FROM t LIMIT 10 OFFSET 5"},
		{"SELECT * FROM t", 0, 5, "SELECT * FROM t"},
	}
	for _, tt := range tests {
		got := ApplyPagination(tt.query, tt.limit, tt.offset)
		if got != tt.want {
			t.Errorf("ApplyPagination(%q, %d, %d) = %q, want %q", tt.query, tt.limit, tt.offset, got, tt.want)
		}
	}
}

func Test_ParseDatabaseType(t *testing.T) {
	tests := []struct {
		input string
		want  DatabaseType
	}{
		{"mysql", MySqlDb},
		{"Mysql", MySqlDb},
		{"postgres", PostgresDb},
		{"POSTGRES", PostgresDb},
		{"postgresql", PostgresDb},
		{"kingbase", KingbaseDb},
		{"kingbase8", KingbaseDb},
		{"sqlite", SqliteDb},
		{"sqlite3", SqliteDb},
		{"tidb", TiDBDb},
		{"TiDB", TiDBDb},
		{"tdsql", TDSQLDb},
		{"TDSQL", TDSQLDb},
		{"polardb", PolarDBMyDb},
		{"polardb-mysql", PolarDBMyDb},
		{"polardb_mysql", PolarDBMyDb},
		{"opengauss", OpenGaussDb},
		{"opengauss-server", OpenGaussDb},
		{"gaussdb", GaussDBDb},
		{"gaussdb-pg", GaussDBDb},
		{"highgo", HighGoDb},
		{"highgodb", HighGoDb},
		{"vastbase", VastbaseDb},
		{"vastbasedb", VastbaseDb},
		{"oceanbase", OceanBaseDb},
		{"oceanbase-mysql", OceanBaseDb},
		{"oceanbase-oracle", OceanBaseOracleDb},
		{"oboracle", OceanBaseOracleDb},
		{"dameng", DamengDb},
		{"dm", DamengDb},
		{"dm8", DamengDb},
	}
	for _, tt := range tests {
		got, err := ParseDatabaseType(tt.input)
		if err != nil || got != tt.want {
			t.Errorf("ParseDatabaseType(%q) = %q, %v; want %q, nil", tt.input, got, err, tt.want)
		}
	}
	if _, err := ParseDatabaseType("unknown"); err == nil {
		t.Error("ParseDatabaseType should fail for unknown type")
	}
}

func Test_DatabaseTypeFamily(t *testing.T) {
	if PostgresDb.Family() != FamilyPostgres {
		t.Errorf("PostgresDb.Family() = %q, want %q", PostgresDb.Family(), FamilyPostgres)
	}
	if KingbaseDb.Family() != FamilyPostgres {
		t.Errorf("KingbaseDb.Family() = %q, want %q", KingbaseDb.Family(), FamilyPostgres)
	}
	if MySqlDb.Family() != FamilyMySQL {
		t.Errorf("MySqlDb.Family() = %q, want %q", MySqlDb.Family(), FamilyMySQL)
	}
	if SqliteDb.Family() != FamilySQLite {
		t.Errorf("SqliteDb.Family() = %q, want %q", SqliteDb.Family(), FamilySQLite)
	}
	if TiDBDb.Family() != FamilyMySQL {
		t.Errorf("TiDBDb.Family() = %q, want %q", TiDBDb.Family(), FamilyMySQL)
	}
	if TDSQLDb.Family() != FamilyMySQL {
		t.Errorf("TDSQLDb.Family() = %q, want %q", TDSQLDb.Family(), FamilyMySQL)
	}
	if PolarDBMyDb.Family() != FamilyMySQL {
		t.Errorf("PolarDBMyDb.Family() = %q, want %q", PolarDBMyDb.Family(), FamilyMySQL)
	}
	if OpenGaussDb.Family() != FamilyPostgres {
		t.Errorf("OpenGaussDb.Family() = %q, want %q", OpenGaussDb.Family(), FamilyPostgres)
	}
	if GaussDBDb.Family() != FamilyPostgres {
		t.Errorf("GaussDBDb.Family() = %q, want %q", GaussDBDb.Family(), FamilyPostgres)
	}
	if HighGoDb.Family() != FamilyPostgres {
		t.Errorf("HighGoDb.Family() = %q, want %q", HighGoDb.Family(), FamilyPostgres)
	}
	if VastbaseDb.Family() != FamilyPostgres {
		t.Errorf("VastbaseDb.Family() = %q, want %q", VastbaseDb.Family(), FamilyPostgres)
	}
	if OceanBaseDb.Family() != FamilyMySQL {
		t.Errorf("OceanBaseDb.Family() = %q, want %q", OceanBaseDb.Family(), FamilyMySQL)
	}
	if OceanBaseOracleDb.Family() != FamilyOracle {
		t.Errorf("OceanBaseOracleDb.Family() = %q, want %q", OceanBaseOracleDb.Family(), FamilyOracle)
	}
	if DamengDb.Family() != FamilyOracle {
		t.Errorf("DamengDb.Family() = %q, want %q", DamengDb.Family(), FamilyOracle)
	}
}

func Test_GetDriverName(t *testing.T) {
	tests := []struct {
		dbType DatabaseType
		want   string
	}{
		{PostgresDb, "postgres"},
		{KingbaseDb, "kingbase"},
		{MySqlDb, "mysql"},
		{SqliteDb, "sqlite"},
		{TiDBDb, "tidb"},
		{TDSQLDb, "tdsql"},
		{PolarDBMyDb, "polardb"},
		{OpenGaussDb, "opengauss"},
		{GaussDBDb, "gaussdb"},
		{HighGoDb, "highgo"},
		{VastbaseDb, "vastbase"},
		{OceanBaseDb, "oceanbase"},
		{OceanBaseOracleDb, "oceanbase-oracle"},
		{DamengDb, "dameng"},
	}
	for _, tt := range tests {
		if got := GetDriverName(tt.dbType); got != tt.want {
			t.Errorf("GetDriverName(%q) = %q, want %q", tt.dbType, got, tt.want)
		}
	}
}

func Test_TiDBDriverRegistered(t *testing.T) {
	if !isDriverRegistered("tidb") {
		t.Error("tidb driver should be registered by init")
	}
}

func Test_TiDBFormatPrepareSQL(t *testing.T) {
	d := NewTiDBDialector(&testConfig{dbType: TiDBDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	if got != src {
		t.Errorf("tidb format prepare sql should keep ?, got: %q", got)
	}
}

func Test_TiDBNeedsReturning(t *testing.T) {
	d := NewTiDBDialector(&testConfig{dbType: TiDBDb})
	if d.NeedsReturning() {
		t.Error("tidb should not need RETURNING")
	}
}

func Test_TDSQLDriverRegistered(t *testing.T) {
	if !isDriverRegistered("tdsql") {
		t.Error("tdsql driver should be registered by init")
	}
}

func Test_TDSQLFormatPrepareSQL(t *testing.T) {
	d := NewTDSQLDialector(&testConfig{dbType: TDSQLDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	if got != src {
		t.Errorf("tdsql format prepare sql should keep ?, got: %q", got)
	}
}

func Test_TDSQLNeedsReturning(t *testing.T) {
	d := NewTDSQLDialector(&testConfig{dbType: TDSQLDb})
	if d.NeedsReturning() {
		t.Error("tdsql should not need RETURNING")
	}
}

func Test_PolarDBDriverRegistered(t *testing.T) {
	if !isDriverRegistered("polardb") {
		t.Error("polardb driver should be registered by init")
	}
}

func Test_PolarDBFormatPrepareSQL(t *testing.T) {
	d := NewPolarDBDialector(&testConfig{dbType: PolarDBMyDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	if got != src {
		t.Errorf("polardb format prepare sql should keep ?, got: %q", got)
	}
}

func Test_PolarDBNeedsReturning(t *testing.T) {
	d := NewPolarDBDialector(&testConfig{dbType: PolarDBMyDb})
	if d.NeedsReturning() {
		t.Error("polardb should not need RETURNING")
	}
}

func Test_OpenGaussDriverRegistered(t *testing.T) {
	if !isDriverRegistered("opengauss") {
		t.Error("opengauss driver should be registered by init")
	}
}

func Test_OpenGaussFormatPrepareSQL(t *testing.T) {
	d := NewOpenGaussDialector(&testConfig{dbType: OpenGaussDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	want := "select * from t where a = $1 and b = $2"
	if got != want {
		t.Errorf("opengauss format prepare sql failed, got: %q want: %q", got, want)
	}
}

func Test_OpenGaussNeedsReturning(t *testing.T) {
	d := NewOpenGaussDialector(&testConfig{dbType: OpenGaussDb})
	if !d.NeedsReturning() {
		t.Error("opengauss should need RETURNING")
	}
}

func Test_OpenGaussDialectorName(t *testing.T) {
	d := NewOpenGaussDialector(&testConfig{dbType: OpenGaussDb})
	if d.Name() != "opengauss" {
		t.Errorf("opengauss dialector name failed, got: %q", d.Name())
	}
}

func Test_GaussDBDriverRegistered(t *testing.T) {
	if !isDriverRegistered("gaussdb") {
		t.Error("gaussdb driver should be registered by init")
	}
}

func Test_GaussDBFormatPrepareSQL(t *testing.T) {
	d := NewGaussDBDialector(&testConfig{dbType: GaussDBDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	want := "select * from t where a = $1 and b = $2"
	if got != want {
		t.Errorf("gaussdb format prepare sql failed, got: %q want: %q", got, want)
	}
}

func Test_GaussDBNeedsReturning(t *testing.T) {
	d := NewGaussDBDialector(&testConfig{dbType: GaussDBDb})
	if !d.NeedsReturning() {
		t.Error("gaussdb should need RETURNING")
	}
}

func Test_GaussDBDialectorName(t *testing.T) {
	d := NewGaussDBDialector(&testConfig{dbType: GaussDBDb})
	if d.Name() != "gaussdb" {
		t.Errorf("gaussdb dialector name failed, got: %q", d.Name())
	}
}

func Test_HighGoDriverRegistered(t *testing.T) {
	if !isDriverRegistered("highgo") {
		t.Error("highgo driver should be registered by init")
	}
}

func Test_HighGoFormatPrepareSQL(t *testing.T) {
	d := NewHighGoDialector(&testConfig{dbType: HighGoDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	want := "select * from t where a = $1 and b = $2"
	if got != want {
		t.Errorf("highgo format prepare sql failed, got: %q want: %q", got, want)
	}
}

func Test_HighGoNeedsReturning(t *testing.T) {
	d := NewHighGoDialector(&testConfig{dbType: HighGoDb})
	if !d.NeedsReturning() {
		t.Error("highgo should need RETURNING")
	}
}

func Test_HighGoDialectorName(t *testing.T) {
	d := NewHighGoDialector(&testConfig{dbType: HighGoDb})
	if d.Name() != "highgo" {
		t.Errorf("highgo dialector name failed, got: %q", d.Name())
	}
}

func Test_VastbaseDriverRegistered(t *testing.T) {
	if !isDriverRegistered("vastbase") {
		t.Error("vastbase driver should be registered by init")
	}
}

func Test_VastbaseFormatPrepareSQL(t *testing.T) {
	d := NewVastbaseDialector(&testConfig{dbType: VastbaseDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	want := "select * from t where a = $1 and b = $2"
	if got != want {
		t.Errorf("vastbase format prepare sql failed, got: %q want: %q", got, want)
	}
}

func Test_VastbaseNeedsReturning(t *testing.T) {
	d := NewVastbaseDialector(&testConfig{dbType: VastbaseDb})
	if !d.NeedsReturning() {
		t.Error("vastbase should need RETURNING")
	}
}

func Test_VastbaseDialectorName(t *testing.T) {
	d := NewVastbaseDialector(&testConfig{dbType: VastbaseDb})
	if d.Name() != "vastbase" {
		t.Errorf("vastbase dialector name failed, got: %q", d.Name())
	}
}

func Test_EffectiveSchema(t *testing.T) {
	tests := []struct {
		params ConnectParams
		want   string
	}{
		{ConnectParams{Schema: "myschema", Type: PostgresDb}, "myschema"},
		{ConnectParams{DBName: "mydb", Type: PostgresDb}, "public"},
		{ConnectParams{DBName: "mydb", Type: MySqlDb}, "mydb"},
		{ConnectParams{DBName: "mydb", Type: SqliteDb}, ""},
		{ConnectParams{DBName: "mydb", Type: TiDBDb}, "mydb"},
		{ConnectParams{DBName: "mydb", Type: TDSQLDb}, "mydb"},
		{ConnectParams{DBName: "mydb", Type: PolarDBMyDb}, "mydb"},
		{ConnectParams{DBName: "mydb", Type: OpenGaussDb}, "public"},
		{ConnectParams{DBName: "mydb", Type: GaussDBDb}, "public"},
		{ConnectParams{DBName: "mydb", Type: HighGoDb}, "public"},
		{ConnectParams{DBName: "mydb", Type: VastbaseDb}, "public"},
		{ConnectParams{DBName: "mydb", Username: "root", Type: OceanBaseDb}, "mydb"},
		{ConnectParams{DBName: "mydb", Username: "SYS", Type: OceanBaseOracleDb}, "SYS"},
		{ConnectParams{DBName: "mydb", Username: "SYSDBA", Type: DamengDb}, "SYSDBA"},
	}
	for _, tt := range tests {
		if got := EffectiveSchema(tt.params); got != tt.want {
			t.Errorf("EffectiveSchema(%+v) = %q, want %q", tt.params, got, tt.want)
		}
	}
}

func Test_GenerateDSN(t *testing.T) {
	pgParams := ConnectParams{Host: "10.0.0.1", Port: 5432, Username: "u", Password: "p", DBName: "db", Type: PostgresDb}
	if got := GenerateDSN(pgParams); got != "host=10.0.0.1 port=5432 user=u password=p dbname=db sslmode=disable" {
		t.Errorf("GenerateDSN postgres = %q", got)
	}
	myParams := ConnectParams{Host: "10.0.0.1", Port: 3306, Username: "u", Password: "p", DBName: "db", Type: MySqlDb}
	if got := GenerateDSN(myParams); got != "u:p@tcp(10.0.0.1:3306)/db?parseTime=true&loc=Local" {
		t.Errorf("GenerateDSN mysql = %q", got)
	}
	sqlParams := ConnectParams{DBName: "test.db", Type: SqliteDb}
	if got := GenerateDSN(sqlParams); got != "test.db?_loc=auto" {
		t.Errorf("GenerateDSN sqlite = %q", got)
	}
	tidbParams := ConnectParams{Host: "10.0.0.1", Port: 4000, Username: "root", Password: "", DBName: "testdb", Type: TiDBDb}
	if got := GenerateDSN(tidbParams); got != "root:@tcp(10.0.0.1:4000)/testdb?parseTime=true&loc=Local" {
		t.Errorf("GenerateDSN tidb = %q", got)
	}
	tdsqlParams := ConnectParams{Host: "10.0.0.1", Port: 3306, Username: "root", Password: "123456", DBName: "testdb", Type: TDSQLDb}
	if got := GenerateDSN(tdsqlParams); got != "root:123456@tcp(10.0.0.1:3306)/testdb?parseTime=true&loc=Local" {
		t.Errorf("GenerateDSN tdsql = %q", got)
	}
	polardbParams := ConnectParams{Host: "10.0.0.1", Port: 3306, Username: "root", Password: "123456", DBName: "testdb", Type: PolarDBMyDb}
	if got := GenerateDSN(polardbParams); got != "root:123456@tcp(10.0.0.1:3306)/testdb?parseTime=true&loc=Local" {
		t.Errorf("GenerateDSN polardb = %q", got)
	}
	opengaussParams := ConnectParams{Host: "10.0.0.1", Port: 5432, Username: "root", Password: "123456", DBName: "testdb", Type: OpenGaussDb}
	if got := GenerateDSN(opengaussParams); got != "host=10.0.0.1 port=5432 user=root password=123456 dbname=testdb sslmode=disable" {
		t.Errorf("GenerateDSN opengauss = %q", got)
	}
	gaussdbParams := ConnectParams{Host: "10.0.0.1", Port: 5432, Username: "root", Password: "123456", DBName: "testdb", Type: GaussDBDb}
	if got := GenerateDSN(gaussdbParams); got != "host=10.0.0.1 port=5432 user=root password=123456 dbname=testdb sslmode=disable" {
		t.Errorf("GenerateDSN gaussdb = %q", got)
	}
	highgoParams := ConnectParams{Host: "10.0.0.1", Port: 5432, Username: "root", Password: "123456", DBName: "testdb", Type: HighGoDb}
	if got := GenerateDSN(highgoParams); got != "host=10.0.0.1 port=5432 user=root password=123456 dbname=testdb sslmode=disable" {
		t.Errorf("GenerateDSN highgo = %q", got)
	}
	vastbaseParams := ConnectParams{Host: "10.0.0.1", Port: 5432, Username: "root", Password: "123456", DBName: "testdb", Type: VastbaseDb}
	if got := GenerateDSN(vastbaseParams); got != "host=10.0.0.1 port=5432 user=root password=123456 dbname=testdb sslmode=disable" {
		t.Errorf("GenerateDSN vastbase = %q", got)
	}
	oceanbaseParams := ConnectParams{Host: "10.0.0.1", Port: 2881, Username: "root", Password: "123456", DBName: "testdb", Type: OceanBaseDb}
	if got := GenerateDSN(oceanbaseParams); got != "root:123456@tcp(10.0.0.1:2881)/testdb?parseTime=true&loc=Local" {
		t.Errorf("GenerateDSN oceanbase = %q", got)
	}
	oboracleParams := ConnectParams{Host: "10.0.0.1", Port: 2881, Username: "SYS", Password: "123456", DBName: "testdb", Type: OceanBaseOracleDb}
	if got := GenerateDSN(oboracleParams); got != "SYS/123456@10.0.0.1:2881/testdb" {
		t.Errorf("GenerateDSN oceanbase-oracle = %q", got)
	}
	damengParams := ConnectParams{Host: "10.0.0.1", Port: 5236, Username: "SYSDBA", Password: "123456", DBName: "testdb", Type: DamengDb}
	if got := GenerateDSN(damengParams); got != "SYSDBA/123456@10.0.0.1:5236/testdb" {
		t.Errorf("GenerateDSN dameng = %q", got)
	}
}

func Test_NewForType(t *testing.T) {
	d, err := NewForType(PostgresDb, &testConfig{dbType: PostgresDb})
	if err != nil || d.Name() != "postgres" {
		t.Errorf("NewForType postgres failed: %v, name=%q", err, d.Name())
	}
	d2, err := NewForType(MySqlDb, &testConfig{dbType: MySqlDb})
	if err != nil || d2.Name() != "mysql" {
		t.Errorf("NewForType mysql failed: %v, name=%q", err, d2.Name())
	}
	d3, err := NewForType(KingbaseDb, &testConfig{dbType: KingbaseDb})
	if err != nil || d3.Name() != "kingbase" {
		t.Errorf("NewForType kingbase failed: %v, name=%q", err, d3.Name())
	}
	d4, err := NewForType(TiDBDb, &testConfig{dbType: TiDBDb})
	if err != nil || d4.Name() != "tidb" {
		t.Errorf("NewForType tidb failed: %v, name=%q", err, d4.Name())
	}
	d5, err := NewForType(TDSQLDb, &testConfig{dbType: TDSQLDb})
	if err != nil || d5.Name() != "tdsql" {
		t.Errorf("NewForType tdsql failed: %v, name=%q", err, d5.Name())
	}
	d6, err := NewForType(PolarDBMyDb, &testConfig{dbType: PolarDBMyDb})
	if err != nil || d6.Name() != "polardb" {
		t.Errorf("NewForType polardb failed: %v, name=%q", err, d6.Name())
	}
	d7, err := NewForType(OpenGaussDb, &testConfig{dbType: OpenGaussDb})
	if err != nil || d7.Name() != "opengauss" {
		t.Errorf("NewForType opengauss failed: %v, name=%q", err, d7.Name())
	}
	d8, err := NewForType(GaussDBDb, &testConfig{dbType: GaussDBDb})
	if err != nil || d8.Name() != "gaussdb" {
		t.Errorf("NewForType gaussdb failed: %v, name=%q", err, d8.Name())
	}
	d9, err := NewForType(HighGoDb, &testConfig{dbType: HighGoDb})
	if err != nil || d9.Name() != "highgo" {
		t.Errorf("NewForType highgo failed: %v, name=%q", err, d9.Name())
	}
	d10, err := NewForType(VastbaseDb, &testConfig{dbType: VastbaseDb})
	if err != nil || d10.Name() != "vastbase" {
		t.Errorf("NewForType vastbase failed: %v, name=%q", err, d10.Name())
	}
	d11, err := NewForType(OceanBaseDb, &testConfig{dbType: OceanBaseDb})
	if err != nil || d11.Name() != "oceanbase" {
		t.Errorf("NewForType oceanbase failed: %v, name=%q", err, d11.Name())
	}
	d12, err := NewForType(OceanBaseOracleDb, &testConfig{dbType: OceanBaseOracleDb})
	if err != nil || d12.Name() != "oceanbase-oracle" {
		t.Errorf("NewForType oceanbase-oracle failed: %v, name=%q", err, d12.Name())
	}
	d13, err := NewForType(DamengDb, &testConfig{dbType: DamengDb})
	if err != nil || d13.Name() != "dameng" {
		t.Errorf("NewForType dameng failed: %v, name=%q", err, d13.Name())
	}
	if _, err := NewForType(DatabaseType("unknown"), &testConfig{}); err == nil {
		t.Error("NewForType should fail for unknown type")
	}
}

type testConfig struct {
	dbType DatabaseType
}

func (c *testConfig) GetDSN() string            { return "" }
func (c *testConfig) GetConnPool() ConnPool      { return nil }
func (c *testConfig) GetMaxTimeout() int          { return 300 }
func (c *testConfig) GetMaxOpen() int             { return 100 }
func (c *testConfig) GetConnectParams() ConnectParams {
	return ConnectParams{Type: c.dbType}
}

func Test_OceanBaseDriverRegistered(t *testing.T) {
	if !isDriverRegistered("oceanbase") {
		t.Error("oceanbase driver should be registered by init")
	}
}

func Test_OceanBaseFormatPrepareSQL(t *testing.T) {
	d := NewOceanBaseDialector(&testConfig{dbType: OceanBaseDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	if got != src {
		t.Errorf("oceanbase format prepare sql should keep ?, got: %q", got)
	}
}

func Test_OceanBaseNeedsReturning(t *testing.T) {
	d := NewOceanBaseDialector(&testConfig{dbType: OceanBaseDb})
	if d.NeedsReturning() {
		t.Error("oceanbase should not need RETURNING")
	}
}

func Test_OceanBaseDialectorName(t *testing.T) {
	d := NewOceanBaseDialector(&testConfig{dbType: OceanBaseDb})
	if d.Name() != "oceanbase" {
		t.Errorf("oceanbase dialector name failed, got: %q", d.Name())
	}
}

func Test_OceanBaseOracleFormatPrepareSQL(t *testing.T) {
	d := NewOceanBaseOracleDialector(&testConfig{dbType: OceanBaseOracleDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	want := "select * from t where a = :1 and b = :2"
	if got != want {
		t.Errorf("oceanbase-oracle format prepare sql failed, got: %q want: %q", got, want)
	}
}

func Test_OceanBaseOracleNeedsReturning(t *testing.T) {
	d := NewOceanBaseOracleDialector(&testConfig{dbType: OceanBaseOracleDb})
	if d.NeedsReturning() {
		t.Error("oceanbase-oracle should not need RETURNING")
	}
}

func Test_OceanBaseOracleDialectorName(t *testing.T) {
	d := NewOceanBaseOracleDialector(&testConfig{dbType: OceanBaseOracleDb})
	if d.Name() != "oceanbase-oracle" {
		t.Errorf("oceanbase-oracle dialector name failed, got: %q", d.Name())
	}
}

func Test_OceanBaseOracleFamily(t *testing.T) {
	d := NewOceanBaseOracleDialector(&testConfig{dbType: OceanBaseOracleDb})
	if d.Family() != FamilyOracle {
		t.Errorf("oceanbase-oracle family failed, got: %q want: %q", d.Family(), FamilyOracle)
	}
}

func Test_OceanBaseOraclePlaceholderStyle(t *testing.T) {
	d := NewOceanBaseOracleDialector(&testConfig{dbType: OceanBaseOracleDb})
	if d.PlaceholderStyle() != PlaceholderColon {
		t.Errorf("oceanbase-oracle placeholder style failed, got: %v want: %v", d.PlaceholderStyle(), PlaceholderColon)
	}
}

func Test_DamengFormatPrepareSQL(t *testing.T) {
	d := NewDamengDialector(&testConfig{dbType: DamengDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	want := "select * from t where a = :1 and b = :2"
	if got != want {
		t.Errorf("dameng format prepare sql failed, got: %q want: %q", got, want)
	}
}

func Test_DamengNeedsReturning(t *testing.T) {
	d := NewDamengDialector(&testConfig{dbType: DamengDb})
	if d.NeedsReturning() {
		t.Error("dameng should not need RETURNING")
	}
}

func Test_DamengDialectorName(t *testing.T) {
	d := NewDamengDialector(&testConfig{dbType: DamengDb})
	if d.Name() != "dameng" {
		t.Errorf("dameng dialector name failed, got: %q", d.Name())
	}
}

func Test_DamengFamily(t *testing.T) {
	d := NewDamengDialector(&testConfig{dbType: DamengDb})
	if d.Family() != FamilyOracle {
		t.Errorf("dameng family failed, got: %q want: %q", d.Family(), FamilyOracle)
	}
}

func Test_DamengPlaceholderStyle(t *testing.T) {
	d := NewDamengDialector(&testConfig{dbType: DamengDb})
	if d.PlaceholderStyle() != PlaceholderColon {
		t.Errorf("dameng placeholder style failed, got: %v want: %v", d.PlaceholderStyle(), PlaceholderColon)
	}
}
