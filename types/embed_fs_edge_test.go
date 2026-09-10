package types

import (
	"os"
	"path/filepath"
	"testing"
)

// Test_NewSqlMappers_EmptyDirPattern 空目录字符串不再 walk 出整个工作区（回归：曾等价于 "."）。
func Test_NewSqlMappers_EmptyDirPattern(t *testing.T) {
	mps := NewSqlMappers("")
	if mps == nil {
		t.Error("NewSqlMappers(\"\") should not return nil")
		return
	}
	// 关键断言：不得因空路径递归扫描当前目录而加载出无关 Mapper
	if len(mps.Mappers) != 0 {
		t.Errorf("empty dir should load 0 mappers, got %d", len(mps.Mappers))
	}
}

// Test_NewSqlMappersFromSources_EmptyPatterns 未指定 pattern 时按当前目录处理（"."）。
func Test_NewSqlMappersFromSources_EmptyPatterns(t *testing.T) {
	dir := writeMixedMapperDir(t)
	// 切到测试目录，验证 Patterns 为空时使用 "." 语义
	wd, err := os.Getwd()
	if err != nil {
		t.Error("getwd failed:", err)
		return
	}
	if err := os.Chdir(dir); err != nil {
		t.Error("chdir failed:", err)
		return
	}
	defer func() { _ = os.Chdir(wd) }()

	mps := NewSqlMappersFromSources(MapperSource{})
	if len(mps.Mappers) != 1 {
		t.Error("empty patterns should default to \".\", got", len(mps.Mappers))
	}
}

// Test_NewSqlMappersFrom_WindowsPathPattern 磁盘路径含反斜杠时仍能正确收集（跨平台回归）。
func Test_NewSqlMappersFrom_WindowsPathPattern(t *testing.T) {
	dir := writeMixedMapperDir(t)
	// filepath.Join 在 Windows 上产生反斜杠；磁盘来源必须兼容
	nested := filepath.Join(dir, "ide")
	if got := len(listXmlFiles(nil, nested)); got != 1 {
		t.Error("listXmlFiles with OS-native nested path, want 1 .xml, got", got)
	}
	// 目录本身（含子目录）仍应递归收集
	if got := len(listXmlFiles(nil, dir)); got != 3 {
		t.Error("listXmlFiles recursive, want 3 .xml, got", got)
	}
}

// Test_listXmlFiles_SingleFilePattern 单个 .xml 文件路径直接采用，不做目录递归。
func Test_listXmlFiles_SingleFilePattern(t *testing.T) {
	dir := writeMixedMapperDir(t)
	single := filepath.Join(dir, "UserInfoMapper.xml")
	files := listXmlFiles(nil, single)
	if len(files) != 1 || files[0] != single {
		t.Error("single file pattern should return itself, got", files)
	}
}
