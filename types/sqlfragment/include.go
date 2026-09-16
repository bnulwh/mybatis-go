package sqlfragment

import (
	"bytes"
	"fmt"

	"github.com/bnulwh/mybatis-go/log"
)

// sqlInclude 对应 MyBatis <include refid="...">：引用 <sql id="..."> 片段。
type sqlInclude struct {
	Sql       string
	Fragments []Node
	Refid     string
}

func init() {
	RegisterNodeParser("include", func(node XmlNode, sns map[string]*SqlElement) (Node, error) {
		return parseSqlIncludeFromXmlNode(node.Attrs, sns)
	})
}

func parseSqlIncludeFromXmlNode(attrs map[string]string, sns map[string]*SqlElement) (*sqlInclude, error) {
	log.Debugf("parse sql include from: %v", toJson(attrs))
	attr, ok := attrs["refid"]
	if ok {
		sn, ok := sns[attr]
		if ok {
			return &sqlInclude{
				Sql:       sn.Sql,
				Fragments: sn.Fragments,
				Refid:     attr,
			}, nil
		}
		return nil, fmt.Errorf("not found sql id=%v", attr)
	}
	return nil, fmt.Errorf("not found refid")
}

// renderFragments 将 include 的片段按给定渲染函数拼接；纯文本 sql 片段直接返回静态文本。
func (in *sqlInclude) renderFragments(render func(f Node, depth int) (string, []string), depth int) (string, []string) {
	if len(in.Fragments) == 0 {
		return in.Sql, []string{}
	}
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Fragments {
		buf.WriteString(" ")
		sqlstr, items := render(item, depth+1)
		buf.WriteString(sqlstr)
		results = append(results, items...)
	}
	return buf.String(), results
}

func (in *sqlInclude) PrepareSqlWithMap(mp map[string]interface{}, depth int) (string, []string) {
	return in.renderFragments(func(f Node, d int) (string, []string) {
		return f.PrepareSqlWithMap(mp, d)
	}, depth)
}

func (in *sqlInclude) GenerateSqlWithMap(mp map[string]interface{}, depth int) string {
	sqlstr, _ := in.renderFragments(func(f Node, d int) (string, []string) {
		return f.GenerateSqlWithMap(mp, d), []string{}
	}, depth)
	return sqlstr
}

func (in *sqlInclude) PrepareSqlWithParam(m interface{}) (string, []string) {
	return in.renderFragments(func(f Node, d int) (string, []string) {
		return f.PrepareSqlWithParam(m)
	}, 0)
}

func (in *sqlInclude) GenerateSqlWithParam(m interface{}) string {
	sqlstr, _ := in.renderFragments(func(f Node, d int) (string, []string) {
		return f.GenerateSqlWithParam(m), []string{}
	}, 0)
	return sqlstr
}

func (in *sqlInclude) PrepareSqlWithSlice(m []interface{}, depth int) (string, []string) {
	return in.renderFragments(func(f Node, d int) (string, []string) {
		return f.PrepareSqlWithSlice(m, d)
	}, depth)
}

func (in *sqlInclude) GenerateSqlWithSlice(m []interface{}, depth int) string {
	sqlstr, _ := in.renderFragments(func(f Node, d int) (string, []string) {
		return f.GenerateSqlWithSlice(m, d), []string{}
	}, depth)
	return sqlstr
}

func (in *sqlInclude) GenerateSqlWithoutParam() string {
	sqlstr, _ := in.renderFragments(func(f Node, d int) (string, []string) {
		return f.GenerateSqlWithoutParam(), []string{}
	}, 0)
	return sqlstr
}

func (in *sqlInclude) Children() []Node {
	return in.Fragments
}

// CollectSlots include 引用的片段视为无条件位置，递归统计（与原 collectSqlSlots 一致）。
func (in *sqlInclude) CollectSlots() []string {
	var out []string
	for _, item := range in.Fragments {
		if item == nil {
			continue
		}
		out = append(out, item.CollectSlots()...)
	}
	return out
}

func (in *sqlInclude) ContainsForEach() bool {
	for _, item := range in.Fragments {
		if item != nil && item.ContainsForEach() {
			return true
		}
	}
	return false
}

// CollectIfTestFields 引用片段内的条件字段不统计（与原 CollectIfTestFieldNames 不下钻 include 一致）。
func (in *sqlInclude) CollectIfTestFields() []string {
	return nil
}
