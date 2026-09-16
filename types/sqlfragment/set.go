package sqlfragment

import (
	"bytes"
	"regexp"
	"strings"
)

// sqlSet 对应 MyBatis <set> 标签：
// 子片段全部为空时不输出任何内容；否则输出 "set " 前缀，
// 并剥离拼接结果的前导/尾随逗号（<if> 片段常见的尾部 "," 会留下多余逗号）。
type sqlSet struct {
	Sql []Node
}

var (
	reLeadingComma  = regexp.MustCompile(`^\s*,+`)
	reTrailingComma = regexp.MustCompile(`,+\s*$`)
)

// trimSetCommas 去除 SQL 片段开头和结尾的逗号（含空白），
// 用于 <set> 内 <if> 片段留下的多余逗号清理。
func trimSetCommas(s string) string {
	s = strings.TrimSpace(reLeadingComma.ReplaceAllString(s, ""))
	s = reTrailingComma.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func (in *sqlSet) PrepareSqlWithMap(mp map[string]interface{}, depth int) (string, []string) {
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
	content := trimSetCommas(buf.String())
	if content == "" {
		return "", []string{}
	}
	return "set " + content, results
}

func (in *sqlSet) GenerateSqlWithMap(mp map[string]interface{}, depth int) string {
	var buf bytes.Buffer
	for _, item := range in.Sql {
		sqlstr := item.GenerateSqlWithMap(mp, depth+1)
		if strings.TrimSpace(sqlstr) != "" {
			buf.WriteString(" ")
			buf.WriteString(sqlstr)
		}
	}
	content := trimSetCommas(buf.String())
	if content == "" {
		return ""
	}
	return "set " + content
}

func (in *sqlSet) PrepareSqlWithParam(m interface{}) (string, []string) {
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
	content := trimSetCommas(buf.String())
	if content == "" {
		return "", []string{}
	}
	return "set " + content, results
}

func (in *sqlSet) GenerateSqlWithParam(m interface{}) string {
	var buf bytes.Buffer
	for _, item := range in.Sql {
		sqlstr := item.GenerateSqlWithParam(m)
		if strings.TrimSpace(sqlstr) != "" {
			buf.WriteString(" ")
			buf.WriteString(sqlstr)
		}
	}
	content := trimSetCommas(buf.String())
	if content == "" {
		return ""
	}
	return "set " + content
}

func (in *sqlSet) PrepareSqlWithSlice(m []interface{}, depth int) (string, []string) {
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
	content := trimSetCommas(buf.String())
	if content == "" {
		return "", []string{}
	}
	return "set " + content, results
}

func (in *sqlSet) GenerateSqlWithSlice(m []interface{}, depth int) string {
	var buf bytes.Buffer
	for _, item := range in.Sql {
		sqlstr := item.GenerateSqlWithSlice(m, depth+1)
		if strings.TrimSpace(sqlstr) != "" {
			buf.WriteString(" ")
			buf.WriteString(sqlstr)
		}
	}
	content := trimSetCommas(buf.String())
	if content == "" {
		return ""
	}
	return "set " + content
}

func (in *sqlSet) GenerateSqlWithoutParam() string {
	var buf bytes.Buffer
	for _, item := range in.Sql {
		sqlstr := item.GenerateSqlWithoutParam()
		if strings.TrimSpace(sqlstr) != "" {
			buf.WriteString(" ")
			buf.WriteString(sqlstr)
		}
	}
	content := trimSetCommas(buf.String())
	if content == "" {
		return ""
	}
	return "set " + content
}

func (in *sqlSet) Children() []Node {
	return in.Sql
}

// CollectSlots 直接文本视为无条件位置，正常统计（递归聚合子片段）。
func (in *sqlSet) CollectSlots() []string {
	var out []string
	for _, item := range in.Sql {
		if item == nil {
			continue
		}
		out = append(out, item.CollectSlots()...)
	}
	return out
}

func (in *sqlSet) ContainsForEach() bool {
	for _, item := range in.Sql {
		if item != nil && item.ContainsForEach() {
			return true
		}
	}
	return false
}

func (in *sqlSet) CollectIfTestFields() []string {
	var names []string
	for _, item := range in.Sql {
		if item == nil {
			continue
		}
		names = append(names, item.CollectIfTestFields()...)
	}
	return names
}

// parseSqlSetNode 解析 <set> 标签，其子元素可以是文本、<if>、<foreach>、<include>、<choose> 等。
func parseSqlSetNode(node XmlNode, sns map[string]*SqlElement) (Node, error) {
	return parseSqlSetFromXmlNode(node.Elements, sns)
}

func parseSqlSetFromXmlNode(elems []XmlElement, sns map[string]*SqlElement) (*sqlSet, error) {
	var sts []Node
	for _, elem := range elems {
		st, err := parseFragmentFromXmlElement(elem, sns)
		if err != nil {
			return nil, err
		}
		sts = append(sts, st)
	}
	return &sqlSet{
		Sql: sts,
	}, nil
}

func init() {
	RegisterNodeParser("set", parseSqlSetNode)
}
