package dialector

import "testing"

// ReservedNames 类型行为测试：大小写不敏感、nil/零值安全、Merge 合并。
func Test_ReservedNames(t *testing.T) {
	r := NewReservedNames("Dual", " JSON_TABLE ", "")
	if !r.Has("dual") || !r.Has("DUAL") || !r.Has("json_table") {
		t.Error("ReservedNames.Has should be case-insensitive and trim spaces")
	}
	if r.Has("json_each") || r.Has("") {
		t.Error("ReservedNames.Has should miss unknown/empty names")
	}
	// nil / 空集合安全
	var nilR *ReservedNames
	if nilR.Has("dual") {
		t.Error("nil ReservedNames.Has should be false")
	}
	empty := NewReservedNames()
	if empty.Has("dual") {
		t.Error("empty ReservedNames.Has should be false")
	}
	// Merge 并集；nil 侧直接返回另一侧
	merged := NewReservedNames("dual").Merge(NewReservedNames("table"))
	if !merged.Has("dual") || !merged.Has("table") {
		t.Error("Merge should contain both sides")
	}
	if got := NewReservedNames("dual").Merge(nil); !got.Has("dual") {
		t.Error("Merge with nil should return the non-nil side")
	}
	if got := (*ReservedNames)(nil).Merge(NewReservedNames("dual")); !got.Has("dual") {
		t.Error("nil Merge should return the other side")
	}
}

// 各数据库方言保留名清单测试：表位置内置函数/哑表必须命中，普通表名不得误命中。
func Test_DialectorReservedTableNames(t *testing.T) {
	cases := []struct {
		name     string
		d        Dialector
		contains []string
		misses   []string
	}{
		{"postgres", NewPostgresDialector(&testConfig{dbType: PostgresDb}),
			[]string{"generate_series", "unnest", "json_populate_recordset", "jsonb_populate_recordset", "xmltable", "rows_from"},
			[]string{"dual", "sys_user"}},
		{"kingbase", NewKingbaseDialector(&testConfig{dbType: KingbaseDb}),
			[]string{"dual", "generate_series", "json_populate_recordset"},
			[]string{"sys_user"}},
		{"mysql", NewMySqlDialector(&testConfig{dbType: MySqlDb}),
			[]string{"dual", "json_table"},
			[]string{"generate_series", "sys_user"}},
		{"tidb", NewTiDBDialector(&testConfig{dbType: TiDBDb}),
			[]string{"dual", "json_table"},
			[]string{"sys_user"}},
		{"sqlite", NewSqliteDialector(&testConfig{dbType: SqliteDb}),
			[]string{"json_each", "json_tree"},
			[]string{"dual", "sys_user"}},
		{"oracle", NewOracleDialector(&testConfig{dbType: OracleDb}),
			[]string{"dual", "table", "xmltable", "json_table"},
			[]string{"sys_user"}},
		{"oceanbase-oracle", NewOceanBaseOracleDialector(&testConfig{dbType: OceanBaseOracleDb}),
			[]string{"dual", "xmltable"},
			[]string{"sys_user"}},
		{"dameng", NewDamengDialector(&testConfig{dbType: DamengDb}),
			[]string{"dual", "table", "xmltable"},
			[]string{"sys_user"}},
		{"mssql", NewMssqlDialector(&testConfig{dbType: MssqlDb}),
			[]string{"openjson", "openrowset", "containstable", "string_split", "inserted", "deleted"},
			[]string{"dual", "sys_user"}},
		{"db2", NewDb2Dialector(&testConfig{dbType: Db2Db}),
			[]string{"unnest", "xmltable", "table"},
			[]string{"dual", "sys_user"}},
		{"gbase8s", NewGBase8sDialector(&testConfig{dbType: GBase8sDb}),
			[]string{"table"},
			[]string{"dual", "sys_user"}},
		{"clickhouse", NewClickHouseDialector(&testConfig{dbType: ClickHouseDb}),
			[]string{"numbers", "generateRandom", "file", "url", "cluster", "input"},
			[]string{"dual", "sys_user"}},
	}
	for _, c := range cases {
		r := c.d.ReservedTableNames()
		for _, n := range c.contains {
			if !r.Has(n) {
				t.Errorf("%s: reserved names should contain %q", c.name, n)
			}
		}
		for _, n := range c.misses {
			if r.Has(n) {
				t.Errorf("%s: reserved names should NOT contain %q", c.name, n)
			}
		}
	}
}

// 家族嵌入继承验证：openGauss/GaussDB/Highgo/Vastbase 继承 PG 清单，
// TDSQL/PolarDB/OceanBase(MySQL 模式) 继承 MySQL 清单。
func Test_DialectorReservedTableNames_familyInherit(t *testing.T) {
	family := []struct {
		name string
		d    Dialector
	}{
		{"opengauss", NewOpenGaussDialector(&testConfig{dbType: OpenGaussDb})},
		{"gaussdb", NewGaussDBDialector(&testConfig{dbType: GaussDBDb})},
		{"highgo", NewHighGoDialector(&testConfig{dbType: HighGoDb})},
		{"vastbase", NewVastbaseDialector(&testConfig{dbType: VastbaseDb})},
	}
	for _, f := range family {
		r := f.d.ReservedTableNames()
		if !r.Has("generate_series") || !r.Has("json_populate_recordset") {
			t.Errorf("%s: should inherit postgres reserved table names", f.name)
		}
	}
	mysqlFamily := []struct {
		name string
		d    Dialector
	}{
		{"tdsql", NewTDSQLDialector(&testConfig{dbType: TDSQLDb})},
		{"polardb", NewPolarDBDialector(&testConfig{dbType: PolarDBMyDb})},
		{"oceanbase", NewOceanBaseDialector(&testConfig{dbType: OceanBaseDb})},
	}
	for _, f := range mysqlFamily {
		r := f.d.ReservedTableNames()
		if !r.Has("dual") || !r.Has("json_table") {
			t.Errorf("%s: should inherit mysql reserved table names", f.name)
		}
	}
}

// BaseDialector 默认实现：无保留名（nil 语义等价，Has 恒 false）。
func Test_BaseDialectorReservedTableNamesDefault(t *testing.T) {
	d := &BaseDialector{}
	if r := d.ReservedTableNames(); r != nil {
		t.Error("BaseDialector.ReservedTableNames should return nil by default")
	}
}
