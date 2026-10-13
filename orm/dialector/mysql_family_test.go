package dialector

import "testing"

func Test_TaurusDBDriverRegistered(t *testing.T) {
	if !isDriverRegistered("taurusdb") {
		t.Error("taurusdb driver should be registered by init")
	}
}

func Test_TaurusDBFormatPrepareSQL(t *testing.T) {
	d := NewTaurusDBDialector(&testConfig{dbType: TaurusDBDb})
	src := "select * from t where a = ? and b = ?"
	if got := d.FormatPrepareSQL(src); got != src {
		t.Errorf("taurusdb FormatPrepareSQL should keep ?, got %q", got)
	}
}

func Test_TaurusDBNeedsReturning(t *testing.T) {
	d := NewTaurusDBDialector(&testConfig{dbType: TaurusDBDb})
	if d.NeedsReturning() {
		t.Error("taurusdb should not need RETURNING")
	}
}

func Test_TaurusDBDialectorName(t *testing.T) {
	d := NewTaurusDBDialector(&testConfig{dbType: TaurusDBDb})
	if d.Name() != "taurusdb" {
		t.Errorf("taurusdb dialector name = %q", d.Name())
	}
}

func Test_UPDRDBDriverRegistered(t *testing.T) {
	if !isDriverRegistered("updrdb") {
		t.Error("updrdb driver should be registered by init")
	}
}

func Test_UPDRDBFormatPrepareSQL(t *testing.T) {
	d := NewUPDRDBDialector(&testConfig{dbType: UPDRDBDb})
	src := "select * from t where a = ? and b = ?"
	if got := d.FormatPrepareSQL(src); got != src {
		t.Errorf("updrdb FormatPrepareSQL should keep ?, got %q", got)
	}
}

func Test_UPDRDBNeedsReturning(t *testing.T) {
	d := NewUPDRDBDialector(&testConfig{dbType: UPDRDBDb})
	if d.NeedsReturning() {
		t.Error("updrdb should not need RETURNING")
	}
}

func Test_UPDRDBDialectorName(t *testing.T) {
	d := NewUPDRDBDialector(&testConfig{dbType: UPDRDBDb})
	if d.Name() != "updrdb" {
		t.Errorf("updrdb dialector name = %q", d.Name())
	}
}

func Test_KingwowDriverRegistered(t *testing.T) {
	if !isDriverRegistered("kingwow") {
		t.Error("kingwow driver should be registered by init")
	}
}

func Test_KingwowFormatPrepareSQL(t *testing.T) {
	d := NewKingwowDialector(&testConfig{dbType: KingwowDb})
	src := "select * from t where a = ? and b = ?"
	if got := d.FormatPrepareSQL(src); got != src {
		t.Errorf("kingwow FormatPrepareSQL should keep ?, got %q", got)
	}
}

func Test_KingwowNeedsReturning(t *testing.T) {
	d := NewKingwowDialector(&testConfig{dbType: KingwowDb})
	if d.NeedsReturning() {
		t.Error("kingwow should not need RETURNING")
	}
}

func Test_KingwowDialectorName(t *testing.T) {
	d := NewKingwowDialector(&testConfig{dbType: KingwowDb})
	if d.Name() != "kingwow" {
		t.Errorf("kingwow dialector name = %q", d.Name())
	}
}
