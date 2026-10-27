package sqlfragment

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/bnulwh/mybatis-go/log"
)

// Node 动态 SQL 片段统一接口（P0-3a）：每种标签一个实现，替代原 tagged-union + switch 分派。
// 渲染方法（7 个）：Map / Param / Slice 三类参数 × Prepare（占位符）/ Generate（值内联）+ 无参生成。
// 遍历方法（4 个）：Children（结构访问）、CollectSlots（必参占位符）、
// ContainsForEach（签名推导）、CollectIfTestFields（<if test> 字段提取）。
type Node interface {
	PrepareSqlWithMap(mp map[string]interface{}, depth int) (string, []string)
	GenerateSqlWithMap(mp map[string]interface{}, depth int) string
	PrepareSqlWithParam(m interface{}) (string, []string)
	GenerateSqlWithParam(m interface{}) string
	PrepareSqlWithSlice(m []interface{}, depth int) (string, []string)
	GenerateSqlWithSlice(m []interface{}, depth int) string
	GenerateSqlWithoutParam() string
	Children() []Node
	CollectSlots() []string
	ContainsForEach() bool
	CollectIfTestFields() []string
}

// NodeParser 节点解析器：XML 标签节点 + <sql id> 片段表 → Node。
type NodeParser func(node XmlNode, sns map[string]*SqlElement) (Node, error)

var (
	nodeParsersMu sync.RWMutex
	nodeParsers   = map[string]NodeParser{}
)

// RegisterNodeParser 注册节点标签解析器（开放封闭：新增标签无需修改既有分派逻辑，
// 标签名经 buildKey 归一化，大小写/空白不敏感）。
func RegisterNodeParser(tag string, parser NodeParser) {
	nodeParsersMu.Lock()
	defer nodeParsersMu.Unlock()
	nodeParsers[buildKey(tag)] = parser
}

func lookupNodeParser(tag string) (NodeParser, bool) {
	nodeParsersMu.RLock()
	defer nodeParsersMu.RUnlock()
	p, ok := nodeParsers[buildKey(tag)]
	return p, ok
}

// containerBase 通用容器骨架：子片段顺序拼接（空片段跳过）、整体为空时不输出任何内容，
// 非空结果经 Wrap 包装（Wrap 为 nil 时仅去空白）。嵌入即得 Node 全部接口方法，
// 新容器节点（<trim> 为首个使用者）只需提供 Wrap；既有 <where>/<set> 保留原手写方法体，不继承本骨架。
type containerBase struct {
	Sql  []Node
	Wrap func(body string) string
}

func (in *containerBase) Children() []Node {
	return in.Sql
}

func (in *containerBase) wrap(body string) string {
	if in.Wrap == nil {
		return strings.TrimSpace(body)
	}
	return in.Wrap(body)
}

func (in *containerBase) PrepareSqlWithMap(mp map[string]interface{}, depth int) (string, []string) {
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Sql {
		sqlstr, items := item.PrepareSqlWithMap(mp, depth+1)
		if strings.TrimSpace(sqlstr) != "" {
			buf.WriteString(" ")
			buf.WriteString(sqlstr)
			results = append(results, items...)
		}
	}
	if strings.TrimSpace(buf.String()) == "" {
		return "", []string{}
	}
	return in.wrap(buf.String()), results
}

func (in *containerBase) GenerateSqlWithMap(mp map[string]interface{}, depth int) string {
	var buf bytes.Buffer
	for _, item := range in.Sql {
		sqlstr := item.GenerateSqlWithMap(mp, depth+1)
		if strings.TrimSpace(sqlstr) != "" {
			buf.WriteString(" ")
			buf.WriteString(sqlstr)
		}
	}
	if strings.TrimSpace(buf.String()) == "" {
		return ""
	}
	return in.wrap(buf.String())
}

