package orm

import (
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// P0-2：<if test> 表达式 AST 端到端（or / == null / not / length / 括号）。

type IfExprRow struct {
	Id   int
	Name string
}

type IfExprMapper struct {
	BaseMapper
	SelectByNameOrStatus func(param map[string]interface{}) ([]IfExprRow, error)
	SelectNameNull       func(param map[string]interface{}) ([]IfExprRow, error)
	SelectNotFlag        func(param map[string]interface{}) ([]IfExprRow, error)
	SelectByIds          func(param map[string]interface{}) ([]IfExprRow, error)
}

const ifExprMapperXML = `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="IfExprMapper">
  <resultMap id="ifExprRow" type="IfExprRow">
    <id column="id" jdbcType="INTEGER" property="id" />
    <result column="name" jdbcType="VARCHAR" property="name" />
  </resultMap>

  <select id="selectByNameOrStatus" parameterType="map" resultMap="ifExprRow">
    select id, name from t_if_expr
    <where>
      <if test="name != null or status != null">name = #{name} or status = #{status}</if>
      <if test="name == null and status == null">1 = 0</if>
    </where>
  </select>

  <select id="selectNameNull" parameterType="map" resultMap="ifExprRow">
    select id, name from t_if_expr
    <where>
      <if test="name == null">name is null</if>
      <if test="name != null">name = #{name}</if>
    </where>
  </select>

  <select id="selectNotFlag" parameterType="map" resultMap="ifExprRow">
    select id, name from t_if_expr where 1 = 1
    <if test="not flag">and name = 'beta'</if>
    <if test="flag == true">and name = 'gamma'</if>
  </select>

  <select id="selectByIds" parameterType="map" resultMap="ifExprRow">
    select id, name from t_if_expr
    <where>
      <if test="ids.length > 0">
        and id in
        <foreach collection="ids" item="item" open="(" separator="," close=")">#{item}</foreach>
      </if>
    </where>
  </select>
</mapper>`

// initIfExprSqlite 初始化 SQLite 并写入测试数据；strictIfTest 为 true 时开启
// strict-if-test（P0-2 后 or / == null 等不再算未识别，注册应成功）。
func initIfExprSqlite(t *testing.T, strictIfTest bool) {
	t.Helper()
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatalf("create mapper dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "IfExprMapper.xml"), []byte(ifExprMapperXML), 0644); err != nil {
		t.Fatalf("write mapper xml failed: %v", err)
	}
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + filepath.Join(dir, "ifexpr.db"),
		"mybatis.mapper-locations": xmlDir,
	}
	if strictIfTest {
		cm["mybatis.configuration.strict-if-test"] = "true"
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Fatalf("initialize sqlite failed: %v", err)
	}
	RegisterModel(new(IfExprRow))
	if _, err := Execute(`CREATE TABLE t_if_expr (id INTEGER PRIMARY KEY, name TEXT, status INTEGER)`); err != nil {
		t.Fatalf("create table failed: %v", err)
	}
	for _, row := range []struct {
		id     int
		name   interface{}
		status interface{}
	}{
		{1, "alpha", 1},
		{2, "beta", 2},
		{3, nil, nil},
		{4, "gamma", nil},
	} {
		if _, err := Execute(`INSERT INTO t_if_expr (id, name, status) VALUES (?, ?, ?)`, row.id, row.name, row.status); err != nil {
			t.Fatalf("insert row %v failed: %v", row.id, err)
		}
	}
	if err := RegisterMapper(new(IfExprMapper)); err != nil {
		t.Fatalf("register mapper failed: %v", err)
	}
}

func names(rows []IfExprRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Name
	}
	return out
}

