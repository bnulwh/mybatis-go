package types

import (
	log "github.com/astaxie/beego/logs"
)

type SqlElement struct {
	Sql string
	Id  string
}

func parseSqlElementFromXmlNode(node xmlNode) *SqlElement {
	log.Debug("begin parse sql element from: %v", ToJson(node))
	defer log.Debug("finish parse sql element from: %v", ToJson(node))
	return &SqlElement{
		Sql: node.Elements[0].Val.(string),
		Id:  node.Id,
	}
}
