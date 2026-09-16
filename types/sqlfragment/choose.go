package sqlfragment

import (
	"fmt"
	"strings"

	"github.com/bnulwh/mybatis-go/log"
)

// sqlChoose 对应 MyBatis <choose>/<when>/<otherwise>：首个满足条件的 when 渲染，否则 otherwise。
type sqlChoose struct {
	Otherwise *simpleSql
	When      []*sqlIfTest
}

func init() {
	RegisterNodeParser("choose", func(node XmlNode, sns map[string]*SqlElement) (Node, error) {
		return parseSqlChooseFromXmlNode(node.Elements)
	})
}

func (in *sqlChoose) PrepareSqlWithMap(mp map[string]interface{}, depth int) (string, []string) {
	log.Debugf("sql choose prepare sql with map: %v", mp)
	for _, item := range in.When {
		if item.checkConditions(mp) {
			return item.PrepareSqlWithMap(mp, depth+1)
		}
	}
	return in.Otherwise.PrepareSqlWithMap(mp, depth+1)
}

func (in *sqlChoose) GenerateSqlWithMap(mp map[string]interface{}, depth int) string {
	log.Debugf("sql choose generate sql with map: %v", mp)
	for _, item := range in.When {
		if item.checkConditions(mp) {
			return item.GenerateSqlWithMap(mp, depth+1)
		}
	}
	return in.Otherwise.GenerateSqlWithMap(mp, depth+1)
}

func (in *sqlChoose) PrepareSqlWithParam(m interface{}) (string, []string) {
	return in.Otherwise.PrepareSqlWithParam(m)
}

func (in *sqlChoose) GenerateSqlWithParam(m interface{}) string {
	return in.Otherwise.GenerateSqlWithParam(m)
}

func (in *sqlChoose) PrepareSqlWithSlice(m []interface{}, depth int) (string, []string) {
	return "", []string{}
}

func (in *sqlChoose) GenerateSqlWithSlice(m []interface{}, depth int) string {
	return ""
}

// GenerateSqlWithoutParam 无参渲染取 otherwise 静态文本（含未替换占位符原样，保持原行为）。
func (in *sqlChoose) GenerateSqlWithoutParam() string {
	return in.Otherwise.Sql
}

func (in *sqlChoose) Children() []Node {
	var children []Node
	for _, w := range in.When {
		children = append(children, w)
	}
	children = append(children, in.Otherwise)
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
	return false
}

// CollectIfTestFields 各 when 条件字段 + when 子片段递归聚合（otherwise 为静态文本，无条件）。
func (in *sqlChoose) CollectIfTestFields() []string {
	var names []string
	for _, w := range in.When {
		if w == nil {
			continue
		}
		names = append(names, w.CollectIfTestFields()...)
	}
	return names
}

func parseSqlChooseFromXmlNode(elems []XmlElement) (*sqlChoose, error) {
	var conds []*sqlIfTest
	var defCond []*simpleSql
	for _, elem := range elems {
		xn := elem.Val.(XmlNode)
		switch strings.ToLower(xn.Name) {
		case "when":
			st, err := parseSqlIfTestFromXmlNode(xn.Attrs, xn.Elements, nil)
			if err != nil {
				return nil, err
			}
			conds = append(conds, st)
		case "otherwise":
			dc := parseSimpleSqlFromText(xn.Elements[0].Val.(string))
			defCond = append(defCond, dc)
		}
	}
	if len(defCond) < 1 {
		return nil, fmt.Errorf("choose sql not contains otherwise")
	}
	return &sqlChoose{
		When:      conds,
		Otherwise: defCond[0],
	}, nil
}
