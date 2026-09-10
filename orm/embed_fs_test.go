package orm

import (
	"embed"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// 内嵌 Mapper XML：本文件所在包编译期即打包进二进制，测试期间不落盘。
//
//go:embed testdata/embeddedmapper
var ormEmbeddedMapperFS embed.FS

type EmbedMpModel struct {
	Id   int
	Name string
}

// EmbedFsMapper 与 testdata/embeddedmapper/EmbedFsMapper.xml 的 namespace 对应。
type EmbedFsMapper struct {
	BaseMapper
	Insert     func(model EmbedMpModel) (int64, error)
	SelectAll  func() ([]EmbedMpModel, error)
	SelectById func(id int) (EmbedMpModel, error)
}

// initEmbedFsSqlite 打开 SQLite 并仅从 embed.FS 装载 Mapper（无任何磁盘 XML）。
func initEmbedFsSqlite(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "embed.db")
	cm := map[string]string{
		"spring.datasource.url": "jdbc:sqlite:" + dbPath,
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return
	}
	if err := RegisterMapperFS(ormEmbeddedMapperFS, "testdata/embeddedmapper"); err != nil {
		t.Errorf("RegisterMapperFS failed: %v", err)
	}
	// resultMap 中 type="EmbedMpModel" 需要先注册模型才能反射构造结果
	RegisterModel(new(EmbedMpModel))
}

