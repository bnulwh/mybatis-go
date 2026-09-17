package orm

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type P1TestModel struct {
	Id         int64  `db:"id,pk"`
	Name       string `db:"name"`
	Version    int    `db:"version,version"`
	CreateTime time.Time `db:"create_time,fill:insert"`
	UpdateTime time.Time `db:"update_time,fill:insert_update"`
	IsRemoved  bool   `db:"is_removed,logic"`
}

func (P1TestModel) TableName() string { return "t_p1_test" }

type P1TestMapper struct {
	BaseMapper
	Insert      func(model *P1TestModel) (int64, error)
	InsertBatch func(models []*P1TestModel) (int64, error)
	UpdateById  func(model *P1TestModel) (int64, error)
	DeleteById  func(id int64) (int64, error)
	SelectById  func(id int64) ([]P1TestModel, error)
	SelectAll   func() ([]P1TestModel, error)
	UpdateNoWhere func() (int64, error)
	DeleteNoWhere func() (int64, error)
}

func initP1Test(t *testing.T, extraProps map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="P1TestMapper">
  <resultMap id="BaseResultMap" type="P1TestModel">
    <id column="id" jdbcType="BIGINT" property="id" />
    <result column="name" jdbcType="VARCHAR" property="name" />
    <result column="version" jdbcType="INTEGER" property="version" />
    <result column="create_time" jdbcType="TIMESTAMP" property="createTime" />
    <result column="update_time" jdbcType="TIMESTAMP" property="updateTime" />
    <result column="is_removed" jdbcType="BOOLEAN" property="isRemoved" />
  </resultMap>
  <insert id="insert" parameterType="P1TestModel" useGeneratedKeys="true" keyProperty="id" keyColumn="id">
    insert into t_p1_test (name, version, create_time, update_time, is_removed)
    values (#{name,jdbcType=VARCHAR}, #{version,jdbcType=INTEGER},
            #{createTime,jdbcType=TIMESTAMP}, #{updateTime,jdbcType=TIMESTAMP},
            #{isRemoved,jdbcType=BOOLEAN})
  </insert>
  <update id="updateById" parameterType="P1TestModel">
    update t_p1_test set name=#{name,jdbcType=VARCHAR}, update_time=#{updateTime,jdbcType=TIMESTAMP}, version=version+1
    where id=#{id,jdbcType=BIGINT} and version=#{version,jdbcType=INTEGER}
  </update>
  <delete id="deleteById" parameterType="java.lang.Long">
    delete from t_p1_test where id=#{id,jdbcType=BIGINT}
  </delete>
  <select id="selectById" parameterType="java.lang.Long" resultMap="BaseResultMap">
    select id, name, version, create_time, update_time, is_removed from t_p1_test where id=#{id,jdbcType=BIGINT}
  </select>
  <select id="selectAll" resultMap="BaseResultMap">
    select id, name, version, create_time, update_time, is_removed from t_p1_test
  </select>
  <update id="updateNoWhere">
    update t_p1_test set name='x'
  </update>
  <delete id="deleteNoWhere">
    delete from t_p1_test
  </delete>
  <insert id="insertBatch" parameterType="list">
    insert into t_p1_test (name, version, create_time, update_time, is_removed)
    values
    <foreach collection="list" item="item" separator=",">
      (#{item.name,jdbcType=VARCHAR}, #{item.version,jdbcType=INTEGER},
       #{item.createTime,jdbcType=TIMESTAMP}, #{item.updateTime,jdbcType=TIMESTAMP},
       #{item.isRemoved,jdbcType=BOOLEAN})
    </foreach>
  </insert>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "P1TestMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	dbPath := filepath.Join(dir, "test.db")
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}
	for k, v := range extraProps {
		cm[k] = v
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return ""
	}
	Execute(`CREATE TABLE IF NOT EXISTS t_p1_test (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(100) NOT NULL,
		version INTEGER NOT NULL DEFAULT 0,
		create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		is_removed BOOLEAN NOT NULL DEFAULT 0
	)`)
	return dir
}

func getP1Mapper(t *testing.T) P1TestMapper {
	t.Helper()
	m := NewMapper("P1TestMapper")
	mp, ok := m.(P1TestMapper)
	if !ok {
		t.Error("cast P1TestMapper failed")
	}
	return mp
}

// --- P1-7: SQL 安全防护 ---

func Test_SafeUpdate(t *testing.T) {
	dir := initP1Test(t, map[string]string{
		"mybatis.configuration.safe-update": "true",
	})
	if dir == "" {
		return
	}
	defer Close()

	RegisterModel(new(P1TestModel))
	RegisterMapper(new(P1TestMapper))
	mp := getP1Mapper(t)

	m := &P1TestModel{Name: "safe_test", Version: 0}
	n, err := mp.Insert(m)
	if err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	if n != 1 {
		t.Errorf("insert affected = %d, want 1", n)
	}

	_, err = mp.UpdateNoWhere()
	if err == nil {
		t.Error("updateNoWhere should be blocked by safe-update")
	} else if err != ErrSafeUpdateBlocked {
		t.Errorf("updateNoWhere error = %v, want ErrSafeUpdateBlocked", err)
	}

	_, err = mp.DeleteNoWhere()
	if err == nil {
		t.Error("deleteNoWhere should be blocked by safe-update")
	} else if err != ErrSafeUpdateBlocked {
		t.Errorf("deleteNoWhere error = %v, want ErrSafeUpdateBlocked", err)
	}

	rows, err := mp.SelectAll()
	if err != nil {
		t.Errorf("selectAll failed: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("selectAll rows = %d, want 1", len(rows))
	}
}

func Test_SafeUpdate_Disabled(t *testing.T) {
	dir := initP1Test(t, nil)
	if dir == "" {
		return
	}
	defer Close()

	RegisterModel(new(P1TestModel))
	RegisterMapper(new(P1TestMapper))
	mp := getP1Mapper(t)

	m := &P1TestModel{Name: "no_safe"}
	mp.Insert(m)

	_, err := mp.UpdateNoWhere()
	if err != nil {
		t.Errorf("updateNoWhere should succeed when safe-update disabled, got: %v", err)
	}
}

func Test_hasWhereClause(t *testing.T) {
	tests := []struct {
		sql  string
		want bool
	}{
		{"UPDATE t SET name='x' WHERE id=1", true},
		{"UPDATE t SET name='x'", false},
		{"DELETE FROM t WHERE id=1", true},
		{"DELETE FROM t", false},
		{"SELECT * FROM t WHERE id=1", true},
		{"UPDATE t SET name='x' -- WHERE in comment", false},
		{"UPDATE t SET name='x' WHERE id in (1,2)", true},
	}
	for _, tt := range tests {
		got := hasWhereClause(tt.sql)
		if got != tt.want {
			t.Errorf("hasWhereClause(%q) = %v, want %v", tt.sql, got, tt.want)
		}
	}
}

// --- P1-6: ID 生成策略 ---

func Test_IdGenerator_Snowflake(t *testing.T) {
	dir := initP1Test(t, map[string]string{
		"mybatis.configuration.id-type":            "snowflake",
		"mybatis.configuration.snowflake-worker-id": "1",
	})
	if dir == "" {
		return
	}
	defer Close()
	ClearHooks()
	SetDefaultIdType("snowflake")

	RegisterModel(new(P1TestModel))
	RegisterMapper(new(P1TestMapper))
	mp := getP1Mapper(t)

	m := &P1TestModel{Name: "snow_test", Version: 0}
	n, err := mp.Insert(m)
	if err != nil {
		t.Errorf("insert with snowflake id failed: %v", err)
		return
	}
	if n != 1 {
		t.Errorf("insert affected = %d, want 1", n)
	}

	rows, err := mp.SelectAll()
	if err != nil {
		t.Errorf("selectAll failed: %v", err)
	}
	if len(rows) == 0 {
		t.Error("selectAll returned 0 rows")
	}
	SetDefaultIdType("")
}

func Test_IdGenerator_UUID(t *testing.T) {
	dir := initP1Test(t, map[string]string{
		"mybatis.configuration.id-type": "uuid",
	})
	if dir == "" {
		return
	}
	defer Close()
	ClearHooks()

	SetDefaultIdType("uuid")
	RegisterModel(new(P1TestModel))
	RegisterMapper(new(P1TestMapper))
	mp := getP1Mapper(t)

	m := &P1TestModel{Name: "uuid_test", Version: 0}
	n, err := mp.Insert(m)
	if err != nil {
		t.Errorf("insert with uuid id failed: %v", err)
		return
	}
	if n != 1 {
		t.Errorf("insert affected = %d, want 1", n)
	}
	SetDefaultIdType("")
}

// --- P1-1: 乐观锁 ---

func Test_OptimisticLock(t *testing.T) {
	dir := initP1Test(t, nil)
	if dir == "" {
		return
	}
	defer Close()
	ClearHooks()
	SetDefaultIdType("")

	RegisterModel(new(P1TestModel))
	RegisterMapper(new(P1TestMapper))
	mp := getP1Mapper(t)

	sf, _ := mp.fetchSqlFunction("updateById")
	sf.HasVersion = true

	m := &P1TestModel{Name: "lock_test", Version: 0}
	n, err := mp.Insert(m)
	if err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	if n != 1 {
		t.Errorf("insert affected = %d, want 1", n)
	}

	rows, _ := mp.SelectAll()
	if len(rows) == 0 {
		t.Error("no rows after insert")
		return
	}
	row := rows[0]

	upd := &P1TestModel{Id: row.Id, Name: "updated_v1", Version: row.Version}
	n, err = mp.UpdateById(upd)
	if err != nil {
		t.Errorf("first update should succeed, got: %v", err)
	}

	stale := &P1TestModel{Id: row.Id, Name: "updated_v2_stale", Version: row.Version}
	_, err = mp.UpdateById(stale)
	if err == nil {
		t.Error("stale version update should fail with optimistic lock error")
	} else if err != ErrOptimisticLock {
		t.Errorf("stale update error = %v, want ErrOptimisticLock", err)
	}
}

// --- P1-2: 自动填充 ---

func Test_AutoFill(t *testing.T) {
	dir := initP1Test(t, nil)
	if dir == "" {
		return
	}
	defer Close()
	ClearHooks()
	SetDefaultIdType("")

	RegisterModel(new(P1TestModel))
	RegisterMapper(new(P1TestMapper))

	RegisterFillHandler("create_time", FillInsert, func(ctx context.Context, field *FieldInfo) interface{} {
		return time.Now()
	})
	RegisterFillHandler("update_time", FillInsertUpdate, func(ctx context.Context, field *FieldInfo) interface{} {
		return time.Now()
	})

	mp := getP1Mapper(t)

	m := &P1TestModel{Name: "fill_test", Version: 0}
	n, err := mp.Insert(m)
	if err != nil {
		t.Errorf("insert with auto-fill failed: %v", err)
		return
	}
	if n != 1 {
		t.Errorf("insert affected = %d, want 1", n)
	}

	rows, _ := mp.SelectAll()
	if len(rows) == 0 {
		t.Error("no rows after insert")
		return
	}
	if rows[0].CreateTime.IsZero() {
		t.Error("create_time should be auto-filled on insert")
	}
	if rows[0].UpdateTime.IsZero() {
		t.Error("update_time should be auto-filled on insert")
	}
	firstUpdate := rows[0].UpdateTime

	time.Sleep(10 * time.Millisecond)
	upd := &P1TestModel{Id: rows[0].Id, Name: "fill_updated", Version: rows[0].Version}
	mp.UpdateById(upd)

	rows2, _ := mp.SelectAll()
	if len(rows2) == 0 {
		t.Error("no rows after update")
		return
	}
	if rows2[0].UpdateTime.Before(firstUpdate) || rows2[0].UpdateTime.Equal(firstUpdate) {
		t.Error("update_time should be auto-filled on update with newer value")
	}
}

// --- P1-4: 批量操作 ---

func Test_InsertBatch(t *testing.T) {
	dir := initP1Test(t, nil)
	if dir == "" {
		return
	}
	defer Close()
	ClearHooks()
	SetDefaultIdType("")

	RegisterModel(new(P1TestModel))
	RegisterMapper(new(P1TestMapper))
	mp := getP1Mapper(t)

	models := []*P1TestModel{
		{Name: "batch_1", Version: 0},
		{Name: "batch_2", Version: 0},
		{Name: "batch_3", Version: 0},
	}
	n, err := mp.InsertBatch(models)
	if err != nil {
		t.Errorf("insertBatch failed: %v", err)
		return
	}
	if n != 3 {
		t.Errorf("insertBatch affected = %d, want 3", n)
	}

	rows, err := mp.SelectAll()
	if err != nil {
		t.Errorf("selectAll failed: %v", err)
	}
	if len(rows) != 3 {
		t.Errorf("selectAll rows = %d, want 3", len(rows))
	}
}

// --- P1-5: 鉴别器 ---

type P1DiscModel struct {
	Id       int64  `db:"id,pk"`
	Type     string `db:"type"`
	Name     string `db:"name"`
	Amount   int64  `db:"amount"`
	Duration int64  `db:"duration"`
}

func (P1DiscModel) TableName() string { return "t_p1_disc" }

type P1DiscMapper struct {
	BaseMapper
	SelectAll func() ([]P1DiscModel, error)
}

func initP1DiscTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="P1DiscMapper">
  <resultMap id="BaseResultMap" type="P1DiscModel">
    <id column="id" property="id" />
    <result column="type" property="type" />
    <result column="name" property="name" />
    <discriminator column="type" javaType="string">
      <case value="video" resultMap="VideoResultMap"/>
      <case value="audio" resultMap="AudioResultMap"/>
    </discriminator>
  </resultMap>
  <resultMap id="VideoResultMap" type="P1DiscModel">
    <id column="id" property="id" />
    <result column="type" property="type" />
    <result column="name" property="name" />
    <result column="duration" property="duration" />
  </resultMap>
  <resultMap id="AudioResultMap" type="P1DiscModel">
    <id column="id" property="id" />
    <result column="type" property="type" />
    <result column="name" property="name" />
    <result column="amount" property="amount" />
  </resultMap>
  <select id="selectAll" resultMap="BaseResultMap">
    select id, type, name, amount, duration from t_p1_disc
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "P1DiscMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	dbPath := filepath.Join(dir, "disc.db")
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return ""
	}
	Execute(`CREATE TABLE IF NOT EXISTS t_p1_disc (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type VARCHAR(20) NOT NULL,
		name VARCHAR(100) NOT NULL,
		amount INTEGER DEFAULT 0,
		duration INTEGER DEFAULT 0
	)`)
	Execute(`INSERT INTO t_p1_disc (type, name, amount, duration) VALUES ('video', 'vid1', 0, 120)`)
	Execute(`INSERT INTO t_p1_disc (type, name, amount, duration) VALUES ('audio', 'aud1', 500, 0)`)
	return dir
}

