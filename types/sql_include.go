package types

import (
	"encoding/xml"
	"fmt"
	log "github.com/sirupsen/logrus"
)

type sqlInclude struct {
	Sql   string
	Refid string
}

func parseSqlIncludeFromXmlNode(attrs map[string]xml.Attr, sns map[string]*SqlElement) *sqlFragment {
	log.Debugf("parse sql include from: %v", ToJson(attrs))
	attr, ok := attrs["refid"]
	if ok {
		sn, ok := sns[attr.Value]
		if ok {
			return &sqlFragment{
				Include: &sqlInclude{
					Sql:   sn.Sql,
					Refid: attr.Value,
				},
				Sql:     nil,
				IfTest:  nil,
				ForLoop: nil,
				Choose:  nil,
				Type:    includeSqlFragment,
			}
		}
		panic(fmt.Sprintf("not found sql id=%v", attr.Value))
	}
	panic("not found refid")
}
