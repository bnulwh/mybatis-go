package dialector

import "testing"

func Test_TeleDBDriverRegistered(t *testing.T) {
	if !isDriverRegistered("teledb") {
		t.Error("teledb driver should be registered by init")
	}
}

func Test_TeleDBFormatPrepareSQL(t *testing.T) {
	d := NewTeleDBDialector(&testConfig{dbType: TeleDBDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	want := "select * from t where a = $1 and b = $2"
	if got != want {
		t.Errorf("teledb format prepare sql failed, got: %q want: %q", got, want)
	}
}

func Test_TeleDBNeedsReturning(t *testing.T) {
	d := NewTeleDBDialector(&testConfig{dbType: TeleDBDb})
	if !d.NeedsReturning() {
		t.Error("teledb should need RETURNING")
	}
}

func Test_TeleDBDialectorName(t *testing.T) {
	d := NewTeleDBDialector(&testConfig{dbType: TeleDBDb})
	if d.Name() != "teledb" {
		t.Errorf("teledb dialector name failed, got: %q", d.Name())
	}
}

func Test_DWSDriverRegistered(t *testing.T) {
	if !isDriverRegistered("dws") {
		t.Error("dws driver should be registered by init")
	}
}

func Test_DWSFormatPrepareSQL(t *testing.T) {
	d := NewDWSDialector(&testConfig{dbType: DWSDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	want := "select * from t where a = $1 and b = $2"
	if got != want {
		t.Errorf("dws format prepare sql failed, got: %q want: %q", got, want)
	}
}

func Test_DWSNeedsReturning(t *testing.T) {
	d := NewDWSDialector(&testConfig{dbType: DWSDb})
	if !d.NeedsReturning() {
		t.Error("dws should need RETURNING")
	}
}

func Test_DWSDialectorName(t *testing.T) {
	d := NewDWSDialector(&testConfig{dbType: DWSDb})
	if d.Name() != "dws" {
		t.Errorf("dws dialector name failed, got: %q", d.Name())
	}
}

func Test_ADBPGDriverRegistered(t *testing.T) {
	if !isDriverRegistered("adbpg") {
		t.Error("adbpg driver should be registered by init")
	}
}

func Test_ADBPGFormatPrepareSQL(t *testing.T) {
	d := NewADBPGDialector(&testConfig{dbType: ADBPGDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	want := "select * from t where a = $1 and b = $2"
	if got != want {
		t.Errorf("adbpg format prepare sql failed, got: %q want: %q", got, want)
	}
}

func Test_ADBPGNeedsReturning(t *testing.T) {
	d := NewADBPGDialector(&testConfig{dbType: ADBPGDb})
	if !d.NeedsReturning() {
		t.Error("adbpg should need RETURNING")
	}
}

func Test_ADBPGDialectorName(t *testing.T) {
	d := NewADBPGDialector(&testConfig{dbType: ADBPGDb})
	if d.Name() != "adbpg" {
		t.Errorf("adbpg dialector name failed, got: %q", d.Name())
	}
}

func Test_SeaboxSQLDriverRegistered(t *testing.T) {
	if !isDriverRegistered("seaboxsql") {
		t.Error("seaboxsql driver should be registered by init")
	}
}

func Test_SeaboxSQLFormatPrepareSQL(t *testing.T) {
	d := NewSeaboxSQLDialector(&testConfig{dbType: SeaboxSQLDb})
	src := "select * from t where a = ? and b = ?"
	got := d.FormatPrepareSQL(src)
	want := "select * from t where a = $1 and b = $2"
	if got != want {
		t.Errorf("seaboxsql format prepare sql failed, got: %q want: %q", got, want)
	}
}

func Test_SeaboxSQLNeedsReturning(t *testing.T) {
	d := NewSeaboxSQLDialector(&testConfig{dbType: SeaboxSQLDb})
	if !d.NeedsReturning() {
		t.Error("seaboxsql should need RETURNING")
	}
}

func Test_SeaboxSQLDialectorName(t *testing.T) {
	d := NewSeaboxSQLDialector(&testConfig{dbType: SeaboxSQLDb})
	if d.Name() != "seaboxsql" {
		t.Errorf("seaboxsql dialector name failed, got: %q", d.Name())
	}
}