func Test_Discriminator(t *testing.T) {
	dir := initP1DiscTest(t)
	if dir == "" {
		return
	}
	defer Close()

	RegisterModel(new(P1DiscModel))
	RegisterMapper(new(P1DiscMapper))

	m := NewMapper("P1DiscMapper")
	mp, ok := m.(P1DiscMapper)
	if !ok {
		t.Error("cast P1DiscMapper failed")
		return
	}

	rows, err := mp.SelectAll()
	if err != nil {
		t.Errorf("selectAll failed: %v", err)
		return
	}
	if len(rows) != 2 {
		t.Errorf("selectAll rows = %d, want 2", len(rows))
		return
	}
	for _, row := range rows {
		if row.Type == "video" {
			if row.Duration != 120 {
				t.Errorf("video row duration = %d, want 120", row.Duration)
			}
		}
		if row.Type == "audio" {
			if row.Amount != 500 {
				t.Errorf("audio row amount = %d, want 500", row.Amount)
			}
		}
	}
}

// --- P1-3: 二级缓存 ---

func Test_SecondLevelCache(t *testing.T) {
	dir := initP1Test(t, map[string]string{
		"mybatis.configuration.cache-enabled":     "true",
		"mybatis.configuration.local-cache-size":   "256",
		"mybatis.configuration.local-cache-ttl":    "3600",
	})
	if dir == "" {
		return
	}
	defer Close()
	ClearHooks()
	SetDefaultIdType("")

	RegisterModel(new(P1TestModel))
	RegisterMapper(new(P1TestMapper))
	mp := getP1Mapper(t)

	m := &P1TestModel{Name: "cache_test", Version: 0}
	mp.Insert(m)

	rows1, err := mp.SelectAll()
	if err != nil {
		t.Errorf("first selectAll failed: %v", err)
	}
	if len(rows1) != 1 {
		t.Errorf("first selectAll rows = %d, want 1", len(rows1))
	}

	rows2, err := mp.SelectAll()
	if err != nil {
		t.Errorf("second selectAll (cached) failed: %v", err)
	}
	if len(rows2) != 1 {
		t.Errorf("second selectAll rows = %d, want 1", len(rows2))
	}

	upd := &P1TestModel{Id: rows1[0].Id, Name: "cache_updated", Version: rows1[0].Version}
	sf, _ := mp.fetchSqlFunction("updateById")
	sf.HasVersion = true
	mp.UpdateById(upd)

	rows3, _ := mp.SelectAll()
	if len(rows3) != 1 || rows3[0].Name != "cache_updated" {
		t.Errorf("after update, name = %q, want 'cache_updated'", rows3[0].Name)
	}
}

func Test_CacheDisabled(t *testing.T) {
	dir := initP1Test(t, nil)
	if dir == "" {
		return
	}
	defer Close()
	ClearHooks()
	SetDefaultIdType("")

	RegisterModel(new(P1TestModel))
	RegisterMapper(new(P1TestMapper))
	mp := getP1Mapper(t)

	m := &P1TestModel{Name: "nocache_test", Version: 0}
	mp.Insert(m)

	rows, _ := mp.SelectAll()
	if len(rows) != 1 {
		t.Errorf("selectAll rows = %d, want 1", len(rows))
	}
}
