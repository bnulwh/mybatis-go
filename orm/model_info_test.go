package orm

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// TagUserModel P0-2 e2e 模型：显式主键 + 显式逻辑删除列（bool）。
type TagUserModel struct {
	UserId    int64  `db:"user_id,pk"`
	UserName  string `db:"user_name"`
	IsRemoved bool   `db:"is_removed,logic"`
}

type TagUserMapper struct {
	BaseMapper
	Insert      func(TagUserModel) (int64, error)
	DeleteById  func(int64) (int64, error)
	SelectById  func(int64) ([]TagUserModel, error)
	SelectList  func() ([]TagUserModel, error)
	SelectCount func() ([]int64, error)
}

// parseFullModel 覆盖全部 tag 选项的解析用例模型。
type parseFullModel struct {
	Id       int64     `db:"id,pk"`
	Name     string    `db:"user_name"`
	Removed  bool      `db:"is_removed,logic"`
	Ver      int       `db:"ver,version"`
	Created  time.Time `db:"created_at,fill:insert"`
	Modified time.Time `db:"modified_at,fill:insert_update"`
	Secret   string    `db:"-"`
	Note     string
}

// parseTableModel 显式 TableName() 方法用例。
type parseTableModel struct {
	Id int64 `db:"id,pk"`
}

func (m parseTableModel) TableName() string { return "my_table" }

// Test_ParseModelInfo db tag 元数据解析：全选项、列名缺省推导、`-` 排除、
// TableName() 显式表名、无 tag 返回 nil、未知选项报错。
func Test_ParseModelInfo(t *testing.T) {
	info, err := ParseModelInfo(new(parseFullModel))
	if err != nil {
		t.Error("parse full model error:", err)
		return
	}
	if info.TableName != "parse_full" {
		t.Error("TableName =", info.TableName, "want parse_full")
	}
	if info.ExplicitTable {
		t.Error("ExplicitTable should be false")
	}
	if info.PrimaryColumn != "id" {
		t.Error("PrimaryColumn =", info.PrimaryColumn, "want id")
	}
	if info.LogicColumn != "is_removed" {
		t.Error("LogicColumn =", info.LogicColumn, "want is_removed")
	}
	if info.VersionColumn != "ver" {
		t.Error("VersionColumn =", info.VersionColumn, "want ver")
	}
	// Secret（db:"-"）不进入字段清单，其余 7 个字段全部映射
	if len(info.Fields) != 7 {
		t.Errorf("Fields count = %d, want 7", len(info.Fields))
	}
	var note *FieldInfo
	for _, f := range info.Fields {
		if f.Name == "Note" {
			note = f
		}
	}
	if note == nil || note.Column != "note" {
		t.Error("untagged field Note should map to column `note`")
	}
	// fill 时机
	if got := info.FillColumns(FillInsert); len(got) != 2 {
		t.Errorf("FillColumns(insert) = %d, want 2", len(got))
	}
	if got := info.FillColumns(FillUpdate); len(got) != 1 {
		t.Errorf("FillColumns(update) = %d, want 1", len(got))
	}

	// TableName() 方法 → 显式表名
	info2, err := ParseModelInfo(new(parseTableModel))
	if err != nil {
		t.Error("parse table model error:", err)
		return
	}
	if !info2.ExplicitTable || info2.TableName != "my_table" {
		t.Error("TableName =", info2.TableName, "explicit =", info2.ExplicitTable, "want my_table/true")
	}

	// 无任何 db tag → (nil, nil)
	info3, err := ParseModelInfo(new(TypeHandlerModel))
	if err != nil || info3 != nil {
		t.Error("plain model should be (nil, nil), got", info3, err)
	}

	// 列名缺省 + pk：db:",pk" → 列名 camelToSnake
	info4, err := ParseModelInfo(new(struct {
		UserId int64 `db:",pk"`
	}))
	if err != nil {
		t.Error("parse empty-col model error:", err)
		return
	}
	if info4.PrimaryColumn != "user_id" {
		t.Error("PrimaryColumn =", info4.PrimaryColumn, "want user_id")
	}

	// 未知选项 → error
	if _, err := ParseModelInfo(new(struct {
		Id int64 `db:"id,bad"`
	})); err == nil {
		t.Error("unknown option should return error")
	}

	// 非 struct 指针 → error
	if _, err := ParseModelInfo(1); err == nil {
		t.Error("non-struct should return error")
	}
}

