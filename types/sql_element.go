package types

import (
	"github.com/bnulwh/mybatis-go/log"
)

type SqlElement struct {
	Sql string
	Id  string
}

func parseSqlElementFromXmlNode(node xmlNode) *SqlElement {
	log.Debugf("begin parse sql element from: %v", ToJson(node))
	defer log.Debugf("finish parse sql element from: %v", ToJson(node))
	return &SqlElement{
		Sql: node.Elements[0].Val.(string),
		Id:  node.Id,
	}
}
