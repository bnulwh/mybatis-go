package types

import (
	"embed"
	"testing"
	"testing/fstest"
)

// testdata 为真实 go:embed 内嵌目录，用于验证 embed.FS 加载路径。
//
//go:embed testdata/embedded
var embeddedMapperFS embed.FS

// Test_NewSqlMappersFrom_EmbedFS embed.FS 直接加载（不落盘）：目录递归 + 非 Mapper XML 剔除。
func Test_NewSqlMappersFrom_EmbedFS(t *testing.T) {
	mps := NewSqlMappersFrom(embeddedMapperFS, "testdata/embedded")
	if mps == nil {
		t.Error("NewSqlMappersFrom returned nil")
		return
	}
	if len(mps.Mappers) != 2 {
		t.Errorf("expect 2 mappers loaded from embed.FS, got %d", len(mps.Mappers))
	}
	// mybatis-config.xml（根标签 configuration、无 namespace）必须被剔除
	for _, m := range mps.Mappers {
		if m.Namespace == "" {
			t.Error("empty-namespace mapper loaded from embed.FS:", m.Filename)
		}
	}
	for _, ns := range []string{"EmbedUserMapper", "EmbedDeptMapper"} {
		if _, ok := mps.NamedMappers[ns]; !ok {
			t.Errorf("namespace %s not found in NamedMappers", ns)
		}
	}
}

// Test_NewSqlMappersFrom_EmbedFSFunctions 内嵌加载的 Mapper 函数定义完整可用。
func Test_NewSqlMappersFrom_EmbedFSFunctions(t *testing.T) {
	mps := NewSqlMappersFrom(embeddedMapperFS, "testdata/embedded")
	mp, ok := mps.NamedMappers["EmbedUserMapper"]
	if !ok {
		t.Error("EmbedUserMapper not loaded")
		return
	}
	if len(mp.Functions) != 2 {
		t.Error("expect 2 functions in EmbedUserMapper, got", len(mp.Functions))
	}
	fn := mp.NamedFunctions["selectembeduserbyid"]
	if fn == nil {
		t.Error("selectEmbedUserById not found in EmbedUserMapper")
		return
	}
	sql, _, err := fn.GenerateSQL(int64(7))
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	if got := collapseSpace(sql); got != "select * from embed_user where user_id = 7" {
		t.Error("unexpected sql from embedded mapper:", got)
	}
}

// Test_NewSqlMappersFrom_SingleFilePattern 单个 .xml 文件路径也可作为 pattern。
func Test_NewSqlMappersFrom_SingleFilePattern(t *testing.T) {
	mps := NewSqlMappersFrom(embeddedMapperFS, "testdata/embedded/EmbedDeptMapper.xml")
	if len(mps.Mappers) != 1 {
		t.Errorf("expect 1 mapper for single-file pattern, got %d", len(mps.Mappers))
		return
	}
	if mps.Mappers[0].Namespace != "EmbedDeptMapper" {
		t.Error("unexpected namespace:", mps.Mappers[0].Namespace)
	}
}

// Test_NewSqlMappersFrom_MultiplePatterns 多个目录/文件 pattern 混合传入，同 namespace 去重。
func Test_NewSqlMappersFrom_MultiplePatterns(t *testing.T) {
	mps := NewSqlMappersFrom(embeddedMapperFS,
		"testdata/embedded",
		"testdata/embedded/EmbedDeptMapper.xml", // 与目录重复，应被去重
	)
	if len(mps.Mappers) != 2 {
		t.Errorf("duplicate pattern should be deduped, got %d mappers", len(mps.Mappers))
	}
}

// Test_NewSqlMappersFromSources_DiskAndEmbed 磁盘目录 + embed.FS 合并加载。
func Test_NewSqlMappersFromSources_DiskAndEmbed(t *testing.T) {
	dir := writeMixedMapperDir(t) // 磁盘：TestLongParamMapper + 非 Mapper XML
	mps := NewSqlMappersFromSources(
		MapperSource{FS: embeddedMapperFS, Patterns: []string{"testdata/embedded"}},
		MapperSource{Patterns: []string{dir}},
	)
	for _, ns := range []string{"EmbedUserMapper", "EmbedDeptMapper", "TestLongParamMapper"} {
		if _, ok := mps.NamedMappers[ns]; !ok {
			t.Errorf("namespace %s missing from merged sources", ns)
		}
	}
	if len(mps.Mappers) != 3 {
		t.Error("expect 3 mappers from merged sources, got", len(mps.Mappers))
	}
}

