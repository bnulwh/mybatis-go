package types

import (
	log "github.com/sirupsen/logrus"
	"reflect"
	"strings"
)

type ResultItem struct {
	Column     string
	Type       reflect.Type
	Property   string
	PrimaryKey bool
}

func parseResultItemFromXmlNode(elem xmlElement) *ResultItem {
	log.Debugf("--parse result item from: %v",ToJson(elem))
	xn := elem.Val.(xmlNode)
	bpk := strings.Compare(xn.Name, "id") == 0
	col := xn.Attrs["column"].Value
	tpn := xn.Attrs["jdbcType"].Value
	pro := xn.Attrs["property"].Value
	typ := parseJdbcTypeFrom(tpn)
	return &ResultItem{
		Column:     col,
		Type:       typ,
		Property:   pro,
		PrimaryKey: bpk,
	}
}
