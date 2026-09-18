package types

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/beevik/etree"
	"github.com/bnulwh/mybatis-go/log"
	yaml "go.yaml.in/yaml/v3"
)

// mybatis / mybatis-plus 配置文件解析：直接给一个 Java 侧配置文件时，
// 先从配置中定位 Mapper XML（mapper-locations / <mappers> / mapperLocations），
// 再按与 NewSqlMappers 相同的链路加载，供 xml2go（及任意生成器）使用。
//
// 支持三种配置形态：
//   - .properties / Spring Boot：mybatis.mapper-locations、mybatis-plus.mapper-locations，
//     及 mybatis[-plus].config-location 指向的 mybatis-config.xml（链式解析）；
//   - .yml / .yaml：mybatis / mybatis-plus 节点下的 mapper-locations（标量 / 行内列表 / 块列表），
//     及 config-location 链式解析；
//   - .xml：mybatis-config.xml（<configuration><mappers><mapper resource|url/></mappers>）
//     与 Spring/MP Spring XML（<property name="mapperLocations" value|<list>）。
//
// 位置解析约定：
//   - classpath*: / classpath: 前缀去除后，相对路径以配置文件所在目录为 classpath 根；
//   - 未命中时依次回退到该目录的父 / 祖父目录（兼容「配置在 mybatis/ 子目录、XML 在资源根」布局）；
//   - 模式支持 * / ? / **（** 跨目录层级），展开结果仅保留 .xml 文件。

// maxConfigChain config-location 链式解析的最大深度（防自引用循环）。
const maxConfigChain = 3

// MyBatisConfigInfo mybatis / mybatis-plus 配置文件的解析结果。
type MyBatisConfigInfo struct {
	// ConfigFile 配置文件绝对路径
	ConfigFile string
	// BaseDir classpath 根（配置文件所在目录），相对 mapper-locations 以此为基准
	BaseDir string
	// Locations 解析出的 mapper 位置模式（原始值，未展开；含 config-location 链式合并结果）
	Locations []string
	// XmlFiles 展开后的 Mapper XML 文件清单（可直接作为 MapperSource.Patterns）
	XmlFiles []string
}

// LoadMappersFromMyBatisConfig 从 mybatis / mybatis-plus 配置文件（.properties / .yml /
// .yaml / .xml）解析 Mapper XML 位置并加载，返回 SqlMappers 与解析明细。
// 未解析到任何有效 Mapper 时返回错误。
func LoadMappersFromMyBatisConfig(configPath string) (*SqlMappers, *MyBatisConfigInfo, error) {
	if strings.TrimSpace(configPath) == "" {
		return nil, nil, errors.New("empty mybatis config path")
	}
	abs, err := filepath.Abs(configPath)
	if err != nil {
		return nil, nil, err
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, nil, fmt.Errorf("mybatis config %v not found: %w", configPath, err)
	}
	baseDir := filepath.Dir(abs)
	locations, err := resolveMapperLocations(abs, baseDir, 0)
	if err != nil {
		return nil, nil, err
	}
	files := ExpandMapperLocations(baseDir, locations)
	info := &MyBatisConfigInfo{
		ConfigFile: abs,
		BaseDir:    baseDir,
		Locations:  locations,
		XmlFiles:   files,
	}
	if len(files) == 0 {
		return nil, info, fmt.Errorf("no mapper xml resolved from config %v (locations: %v)", configPath, locations)
	}
	mps := NewSqlMappersFromSources(MapperSource{Patterns: files})
	if len(mps.Mappers) == 0 {
		return mps, info, fmt.Errorf("mapper xml resolved (%v files) but none parsed successfully from config %v", len(files), configPath)
	}
	return mps, info, nil
}

