package types

import (
	"strings"

	"github.com/bnulwh/mybatis-go/log"
)

type SqlElement struct {
	Sql       string
	Fragments []*sqlFragment
	Id        string
}

func parseSqlElementFromXmlNode(node xmlNode) *SqlElement {
	log.Debugf("begin parse sql element from: %v", ToJson(node))
	defer log.Debugf("finish parse sql element from: %v", ToJson(node))
	var buf strings.Builder
	var frags []*sqlFragment
	for _, elem := range node.Elements {
		switch elem.ElementType {
		case xmlTextElem:
			buf.WriteString(elem.Val.(string))
		case xmlNodeElem:
			// 嵌套标签（如 <where>/<if>/<foreach> 等）解析为片段，供 <include> 引用时渲染
			st, err := parsesqlFragmentFromXmlElement(elem, nil)
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
