package orm

import (
	"path/filepath"
	"testing"
	"testing/fstest"

	_ "modernc.org/sqlite"
)

// E3：namespace 冲突检测、NamespacePrefix 加载、RegisterNamespaceAlias 别名 —— 端到端验证。

type E3MpModel struct {
	Id   int
	Name string
}

// E3AliasMapper 模拟「旧名结构体」：XML namespace 已改为 E3UserMapper，靠别名绑定不失效。
type E3AliasMapper struct {
	BaseMapper
	SelectAll func() ([]E3MpModel, error)
}

// QualityE3PlatformMapper 模拟 NamespacePrefix=Quality 来源的绑定结构体（前缀+短名）。
type QualityE3PlatformMapper struct {
	BaseMapper
	SelectAll func() ([]E3MpModel, error)
}

func initE3Sqlite(t *testing.T, name string) {
	dir := t.TempDir()
	cm := map[string]string{
		"spring.datasource.url": "jdbc:sqlite:" + filepath.Join(dir, name),
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
	}
}

// Test_RegisterNamespaceAlias_BindAndExec 别名端到端：旧名结构体经别名绑定到改名后的 XML 并可执行。
func Test_RegisterNamespaceAlias_BindAndExec(t *testing.T) {
	initE3Sqlite(t, "e3-alias.db")
	defer Close()

	mfs := fstest.MapFS{
		"mapper/E3UserMapper.xml": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="E3UserMapper">
  <resultMap id="BaseResultMap" type="E3MpModel">
    <id column="id" jdbcType="INTEGER" property="id" />
    <result column="name" jdbcType="VARCHAR" property="name" />
  </resultMap>
  <select id="selectAll" resultMap="BaseResultMap">
    select id, name from t_e3_alias
  </select>
</mapper>`)},
	}
	if err := RegisterMapperSources(MapperSource{FS: mfs, Patterns: []string{"mapper"}}); err != nil {
		t.Errorf("RegisterMapperSources failed: %v", err)
		return
	}
	RegisterModel(new(E3MpModel))
	if err := RegisterNamespaceAlias("E3AliasMapper", "E3UserMapper"); err != nil {
		t.Errorf("RegisterNamespaceAlias failed: %v", err)
		return
	}
	if err := RegisterMapper(new(E3AliasMapper)); err != nil {
		t.Errorf("RegisterMapper failed: %v", err)
		return
	}
	if _, err := Execute(`CREATE TABLE t_e3_alias (
		id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	if _, err := Execute(`INSERT INTO t_e3_alias (name) VALUES ('via-alias')`); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	mp := NewMapperPtr("E3AliasMapper").(*E3AliasMapper)
	rows, err := mp.SelectAll()
	if err != nil {
		t.Errorf("SelectAll via alias failed: %v", err)
		return
	}
	if len(rows) != 1 || rows[0].Name != "via-alias" {
		t.Errorf("SelectAll via alias got %+v, want 1 row named via-alias", rows)
	}
}

// Test_RegisterNamespaceAlias_Errors 别名注册的失败路径：空别名 / 目标未加载。
func Test_RegisterNamespaceAlias_Errors(t *testing.T) {
	initE3Sqlite(t, "e3-alias-err.db")
	defer Close()

	if err := RegisterNamespaceAlias("", "E3UserMapper"); err == nil {
		t.Error("empty alias should return error")
	}
	if err := RegisterNamespaceAlias("SomeAlias", "NoSuchMapperAtAll"); err == nil {
		t.Error("alias to unknown mapper should return error")
	}
}

// Test_MapperSource_NamespacePrefix_BindAndExec 前缀加载端到端：
// 结构体命名为 前缀+短名 即可绑定带前缀来源的 XML，裸短名键不注册。
func Test_MapperSource_NamespacePrefix_BindAndExec(t *testing.T) {
	initE3Sqlite(t, "e3-prefix.db")
	defer Close()

	mfs := fstest.MapFS{
		"mapper/E3PlatformMapper.xml": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="E3PlatformMapper">
  <resultMap id="BaseResultMap" type="E3MpModel">
    <id column="id" jdbcType="INTEGER" property="id" />
    <result column="name" jdbcType="VARCHAR" property="name" />
  </resultMap>
  <select id="selectAll" resultMap="BaseResultMap">
    select id, name from t_e3_prefix
  </select>
</mapper>`)},
	}
	if err := RegisterMapperSources(
		MapperSource{FS: mfs, Patterns: []string{"mapper"}, NamespacePrefix: "Quality"},
	); err != nil {
		t.Errorf("RegisterMapperSources failed: %v", err)
		return
	}
	RegisterModel(new(E3MpModel))
	if err := RegisterMapper(new(QualityE3PlatformMapper)); err != nil {
		t.Errorf("RegisterMapper failed: %v", err)
		return
	}
	if gCache.sqls == nil {
		t.Error("sqls should be set")
		return
	}
	// 带前缀来源不注册裸短名键
	if _, ok := gCache.sqls.NamedMappers["E3PlatformMapper"]; ok {
		t.Error("bare short-name key should not be registered for prefixed source")
	}
	if mp := gCache.sqls.NamedMappers["QualityE3PlatformMapper"]; mp == nil || mp.Namespace != "Quality.E3PlatformMapper" {
		t.Errorf("prefixed binding key missing or namespace wrong: %+v", mp)
		return
	}
	if _, err := Execute(`CREATE TABLE t_e3_prefix (
		id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	if _, err := Execute(`INSERT INTO t_e3_prefix (name) VALUES ('via-prefix')`); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	mp := NewMapperPtr("QualityE3PlatformMapper").(*QualityE3PlatformMapper)
	rows, err := mp.SelectAll()
	if err != nil {
		t.Errorf("SelectAll via prefixed source failed: %v", err)
		return
	}
	if len(rows) != 1 || rows[0].Name != "via-prefix" {
		t.Errorf("SelectAll via prefixed source got %+v, want 1 row named via-prefix", rows)
	}
}

// Test_NamespaceConflicts_orm orm 层冲突查询：同 namespace 多文件被记录（后者覆盖前者）。
func Test_NamespaceConflicts_orm(t *testing.T) {
	initE3Sqlite(t, "e3-conflict.db")
	defer Close()

	mfs := fstest.MapFS{
		"a/E3DupMapper.xml": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="E3DupMapper">
  <select id="fromOne" resultType="int">select 1</select>
</mapper>`)},
		"b/E3DupMapper.xml": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="E3DupMapper">
  <select id="fromTwo" resultType="int">select 2</select>
</mapper>`)},
	}
	if err := RegisterMapperSources(
		MapperSource{FS: mfs, Patterns: []string{"a"}},
		MapperSource{FS: mfs, Patterns: []string{"b"}},
	); err != nil {
		t.Errorf("RegisterMapperSources failed: %v", err)
		return
	}
	conflicts := NamespaceConflicts()
	if len(conflicts) != 1 {
		t.Errorf("NamespaceConflicts want 1, got %d", len(conflicts))
		return
	}
	if conflicts[0].Namespace != "E3DupMapper" || len(conflicts[0].Files) != 2 {
		t.Errorf("unexpected conflict record: %+v", conflicts[0])
	}
}
