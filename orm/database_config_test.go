package orm

import "testing"

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
}