// resolveMapperLocations 解析单个配置文件中的 mapper 位置；
// 仅有 config-location 时沿链递归（classpath 根保持为最外层配置所在目录）。
func resolveMapperLocations(cfgPath, baseDir string, depth int) ([]string, error) {
	if depth > maxConfigChain {
		return nil, fmt.Errorf("mybatis config-location chain too deep (max %v): %v", maxConfigChain, cfgPath)
	}
	body, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}
	var locations, configLocs []string
	switch strings.ToLower(filepath.Ext(cfgPath)) {
	case ".properties":
		locations, configLocs = parseMyBatisProperties(body)
	case ".yml", ".yaml":
		locations, configLocs = parseMyBatisYaml(body)
	case ".xml":
		locations, configLocs = parseMyBatisXml(body)
	default:
		return nil, fmt.Errorf("unsupported mybatis config file %v (expect .properties/.yml/.yaml/.xml)", cfgPath)
	}
	if len(locations) > 0 {
		return locations, nil
	}
	for _, cl := range configLocs {
		p, ok := resolveChainedConfig(cl, baseDir)
		if !ok {
			log.Warnf("mybatis config-location %v not found (base %v), skipped", cl, baseDir)
			continue
		}
		sub, err := resolveMapperLocations(p, baseDir, depth+1)
		if err != nil {
			log.Warnf("resolve chained mybatis config %v failed: %v", p, err)
			continue
		}
		if len(sub) > 0 {
			return sub, nil
		}
	}
	return locations, nil
}

// resolveChainedConfig 把 config-location 值解析为磁盘路径并确认存在。
func resolveChainedConfig(loc, baseDir string) (string, bool) {
	clean, absolute := stripLocationPrefix(loc)
	if strings.TrimSpace(clean) == "" {
		return "", false
	}
	p := clean
	if !absolute {
		p = filepath.Join(baseDir, clean)
	}
	if _, err := os.Stat(p); err != nil {
		return "", false
	}
	return p, true
}

// parseMyBatisProperties 解析 Spring Boot 风格 .properties：
// mybatis[-plus].mapper-locations / mybatis[-plus].config-location（兼容驼峰与中划线）。
func parseMyBatisProperties(body []byte) (locations, configLocs []string) {
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' || line[0] == '!' {
			continue
		}
		pos := strings.Index(line, "=")
		if pos <= 0 {
			pos = strings.Index(line, ":")
		}
		if pos <= 0 {
			continue
		}
		key := normalizeCfgKey(line[:pos])
		val := strings.TrimSpace(strings.Trim(strings.TrimSpace(line[pos+1:]), "'\""))
		switch key {
		case "mybatis.mapperlocations", "mybatisplus.mapperlocations":
			locations = append(locations, splitLocationList(val)...)
		case "mybatis.configlocation", "mybatisplus.configlocation":
			configLocs = append(configLocs, splitLocationList(val)...)
		}
	}
	return
}

// parseMyBatisYaml 解析 Spring Boot 风格 .yml/.yaml：
// mybatis / mybatis-plus 节点下的 mapper-locations（标量 / 行内列表 / 块列表）与 config-location。
func parseMyBatisYaml(body []byte) (locations, configLocs []string) {
	var doc map[string]interface{}
	if err := yaml.Unmarshal(body, &doc); err != nil {
		log.Warnf("parse mybatis yaml config failed: %v", err)
		return nil, nil
	}
	for top, sub := range doc {
		switch normalizeCfgKey(top) {
		case "mybatis", "mybatisplus":
			m, ok := sub.(map[string]interface{})
			if !ok {
				continue
			}
			for k, v := range m {
				switch normalizeCfgKey(k) {
				case "mapperlocations":
					locations = append(locations, yamlStringList(v)...)
				case "configlocation":
					configLocs = append(configLocs, yamlStringList(v)...)
				}
			}
		}
	}
	return
}

// yamlStringList 把 yaml 值（标量 / 列表）规整为字符串列表（逐项再按逗号分号拆分）。
func yamlStringList(v interface{}) []string {
	switch t := v.(type) {
	case string:
		return splitLocationList(t)
	case []interface{}:
		var out []string
		for _, it := range t {
			if s, ok := it.(string); ok {
				out = append(out, splitLocationList(s)...)
			}
		}
		return out
	case nil:
		return nil
	default:
		return splitLocationList(fmt.Sprintf("%v", t))
	}
}

