package sqlfragment

import (
	"bytes"
	"regexp"
	"strings"
)

// sqlWhere 对应 MyBatis <where> 标签：
// 子片段全部为空时不输出任何内容；否则输出 "where " 前缀，
// 并去除首个条件的开头 AND/OR（大小写不敏感）。
type sqlWhere struct {
	Sql []Node
}

var reLeadingAndOr = regexp.MustCompile(`(?i)^\s*(and|or)\s+`)

// trimLeadingAndOr 去除 SQL 片段开头的 AND / OR（含前导空白），
// 用于 <where> 内首个条件的 AND/OR 剥离。
func trimLeadingAndOr(s string) string {
	return strings.TrimSpace(reLeadingAndOr.ReplaceAllString(s, ""))
}

func (in *sqlWhere) PrepareSqlWithMap(mp map[string]interface{}, depth int) (string, []string) {
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
	return "where " + trimLeadingAndOr(buf.String()), results
}

func (in *sqlWhere) GenerateSqlWithMap(mp map[string]interface{}, depth int) string {
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
	return "where " + trimLeadingAndOr(buf.String())
}

func (in *sqlWhere) PrepareSqlWithParam(m interface{}) (string, []string) {
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
	return "where " + trimLeadingAndOr(buf.String()), results
}

func (in *sqlWhere) GenerateSqlWithParam(m interface{}) string {
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
	return "where " + trimLeadingAndOr(buf.String())
}

func (in *sqlWhere) PrepareSqlWithSlice(m []interface{}, depth int) (string, []string) {
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
	return "where " + trimLeadingAndOr(buf.String()), results
}

func (in *sqlWhere) GenerateSqlWithSlice(m []interface{}, depth int) string {
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
	return "where " + trimLeadingAndOr(buf.String())
}

func (in *sqlWhere) GenerateSqlWithoutParam() string {
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
	return "where " + trimLeadingAndOr(buf.String())
}

func (in *sqlWhere) Children() []Node {
	return in.Sql
}

// CollectSlots 直接文本视为无条件位置，正常统计（递归聚合子片段）。
func (in *sqlWhere) CollectSlots() []string {
	var out []string
	for _, item := range in.Sql {
		if item == nil {
			continue
		}
		out = append(out, item.CollectSlots()...)
	}
	return out
}

func (in *sqlWhere) ContainsForEach() bool {
	for _, item := range in.Sql {
		if item != nil && item.ContainsForEach() {
			return true
		}
	}
	return false
}

func (in *sqlWhere) CollectIfTestFields() []string {
	var names []string
	for _, item := range in.Sql {
		if item == nil {
			continue
		}
		names = append(names, item.CollectIfTestFields()...)
	}
	return names
}

// parseSqlWhereNode 解析 <where> 标签，其子元素可以是文本、<if>、<foreach>、<include>、<choose> 等。
func parseSqlWhereNode(node XmlNode, sns map[string]*SqlElement) (Node, error) {
	return parseSqlWhereFromXmlNode(node.Elements, sns)
}

func parseSqlWhereFromXmlNode(elems []XmlElement, sns map[string]*SqlElement) (*sqlWhere, error) {
	var sts []Node
	for _, elem := range elems {
		st, err := parseFragmentFromXmlElement(elem, sns)
		if err != nil {
			return nil, err
		}
		sts = append(sts, st)
	}
	return &sqlWhere{
		Sql: sts,
	}, nil
}

func init() {
	RegisterNodeParser("where", parseSqlWhereNode)
}
