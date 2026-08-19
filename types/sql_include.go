package types

import (
	"bytes"
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
)

type sqlInclude struct {
	Sql       string
	Fragments []*sqlFragment
	Refid     string
}

func parseSqlIncludeFromXmlNode(attrs map[string]string, sns map[string]*SqlElement) (*sqlFragment, error) {
	log.Debugf("parse sql include from: %v", ToJson(attrs))
	attr, ok := attrs["refid"]
	if ok {
		sn, ok := sns[attr]
		if ok {
			return &sqlFragment{
				Include: &sqlInclude{
					Sql:       sn.Sql,
					Fragments: sn.Fragments,
					Refid:     attr,
				},
				Sql:     nil,
				IfTest:  nil,
				ForLoop: nil,
				Choose:  nil,
				Where:   nil,
				Type:    includeSqlFragment,
			}, nil
		}
		return nil, fmt.Errorf("not found sql id=%v", attr)
	}
	return nil, fmt.Errorf("not found refid")
}

// renderFragments 将 include 的片段按给定渲染函数拼接；纯文本 sql 片段直接返回静态文本。
func (in *sqlInclude) renderFragments(render func(f *sqlFragment, depth int) (string, []string), depth int) (string, []string) {
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

func (in *sqlInclude) prepareSqlWithMap(mp map[string]interface{}, depth int) (string, []string) {
	return in.renderFragments(func(f *sqlFragment, d int) (string, []string) {
		return f.prepareSqlWithMap(mp, d)
	}, depth)
}

func (in *sqlInclude) generateSqlWithMap(mp map[string]interface{}, depth int) string {
	sqlstr, _ := in.renderFragments(func(f *sqlFragment, d int) (string, []string) {
		return f.generateSqlWithMap(mp, d), []string{}
	}, depth)
	return sqlstr
}

func (in *sqlInclude) prepareSqlWithParam(m interface{}) (string, []string) {
	return in.renderFragments(func(f *sqlFragment, d int) (string, []string) {
		return f.prepareSqlWithParam(m)
	}, 0)
}

func (in *sqlInclude) generateSqlWithParam(m interface{}) string {
	sqlstr, _ := in.renderFragments(func(f *sqlFragment, d int) (string, []string) {
		return f.generateSqlWithParam(m), []string{}
	}, 0)
	return sqlstr
}

func (in *sqlInclude) prepareSqlWithSlice(m []interface{}, depth int) (string, []string) {
	return in.renderFragments(func(f *sqlFragment, d int) (string, []string) {
		return f.prepareSqlWithSlice(m, d)
	}, depth)
}

func (in *sqlInclude) generateSqlWithSlice(m []interface{}, depth int) string {
	sqlstr, _ := in.renderFragments(func(f *sqlFragment, d int) (string, []string) {
		return f.generateSqlWithSlice(m, d), []string{}
	}, depth)
	return sqlstr
}

func (in *sqlInclude) generateSqlWithoutParam() string {
	sqlstr, _ := in.renderFragments(func(f *sqlFragment, d int) (string, []string) {
		return f.generateSqlWithoutParam(), []string{}
	}, 0)
	return sqlstr
}
