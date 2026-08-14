package orm

import "testing"

func Test_KingbaseDialectorFormatPrepareSQL(t *testing.T) {
	d := NewKingbaseDialector(&Config{})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	want := "select * from t where a = $1 and b = $2"
	if got != want {
		t.Errorf("format prepare sql failed, got: %q want: %q", got, want)
	}
}

func Test_KingbaseDialectorName(t *testing.T) {
	d := NewKingbaseDialector(&Config{})
	if d.Name() != "kingbase" {
		t.Errorf("dialector name failed, got: %q", d.Name())
	}
}

func Test_KingbaseDriverRegistered(t *testing.T) {
	if !isDriverRegistered("kingbase") {
		t.Error("kingbase driver should be registered by init")
	}
}

func Test_KingbaseConfigGenerateDSN(t *testing.T) {
	cfg := newDatabaseConfig("kingbase", "10.1.2.3", 54321, "system", "secret", "testdb")
	if cfg.DriverName() != "kingbase" {
		t.Errorf("driver name failed, got: %q", cfg.DriverName())
	}
	want := "host=10.1.2.3 port=54321 user=system password=secret dbname=testdb sslmode=disable"
	if got := cfg.GenerateDSN(); got != want {
		t.Errorf("generate dsn failed, got: %q want: %q", got, want)
	}
}
