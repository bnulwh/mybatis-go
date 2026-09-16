package sqlfragment

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/bnulwh/mybatis-go/log"
)

// sqlFragmentParam 占位符参数（#{name,jdbcType=...} / ${name}）。
type sqlFragmentParam struct {
	Name     string
	TypeName string
	Type     reflect.Type
	Origin   string
	Raw      bool // ${...} 原始替换：不入占位符参数、不加引号
}

// simpleSql 静态文本节点：含 #{} / ${} 占位符的 SQL 文本。
type simpleSql struct {
	Sql    string
	Params []sqlFragmentParam
}

func (in *simpleSql) PrepareSqlWithMap(mp map[string]interface{}, depth int) (string, []string) {
	log.Debugf("simple sql prepare sql with map: %v", mp)
	sqlstr := in.Sql
	var results []string
	for _, param := range in.Params {
		val, ok := lookupParam(mp, param.Name)
		if !ok {
			log.Warnf("not found %v in map", param.Name)
			continue
		}
		if param.Raw {
			// ${...} 原始注入：直接拼接值，不入占位符参数
			sqlstr = strings.ReplaceAll(sqlstr, param.Origin, rawFormatValue(val))
			continue
		}
		sqlstr = strings.ReplaceAll(sqlstr, param.Origin, "?")
		results = append(results, formatValue(val))
	}
	return sqlstr, results
}

func (in *simpleSql) GenerateSqlWithMap(mp map[string]interface{}, depth int) string {
	log.Debugf("simple sql generate sql with map: %v", mp)
	sqlstr := in.Sql
	for _, param := range in.Params {
		val, ok := lookupParam(mp, param.Name)
		if !ok {
			log.Warnf("not found %v in map", param.Name)
			continue
		}
		var valstr string
		if param.Raw {
			// ${...} 原始注入：不加引号
			valstr = rawFormatValue(val)
		} else {
			valstr = formatValue(val)
		}
		sqlstr = strings.ReplaceAll(sqlstr, param.Origin, valstr)
	}
	return sqlstr
}

func (in *simpleSql) PrepareSqlWithParam(m interface{}) (string, []string) {
	log.Debugf("simple sql prepare sql with param: %v", m)
	sqlstr := in.Sql
	var results []string
	for _, param := range in.Params {
		if param.Raw {
			sqlstr = strings.ReplaceAll(sqlstr, param.Origin, rawFormatValue(m))
			continue
		}
		sqlstr = strings.ReplaceAll(sqlstr, param.Origin, "?")
		results = append(results, formatValue(m))
	}
	return sqlstr, results
}

func (in *simpleSql) GenerateSqlWithParam(m interface{}) string {
	log.Debugf("simple sql generate sql with param: %v", m)
	sqlstr := in.Sql
	for _, param := range in.Params {
		var valstr string
		if param.Raw {
			valstr = rawFormatValue(m)
		} else {
			valstr = formatValue(m)
		}
		sqlstr = strings.ReplaceAll(sqlstr, param.Origin, valstr)
	}
	return sqlstr
}

func (in *simpleSql) PrepareSqlWithSlice(m []interface{}, depth int) (string, []string) {
	log.Debugf("simple sql generate sql with slice : %v depth: %v", m, depth)
	if len(in.Params) == 0 {
		return in.Sql, []string{}
	}
	panic("simple sql has param not replaced!!!!")
}

func (in *simpleSql) GenerateSqlWithSlice(m []interface{}, depth int) string {
	log.Debugf("simple sql generate sql with slice : %v depth: %v", m, depth)
	if len(in.Params) == 0 {
		return in.Sql
	}
	panic("simple sql has param not replaced!!!!")
}

func (in *simpleSql) GenerateSqlWithoutParam() string {
	log.Debugf("simple sql generate sql without param")
	return in.Sql
}

func (in *simpleSql) Children() []Node {
	return nil
}

// CollectSlots 本节点直接贡献的占位符名（无条件位置，全部统计）。
func (in *simpleSql) CollectSlots() []string {
	var out []string
	for _, p := range in.Params {
		out = append(out, p.Name)
	}
	return out
}

func (in *simpleSql) ContainsForEach() bool {
	return false
}

func (in *simpleSql) CollectIfTestFields() []string {
	return nil
}

// parseSimpleSqlFromText 解析文本为 simpleSql：提取 #{} / ${} 占位符参数。
func parseSimpleSqlFromText(text string) *simpleSql {
	return &simpleSql{
		Sql:    text,
		Params: parseSqlFragmentParamFromText(text),
	}
}

func parseSqlFragmentParamFromText(text string) []sqlFragmentParam {
	re := regexp.MustCompile(`[#$][{][\s]*([\w.]+)[\s]*(,[\s]*([\w.]+)[\s]*=[\s]*([\w.]+)[\s]*)*[}]`)
	matches := re.FindAllStringSubmatch(text, -1)
	var stps []sqlFragmentParam
	for _, match := range matches {
		raw := strings.HasPrefix(match[0], "${")
		if len(match) == 2 {
			stps = append(stps, sqlFragmentParam{
				Origin:   match[0],
				Name:     match[1],
				TypeName: "",
				Raw:      raw,
			})
		} else if len(match) == 5 {
			if len(match[4]) > 0 {
				stps = append(stps, sqlFragmentParam{
					Origin:   match[0],
					Name:     match[1],
					TypeName: match[4],
					Type:     parseJdbcTypeFrom(match[4]),
					Raw:      raw,
				})
			} else {
				stps = append(stps, sqlFragmentParam{
					Origin:   match[0],
					Name:     match[1],
					TypeName: "",
					Raw:      raw,
				})
			}
		}
	}
	return stps
}