// Test_NewSqlMappersFromSources_LaterOverrides 同 namespace 时后列来源覆盖先列来源。
func Test_NewSqlMappersFromSources_LaterOverrides(t *testing.T) {
	first := fstest.MapFS{
		"m/User.xml": {Data: []byte(`<mapper namespace="DupMapper">
			<select id="first">select 1</select>
		</mapper>`)},
	}
	second := fstest.MapFS{
		"m/User.xml": {Data: []byte(`<mapper namespace="DupMapper">
			<select id="second">select 2</select>
		</mapper>`)},
	}
	mps := NewSqlMappersFromSources(
		MapperSource{FS: first, Patterns: []string{"m"}},
		MapperSource{FS: second, Patterns: []string{"m"}},
	)
	if len(mps.Mappers) != 2 {
		t.Error("both definitions should be kept in Mappers, got", len(mps.Mappers))
	}
	mp, ok := mps.NamedMappers["DupMapper"]
	if !ok {
		t.Error("DupMapper not found")
		return
	}
	if _, ok := mp.NamedFunctions["second"]; !ok {
		t.Error("later source should override earlier one, got functions:", mp.NamedFunctions)
	}
}

// Test_NewSqlMappersFrom_MapFS fstest.MapFS 亦可直接加载（纯内存，不触盘）。
func Test_NewSqlMappersFrom_MapFS(t *testing.T) {
	mfs := fstest.MapFS{
		"mapper/User.xml": {Data: []byte(`<mapper namespace="MapFSTestMapper">
			<select id="selectOne" resultType="map">select * from t</select>
		</mapper>`)},
		"mapper/readme.txt": {Data: []byte("ignored")},
	}
	mps := NewSqlMappersFrom(mfs, "mapper")
	if len(mps.Mappers) != 1 {
		t.Error("expect 1 mapper from MapFS, got", len(mps.Mappers))
		return
	}
	if _, ok := mps.NamedMappers["MapFSTestMapper"]; !ok {
		t.Error("MapFSTestMapper not loaded")
	}
}

// Test_NewSqlMappersFrom_MissingPattern 不存在的 pattern 不 panic，仅返回空集合。
func Test_NewSqlMappersFrom_MissingPattern(t *testing.T) {
	mps := NewSqlMappersFrom(embeddedMapperFS, "testdata/no-such-dir")
	if mps == nil {
		t.Error("NewSqlMappersFrom should not return nil for missing pattern")
		return
	}
	if len(mps.Mappers) != 0 {
		t.Error("missing pattern should yield 0 mappers, got", len(mps.Mappers))
	}
}

// Test_NewSqlMappersFrom_NilFS nil FS 表示磁盘文件系统（与 MapperSource 语义一致），
// 指向不存在的磁盘路径时应安全返回空集合而非 panic。
func Test_NewSqlMappersFrom_NilFS(t *testing.T) {
	mps := NewSqlMappersFrom(nil, "testdata/no-such-dir-on-disk")
	if mps == nil {
		t.Error("NewSqlMappersFrom should not return nil")
		return
	}
	if len(mps.Mappers) != 0 {
		t.Error("nil FS + missing disk path should yield 0 mappers, got", len(mps.Mappers))
	}
}

// Test_NewSqlMappers_DiskUnchanged NewSqlMappers 磁盘路径行为保持兼容（回归）。
func Test_NewSqlMappers_DiskUnchanged(t *testing.T) {
	dir := writeMixedMapperDir(t)
	mps := NewSqlMappers(dir)
	if len(mps.Mappers) != 1 {
		t.Error("disk loading regression: expect 1 mapper, got", len(mps.Mappers))
		return
	}
	if _, ok := mps.NamedMappers["TestLongParamMapper"]; !ok {
		t.Error("TestLongParamMapper not loaded from disk")
	}
}