// parseMyBatisXml 解析两类 XML 配置：
// ① mybatis-config.xml：<configuration><mappers><mapper resource|url/></mappers>；
// ② Spring / MyBatis-Plus Spring XML：<property name="mapperLocations" value="..."/>
//   或嵌套 <list><value>...</value></list>（含 configLocation 链式入口）。
func parseMyBatisXml(body []byte) (locations, configLocs []string) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(body); err != nil {
		log.Warnf("parse mybatis xml config failed: %v", err)
		return nil, nil
	}
	root := doc.Root()
	if root == nil {
		return nil, nil
	}
	if strings.EqualFold(root.Tag, "configuration") {
		for _, m := range root.FindElements("./mappers/mapper") {
			if res := strings.TrimSpace(m.SelectAttrValue("resource", "")); res != "" {
				locations = append(locations, res)
				continue
			}
			if u := strings.TrimSpace(m.SelectAttrValue("url", "")); u != "" {
				locations = append(locations, u)
			}
		}
		// <package name="..."/> 指向 Java 注解接口包（无 XML），跳过
	}
	for _, p := range root.FindElements("//property") {
		switch normalizeCfgKey(p.SelectAttrValue("name", "")) {
		case "mapperlocations":
			if v := strings.TrimSpace(p.SelectAttrValue("value", "")); v != "" {
				locations = append(locations, splitLocationList(v)...)
			}
			values := append(p.FindElements("./list/value"), p.FindElements("./array/value")...)
			for _, lv := range values {
				if t := strings.TrimSpace(lv.Text()); t != "" {
					locations = append(locations, splitLocationList(t)...)
				}
			}
		case "configlocation":
			if v := strings.TrimSpace(p.SelectAttrValue("value", "")); v != "" {
				configLocs = append(configLocs, splitLocationList(v)...)
			}
		}
	}
	return
}

// ExpandMapperLocations 把 mapper 位置模式（mapper-locations / resource 值）展开为
// 具体 XML 文件清单。相对模式以 baseDir 为基准；单条模式未命中时依次回退到
// baseDir 的父 / 祖父目录。结果去重、排序。
func ExpandMapperLocations(baseDir string, locations []string) []string {
	bases := locationFallbackBases(baseDir)
	seen := map[string]bool{}
	var files []string
	for _, loc := range locations {
		clean, absolute := stripLocationPrefix(loc)
		if strings.TrimSpace(clean) == "" {
			continue
		}
		var candidates []string
		if absolute {
			candidates = []string{clean}
		} else {
			for _, b := range bases {
				candidates = append(candidates, filepath.Join(b, clean))
			}
		}
		for _, cand := range candidates {
			found := expandOneLocation(cand)
			if len(found) == 0 {
				continue
			}
			for _, f := range found {
				if !seen[f] {
					seen[f] = true
					files = append(files, f)
				}
			}
			break
		}
	}
	sort.Strings(files)
	return files
}

// expandOneLocation 展开单个已拼接基准目录的位置：glob 模式走通配匹配，
// 目录递归收集 .xml，具体 .xml 文件校验存在后直接采用。
func expandOneLocation(cand string) []string {
	if hasGlobMeta(cand) {
		return globXmlFiles(cand)
	}
	st, err := os.Stat(cand)
	if err != nil {
		return nil
	}
	if !st.IsDir() && !strings.EqualFold(filepath.Ext(cand), ".xml") {
		log.Warnf("mapper location %v is not a directory or .xml file, skipped", cand)
		return nil
	}
	return listXmlFiles(nil, cand)
}

// locationFallbackBases 相对模式的候选基准：baseDir → 父目录 → 祖父目录。
func locationFallbackBases(baseDir string) []string {
	var bases []string
	d := baseDir
	for i := 0; i < 3 && d != ""; i++ {
		bases = append(bases, d)
		p := filepath.Dir(d)
		if p == d {
			break
		}
		d = p
	}
	return bases
}

