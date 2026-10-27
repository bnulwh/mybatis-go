package orm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// <bind> 端到端模型 / Mapper（P0-3）

type SqliteBindModel struct {
	Id   int
	Name string
}

// SqliteBindParam struct 参数：<bind> 引用字段（parameterType 走 struct 渲染路径）。
type SqliteBindParam struct {
	Keyword string
}

type SqliteBindMapper struct {
	BaseMapper
	// 字符串拼接绑定：<bind name="pattern" value="'%' + keyword + '%'"/>
	SelectByKeyword func(keyword string) ([]SqliteBindModel, error)
	// 纯字面量绑定：语句无调用方参数，bind 仍须在渲染期求值
	SelectByLiteral func() ([]SqliteBindModel, error)
	// map 参数 + 绑定：调用方 map 不应被写入绑定变量（不产生副作用）
	SelectByMap func(param map[string]interface{}) ([]SqliteBindModel, error)
	// struct 参数 + 绑定：字段引用（keyword）经 struct → map 转换后参与求值
	SelectByParam func(param SqliteBindParam) ([]SqliteBindModel, error)
}

const sqliteBindXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="SqliteBindMapper">
  <resultMap id="BaseResultMap" type="SqliteBindModel">
    <id column="id" jdbcType="INTEGER" property="id" />
    <result column="name" jdbcType="VARCHAR" property="name" />
  </resultMap>
  <select id="selectByKeyword" resultMap="BaseResultMap">
    <bind name="pattern" value="'%' + keyword + '%'"/>
    select id, name from t_bind where name like #{pattern} order by id
  </select>
  <select id="selectByLiteral" resultMap="BaseResultMap">
    <bind name="literal" value="'beta'"/>
    select id, name from t_bind where name = #{literal} order by id
  </select>
  <select id="selectByMap" parameterType="map" resultMap="BaseResultMap">
    <bind name="pattern" value="'%' + keyword + '%'"/>
    <bind name="minId" value="1 + 1"/>
    select id, name from t_bind where name like #{pattern} and id >= #{minId} order by id
  </select>
  <select id="selectByParam" parameterType="SqliteBindParam" resultMap="BaseResultMap">
    <bind name="pattern" value="'%' + keyword + '%'"/>
    select id, name from t_bind where name like #{pattern} order by id
  </select>
</mapper>`

func initSqliteBindTest(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "SqliteBindMapper.xml"), []byte(sqliteBindXML), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return
	}
	dbPath := filepath.Join(dir, "bind.db")
	if err := InitializeFromSettings(map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return
	}
	RegisterModel(new(SqliteBindModel))
	RegisterModel(new(SqliteBindParam))
	RegisterModel(new(SqliteBindParam))
	if _, err := Execute(`CREATE TABLE t_bind (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	for _, name := range []string{"alpha", "beta", "gamma"} {
		if _, err := Execute(`INSERT INTO t_bind (name) VALUES (?)`, name); err != nil {
			t.Errorf("insert %v failed: %v", name, err)
			return
		}
	}
}

// Test_SqliteBindVariable <bind> 拼接绑定 / 纯字面量绑定的端到端行为。
func Test_SqliteBindVariable(t *testing.T) {
	initSqliteBindTest(t)
	defer Close()

	if err := RegisterMapper(new(SqliteBindMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mapper := NewMapper("SqliteBindMapper").(SqliteBindMapper)

	rows, err := mapper.SelectByKeyword("et")
	if err != nil {
		t.Errorf("SelectByKeyword failed: %v", err)
		return
	}
	if len(rows) != 1 || rows[0].Name != "beta" {
		t.Errorf("SelectByKeyword(et) = %v, want [beta]", rows)
	}

	rows, err = mapper.SelectByLiteral()
	if err != nil {
		t.Errorf("SelectByLiteral failed: %v", err)
		return
	}
	if len(rows) != 1 || rows[0].Name != "beta" {
		t.Errorf("SelectByLiteral() = %v, want [beta]", rows)
	}

	rows, err = mapper.SelectByParam(SqliteBindParam{Keyword: "amm"})
	if err != nil {
		t.Errorf("SelectByParam failed: %v", err)
		return
	}
	if len(rows) != 1 || rows[0].Name != "gamma" {
		t.Errorf("SelectByParam(amm) = %v, want [gamma]", rows)
	}
}

// Test_SqliteBindMapParam map 参数绑定：绑定变量生效且不污染调用方 map。
func Test_SqliteBindMapParam(t *testing.T) {
	initSqliteBindTest(t)
	defer Close()

	if err := RegisterMapper(new(SqliteBindMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mapper := NewMapper("SqliteBindMapper").(SqliteBindMapper)
	param := map[string]interface{}{"keyword": "mm"}
	rows, err := mapper.SelectByMap(param)
	if err != nil {
		t.Errorf("SelectByMap failed: %v", err)
		return
	}
	if len(rows) != 1 || rows[0].Name != "gamma" {
		t.Errorf("SelectByMap(mm) = %v, want [gamma]", rows)
	}
	if _, ok := param["pattern"]; ok {
		t.Errorf("caller map must not be polluted by <bind>, got %v", param)
	}
}

// Test_UnsupportedTag_Registration 未注册标签：默认 Warn 可启动，strict-unknown-tag 注册期报错。
func Test_UnsupportedTag_Registration(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="UnknownTagMapper">
  <insert id="insert" parameterType="map">
    insert into t_unknown (name) values ('x')
    <selectKey keyProperty="id" resultType="int">select 1</selectKey>
  </insert>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "UnknownTagMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return
	}
	dbPath := filepath.Join(dir, "unknown.db")
	settings := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}
	if err := InitializeFromSettings(settings); err != nil {
		t.Errorf("default mode should tolerate unsupported tags, got: %v", err)
		return
	}
	defer Close()
	if err := RegisterMapper(new(UnknownTagMapper)); err != nil {
		t.Errorf("default mode register mapper failed: %v", err)
	}
	sf := gCache.sqls.NamedMappers["unknowntagmapper"].NamedFunctions["insert"]
	if sf == nil {
		t.Error("statement insert not parsed")
		return
	}
	if tags := sf.UnsupportedTags; len(tags) != 1 || tags[0] != "selectKey" {
		t.Errorf("UnsupportedTags = %v, want [selectKey]", tags)
	}

	// 严格模式：注册期直接失败
	SetStrictUnknownTags(true)
	defer SetStrictUnknownTags(false)
	Close()
	err := InitializeFromSettings(settings)
	if err == nil {
		t.Error("strict-unknown-tag should fail registration for unsupported tag")
		return
	}
	if !strings.Contains(err.Error(), "unsupported xml tags") {
		t.Errorf("strict error should mention unsupported xml tags, got: %v", err)
	}
}

// UnknownTagMapper 仅用于未注册标签注册期测试。
type UnknownTagMapper struct {
	BaseMapper
	Insert func(param map[string]interface{}) (int64, error)
}
