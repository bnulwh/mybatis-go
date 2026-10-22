package orm

import (
	"testing"

	"github.com/bnulwh/mybatis-go/orm/dialector"
)

// F1：MySQL→PG 族函数翻译的 orm 层开关、数据源门控与配置解析。

// Test_ApplyFunctionRewrite_PGFamily PG 族数据源 + 开关开启时翻译生效。
func Test_ApplyFunctionRewrite_PGFamily(t *testing.T) {
	defer SetFunctionRewrite(FunctionRewrite()) // 恢复原值
	SetFunctionRewrite(true)

	cfg := newDatabaseConfig("postgres", "localhost", 5432, "u", "p", "db")
	d, err := dialector.NewForType(dialector.PostgresDb, cfg)
	if err != nil {
		t.Errorf("NewForType failed: %v", err)
		return
	}
	if d.Family() != dialector.FamilyPostgres {
		t.Errorf("postgres dialector family want FamilyPostgres, got %v", d.Family())
		return
	}
	db := &DB{Config: &Config{Dialector: d}}

	in := "SELECT ifnull(a, 0), date_format(d, '%Y-%m-%d') FROM `t` WHERE find_in_set(id, ids) > 0"
	want := `SELECT coalesce(a, 0), to_char(d, 'YYYY-MM-DD') FROM "t" WHERE position(','||id||',' in ','||ids||',') > 0`
	if got := db.applyFunctionRewrite(in); got != want {
		t.Errorf("applyFunctionRewrite = %q, want %q", got, want)
	}
}

// Test_ApplyFunctionRewrite_SwitchOff 开关关闭（默认）时不翻译，存量行为不变。
func Test_ApplyFunctionRewrite_SwitchOff(t *testing.T) {
	defer SetFunctionRewrite(FunctionRewrite())
	SetFunctionRewrite(false)

	cfg := newDatabaseConfig("postgres", "localhost", 5432, "u", "p", "db")
	d, _ := dialector.NewForType(dialector.PostgresDb, cfg)
	db := &DB{Config: &Config{Dialector: d}}

	in := "SELECT ifnull(a, 0) FROM t"
	if got := db.applyFunctionRewrite(in); got != in {
		t.Errorf("switch off should keep sql as-is, got %q", got)
	}
}

// Test_ApplyFunctionRewrite_NonPGFamily 非 PG 族数据源（MySQL 原生语法）不翻译。
func Test_ApplyFunctionRewrite_NonPGFamily(t *testing.T) {
	defer SetFunctionRewrite(FunctionRewrite())
	SetFunctionRewrite(true)

	cfg := newDatabaseConfig("mysql", "localhost", 3306, "u", "p", "db")
	d, err := dialector.NewForType(dialector.MySqlDb, cfg)
	if err != nil {
		t.Errorf("NewForType failed: %v", err)
		return
	}
	db := &DB{Config: &Config{Dialector: d}}

	in := "SELECT ifnull(a, 0), `col` FROM t"
	if got := db.applyFunctionRewrite(in); got != in {
		t.Errorf("mysql family should keep sql as-is, got %q", got)
	}
}

// Test_ConfigParseFunctionRewrite 配置 mybatis.configuration.function-rewrite 解析与套用。
func Test_ConfigParseFunctionRewrite(t *testing.T) {
	defer SetFunctionRewrite(FunctionRewrite())

	cm := map[string]string{
		"spring.datasource.url":                  "jdbc:sqlite:test.db",
		"mybatis.configuration.function-rewrite": "true",
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("InitializeFromSettings failed: %v", err)
		return
	}
	defer Close()
	if !FunctionRewrite() {
		t.Error("function-rewrite=true should enable FunctionRewrite")
	}

	cm2 := map[string]string{
		"spring.datasource.url": "jdbc:sqlite:test.db",
	}
	if err := InitializeFromSettings(cm2); err != nil {
		t.Errorf("InitializeFromSettings failed: %v", err)
		return
	}
	defer Close()
	if FunctionRewrite() {
		t.Error("function-rewrite default should be off")
	}
}