// Test_RegisterModelInfoCache 注册期元数据入缓存：5 键命中、无 tag 模型存 nil、
// GetModelInfo 查询、provider 回调按显式元数据门控。
func Test_RegisterModelInfoCache(t *testing.T) {
	RegisterModel(new(TagUserModel))
	RegisterModel(new(TypeHandlerModel))

	gCache.models.mu.RLock()
	tagInfo := gCache.models.Infos["TagUserModel"]
	plainInfo := gCache.models.Infos["TypeHandlerModel"]
	shortInfo := gCache.models.Infos["tagusermodel"]
	gCache.models.mu.RUnlock()
	if tagInfo == nil {
		t.Error("TagUserModel info should be cached")
		return
	}
	if plainInfo != nil {
		t.Error("model without db tags should cache nil info")
	}
	if shortInfo == nil {
		t.Error("lower(full-name) key should be cached")
	}

	// GetModelInfo
	if got := GetModelInfo(new(TagUserModel)); got == nil || got.TableName != "tag_user" {
		t.Error("GetModelInfo =", got, "want table tag_user")
	}
	if got := GetModelInfo(new(TypeHandlerModel)); got != nil {
		t.Error("GetModelInfo on plain model should be nil")
	}

	// provider 回调：显式主键/逻辑删除列 → 返回 tag 推导结构
	ts := modelStructureProviderCallback("TagUserModel")
	if ts == nil {
		t.Error("provider callback should return structure for TagUserModel")
		return
	}
	if ts.Table != "tag_user" {
		t.Error("provider Table =", ts.Table, "want tag_user")
	}
	if ts.PrimaryColumn == nil || ts.PrimaryColumn.Name != "user_id" {
		t.Error("provider PrimaryColumn want user_id")
	}
	if ts.LogicColumn == nil || ts.LogicColumn.Name != "is_removed" {
		t.Error("provider LogicColumn want is_removed")
	}
	if modelStructureProviderCallback("NoSuchModel") != nil {
		t.Error("provider callback should return nil for unknown model")
	}

	// 仅列改名（无显式表名/主键/逻辑删除）→ 门控不放行，保持 resultMap 推导
	type renameOnlyModel struct {
		Name string `db:"user_name"`
	}
	RegisterModel(new(renameOnlyModel))
	if modelStructureProviderCallback("renameOnlyModel") != nil {
		t.Error("rename-only model should not pass explicit-meta gate")
	}
}

// initTagUserTest 初始化 SQLite 数据源（TagUser 专用 mapper，XML 不含任何 MP 内置方法，
// 由加载期 ensureMPBuiltinCRUD 经 provider 元数据在内存补生成）。
func initTagUserTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="TagUserMapper">
  <resultMap id="BaseResultMap" type="TagUserModel">
    <id column="user_id" jdbcType="INTEGER" property="userId" />
    <result column="user_name" jdbcType="VARCHAR" property="userName" />
  </resultMap>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "TagUserMapper.xml"), []byte(xml), 0644); err != nil {
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

// Test_SqliteMPBuiltinWithTag SQLite 端到端：XML 仅有 resultMap，MP 内置 CRUD 由
// struct db tag 元数据（provider）在加载期内存补生成；deleteById 走显式逻辑删除列
// is_removed（update set is_removed=true），select 自动过滤 is_removed = false。
func Test_SqliteMPBuiltinWithTag(t *testing.T) {
	// 先注册模型：provider 在 mapper XML 加载期查询元数据缓存
	RegisterModel(new(TagUserModel))
	dir := initTagUserTest(t)
	if dir == "" {
		return
	}
	defer Close()

	if _, err := Execute(`CREATE TABLE tag_user (
		user_id INTEGER PRIMARY KEY,
		user_name TEXT,
		is_removed BOOLEAN NOT NULL DEFAULT 0)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	if err := RegisterMapper(new(TagUserMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("TagUserMapper").(TagUserMapper)

	// insert：逻辑删除列不进入 insert 列清单
	if _, err := mp.Insert(TagUserModel{UserId: 1, UserName: "u1"}); err != nil {
		t.Error("insert u1 failed:", err)
		return
	}
	if _, err := mp.Insert(TagUserModel{UserId: 2, UserName: "u2"}); err != nil {
		t.Error("insert u2 failed:", err)
		return
	}
	// selectList：自动过滤 is_removed = false
	list, err := mp.SelectList()
	if err != nil {
		t.Error("select list failed:", err)
		return
	}
	if len(list) != 2 {
		t.Errorf("select list count = %d, want 2", len(list))
		return
	}
	// selectById：命中未删除行
	rows, err := mp.SelectById(1)
	if err != nil {
		t.Error("select by id failed:", err)
		return
	}
	if len(rows) != 1 || rows[0].UserName != "u1" || rows[0].IsRemoved {
		t.Errorf("select by id = %+v, want u1 not removed", rows)
		return
	}
	// deleteById → update tag_user set is_removed=true where user_id=1
	affected, err := mp.DeleteById(1)
	if err != nil {
		t.Error("delete by id failed:", err)
		return
	}
	if affected != 1 {
		t.Errorf("delete by id affected = %d, want 1", affected)
		return
	}
	// 逻辑删除后：list 只剩 u2、selectById(1) 无行
	list2, err := mp.SelectList()
	if err != nil {
		t.Error("select list after delete failed:", err)
		return
	}
	if len(list2) != 1 || list2[0].UserName != "u2" {
		t.Errorf("select list after delete = %+v, want only u2", list2)
		return
	}
	rows2, err := mp.SelectById(1)
	if err != nil {
		t.Error("select by id after delete failed:", err)
		return
	}
	if len(rows2) != 0 {
		t.Errorf("select by id after delete = %d rows, want 0", len(rows2))
		return
	}
	// selectCount 过滤已删除行
	counts, err := mp.SelectCount()
	if err != nil {
		t.Error("select count failed:", err)
		return
	}
	if len(counts) != 1 || counts[0] != 1 {
		t.Errorf("select count = %v, want [1]", counts)
		return
	}
	// 物理行仍在，仅标记位翻转（软删除而非物理删除）
	rs, err := Query("select is_removed from tag_user where user_id = 1")
	if err != nil {
		t.Error("raw query failed:", err)
		return
	}
	if len(rs) != 1 {
		t.Errorf("raw query rows = %d, want 1", len(rs))
		return
	}
	switch v := rs[0]["is_removed"].(type) {
	case bool:
		if !v {
			t.Error("is_removed = false, want true")
		}
	case int64:
		if v != 1 {
			t.Errorf("is_removed = %d, want 1", v)
		}
	default:
		t.Errorf("is_removed = %v (%T), want true", rs[0]["is_removed"], rs[0]["is_removed"])
	}
}
