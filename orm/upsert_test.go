package orm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bnulwh/mybatis-go/orm/dialector"
	"github.com/bnulwh/mybatis-go/types"

	_ "modernc.org/sqlite"
)

type UpsertTestModel struct {
	Id   int64  `db:"id,pk"`
	Name string `db:"name"`
}

func (UpsertTestModel) TableName() string { return "t_upsert_test" }

type UpsertTestMapper struct {
	BaseMapper
	Insert         func(UpsertTestModel) (int64, error)
	InsertOrUpdate func(UpsertTestModel) (int64, error)
	SelectById     func(int64) ([]UpsertTestModel, error)
	SelectList     func() ([]UpsertTestModel, error)
}

func initUpsertTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="UpsertTestMapper">
  <resultMap id="BaseResultMap" type="UpsertTestModel">
    <id column="id" jdbcType="BIGINT" property="id" />
    <result column="name" jdbcType="VARCHAR" property="name" />
  </resultMap>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "UpsertTestMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	dbPath := filepath.Join(dir, "test.db")
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}
	RegisterModel(new(UpsertTestModel))
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return ""
	}
	Execute(`CREATE TABLE IF NOT EXISTS t_upsert_test (
		id INTEGER PRIMARY KEY,
		name VARCHAR(100) NOT NULL
	)`)
	return dir
}

func Test_GenerateUpsertSQL_Postgres(t *testing.T) {
	args := types.UpsertSQLArgs{
		Table:      "sys_user",
		PkColumn:   "id",
		Columns:    []string{"id", "user_name"},
		Properties: []string{"id", "userName"},
		JdbcTypes:  []string{"BIGINT", "VARCHAR"},
	}
	sql := generateUpsertSQL(dialector.FamilyPostgres, args)
	if sql == "" {
		t.Fatal("postgres upsert sql should not be empty")
	}
	if !strings.Contains(sql, "insert into sys_user") {
		t.Error("missing insert into")
	}
	if !strings.Contains(sql, "on conflict (id) do update set") {
		t.Error("missing on conflict clause")
	}
	if !strings.Contains(sql, "user_name=EXCLUDED.user_name") {
		t.Error("missing EXCLUDED reference")
	}
}

func Test_GenerateUpsertSQL_SQLite(t *testing.T) {
	args := types.UpsertSQLArgs{
		Table:      "sys_user",
		PkColumn:   "id",
		Columns:    []string{"id", "user_name"},
		Properties: []string{"id", "userName"},
		JdbcTypes:  []string{"BIGINT", "VARCHAR"},
	}
	sql := generateUpsertSQL(dialector.FamilySQLite, args)
	if sql == "" {
		t.Fatal("sqlite upsert sql should not be empty")
	}
	if !strings.Contains(sql, "on conflict (id) do update set") {
		t.Error("sqlite should use ON CONFLICT syntax")
	}
}

func Test_GenerateUpsertSQL_MySQL(t *testing.T) {
	args := types.UpsertSQLArgs{
		Table:      "sys_user",
		PkColumn:   "id",
		Columns:    []string{"id", "user_name"},
		Properties: []string{"id", "userName"},
		JdbcTypes:  []string{"BIGINT", "VARCHAR"},
	}
	sql := generateUpsertSQL(dialector.FamilyMySQL, args)
	if sql == "" {
		t.Fatal("mysql upsert sql should not be empty")
	}
	if !strings.Contains(sql, "insert into sys_user") {
		t.Error("missing insert into")
	}
	if !strings.Contains(sql, "on duplicate key update") {
		t.Error("missing on duplicate key update")
	}
	if !strings.Contains(sql, "user_name=VALUES(user_name)") {
		t.Error("missing VALUES() reference")
	}
}

func Test_GenerateUpsertSQL_MSSQL(t *testing.T) {
	args := types.UpsertSQLArgs{
		Table:      "sys_user",
		PkColumn:   "id",
		Columns:    []string{"id", "user_name"},
		Properties: []string{"id", "userName"},
		JdbcTypes:  []string{"BIGINT", "VARCHAR"},
	}
	sql := generateUpsertSQL(dialector.FamilyMSSQL, args)
	if sql == "" {
		t.Fatal("mssql upsert sql should not be empty")
	}
	if !strings.Contains(sql, "merge into sys_user") {
		t.Error("missing merge into")
	}
	if !strings.Contains(sql, "when matched then update set") {
		t.Error("missing when matched")
	}
	if !strings.Contains(sql, "when not matched then insert") {
		t.Error("missing when not matched")
	}
}

