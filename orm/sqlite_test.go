package orm

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// SQLite 专用模型/Mapper（独立命名，避免与 demo 或其他测试冲突）

type SqliteTestModel struct {
	Id         int
	Name       string
	CreateTime time.Time
}

type SqliteTestMapper struct {
	BaseMapper
	Insert    func(model SqliteTestModel) (int64, error)
	SelectAll func() ([]SqliteTestModel, error)
	StreamAll func() (*RowStream, error)
}

func Test_parseAddr_sqlite(t *testing.T) {
	mp := map[string]string{"spring.datasource.url": "jdbc:sqlite:test.db"}
	tp, host, port, db, err := parseAddr(mp)
	if tp != "sqlite" || host != "" || port != 0 || db != "test.db" || err != nil {
		t.Error("test parseAddr(jdbc:sqlite:test.db) failed.")
	}
	mp["spring.datasource.url"] = "jdbc:sqlite:/abs/path/data.db"
	_, _, _, db1, err := parseAddr(mp)
	if db1 != "/abs/path/data.db" || err != nil {
		t.Error("test parseAddr(jdbc:sqlite:/abs/path) failed.")
	}
	mp["spring.datasource.url"] = "jdbc:sqlite:file:data.db"
	_, _, _, db2, err := parseAddr(mp)
	if db2 != "data.db" || err != nil {
		t.Error("test parseAddr(jdbc:sqlite:file:) failed.")
	}
	mp["spring.datasource.url"] = "jdbc:sqlite::memory:"
	_, _, _, db3, err := parseAddr(mp)
	if db3 != ":memory:" || err != nil {
		t.Error("test parseAddr(jdbc:sqlite::memory:) failed.")
	}
}

// 初始化 SQLite 数据源并返回 mapper XML 目录
func initSqliteTest(t *testing.T) string {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="SqliteTestMapper">
  <resultMap id="BaseResultMap" type="SqliteTestModel">
    <id column="id" jdbcType="INTEGER" property="id" />
    <result column="name" jdbcType="VARCHAR" property="name" />
    <result column="create_time" jdbcType="TIMESTAMP" property="createTime" />
  </resultMap>
  <insert id="insert" parameterType="SqliteTestModel">
    insert into t_sqlite (name, create_time)
    values (#{name,jdbcType=VARCHAR}, #{createTime,jdbcType=TIMESTAMP})
  </insert>
  <select id="selectAll" resultMap="BaseResultMap">
    select id, name, create_time from t_sqlite
  </select>
  <select id="streamAll" resultMap="BaseResultMap">
    select id, name, create_time from t_sqlite order by id
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "SqliteTestMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	dbPath := filepath.Join(dir, "test.db")
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return ""
	}
	return dir
}

func Test_SqliteQueryExecute(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	if _, err := Execute(`CREATE TABLE t_sqlite (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		create_time DATETIME)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	tm := time.Date(2026, 8, 13, 17, 0, 0, 0, time.UTC)
	if _, err := Execute(`INSERT INTO t_sqlite (name, create_time) VALUES (?, ?)`, "hello", tm); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	res, err := Query(`SELECT id, name, create_time FROM t_sqlite`)
	if err != nil {
		t.Errorf("query failed: %v", err)
		return
	}
	if len(res) != 1 {
		t.Errorf("query result count failed, got %d", len(res))
		return
	}
	if res[0]["name"] != "hello" {
		t.Errorf("query name failed, got %v", res[0]["name"])
	}
	ct, ok := res[0]["create_time"].(string)
	if !ok {
		t.Errorf("query create_time should be string, got %T", res[0]["create_time"])
	} else if _, err := time.Parse(time.RFC3339Nano, ct); err != nil {
		t.Errorf("query create_time %q not RFC3339: %v", ct, err)
	}
}

func Test_SqliteMapper(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	if _, err := Execute(`CREATE TABLE t_sqlite (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		create_time DATETIME)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	RegisterModel(new(SqliteTestModel))
	if err := RegisterMapper(new(SqliteTestMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("SqliteTestMapper").(SqliteTestMapper)
	tm := time.Date(2026, 8, 13, 17, 0, 0, 0, time.UTC)
	if _, err := mp.Insert(SqliteTestModel{Name: "mapper_user", CreateTime: tm}); err != nil {
		t.Errorf("mapper insert failed: %v", err)
		return
	}
	rs, err := mp.SelectAll()
	if err != nil {
		t.Errorf("mapper select failed: %v", err)
		return
	}
	if len(rs) != 1 {
		t.Errorf("mapper select count failed, got %d", len(rs))
		return
	}
	if rs[0].Name != "mapper_user" {
		t.Errorf("mapper select name failed, got %v", rs[0].Name)
	}
	if !rs[0].CreateTime.Equal(tm) {
		t.Errorf("mapper select create_time failed, got %v want %v", rs[0].CreateTime, tm)
	}
}

func Test_SqliteStreamMapper(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	if _, err := Execute(`CREATE TABLE t_sqlite (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		create_time DATETIME)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	RegisterModel(new(SqliteTestModel))
	if err := RegisterMapper(new(SqliteTestMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("SqliteTestMapper").(SqliteTestMapper)
	tm := time.Date(2026, 8, 13, 17, 0, 0, 0, time.UTC)
	for _, n := range []string{"stream_1", "stream_2", "stream_3"} {
		if _, err := mp.Insert(SqliteTestModel{Name: n, CreateTime: tm}); err != nil {
			t.Errorf("mapper insert failed: %v", err)
			return
		}
	}

	// P4-2：Mapper 流式 select —— 返回 *RowStream，逐行 Scan，不整表进内存
	st, err := mp.StreamAll()
	if err != nil {
		t.Errorf("mapper stream failed: %v", err)
		return
	}
	defer st.Close()
	var got []SqliteTestModel
	for st.Next() {
		var m SqliteTestModel
		if err := st.Scan(&m); err != nil {
			t.Errorf("stream scan failed: %v", err)
			return
		}
		got = append(got, m)
	}
	if err := st.Err(); err != nil {
		t.Errorf("stream err failed: %v", err)
		return
	}
	if len(got) != 3 {
		t.Errorf("stream count failed, got %d want 3", len(got))
		return
	}
	if got[0].Name != "stream_1" || got[1].Name != "stream_2" || got[2].Name != "stream_3" {
		t.Errorf("stream names failed, got %v", got)
	}
	if !got[0].CreateTime.Equal(tm) {
		t.Errorf("stream create_time failed, got %v want %v", got[0].CreateTime, tm)
	}
	if st.Count() != 3 {
		t.Errorf("stream Count() failed, got %d want 3", st.Count())
	}
}

func Test_SqliteTableStructure(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	if _, err := Execute(`CREATE TABLE t_sqlite (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		create_time DATETIME)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	tns, err := fetchTables("")
	if err != nil {
		t.Errorf("fetchTables failed: %v", err)
		return
	}
	found := false
	for _, tn := range tns {
		if tn == "t_sqlite" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("fetchTables not found t_sqlite, got %v", tns)
		return
	}
	pts, err := newTableStruct("", "t_sqlite")
	if err != nil {
		t.Errorf("newTableStruct failed: %v", err)
		return
	}
	if pts.PrimaryColumn == nil || pts.PrimaryColumn.Name != "id" {
		t.Errorf("primary key detect failed, got %v", pts.PrimaryColumn)
	}
	if len(pts.Columns) != 3 {
		t.Errorf("column count failed, got %d", len(pts.Columns))
	}
}
