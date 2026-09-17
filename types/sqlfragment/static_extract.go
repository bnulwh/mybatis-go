package sqlfragment

import (
	"bytes"
	"strings"
)

// StaticParam 静态 SQL 的占位符参数（按语句中出现顺序，S1）。
type StaticParam struct {
	Name     string // #{name,...} 的 name
	JdbcType string // jdbcType 属性（可空）
}

// ExtractStaticSQL 判定片段树是否为「纯静态 SQL」（S1 生成器输入）：
// 仅由静态文本与纯文本 <include> 组成——不含 <if>/<where>/<set>/<trim>/<choose>/<foreach>
// 等动态节点，也不含 ${} 原始注入。静态时返回把 #{} 统一替换为 ? 的 SQL
// 与按出现顺序的参数列表；非静态（或无可提取文本）返回 ok=false。
// 拼接语义与 PrepareSqlWithMap 保持一致：片段间以空格连接。
func ExtractStaticSQL(items []Node) (string, []StaticParam, bool) {
	var buf bytes.Buffer
	var params []StaticParam
	for _, it := range items {
		if it == nil {
			continue
		}
		text, ps, ok := extractStaticNode(it)
		if !ok {
			return "", nil, false
		}
		buf.WriteString(" ")
		buf.WriteString(text)
		params = append(params, ps...)
	}
	sqlstr := strings.TrimSpace(buf.String())
	if sqlstr == "" {
		return "", nil, false
	}
	return sqlstr, params, true
}

// extractStaticNode 单节点静态提取：simpleSql 直接提取；
// sqlInclude 无嵌套片段时取其静态文本（<sql> 纯文本场景），有嵌套片段时递归
// （嵌套片段须全为静态文本，出现动态标签即非静态）；其余节点一律视为非静态。
func extractStaticNode(n Node) (string, []StaticParam, bool) {
	switch v := n.(type) {
	case *simpleSql:
		return extractSimpleSql(v)
	case *sqlInclude:
		if len(v.Fragments) == 0 {
			return extractSimpleSql(parseSimpleSqlFromText(v.Sql))
		}
		var buf bytes.Buffer
		var params []StaticParam
		for _, f := range v.Fragments {
			if f == nil {
				continue
			}
			text, ps, ok := extractStaticNode(f)
			if !ok {
				return "", nil, false
			}
			buf.WriteString(" ")
			buf.WriteString(text)
			params = append(params, ps...)
		}
		return strings.TrimSpace(buf.String()), params, true
	}
	return "", nil, false
}

// extractSimpleSql 静态文本提取：#{...} 按出现顺序替换为 ?（与 PrepareSqlWithMap
// 的逐个 Replace 语义一致，重复占位符按位推进），${...} 判非静态。
// 纯空白且无占位符的文本（标签间换行等）合法返回空文本。
func extractSimpleSql(s *simpleSql) (string, []StaticParam, bool) {
	sqlstr := s.Sql
	var params []StaticParam
	for _, p := range s.Params {
		if p.Raw {
			return "", nil, false // ${...} 原始注入：无法静态参数化
		}
		sqlstr = strings.Replace(sqlstr, p.Origin, "?", 1)
		params = append(params, StaticParam{Name: p.Name, JdbcType: p.TypeName})
	}
	return sqlstr, params, true
}
