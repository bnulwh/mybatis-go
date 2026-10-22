package types

import (
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/utils"
)

// NamespaceConflict 记录同一 namespace 被多个 XML 文件加载的情况（E3 可观测性）。
// 覆盖行为不变（后者覆盖前者），此处仅告警与记录，便于加载期发现「共享 XML 与本地
// Mapper 同名互踩」「来源顺序写反导致覆盖未按预期生效」等问题。
type NamespaceConflict struct {
	Namespace string
	Files     []string // 按加载顺序，最后一个文件生效
}

type SqlMappers struct {
	Mappers      []SqlMapper
	NamedMappers map[string]*SqlMapper
	// NamespaceConflicts 为加载期检测到的「同 namespace 多文件」记录（E3，按 namespace 排序）。
	NamespaceConflicts []NamespaceConflict
}

//func NewSqlMappersEx(ds *DatabaseStructure) *SqlMappers {
//	var mps []SqlMapper
//	nmp := map[string]*SqlMapper{}
//	for _,tableName := range ds.TableList{
//
//	}
//}

func NewSqlMappers(dir string) *SqlMappers {
	return NewSqlMappersFromSources(MapperSource{Patterns: []string{dir}})
}

// NewSqlMappersFrom 从任意文件系统加载 Mapper XML，支持 go:embed 内嵌的只读文件系统。
// 典型用法：
//
//	//go:embed resources/mapper/*.xml
//	var mapperFS embed.FS
//
//	mps := types.NewSqlMappersFrom(mapperFS, "resources/mapper")
//
// 每个 pattern 既可以是目录（递归收集其下全部 .xml），也可以是单个 .xml 文件路径，
// 与 mybatis.mapper-locations 的语义保持一致。
func NewSqlMappersFrom(fsys fs.FS, patterns ...string) *SqlMappers {
	return NewSqlMappersFromSources(MapperSource{FS: fsys, Patterns: patterns})
}

// NewSqlMappersFromSources 从多个来源（磁盘目录 / embed.FS / 内存）加载 Mapper。
// 同一 namespace 重复出现时后者覆盖前者（E3：覆盖时 log.Warnf 并记入 NamespaceConflicts）。
// 来源指定 NamespacePrefix 时该来源所有 namespace 加「prefix.」前缀，绑定键改为
// 「prefix + 短名」，可与无前缀同名 Mapper 并存互不覆盖（E3）。
func NewSqlMappersFromSources(sources ...MapperSource) *SqlMappers {
	var mps []SqlMapper
	nmp := map[string]*SqlMapper{}
	nsFiles := map[string][]string{}
	for _, src := range sources {
		prefix := normalizeNamespacePrefix(src.NamespacePrefix)
		for _, info := range collectMapperFiles(src) {
			start := time.Now()
			mp := loadMapperFrom(info)
			count := 0
			if mp != nil {
				mps = append(mps, *mp)
				// E3：同 namespace 多文件检测——后者覆盖前者，显式告警提升可观测性
				if prev := nsFiles[mp.Namespace]; len(prev) > 0 {
					log.Warnf("duplicate mapper namespace %q: %s overrides %s (later source wins)",
						mp.Namespace, mp.Filename, strings.Join(prev, ", "))
				}
				nsFiles[mp.Namespace] = append(nsFiles[mp.Namespace], mp.Filename)
				nmp[mp.Namespace] = mp
				if prefix != "" {
					// E3 NamespacePrefix：带前缀来源只注册「前缀 + 短名」绑定键，
					// 不注册裸短名键，避免与无前缀同名 Mapper 相互覆盖；
					// Go 结构体命名为 前缀+短名（如 QualityPlatformDisplayMapper）即可绑定。
					bname := prefix + GetShortName(mp.Namespace)
					nmp[bname] = mp
					nmp[buildKey(bname)] = mp
				} else {
					sname := GetShortName(mp.Namespace)
					nmp[sname] = mp
					nmp[buildKey(sname)] = mp
					nmp[strings.ToLower(sname)] = mp
				}
				count = len(mp.Functions)
			}
			// 5.2：每文件一行汇总（替代逐条刷屏），便于看清解析耗时与条数
			log.Debugf("parsed mapper %s: %d statements in %s", info.name, count, time.Since(start))
		}
	}
	return &SqlMappers{
		Mappers:            mps,
		NamedMappers:       nmp,
		NamespaceConflicts: buildNamespaceConflicts(nsFiles),
	}
}

// normalizeNamespacePrefix 规整 NamespacePrefix：去两端空白与点号（"  Quality. " → "Quality"）。
func normalizeNamespacePrefix(prefix string) string {
	return strings.Trim(strings.TrimSpace(prefix), ".")
}

