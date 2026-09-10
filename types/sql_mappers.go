package types

import (
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/utils"
)

type SqlMappers struct {
	Mappers      []SqlMapper
	NamedMappers map[string]*SqlMapper
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
// 同一 namespace 重复出现时后者覆盖前者。
func NewSqlMappersFromSources(sources ...MapperSource) *SqlMappers {
	var mps []SqlMapper
	nmp := map[string]*SqlMapper{}
	for _, src := range sources {
		for _, info := range collectMapperFiles(src) {
			start := time.Now()
			mp := loadMapperFrom(info)
			count := 0
			if mp != nil {
				mps = append(mps, *mp)
				nmp[mp.Namespace] = mp
				sname := GetShortName(mp.Namespace)
				nmp[sname] = mp
				nmp[buildKey(sname)] = mp
				nmp[strings.ToLower(sname)] = mp
				count = len(mp.Functions)
			}
			// 5.2：每文件一行汇总（替代逐条刷屏），便于看清解析耗时与条数
			log.Debugf("parsed mapper %s: %d statements in %s", info.name, count, time.Since(start))
		}
	}
	return &SqlMappers{
		Mappers:      mps,
		NamedMappers: nmp,
	}
}

// mapperInfo 描述一个待加载的 Mapper XML：name 用于日志与 SqlMapper.Filename，
// content 非空时直接解析（embed.FS / 内存来源），为 nil 时按 name 从磁盘读取。
type mapperInfo struct {
	name    string
	content []byte
}

// MapperSource 描述一个 Mapper 来源：FS 为空表示磁盘文件系统，Patterns 为目录或 .xml 文件路径。
// Patterns 为空时等价于当前目录。用于合并磁盘目录与 embed.FS 等多来源加载。
type MapperSource struct {
	FS       fs.FS
	Patterns []string
}

// collectMapperFiles 收集来源下全部 .xml 路径；目录递归，文件直接采用（仅 .xml）。
// 当来源指定了 fs.FS 时立即读出内容驻留内存（embed.FS 为只读内存 FS，无需落盘）。
// 单个来源出错只记日志跳过，不影响其余来源。
func collectMapperFiles(src MapperSource) []mapperInfo {
	patterns := src.Patterns
	if len(patterns) == 0 {
		patterns = []string{"."}
	}
	var out []mapperInfo
	seen := map[string]bool{}
	for _, pattern := range patterns {
		for _, name := range listXmlFiles(src.FS, pattern) {
			if seen[name] {
				continue
			}
			seen[name] = true
			info := mapperInfo{name: name}
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
