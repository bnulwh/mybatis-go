package types

import (
	"bytes"
	"regexp"
	"strings"
)

// sqlSet 对应 MyBatis <set> 标签：
// 子片段全部为空时不输出任何内容；否则输出 "set " 前缀，
// 并剥离拼接结果的前导/尾随逗号（<if> 片段常见的尾部 "," 会留下多余逗号）。
type sqlSet struct {
	Sql []*sqlFragment
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

func (in *sqlSet) prepareSqlWithMap(mp map[string]interface{}, depth int) (string, []string) {
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Sql {
		sqlstr, items := item.prepareSqlWithMap(mp, depth+1)
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

func (in *sqlSet) generateSqlWithMap(mp map[string]interface{}, depth int) string {
	var buf bytes.Buffer
	for _, item := range in.Sql {
		sqlstr := item.generateSqlWithMap(mp, depth+1)
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

func (in *sqlSet) prepareSqlWithParam(m interface{}) (string, []string) {
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Sql {
		sqlstr, items := item.prepareSqlWithParam(m)
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

func (in *sqlSet) generateSqlWithParam(m interface{}) string {
	var buf bytes.Buffer
	for _, item := range in.Sql {
		sqlstr := item.generateSqlWithParam(m)
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

func (in *sqlSet) prepareSqlWithSlice(m []interface{}, depth int) (string, []string) {
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Sql {
		sqlstr, items := item.prepareSqlWithSlice(m, depth+1)
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

func (in *sqlSet) generateSqlWithSlice(m []interface{}, depth int) string {
	var buf bytes.Buffer
	for _, item := range in.Sql {
		sqlstr := item.generateSqlWithSlice(m, depth+1)
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

func (in *sqlSet) generateSqlWithoutParam() string {
	var buf bytes.Buffer
	for _, item := range in.Sql {
		sqlstr := item.generateSqlWithoutParam()
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

// parseSqlSetFromXmlNode 解析 <set> 标签，其子元素可以是文本、<if>、<foreach>、<include>、<choose> 等。
func parseSqlSetFromXmlNode(elems []xmlElement, sns map[string]*SqlElement) (*sqlFragment, error) {
	var sts []*sqlFragment
	for _, elem := range elems {
		st, err := parsesqlFragmentFromXmlElement(elem, sns)
		if err != nil {
			return nil, err
		}
		sts = append(sts, st)
	}
	return &sqlFragment{
		Where: nil,
		Set: &sqlSet{
			Sql: sts,
		},
		Sql:     nil,
		IfTest:  nil,
		ForLoop: nil,
		Include: nil,
		Choose:  nil,
		Type:    setSqlFragment,
	}, nil
}
