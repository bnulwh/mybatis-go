package orm

import (
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// GenKeyModel/GenKeyMapper：useGeneratedKeys 回填专用模型/Mapper（S-11），独立命名避免与既有测试冲突。

type GenKeyModel struct {
	Id   int64
	Name string
}

type GenKeyMapper struct {
	BaseMapper
	InsertPtr func(model *GenKeyModel) (int64, error)
}

const genKeyMapperXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="GenKeyMapper">
  <insert id="insertPtr" parameterType="GenKeyModel" useGeneratedKeys="true" keyProperty="id">
    insert into t_gen_key (name) values (#{name})
  </insert>
</mapper>
`

// Test_SqliteGeneratedKeysBackfill 端到端验证：useGeneratedKeys + keyProperty
// 在 insert 后把自增主键回填到入参 struct（指针）字段（S-11）。
func Test_SqliteGeneratedKeysBackfill(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "GenKeyMapper.xml"), []byte(genKeyMapperXML), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return
	}
	dbPath := filepath.Join(dir, "test.db")
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return
	}
	defer Close()

	if _, err := Execute(`CREATE TABLE t_gen_key (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	RegisterModel(new(GenKeyModel))
	if err := RegisterMapper(new(GenKeyMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("GenKeyMapper").(GenKeyMapper)

	m := &GenKeyModel{Name: "gen_user"}
	if _, err := mp.InsertPtr(m); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	if m.Id != 1 {
		t.Errorf("backfilled Id = %v, want 1", m.Id)
	}
	m2 := &GenKeyModel{Name: "gen_user2"}
	if _, err := mp.InsertPtr(m2); err != nil {
		t.Errorf("insert2 failed: %v", err)
		return
	}
	if m2.Id != 2 {
		t.Errorf("backfilled Id2 = %v, want 2", m2.Id)
	}
}