// Test_RegisterMapperFS_EndToEnd embed.FS 内嵌 Mapper 全流程：
// 加载 → 绑定 → 建表 → 插入 → 查询，全程不依赖磁盘 XML。
func Test_RegisterMapperFS_EndToEnd(t *testing.T) {
	initEmbedFsSqlite(t)
	defer Close()

	if err := RegisterMapper(new(EmbedFsMapper)); err != nil {
		t.Errorf("RegisterMapper failed: %v", err)
		return
	}
	if _, err := Execute(`CREATE TABLE t_embed (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	mp := NewMapperPtr("EmbedFsMapper").(*EmbedFsMapper)
	if mp == nil {
		t.Error("NewMapperPtr returned nil")
		return
	}
	id, err := mp.Insert(EmbedMpModel{Name: "embedded-insert"})
	if err != nil {
		t.Errorf("insert via embedded mapper failed: %v", err)
		return
	}
	if id <= 0 {
		t.Error("insert should return positive id, got", id)
	}
	rows, err := mp.SelectAll()
	if err != nil {
		t.Errorf("SelectAll via embedded mapper failed: %v", err)
		return
	}
	if len(rows) != 1 || rows[0].Name != "embedded-insert" {
		t.Errorf("SelectAll got %+v, want 1 row named embedded-insert", rows)
	}
	one, err := mp.SelectById(int(id))
	if err != nil {
		t.Errorf("SelectById via embedded mapper failed: %v", err)
		return
	}
	if one.Name != "embedded-insert" {
		t.Errorf("SelectById got %+v, want name embedded-insert", one)
	}
}

// Test_RegisterMapperFS_LoadBeforeRegister 先加载 XML、后注册结构体（推荐顺序）同样可用。
func Test_RegisterMapperFS_LoadBeforeRegister(t *testing.T) {
	initEmbedFsSqlite(t)
	defer Close()

	// RegisterMapperFS 已在上方完成，此处仅注册结构体；注册时应自动重新绑定
	if err := RegisterMapper(new(EmbedFsMapper)); err != nil {
		t.Errorf("RegisterMapper failed: %v", err)
		return
	}
	mp := NewMapperPtr("EmbedFsMapper").(*EmbedFsMapper)
	if mp.SelectAll == nil {
		t.Error("SelectAll should be bound after RegisterMapper")
	}
}

// Test_RegisterMapperFS_RegisterBeforeLoad 先注册结构体、后加载 XML 也可用：
// RegisterMapper 时 XML 未就绪不报错，RegisterMapperFS 后需 ReloadMappers 补绑。
func Test_RegisterMapperFS_RegisterBeforeLoad(t *testing.T) {
	dir := t.TempDir()
	cm := map[string]string{
		"spring.datasource.url": "jdbc:sqlite:" + filepath.Join(dir, "embed2.db"),
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return
	}
	defer Close()

	// 无 XML 时注册结构体：不报错，SqlFunc 尚未绑定
	if err := RegisterMapper(new(EmbedFsMapper)); err != nil {
		t.Errorf("RegisterMapper without xml should not fail: %v", err)
		return
	}
	if err := RegisterMapperFS(ormEmbeddedMapperFS, "testdata/embeddedmapper"); err != nil {
		t.Errorf("RegisterMapperFS failed: %v", err)
		return
	}
	if err := ReloadMappers(); err != nil {
		t.Errorf("ReloadMappers failed: %v", err)
		return
	}
	// 绑定成功后应能真正执行
	if _, err := Execute(`CREATE TABLE t_embed (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	mp := NewMapperPtr("EmbedFsMapper").(*EmbedFsMapper)
	if _, err := mp.Insert(EmbedMpModel{Name: "late-load"}); err != nil {
		t.Errorf("insert after ReloadMappers failed: %v", err)
	}
}

// Test_RegisterMapperFS_ReplacesPrevious 再次 RegisterMapperFS 会替换既有 SQL 定义。
func Test_RegisterMapperFS_ReplacesPrevious(t *testing.T) {
	dir := t.TempDir()
	cm := map[string]string{
		"spring.datasource.url": "jdbc:sqlite:" + filepath.Join(dir, "embed3.db"),
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return
	}
	defer Close()

	if err := RegisterMapperFS(ormEmbeddedMapperFS, "testdata/embeddedmapper"); err != nil {
		t.Errorf("first RegisterMapperFS failed: %v", err)
		return
	}
	// 第二次加载另一批 Mapper，旧定义被替换
	if err := RegisterMapperFS(ormEmbeddedMapperFS, "testdata/embeddedmapper"); err != nil {
		t.Errorf("second RegisterMapperFS failed: %v", err)
		return
	}
	if gCache.sqls == nil {
		t.Error("sqls should be set")
		return
	}
	if _, ok := gCache.sqls.NamedMappers["EmbedFsMapper"]; !ok {
		t.Error("EmbedFsMapper missing after reload")
	}
}

// Test_RegisterMapperFS_InvalidXML 无效 XML / 非 Mapper XML 不 panic，仅记日志跳过。
func Test_RegisterMapperFS_InvalidXML(t *testing.T) {
	dir := t.TempDir()
	cm := map[string]string{
		"spring.datasource.url": "jdbc:sqlite:" + filepath.Join(dir, "embed4.db"),
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return
	}
	defer Close()

	// 指向一个不含任何 XML 的内嵌目录：应正常返回，不 panic
	if err := RegisterMapperFS(ormEmbeddedMapperFS, "testdata/no-such-dir"); err != nil {
		t.Errorf("RegisterMapperFS with missing pattern should not fail: %v", err)
	}
}

// Test_RegisterMapperSources_DiskOverridesEmbed 磁盘目录覆盖内嵌 Mapper（便于本地热修 SQL）。
func Test_RegisterMapperSources_DiskOverridesEmbed(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "override.db")
	cm := map[string]string{
		"spring.datasource.url": "jdbc:sqlite:" + dbPath,
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return
	}
	defer Close()

	// 磁盘覆盖目录：同名 namespace，selectAll 改为返回固定表
	overrideDir := filepath.Join(dir, "override")
	if err := os.MkdirAll(overrideDir, 0755); err != nil {
		t.Errorf("mkdir failed: %v", err)
		return
	}
	overrideXML := `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="EmbedFsMapper">
  <resultMap id="BaseResultMap" type="EmbedMpModel">
    <id column="id" jdbcType="INTEGER" property="id" />
    <result column="name" jdbcType="VARCHAR" property="name" />
  </resultMap>
  <select id="selectAll" resultMap="BaseResultMap">
    select id, name from t_embed_override
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(overrideDir, "EmbedFsMapper.xml"), []byte(overrideXML), 0644); err != nil {
		t.Errorf("write override xml failed: %v", err)
		return
	}
	if err := RegisterMapperSources(
		MapperSource{FS: ormEmbeddedMapperFS, Patterns: []string{"testdata/embeddedmapper"}},
		MapperSource{Patterns: []string{overrideDir}},
	); err != nil {
		t.Errorf("RegisterMapperSources failed: %v", err)
		return
	}
	if err := RegisterMapper(new(EmbedFsMapper)); err != nil {
		t.Errorf("RegisterMapper failed: %v", err)
		return
	}
	if _, err := Execute(`CREATE TABLE t_embed_override (
		id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	if _, err := Execute(`INSERT INTO t_embed_override (name) VALUES (?)`, "from-disk"); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	mp := NewMapperPtr("EmbedFsMapper").(*EmbedFsMapper)
	rows, err := mp.SelectAll()
	if err != nil {
		t.Errorf("SelectAll failed: %v", err)
		return
	}
	if len(rows) != 1 || rows[0].Name != "from-disk" {
		t.Errorf("disk override not effective, got %+v", rows)
	}
}
