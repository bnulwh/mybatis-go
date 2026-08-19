package types

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testConfigXML = `<?xml version="1.0" encoding="UTF-8" ?>
<!DOCTYPE configuration
PUBLIC "-//mybatis.org//DTD Config 3.0//EN"
"http://mybatis.org/dtd/mybatis-3-config.dtd">
<configuration>
    <settings>
        <setting name="cacheEnabled" value="true" />
    </settings>
</configuration>
`

const testNonMapperXML = `<?xml version="1.0" encoding="UTF-8"?>
<project version="4">
	<component name="ProjectRunConfigurationManager" />
</project>
`

// writeMixedMapperDir 生成一个同时含 Mapper XML 与非 Mapper XML 的目录。
func writeMixedMapperDir(t *testing.T) string {
	dir := t.TempDir()
	files := map[string]string{
		"mybatis-config.xml":  testConfigXML,
		"UserInfoMapper.xml":  testLongParamMapperXML,
		"ide/project.xml":     testNonMapperXML,
		"short":               "not xml",
	}
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Error("mkdir failed:", err)
			return ""
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Error("write file failed:", err)
			return ""
		}
	}
	return dir
}

// Test_filterMapperFiles_Suffix 只收集 .xml 文件，短文件名（如 "short"）不 panic（S-10）。
func Test_filterMapperFiles_Suffix(t *testing.T) {
	dir := writeMixedMapperDir(t)
	files := filterMapperFiles(dir)
	if len(files) != 3 {
		t.Error("filterMapperFiles should find 3 xml files, got", len(files), files)
	}
	for _, f := range files {
		if !strings.HasSuffix(strings.ToLower(f), ".xml") {
			t.Error("non-xml file collected:", f)
		}
	}
}

// Test_LoadMapper_SkipNonMapper loadMapper 对 mybatis-config.xml / 其他非 Mapper XML 返回 nil（S-10）。
func Test_LoadMapper_SkipNonMapper(t *testing.T) {
	dir := writeMixedMapperDir(t)
	if mp := loadMapper(filepath.Join(dir, "mybatis-config.xml")); mp != nil {
		t.Error("loadMapper(mybatis-config.xml) should return nil, got namespace:", mp.Namespace)
	}
	if mp := loadMapper(filepath.Join(dir, "ide", "project.xml")); mp != nil {
		t.Error("loadMapper(project.xml) should return nil, got namespace:", mp.Namespace)
	}
	// 正常 Mapper 仍可加载
	mp := loadMapper(filepath.Join(dir, "UserInfoMapper.xml"))
	if mp == nil {
		t.Error("loadMapper(UserInfoMapper.xml) should load")
		return
	}
	if mp.Namespace != "TestLongParamMapper" {
		t.Error("loaded namespace =", mp.Namespace, "want TestLongParamMapper")
	}
}

// Test_NewSqlMappers_SkipConfig samples 真实回归：mybatis-config.xml 不再被当作 Mapper 加载（S-10）。
func Test_NewSqlMappers_SkipConfig(t *testing.T) {
	mps := NewSqlMappers("../samples")
	if mps == nil || len(mps.Mappers) == 0 {
		t.Error("load samples failed")
		return
	}
	for _, m := range mps.Mappers {
		if m.Namespace == "" {
			t.Error("empty-namespace mapper loaded from samples:", m.Filename)
		}
		if strings.Contains(m.Filename, "mybatis-config") {
			t.Error("mybatis-config.xml loaded as mapper:", m.Filename)
		}
	}
	if _, ok := mps.NamedMappers[""]; ok {
		t.Error("NamedMappers should not contain empty-namespace key")
	}
}