func (in *containerBase) PrepareSqlWithParam(m interface{}) (string, []string) {
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Sql {
		sqlstr, items := item.PrepareSqlWithParam(m)
		if strings.TrimSpace(sqlstr) != "" {
			buf.WriteString(" ")
			buf.WriteString(sqlstr)
			results = append(results, items...)
		}
	}
	if strings.TrimSpace(buf.String()) == "" {
		return "", []string{}
	}
	return in.wrap(buf.String()), results
}

func (in *containerBase) GenerateSqlWithParam(m interface{}) string {
	var buf bytes.Buffer
	for _, item := range in.Sql {
		sqlstr := item.GenerateSqlWithParam(m)
		if strings.TrimSpace(sqlstr) != "" {
			buf.WriteString(" ")
			buf.WriteString(sqlstr)
		}
	}
	if strings.TrimSpace(buf.String()) == "" {
		return ""
	}
	return in.wrap(buf.String())
}

func (in *containerBase) PrepareSqlWithSlice(m []interface{}, depth int) (string, []string) {
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Sql {
		sqlstr, items := item.PrepareSqlWithSlice(m, depth+1)
		if strings.TrimSpace(sqlstr) != "" {
			buf.WriteString(" ")
			buf.WriteString(sqlstr)
			results = append(results, items...)
		}
	}
	if strings.TrimSpace(buf.String()) == "" {
		return "", []string{}
	}
	return in.wrap(buf.String()), results
}

func (in *containerBase) GenerateSqlWithSlice(m []interface{}, depth int) string {
	var buf bytes.Buffer
	for _, item := range in.Sql {
		sqlstr := item.GenerateSqlWithSlice(m, depth+1)
		if strings.TrimSpace(sqlstr) != "" {
			buf.WriteString(" ")
			buf.WriteString(sqlstr)
		}
	}
	if strings.TrimSpace(buf.String()) == "" {
		return ""
	}
	return in.wrap(buf.String())
}

func (in *containerBase) GenerateSqlWithoutParam() string {
	var buf bytes.Buffer
	for _, item := range in.Sql {
		sqlstr := item.GenerateSqlWithoutParam()
		if strings.TrimSpace(sqlstr) != "" {
			buf.WriteString(" ")
			buf.WriteString(sqlstr)
		}
	}
	if strings.TrimSpace(buf.String()) == "" {
		return ""
	}
	return in.wrap(buf.String())
}

// CollectSlots 直接文本视为无条件位置，正常统计（递归聚合子片段）。
func (in *containerBase) CollectSlots() []string {
	var out []string
	for _, item := range in.Sql {
		if item == nil {
			continue
		}
		out = append(out, item.CollectSlots()...)
	}
	return out
}

func (in *containerBase) ContainsForEach() bool {
	for _, item := range in.Sql {
		if item != nil && item.ContainsForEach() {
			return true
		}
	}
	return false
}

func (in *containerBase) CollectIfTestFields() []string {
	var names []string
	for _, item := range in.Sql {
		if item == nil {
			continue
		}
		names = append(names, item.CollectIfTestFields()...)
	}
	return names
}

