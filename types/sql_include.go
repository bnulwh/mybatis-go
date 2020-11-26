package types

import (
	"encoding/xml"
	"fmt"
	"github.com/astaxie/beego/logs"
)

type SqlInclude struct {
	Sql   string
	Refid string
}

func parseSqlIncludeFromXmlNode(attrs map[string]xml.Attr, sns map[string]*SqlElement) *SqlFragment {
	logs.Debug("parse sql include from: %v",ToJson(attrs))
	attr, ok := attrs["refid"]
	if ok {
		sn, ok := sns[attr.Value]
		if ok {
			return &SqlFragment{
				Include: &SqlInclude{
					Sql:   sn.Sql,
					Refid: attr.Value,
				},
				Sql:     nil,
				IfTest:  nil,
				ForLoop: nil,
				Choose:  nil,
				Type:    IncludeSQL,
			}
		}
		panic(fmt.Sprintf("not found sql id=%v", attr.Value))
	}
	panic("not found refid")
}
