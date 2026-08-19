package types

import (
	"bytes"
	"regexp"
	"strings"
)

// sqlWhere 对应 MyBatis <where> 标签：
// 子片段全部为空时不输出任何内容；否则输出 "where " 前缀，
// 并去除首个条件的开头 AND/OR（大小写不敏感）。
type sqlWhere struct {
	Sql []*sqlFragment
}

var reLeadingAndOr = regexp.MustCompile(`(?i)^\s*(and|or)\s+`)

// trimLeadingAndOr 去除 SQL 片段开头的 AND / OR（含前导空白），
// 用于 <where> 内首个条件的 AND/OR 剥离。
func trimLeadingAndOr(s string) string {
	return strings.TrimSpace(reLeadingAndOr.ReplaceAllString(s, ""))
}

func (in *sqlWhere) prepareSqlWithMap(mp map[string]interface{}, depth int) (string, []string) {
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
	if strings.TrimSpace(buf.String()) == "" {
		return "", []string{}
	}
	return "where " + trimLeadingAndOr(buf.String()), results
}

func (in *sqlWhere) generateSqlWithMap(mp map[string]interface{}, depth int) string {
	var buf bytes.Buffer
	for _, item := range in.Sql {
		sqlstr := item.generateSqlWithMap(mp, depth+1)
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

func (in *sqlWhere) prepareSqlWithParam(m interface{}) (string, []string) {
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
	if strings.TrimSpace(buf.String()) == "" {
		return "", []string{}
	}
	return "where " + trimLeadingAndOr(buf.String()), results
}

func (in *sqlWhere) generateSqlWithParam(m interface{}) string {
	var buf bytes.Buffer
	for _, item := range in.Sql {
		sqlstr := item.generateSqlWithParam(m)
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

func (in *sqlWhere) prepareSqlWithSlice(m []interface{}, depth int) (string, []string) {
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
	if strings.TrimSpace(buf.String()) == "" {
		return "", []string{}
	}
	return "where " + trimLeadingAndOr(buf.String()), results
}

func (in *sqlWhere) generateSqlWithSlice(m []interface{}, depth int) string {
	var buf bytes.Buffer
	for _, item := range in.Sql {
		sqlstr := item.generateSqlWithSlice(m, depth+1)
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

func (in *sqlWhere) generateSqlWithoutParam() string {
	var buf bytes.Buffer
	for _, item := range in.Sql {
		sqlstr := item.generateSqlWithoutParam()
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

// parseSqlWhereFromXmlNode 解析 <where> 标签，其子元素可以是文本、<if>、<foreach>、<include>、<choose> 等。
func parseSqlWhereFromXmlNode(elems []xmlElement, sns map[string]*SqlElement) (*sqlFragment, error) {
	var sts []*sqlFragment
	for _, elem := range elems {
		st, err := parsesqlFragmentFromXmlElement(elem, sns)
		if err != nil {
			return nil, err
		}
		sts = append(sts, st)
	}
	return &sqlFragment{
		Where: &sqlWhere{
			Sql: sts,
		},
		Sql:     nil,
		IfTest:  nil,
		ForLoop: nil,
		Include: nil,
		Choose:  nil,
		Type:    whereSqlFragment,
	}, nil
}