// parseFragmentFromXmlElement 单个 XML 子元素 → Node：文本元素解析为静态 SQL，
// 节点元素按标签查注册表。
func parseFragmentFromXmlElement(elem XmlElement, sns map[string]*SqlElement) (Node, error) {
	log.Debugf("++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
	log.Debugf("++begin parse sql fragment from element: %v", toJson(elem))
	defer log.Debugf("++finish parse sql fragment from element: %v", toJson(elem))
	switch elem.ElementType {
	case XmlTextElem:
		return parseSimpleSqlFromText(elem.Val.(string)), nil
	case XmlNodeElem:
		xn := elem.Val.(XmlNode)
		parser, ok := lookupNodeParser(xn.Name)
		if !ok {
			// P0-3：未注册标签不再丢弃——保留占位节点，注册期可定位并汇总
			// （默认 Warn，strict-unknown-tag 下报错），避免「能启动、一调用才挂」。
			log.Debugf("unsupported xml tag <%v> in statement body, recorded for registration check", xn.Name)
			return &unsupportedTagNode{Tag: xn.Name}, nil
		}
		return parser(xn, sns)
	}
	return nil, fmt.Errorf("wrong type of element type %v", elem.ElementType)
}

// unsupportedTagNode 未注册标签占位节点（P0-3）：不产出任何 SQL，但保留在片段树中，
// 使注册期能带上 mapper/语句上下文汇总告警；旧实现仅一条无上下文 error 日志后丢弃。
type unsupportedTagNode struct {
	Tag string
}

func (in *unsupportedTagNode) PrepareSqlWithMap(map[string]interface{}, int) (string, []string) {
	return "", nil
}

func (in *unsupportedTagNode) GenerateSqlWithMap(map[string]interface{}, int) string { return "" }

func (in *unsupportedTagNode) PrepareSqlWithParam(interface{}) (string, []string) { return "", nil }

func (in *unsupportedTagNode) GenerateSqlWithParam(interface{}) string { return "" }

func (in *unsupportedTagNode) PrepareSqlWithSlice([]interface{}, int) (string, []string) {
	return "", nil
}

func (in *unsupportedTagNode) GenerateSqlWithSlice([]interface{}, int) string { return "" }

func (in *unsupportedTagNode) GenerateSqlWithoutParam() string { return "" }

func (in *unsupportedTagNode) Children() []Node { return nil }

func (in *unsupportedTagNode) CollectSlots() []string { return nil }

func (in *unsupportedTagNode) ContainsForEach() bool { return false }

func (in *unsupportedTagNode) CollectIfTestFields() []string { return nil }

func (in *unsupportedTagNode) UnsupportedIfTests() []string { return nil }

// UnsupportedTag 返回未注册标签名。
func (in *unsupportedTagNode) UnsupportedTag() string { return in.Tag }

// walkNodeTree 深度优先遍历片段树（含嵌套子片段）。
func walkNodeTree(items []Node, fn func(Node)) {
	for _, it := range items {
		if it == nil {
			continue
		}
		fn(it)
		walkNodeTree(it.Children(), fn)
	}
}

// CollectUnsupportedTags 递归收集片段树中未注册的标签名（去重、按首现序，P0-3）。
func CollectUnsupportedTags(items []Node) []string {
	seen := map[string]bool{}
	var out []string
	walkNodeTree(items, func(n Node) {
		p, ok := n.(interface{ UnsupportedTag() string })
		if !ok {
			return
		}
		tag := p.UnsupportedTag()
		k := buildKey(tag)
		if k == "" || seen[k] {
			return
		}
		seen[k] = true
		out = append(out, tag)
	})
	return out
}

// CollectBoundNames 递归收集 <bind> 绑定的变量名（去重、按首现序）。
// 这些名字由语句自身产生，不从调用方参数取值（1.1/1.2 的 Slots 需剔除）。
func CollectBoundNames(items []Node) []string {
	return collectNodeStrings(items, func(n Node) []string {
		if p, ok := n.(interface{ BoundName() string }); ok {
			return []string{p.BoundName()}
		}
		return nil
	})
}

// CollectBindRefs 递归收集 <bind> 表达式引用的参数路径（去重、按首现序）。
func CollectBindRefs(items []Node) []string {
	return collectNodeStrings(items, func(n Node) []string {
		if p, ok := n.(interface{ BindRefs() []string }); ok {
			return p.BindRefs()
		}
		return nil
	})
}

// ContainsBind 判断片段树中是否存在 <bind>。
func ContainsBind(items []Node) bool {
	found := false
	walkNodeTree(items, func(n Node) {
		if _, ok := n.(interface{ BoundName() string }); ok {
			found = true
		}
	})
	return found
}

func collectNodeStrings(items []Node, pick func(Node) []string) []string {
	seen := map[string]bool{}
	var out []string
	walkNodeTree(items, func(n Node) {
		for _, s := range pick(n) {
			k := buildKey(s)
			if k == "" || seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, s)
		}
	})
	return out
}

