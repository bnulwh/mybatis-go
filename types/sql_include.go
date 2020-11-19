package types

import(
	"bytes"
	"encoding/xml"
	"fmt"
	log "github.com/astaxie/beego/logs"
	"reflect"
	"strings"
	"time"
)

type SqlInclude struct{
	Sql string
	Refid string
}

func parseSqlIncludeFromXmlNode(attrs map[string]xml.Attr,sns map[string]*SqlElement) *SqlText{
	attr,ok := attrs["refid"]
	if ok{
		sn,ok := sns[attr.Value]
		if ok{
			return &SqlText{
				Include: &SqlInclude{
					Sql: sn.Sql,
					Refid: attr.Value,
				},
				Sql: nil,
				IfTest: nil,
				ForLoop: nil,
				Choose: nil,
				Type: IncludeSQL,
			}
		}
		panic(fmt.Sprintf("not found sql id=%v",attr.Value))
	}
	panic("not found refid")
}