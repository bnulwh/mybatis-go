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
	}
	for _, tt := range tests {
		if got := GetDriverName(tt.dbType); got != tt.want {
			t.Errorf("GetDriverName(%q) = %q, want %q", tt.dbType, got, tt.want)
		}
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