// Test_SqliteIfExpr_OrAndNull or / == null 组合条件（旧实现 or 分支会被静默丢弃）。
func Test_SqliteIfExpr_OrAndNull(t *testing.T) {
	initIfExprSqlite(t, true) // 开启 strict-if-test：证明这些写法不再算未识别
	defer Close()
	defer SetStrictIfTest(false)
	mp := NewMapper("IfExprMapper").(IfExprMapper)

	rows, err := mp.SelectByNameOrStatus(map[string]interface{}{})
	if err != nil {
		t.Fatalf("empty params select failed: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("no params should match nothing, got %v", names(rows))
	}

	// 仅 name 分支命中（status 显式传 nil → SQL 参数为 null）
	rows, err = mp.SelectByNameOrStatus(map[string]interface{}{"name": "alpha", "status": nil})
	if err != nil {
		t.Fatalf("select by name failed: %v", err)
	}
	if len(rows) != 1 || rows[0].Name != "alpha" {
		t.Errorf("select by name = %v, want [alpha]", names(rows))
	}

	// 仅 status 分支命中（旧实现 or 分支被丢弃时这条永远查不到）
	rows, err = mp.SelectByNameOrStatus(map[string]interface{}{"name": nil, "status": 2})
	if err != nil {
		t.Fatalf("select by status failed: %v", err)
	}
	if len(rows) != 1 || rows[0].Name != "beta" {
		t.Errorf("select by status = %v, want [beta]", names(rows))
	}

	rows, err = mp.SelectByNameOrStatus(map[string]interface{}{"name": "alpha", "status": 2})
	if err != nil {
		t.Fatalf("select by name+status failed: %v", err)
	}
	if len(rows) != 2 || rows[0].Name != "alpha" || rows[1].Name != "beta" {
		t.Errorf("select by name+status = %v, want [alpha beta] (or 语义)", names(rows))
	}
}

// Test_SqliteIfExpr_NameNull == null / != null（含空串的 legacy 有效值语义）。
func Test_SqliteIfExpr_NameNull(t *testing.T) {
	initIfExprSqlite(t, false)
	defer Close()
	mp := NewMapper("IfExprMapper").(IfExprMapper)

	rows, err := mp.SelectNameNull(map[string]interface{}{})
	if err != nil {
		t.Fatalf("select null name failed: %v", err)
	}
	if len(rows) != 1 || rows[0].Id != 3 {
		t.Errorf("missing name should match id=3, got %v", names(rows))
	}

	// 空串按 legacy「有效值」语义视作 null（与 `!= null` 对称）
	rows, err = mp.SelectNameNull(map[string]interface{}{"name": ""})
	if err != nil {
		t.Fatalf("select empty name failed: %v", err)
	}
	if len(rows) != 1 || rows[0].Id != 3 {
		t.Errorf("empty name should match id=3, got %v", names(rows))
	}

	rows, err = mp.SelectNameNull(map[string]interface{}{"name": "alpha"})
	if err != nil {
		t.Fatalf("select alpha failed: %v", err)
	}
	if len(rows) != 1 || rows[0].Id != 1 {
		t.Errorf("name=alpha should match id=1, got %v", names(rows))
	}
}

// Test_SqliteIfExpr_NotFlag not / 布尔字面量比较。
func Test_SqliteIfExpr_NotFlag(t *testing.T) {
	initIfExprSqlite(t, false)
	defer Close()
	mp := NewMapper("IfExprMapper").(IfExprMapper)

	rows, err := mp.SelectNotFlag(map[string]interface{}{"flag": false})
	if err != nil {
		t.Fatalf("flag=false select failed: %v", err)
	}
	if len(rows) != 1 || rows[0].Name != "beta" {
		t.Errorf("flag=false = %v, want [beta]", names(rows))
	}

	rows, err = mp.SelectNotFlag(map[string]interface{}{"flag": true})
	if err != nil {
		t.Fatalf("flag=true select failed: %v", err)
	}
	if len(rows) != 1 || rows[0].Name != "gamma" {
		t.Errorf("flag=true = %v, want [gamma]", names(rows))
	}
}

// Test_SqliteIfExpr_Length 集合长度 + foreach 组合。
func Test_SqliteIfExpr_Length(t *testing.T) {
	initIfExprSqlite(t, false)
	defer Close()
	mp := NewMapper("IfExprMapper").(IfExprMapper)

	rows, err := mp.SelectByIds(map[string]interface{}{"ids": []interface{}{1, 4}})
	if err != nil {
		t.Fatalf("select by ids failed: %v", err)
	}
	if len(rows) != 2 || rows[0].Id != 1 || rows[1].Id != 4 {
		t.Errorf("ids=[1,4] = %v, want ids [1 4]", rows)
	}

	rows, err = mp.SelectByIds(map[string]interface{}{"ids": []interface{}{}})
	if err != nil {
		t.Fatalf("select by empty ids failed: %v", err)
	}
	if len(rows) != 4 {
		t.Errorf("empty ids should not filter, got %d rows", len(rows))
	}
}
