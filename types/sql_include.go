package types

import (
	"fmt"
	log "github.com/bnulwh/logrus"
)

type sqlInclude struct {
	Sql   string
	Refid string
}

func parseSqlIncludeFromXmlNode(attrs map[string]string, sns map[string]*SqlElement) *sqlFragment {
	log.Debugf("parse sql include from: %v", ToJson(attrs))
	attr, ok := attrs["refid"]
	if ok {
		sn, ok := sns[attr]
		if ok {
			return &sqlFragment{
				Include: &sqlInclude{
					Sql:   sn.Sql,
					Refid: attr,
				},
				Sql:     nil,
				IfTest:  nil,
				ForLoop: nil,
				Choose:  nil,
				Type:    includeSqlFragment,
			}
		}
		panic(fmt.Sprintf("not found sql id=%v", attr))
	}
	panic("not found refid")
}
