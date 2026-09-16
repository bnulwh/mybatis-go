package sqlfragment

import (
	"strings"

	"github.com/bnulwh/mybatis-go/log"
)

// SqlElement 对应 Mapper 中的 <sql id="..."> 片段：静态文本 + 可选嵌套片段，
// 供 <include refid="..."/> 引用。
type SqlElement struct {
	Sql       string
	Fragments []Node
	Id        string
}

// ParseSqlElementFromXmlNode 解析 <sql> 节点：文本拼接为静态 Sql，
// 嵌套标签（如 <where>/<if>/<foreach> 等）解析为片段，供 <include> 引用时渲染。
func ParseSqlElementFromXmlNode(node XmlNode) *SqlElement {
	log.Debugf("begin parse sql element from: %v", toJson(node))
	defer log.Debugf("finish parse sql element from: %v", toJson(node))
	var buf strings.Builder
	var frags []Node
	for _, elem := range node.Elements {
		switch elem.ElementType {
		case XmlTextElem:
			buf.WriteString(elem.Val.(string))
		case XmlNodeElem:
			// 嵌套标签（如 <where>/<if>/<foreach> 等）解析为片段，供 <include> 引用时渲染
			st, err := parseFragmentFromXmlElement(elem, nil)
			if err != nil {
				log.Warnf("parse nested element in sql element %v failed: %v", node.Id, err)
				continue
			}
			frags = append(frags, st)
		}
	}
	return &SqlElement{
		Sql:       buf.String(),
		Fragments: frags,
		Id:        node.Id,
	}
}
