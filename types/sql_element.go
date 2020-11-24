package types

import(
	log "github.com/astaxie/beego/logs"
)

type SqlElement struct{
	Sql string
	Id string
}

func parseSqlElementFromXmlNode(node xmlNode) *SqlElement{
	log.Info("parse sql element from %v",node)
	return &SqlElement{
		Sql: node.Elements[0].Val.(string),
		Id: node.Id,
	}
}