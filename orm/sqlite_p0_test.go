package orm

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// P0 端到端验收：共享 Java XML（无 parameterType）零改造注册 + 变参不 panic + SQLite 真实执行

// MdmOrgRaw 语义：语句无 parameterType（1.1 焦点），resultType 用 map 避免卷入 TODO M-04
// （自定义 resultType 短类名解析不在本计划范围）。
type MdmOrgRawMapper struct {
	BaseMapper
	// 1.1：语句无 parameterType、Go 函数有参 → 注册成功（单 map 参走顶层键，Java POJO 语义）
	SelectByOrgCode func(params map[string]interface{}) ([]map[string]interface{}, error)
	// 1.3：变参 + tag 长度 > NumIn → 注册不 panic、运行按 tag 名打包
	SelectByOrgCodeV func(args ...interface{}) ([]map[string]interface{}, error) `args:"orgCode,deleted"`
}

func initP0Sqlite(t *testing.T) string {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	// 与 samples/mapper/mdm/MdmOrgRawMapper.xml 同构：无 parameterType、含 #{} 占位符
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="com.cncec.project.mdm.mapper.MdmOrgRawMapper">
  <select id="selectByOrgCode" resultType="map">
    select org_code, deleted_at as deleted_at from mdm_org_raw where org_code = #{orgCode} and deleted_at is null
  </select>
  <select id="selectByOrgCodeV" resultType="map">
    select org_code from mdm_org_raw where org_code = #{orgCode} and deleted_at = #{deleted}
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "MdmOrgRawMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	dbPath := filepath.Join(dir, "p0.db")
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

func Test_P0_SharedJavaXml_NoParameterType(t *testing.T) {
	dir := initP0Sqlite(t)
	if dir == "" {
		return
	}
	defer Close()
	if _, err := Execute(`CREATE TABLE mdm_org_raw (org_code TEXT, deleted_at TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	if _, err := Execute(`INSERT INTO mdm_org_raw (org_code, deleted_at) VALUES ('O1', NULL), ('O2', '2026-01-01')`); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	// 注册不得因「无 parameterType」失败（1.1 核心验收）；变参函数不得 panic（1.3 验收）
	if err := RegisterMapper(new(MdmOrgRawMapper)); err != nil {
		t.Errorf("register shared java xml mapper failed: %v", err)
		return
	}
	mp := NewMapper("MdmOrgRawMapper").(MdmOrgRawMapper)

	// 1.1 + 1.2：map 参数按名绑定
	rs, err := mp.SelectByOrgCode(map[string]interface{}{"orgCode": "O1"})
	if err != nil {
		t.Errorf("selectByOrgCode failed: %v", err)
		return
	}
	if len(rs) != 1 {
		t.Errorf("selectByOrgCode expect 1 row, got %v", rs)
	}
	if v, ok := rs[0]["org_code"].(string); !ok || v != "O1" {
		t.Errorf("row org_code = %v, want O1", rs[0]["org_code"])
	}

	// 1.3：变参 + args tag 按名打包（orgCode / deleted 两键），运行不 panic
	rs2, err := mp.SelectByOrgCodeV("O1", "2026-01-01")
	if err != nil {
		t.Errorf("selectByOrgCodeV (variadic) failed: %v", err)
		return
	}
	// O1 的 deleted_at 是 NULL，第二个参数传 '2026-01-01' 应查不到（deleted='NULL' 文本不匹配）
	if len(rs2) != 0 {
		t.Errorf("variadic binding: expect 0 row for O1+2026-01-01 (deleted_at NULL), got %v", rs2)
	}
}