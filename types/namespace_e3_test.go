package types

import (
	"testing"
	"testing/fstest"
)

// E3：namespace 冲突检测与 NamespacePrefix 加载。

// dupNamespaceFS 构造两个目录、同 namespace、不同语句的 MapFS。
func dupNamespaceFS() fstest.MapFS {
	return fstest.MapFS{
		"a/One.xml": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="DupNsMapper">
  <select id="fromOne" resultType="int">select 1</select>
</mapper>`)},
		"b/Two.xml": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="DupNsMapper">
  <select id="fromTwo" resultType="int">select 2</select>
</mapper>`)},
	}
}

// Test_NewSqlMappersFromSources_DuplicateNamespace 同 namespace 多文件：
// 覆盖行为不变（后者覆盖前者），但记入 NamespaceConflicts（E3）。
func Test_NewSqlMappersFromSources_DuplicateNamespace(t *testing.T) {
	mfs := dupNamespaceFS()
	mps := NewSqlMappersFromSources(
		MapperSource{FS: mfs, Patterns: []string{"a"}},
		MapperSource{FS: mfs, Patterns: []string{"b"}},
	)
	if mps == nil {
		t.Error("NewSqlMappersFromSources returned nil")
		return
	}
	// 后者覆盖前者：生效的是 b/Two.xml
	mp, ok := mps.NamedMappers["DupNsMapper"]
	if !ok {
		t.Error("DupNsMapper missing in NamedMappers")
		return
	}
	if _, ok := mp.NamedFunctions["fromTwo"]; !ok {
		t.Error("later source should override, fromTwo missing")
	}
	if _, ok := mp.NamedFunctions["fromOne"]; ok {
		t.Error("later source should override, fromOne should be replaced")
	}
	// 冲突记录：1 条，含两个文件、按加载顺序
	if len(mps.NamespaceConflicts) != 1 {
		t.Errorf("NamespaceConflicts want 1, got %d", len(mps.NamespaceConflicts))
		return
	}
	cf := mps.NamespaceConflicts[0]
	if cf.Namespace != "DupNsMapper" {
		t.Errorf("conflict namespace want DupNsMapper, got %s", cf.Namespace)
	}
	if len(cf.Files) != 2 || cf.Files[0] != "a/One.xml" || cf.Files[1] != "b/Two.xml" {
		t.Errorf("conflict files want [a/One.xml b/Two.xml], got %v", cf.Files)
	}
}

// Test_NewSqlMappersFromSources_NoConflict 无重复 namespace 时 NamespaceConflicts 为空。
func Test_NewSqlMappersFromSources_NoConflict(t *testing.T) {
	mfs := fstest.MapFS{
		"m/One.xml": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="OneMapper">
  <select id="sel" resultType="int">select 1</select>
</mapper>`)},
	}
	mps := NewSqlMappersFromSources(MapperSource{FS: mfs, Patterns: []string{"m"}})
	if len(mps.NamespaceConflicts) != 0 {
		t.Errorf("NamespaceConflicts should be empty, got %v", mps.NamespaceConflicts)
	}
}

// Test_MapperSource_NamespacePrefix 带前缀来源：namespace 加 prefix. 前缀，
// 绑定键为「前缀 + 短名」，不注册裸短名键，与无前缀同名 Mapper 并存互不覆盖（E3）。
func Test_MapperSource_NamespacePrefix(t *testing.T) {
	plainFS := fstest.MapFS{
		"m/PlatformDisplayMapper.xml": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="PlatformDisplayMapper">
  <select id="selectA" resultType="int">select 1</select>
</mapper>`)},
	}
	prefixedFS := fstest.MapFS{
		"m/PlatformDisplayMapper.xml": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="PlatformDisplayMapper">
  <select id="selectB" resultType="int">select 2</select>
</mapper>`)},
	}
	mps := NewSqlMappersFromSources(
		MapperSource{FS: plainFS, Patterns: []string{"m"}},
		MapperSource{FS: prefixedFS, Patterns: []string{"m"}, NamespacePrefix: "Quality"},
	)
	// 无前缀：裸短名键指向 plainFS 版本
	plain, ok := mps.NamedMappers["PlatformDisplayMapper"]
	if !ok {
		t.Error("unprefixed PlatformDisplayMapper missing")
		return
	}
	if _, ok := plain.NamedFunctions["selectA"]; !ok {
		t.Error("unprefixed mapper should keep selectA")
	}
	// 带前缀：绑定键 Quality+短名 指向 prefixedFS 版本，namespace 带 Quality. 前缀
	prefixed, ok := mps.NamedMappers["QualityPlatformDisplayMapper"]
	if !ok {
		t.Error("prefixed binding key QualityPlatformDisplayMapper missing")
		return
	}
	if prefixed.Namespace != "Quality.PlatformDisplayMapper" {
		t.Errorf("prefixed namespace want Quality.PlatformDisplayMapper, got %s", prefixed.Namespace)
	}
	if _, ok := prefixed.NamedFunctions["selectB"]; !ok {
		t.Error("prefixed mapper should have selectB")
	}
	// 带前缀来源不注册裸短名键：不与无前缀同名 Mapper 互相覆盖
	if mps.NamedMappers["PlatformDisplayMapper"] != plain {
		t.Error("bare short-name key should keep pointing to unprefixed mapper")
	}
	// 两个 namespace 不同，不算冲突
	if len(mps.NamespaceConflicts) != 0 {
		t.Errorf("different namespaces should not conflict, got %v", mps.NamespaceConflicts)
	}
	// 函数 Owner 与带前缀 namespace 一致（二级缓存键消费点）
	for _, fn := range prefixed.Functions {
		if fn.Owner != "Quality.PlatformDisplayMapper" {
			t.Errorf("function owner want Quality.PlatformDisplayMapper, got %s", fn.Owner)
		}
	}
}

// Test_MapperSource_NamespacePrefixTrim 前缀两端空白与点号被规整（"  Quality. " → "Quality"）。
func Test_MapperSource_NamespacePrefixTrim(t *testing.T) {
	mfs := fstest.MapFS{
		"m/TinyMapper.xml": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="TinyMapper">
  <select id="sel" resultType="int">select 1</select>
</mapper>`)},
	}
	mps := NewSqlMappersFromSources(
		MapperSource{FS: mfs, Patterns: []string{"m"}, NamespacePrefix: "  Quality. "},
	)
	mp, ok := mps.NamedMappers["QualityTinyMapper"]
	if !ok {
		t.Error("trimmed prefix binding key QualityTinyMapper missing")
		return
	}
	if mp.Namespace != "Quality.TinyMapper" {
		t.Errorf("prefixed namespace want Quality.TinyMapper, got %s", mp.Namespace)
	}
}
