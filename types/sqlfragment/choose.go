package sqlfragment

import (
	"fmt"
	"strings"

	"github.com/bnulwh/mybatis-go/log"
)

// sqlChoose 对应 MyBatis <choose>/<when>/<otherwise>：首个满足条件的 when 渲染，否则 otherwise。
// otherwise 为片段树 []Node（A1 同族修复：支持嵌套 <if>/<include> 与文本混排，不再断言纯文本）。
type sqlChoose struct {
	Otherwise []Node
	When      []*sqlIfTest
}

func init() {
	RegisterNodeParser("choose", func(node XmlNode, sns map[string]*SqlElement) (Node, error) {
		return parseSqlChooseFromXmlNode(node.Elements, sns)
	})
}

func (in *sqlChoose) PrepareSqlWithMap(mp map[string]interface{}, depth int) (string, []string) {
	log.Debugf("sql choose prepare sql with map: %v", mp)
	for _, item := range in.When {
		if item.checkConditions(mp) {
			return item.PrepareSqlWithMap(mp, depth+1)
		}
	}
	return in.prepareOtherwise(func(f Node, d int) (string, []string) {
		return f.PrepareSqlWithMap(mp, d)
	}, depth)
}

func (in *sqlChoose) GenerateSqlWithMap(mp map[string]interface{}, depth int) string {
	log.Debugf("sql choose generate sql with map: %v", mp)
	for _, item := range in.When {
		if item.checkConditions(mp) {
			return item.GenerateSqlWithMap(mp, depth+1)
		}
	}
	sqlstr, _ := in.prepareOtherwise(func(f Node, d int) (string, []string) {
		return f.GenerateSqlWithMap(mp, d), []string{}
	}, depth)
	return sqlstr
}

func (in *sqlChoose) PrepareSqlWithParam(m interface{}) (string, []string) {
	return in.prepareOtherwise(func(f Node, d int) (string, []string) {
		return f.PrepareSqlWithParam(m)
	}, 0)
}

func (in *sqlChoose) GenerateSqlWithParam(m interface{}) string {
	sqlstr, _ := in.prepareOtherwise(func(f Node, d int) (string, []string) {
		return f.GenerateSqlWithParam(m), []string{}
	}, 0)
	return sqlstr
}

func (in *sqlChoose) PrepareSqlWithSlice(m []interface{}, depth int) (string, []string) {
	return "", []string{}
}

func (in *sqlChoose) GenerateSqlWithSlice(m []interface{}, depth int) string {
	return ""
}

// GenerateSqlWithoutParam 无参渲染取 otherwise 各节点静态文本拼接
// （含未替换占位符原样，保持原行为）。
func (in *sqlChoose) GenerateSqlWithoutParam() string {
	var buf strings.Builder
	for _, item := range in.Otherwise {
		buf.WriteString(" ")
		buf.WriteString(item.GenerateSqlWithoutParam())
	}
	return buf.String()
}

func (in *sqlChoose) Children() []Node {
	var children []Node
	for _, w := range in.When {
		children = append(children, w)
	}
	children = append(children, in.Otherwise...)
	return children
}

// CollectSlots 条件分支内占位符不统计（运行时求值，不视为必参）。
func (in *sqlChoose) CollectSlots() []string {
	return nil
}

func (in *sqlChoose) ContainsForEach() bool {
	for _, w := range in.When {
		if w != nil && w.ContainsForEach() {
			return true
		}
	}
	for _, o := range in.Otherwise {
		if o != nil && o.ContainsForEach() {
			return true
		}
	}
	return false
}

// CollectIfTestFields 各 when 条件字段 + when/otherwise 子片段递归聚合。
func (in *sqlChoose) CollectIfTestFields() []string {
	var names []string
	for _, w := range in.When {
		if w == nil {
			continue
		}
		names = append(names, w.CollectIfTestFields()...)
	}
	for _, o := range in.Otherwise {
		if o == nil {
			continue
		}
		names = append(names, o.CollectIfTestFields()...)
	}
	return names
}

// prepareOtherwise 按文档序渲染 otherwise 片段树。
func (in *sqlChoose) prepareOtherwise(render func(f Node, depth int) (string, []string), depth int) (string, []string) {
	var buf strings.Builder
	var results []string
	for _, item := range in.Otherwise {
		buf.WriteString(" ")
		sqlstr, items := render(item, depth+1)
		buf.WriteString(sqlstr)
		results = append(results, items...)
	}
	return buf.String(), results
}

// parseSqlChooseFromXmlNode 解析 <choose>：when/otherwise 子标签经统一注册表解析
// （A1 同族修复：otherwise 支持嵌套标签；choose 直接子文本安全跳过，不再断言 XmlNode）。
func parseSqlChooseFromXmlNode(elems []XmlElement, sns map[string]*SqlElement) (*sqlChoose, error) {
	var conds []*sqlIfTest
	var defCond []Node
	for _, elem := range elems {
		if elem.ElementType != XmlNodeElem {
			continue
		}
		xn, ok := elem.Val.(XmlNode)
		if !ok {
			continue
		}
		switch strings.ToLower(xn.Name) {
		case "when":
			st, err := parseSqlIfTestFromXmlNode(xn.Attrs, xn.Elements, sns)
			if err != nil {
				return nil, err
			}
			conds = append(conds, st)
		case "otherwise":
			nds, err := ParseFragments(xn.Elements, sns)
			if err != nil {
				return nil, err
			}
			defCond = nds
		}
	}
	if len(defCond) < 1 {
		return nil, fmt.Errorf("choose sql not contains otherwise")
	}
	return &sqlChoose{
		When:      conds,
		Otherwise: defCond,
	}, nil
}