// ParseFragments 将 Mapper 语句体子元素解析为片段列表：
// 单个元素解析失败仅记日志跳过（保持既有 log-and-continue 容错），不中断整体。
func ParseFragments(elems []XmlElement, sns map[string]*SqlElement) ([]Node, error) {
	var sts []Node
	for _, elem := range elems {
		st, err := parseFragmentFromXmlElement(elem, sns)
		if err != nil {
			log.Errorf("parse error:%v", err)
			continue
		}
		sts = append(sts, st)
	}
	return sts, nil
}

// CollectSqlSlots 收集无条件位置的占位符名（#{} / ${}，去重、按首现序）。
// 推导规则（1.1）：<if>/<choose>/<foreach> 体内的占位符不统计——条件体由运行时求值，
// 不视为必参（selectAll 类「占位符全在 if 体内」语句保持无参调用契约）；
// <include>/<where>/<set> 的直接文本视为无条件位置，正常统计。
func CollectSqlSlots(items []Node) []string {
	seen := map[string]bool{}
	var out []string
	for _, it := range items {
		if it == nil {
			continue
		}
		for _, name := range it.CollectSlots() {
			k := buildKey(name)
			if k != "" && !seen[k] {
				seen[k] = true
				out = append(out, name)
			}
		}
	}
	return out
}

// CollectAllSqlSlots 递归收集片段树中全部占位符名（#{} / ${}，含 <if>/<choose>/
// <foreach>/<where>/<set>/<include> 体内，去重、按首现序）。
// 与 CollectSqlSlots（仅无条件位置）互补：前者服务运行期 Need 推导与按位绑定（1.1/1.2）；
// 本函数服务 xml2go 签名推导（E1）——动态体内占位符同样是调用方需要提供的参数，
// 存在「仅在动态体内出现的占位符」（len(AllSlots) > len(Slots)）时签名应退化为单 map 按名取数。
func CollectAllSqlSlots(items []Node) []string {
	seen := map[string]bool{}
	var out []string
	var walk func(nodes []Node)
	walk = func(nodes []Node) {
		for _, it := range nodes {
			if it == nil {
				continue
			}
			for _, name := range it.CollectSlots() {
				k := buildKey(name)
				if k != "" && !seen[k] {
					seen[k] = true
					out = append(out, name)
				}
			}
			walk(it.Children())
		}
	}
	walk(items)
	return out
}

// CollectIfTestFieldNames 从片段树递归提取所有 <if test> 条件引用的字段名（4.3）。
// 条件字段名去除 .length 后缀、跳过 params. 前缀（通用 map 参数，无法静态校验），
// 全局去重后返回。<include> 引用片段内条件不下钻（保持原行为）。
func CollectIfTestFieldNames(items []Node) []string {
	seen := map[string]bool{}
	var names []string
	for _, it := range items {
		if it == nil {
			continue
		}
		for _, name := range it.CollectIfTestFields() {
			if name != "" && !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names
}

// ContainsForEach 递归检查片段树（含 if/include/choose/where/set 嵌套）中是否含 <foreach>。
func ContainsForEach(items []Node) bool {
	for _, it := range items {
		if it != nil && it.ContainsForEach() {
			return true
		}
	}
	return false
}

// CollectUnsupportedIfTests 从片段树递归提取所有 <if>/<when test> 中未识别的
// 子表达式（A3）。经 Children() 遍历覆盖嵌套（if/choose/where/set 等），
// 未识别表达式全局去重后按首现序返回；无未识别时返回 nil。
func CollectUnsupportedIfTests(items []Node) []string {
	seen := map[string]bool{}
	var out []string
	var walk func(nodes []Node)
	walk = func(nodes []Node) {
		for _, it := range nodes {
			if it == nil {
				continue
			}
			if p, ok := it.(interface{ UnsupportedIfTests() []string }); ok {
				for _, u := range p.UnsupportedIfTests() {
					if u != "" && !seen[u] {
						seen[u] = true
						out = append(out, u)
					}
				}
			}
			walk(it.Children())
		}
	}
	walk(items)
	return out
}