// buildNamespaceConflicts 汇总「同 namespace 多文件」记录（E3：按 namespace 排序，结果确定）。
func buildNamespaceConflicts(nsFiles map[string][]string) []NamespaceConflict {
	var conflicts []NamespaceConflict
	for ns, files := range nsFiles {
		if len(files) > 1 {
			conflicts = append(conflicts, NamespaceConflict{Namespace: ns, Files: files})
		}
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].Namespace < conflicts[j].Namespace })
	return conflicts
}

// mapperInfo 描述一个待加载的 Mapper XML：name 用于日志与 SqlMapper.Filename，
// content 非空时直接解析（embed.FS / 内存来源），为 nil 时按 name 从磁盘读取。
// nsPrefix 非空时（E3 NamespacePrefix）加载出的 namespace 加「prefix.」前缀。
type mapperInfo struct {
	name     string
	content  []byte
	nsPrefix string
}

// MapperSource 描述一个 Mapper 来源：FS 为空表示磁盘文件系统，Patterns 为目录或 .xml 文件路径。
// Patterns 为空时等价于当前目录。用于合并磁盘目录与 embed.FS 等多来源加载。
//
// NamespacePrefix 非空时（E3）：该来源下所有 Mapper 的 namespace 加「prefix.」前缀，
// 绑定键改为「prefix + 短名」（不注册裸短名键），与无前缀同名 Mapper 并存互不覆盖；
// 对应 Go 结构体命名为 prefix+短名（如 prefix=Quality、短名 PlatformDisplayMapper →
// 结构体 QualityPlatformDisplayMapper）。prefix 应为合法 Go 标识符段（不含点，
// 含点时命名空间仍生效，但绑定键无法与 Go 结构体名对齐）。
type MapperSource struct {
	FS              fs.FS
	Patterns        []string
	NamespacePrefix string
}

// collectMapperFiles 收集来源下全部 .xml 路径；目录递归，文件直接采用（仅 .xml）。
// 当来源指定了 fs.FS 时立即读出内容驻留内存（embed.FS 为只读内存 FS，无需落盘）。
// 单个来源出错只记日志跳过，不影响其余来源。
// 来源指定 NamespacePrefix 时（E3）随 mapperInfo 下发，由 loadMapperFrom 统一加前缀
// （函数 Owner / MP 内置 CRUD 等 namespace 消费点保持一致）。
func collectMapperFiles(src MapperSource) []mapperInfo {
	patterns := src.Patterns
	if len(patterns) == 0 {
		patterns = []string{"."}
	}
	prefix := normalizeNamespacePrefix(src.NamespacePrefix)
	var out []mapperInfo
	seen := map[string]bool{}
	for _, pattern := range patterns {
		for _, name := range listXmlFiles(src.FS, pattern) {
			if seen[name] {
				continue
			}
			seen[name] = true
			info := mapperInfo{name: name, nsPrefix: prefix}
			if src.FS != nil {
				content, err := fs.ReadFile(src.FS, name)
				if err != nil {
					log.Warnf("read mapper %v from fs failed: %v", name, err)
					continue
				}
				info.content = content
			}
			out = append(out, info)
		}
	}
	return out
}

// listXmlFiles 在 fsys（nil 表示磁盘文件系统）下展开 pattern：
// pattern 为 .xml 文件时仅返回自身；为目录时递归返回其下全部 .xml。
func listXmlFiles(fsys fs.FS, pattern string) []string {
	if pattern == "" {
		return nil
	}
	if strings.HasSuffix(strings.ToLower(pattern), ".xml") {
		return []string{pattern}
	}
	var files []string
	filter := func(p string, info fs.DirEntry, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(p), ".xml") {
			files = append(files, p)
		}
		return nil
	}
	var err error
	if fsys == nil {
		err = filepath.WalkDir(pattern, filter)
	} else {
		// embed.FS / io/fs 路径始终用正斜杠，避免 Windows 下 filepath.Join 产生反斜杠
		err = fs.WalkDir(fsys, path.Clean(filepath.ToSlash(pattern)), filter)
	}
	if err != nil {
		log.Warnf("walk dir %v failed: %v", pattern, err)
	}
	return files
}

func (in *SqlMappers) GenerateFiles(dir, pkg string) {
	err := utils.MakeDirAll(dir)
	if err != nil {
		return
	}
	err = utils.MakeDirAll(filepath.Join(dir, "mapper"))
	if err != nil {
		return
	}
	err = utils.MakeDirAll(filepath.Join(dir, "models"))
	if err != nil {
		return
	}
	for _, mapper := range in.Mappers {
		mapper.GenerateFiles(dir, pkg)
	}
}

// filterMapperFiles 递归收集 dir 下全部 .xml 文件路径（磁盘文件系统）。
func filterMapperFiles(dir string) []string {
	return listXmlFiles(nil, dir)
}