// stripLocationPrefix 去掉 classpath*: / classpath: / file: 前缀并归一化为本机路径，
// 返回归一结果与是否绝对路径。
func stripLocationPrefix(loc string) (clean string, absolute bool) {
	l := strings.TrimSpace(loc)
	lower := strings.ToLower(l)
	switch {
	case strings.HasPrefix(lower, "classpath*:"):
		l = strings.TrimLeft(l[len("classpath*:"):], `/\`)
	case strings.HasPrefix(lower, "classpath:"):
		l = strings.TrimLeft(l[len("classpath:"):], `/\`)
	case strings.HasPrefix(lower, "file:"):
		l = fileURLToPath(l[len("file:"):])
	}
	clean = filepath.FromSlash(l)
	return clean, filepath.IsAbs(clean)
}

// fileURLToPath 把 file: URL 余部转成本机路径：
// file:///D:/x → D:/x（Windows 盘符），file:///srv/x → /srv/x（POSIX），
// file://server/share 形态保留（Windows UNC）。
func fileURLToPath(rest string) string {
	if strings.HasPrefix(rest, "///") {
		rest = rest[2:]
	}
	if u, err := url.PathUnescape(rest); err == nil {
		rest = u
	}
	if len(rest) > 2 && rest[0] == '/' && rest[2] == ':' {
		rest = rest[1:]
	}
	return rest
}

// splitLocationList 按逗号 / 分号拆分位置列表（Spring 多值写法）。
func splitLocationList(v string) []string {
	var out []string
	for _, it := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ';' }) {
		if s := strings.TrimSpace(it); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// normalizeCfgKey 键名归一化：小写并去掉 - / _（兼容 mapper-locations / mapperLocations）。
func normalizeCfgKey(k string) string {
	k = strings.ToLower(strings.TrimSpace(k))
	k = strings.ReplaceAll(k, "-", "")
	return strings.ReplaceAll(k, "_", "")
}

// hasGlobMeta 判断路径是否含通配符。
func hasGlobMeta(p string) bool {
	return strings.ContainsAny(p, "*?[")
}

// globXmlFiles 按通配模式收集 .xml 文件（支持 * / ? / **）：
// 从模式中第一个含通配符目录段之前的静态根开始 WalkDir，整路径正则匹配。
func globXmlFiles(pattern string) []string {
	re, err := globRegexp(filepath.ToSlash(pattern))
	if err != nil {
		log.Warnf("build glob regexp from %v failed: %v", pattern, err)
		return nil
	}
	root := globStaticRoot(pattern)
	var files []string
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		slash := filepath.ToSlash(p)
		if re.MatchString(slash) && strings.EqualFold(filepath.Ext(slash), ".xml") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		log.Warnf("walk glob root %v failed: %v", root, err)
	}
	return files
}

// globRegexp 把通配模式编译为整路径正则：**/ → 跨零或多层目录，** → 任意，* / ? 单段内。
func globRegexp(pattern string) (*regexp.Regexp, error) {
	var sb strings.Builder
	sb.WriteString("^")
	rs := []rune(pattern)
	for i := 0; i < len(rs); i++ {
		switch rs[i] {
		case '*':
			if i+1 < len(rs) && rs[i+1] == '*' {
				if i+2 < len(rs) && rs[i+2] == '/' {
					sb.WriteString("(?:[^/]+/)*")
					i += 2
				} else {
					sb.WriteString(".*")
					i++
				}
			} else {
				sb.WriteString("[^/]*")
			}
		case '?':
			sb.WriteString("[^/]")
		default:
			sb.WriteString(regexp.QuoteMeta(string(rs[i])))
		}
	}
	sb.WriteString("$")
	return regexp.Compile(sb.String())
}

// globStaticRoot 取模式中第一个含通配符目录段之前的静态根目录（相对模式无静态前缀时为 "."）。
// Windows 盘符卷名（如 D:）单独拼接（filepath.Join 会把 "D:" 当相对路径）。
func globStaticRoot(pattern string) string {
	vol := filepath.VolumeName(pattern)
	rest := pattern[len(vol):]
	dir := ""
	if idx := strings.LastIndexAny(rest, `/\`); idx >= 0 {
		dir = rest[:idx+1]
	}
	segs := strings.FieldsFunc(dir, func(r rune) bool { return r == '/' || r == '\\' })
	var keep []string
	for _, s := range segs {
		if hasGlobMeta(s) {
			break
		}
		keep = append(keep, s)
	}
	if len(keep) == 0 {
		if vol != "" {
			return vol + string(filepath.Separator)
		}
		return "."
	}
	if vol != "" {
		return vol + string(filepath.Separator) + filepath.Join(keep...)
	}
	return filepath.Join(keep...)
}
