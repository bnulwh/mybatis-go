package types

import(
	"bytes"
	"fmt"
	log "github.com/astaxie/beego/logs"
	"reflect"
	"strings"
	"time"
)

type ResultItem struct{
	Column string
	Type reflect.Type
	Property string
	PrimaryKey bool
}

func parseResultItemFromXmlNode(elem xmlElement) *ResultItem{
	xn := elem.Val.(xmlNode)
	bpk := strings.Compare(xn.Name,"id")==0
	col := xn.Attrs["column"].Value
	tpn := xn.Attrs["jdbcType"].Value
	pro := xn.Attrs["property"].Value
	typ := parseTypeFrom(tpn)
	return &ResultItem{
		Column: col,
		Type: typ,
		Property: pro,
		PrimaryKey: bpk,
	}
}