func Test_GenerateUpsertSQL_Oracle(t *testing.T) {
	args := types.UpsertSQLArgs{
		Table:      "sys_user",
		PkColumn:   "id",
		Columns:    []string{"id", "user_name"},
		Properties: []string{"id", "userName"},
		JdbcTypes:  []string{"BIGINT", "VARCHAR"},
	}
	sql := generateUpsertSQL(dialector.FamilyOracle, args)
	if sql == "" {
		t.Fatal("oracle upsert sql should not be empty")
	}
	if !strings.Contains(sql, "merge into sys_user") {
		t.Error("missing merge into")
	}
	if !strings.Contains(sql, "from dual") {
		t.Error("missing from dual")
	}
}

func Test_GenerateUpsertSQL_Unsupported(t *testing.T) {
	args := types.UpsertSQLArgs{
		Table:      "sys_user",
		PkColumn:   "id",
		Columns:    []string{"id", "user_name"},
		Properties: []string{"id", "userName"},
		JdbcTypes:  []string{"BIGINT", "VARCHAR"},
	}
	sql := generateUpsertSQL(dialector.FamilyClickHouse, args)
	if sql != "" {
		t.Error("unsupported family should return empty string")
	}
}

func Test_GenerateUpsertSQL_NoPKInUpdate(t *testing.T) {
	args := types.UpsertSQLArgs{
		Table:      "sys_user",
		PkColumn:   "id",
		Columns:    []string{"id", "user_name"},
		Properties: []string{"id", "userName"},
		JdbcTypes:  []string{"BIGINT", "VARCHAR"},
	}
	for _, family := range []dialector.DatabaseFamily{dialector.FamilyPostgres, dialector.FamilyMySQL, dialector.FamilyMSSQL, dialector.FamilyOracle} {
		sql := generateUpsertSQL(family, args)
		if strings.Contains(sql, "id=EXCLUDED.id") || strings.Contains(sql, "id=VALUES(id)") || strings.Contains(sql, "id=source.id") {
			t.Errorf("family %v: pk column should not be in update set, sql: %v", family, sql)
		}
	}
}

func Test_SqliteInsertOrUpdate(t *testing.T) {
	dir := initUpsertTest(t)
	if dir == "" {
		return
	}
	defer Close()

	if err := RegisterMapper(new(UpsertTestMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("UpsertTestMapper").(UpsertTestMapper)

	n, err := mp.Insert(UpsertTestModel{Id: 1, Name: "alice"})
	if err != nil {
		t.Error("insert failed:", err)
		return
	}
	if n != 1 {
		t.Errorf("insert affected = %d, want 1", n)
		return
	}

	rows, err := mp.SelectById(1)
	if err != nil {
		t.Error("selectById failed:", err)
		return
	}
	if len(rows) != 1 || rows[0].Name != "alice" {
		t.Errorf("selectById = %+v, want alice", rows)
		return
	}

	n, err = mp.InsertOrUpdate(UpsertTestModel{Id: 1, Name: "bob"})
	if err != nil {
		t.Error("insertOrUpdate (update) failed:", err)
		return
	}
	if n != 1 {
		t.Errorf("insertOrUpdate (update) affected = %d, want 1", n)
	}

	rows2, err := mp.SelectById(1)
	if err != nil {
		t.Error("selectById after upsert failed:", err)
		return
	}
	if len(rows2) != 1 || rows2[0].Name != "bob" {
		t.Errorf("after upsert name = %q, want bob", rows2[0].Name)
	}

	n, err = mp.InsertOrUpdate(UpsertTestModel{Id: 2, Name: "charlie"})
	if err != nil {
		t.Error("insertOrUpdate (insert) failed:", err)
		return
	}
	if n != 1 {
		t.Errorf("insertOrUpdate (insert) affected = %d, want 1", n)
	}

	rows3, err := mp.SelectById(2)
	if err != nil {
		t.Error("selectById after upsert insert failed:", err)
		return
	}
	if len(rows3) != 1 || rows3[0].Name != "charlie" {
		t.Errorf("after upsert insert name = %q, want charlie", rows3[0].Name)
	}

	all, err := mp.SelectList()
	if err != nil {
		t.Error("selectList failed:", err)
		return
	}
	if len(all) != 2 {
		t.Errorf("selectList count = %d, want 2", len(all))
	}
}
