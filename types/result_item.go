package types

import (
	log "github.com/bnulwh/logrus"
	"reflect"
	"strings"
)

type ResultItem struct {
	Column     string
	Type       reflect.Type
	Property   string
	PrimaryKey bool
}

//<id column="id" jdbcType="INTEGER" property="id" />
//    <result column="created_by" jdbcType="VARCHAR" property="createdBy" />
//
func parseResultItemFromXmlNode(elem xmlElement) *ResultItem {
	log.Debugf("--parse result item from: %v", ToJson(elem))
	xn := elem.Val.(xmlNode)
	bpk := strings.Compare(xn.Name, "id") == 0
	col := xn.Attrs["column"]
	tpn := xn.Attrs["jdbcType"]
	pro := xn.Attrs["property"]
	typ := ParseJdbcTypeFrom(tpn)
	return &ResultItem{
		Column:     col,
		Type:       typ,
		Property:   pro,
		PrimaryKey: bpk,
	}
}
