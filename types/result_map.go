package types

import (
	"bytes"
	"fmt"
	log "github.com/astaxie/beego/logs"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type ResultMap struct {
	Id          string
	TypeName    string
	Results     []*ResultItem
	ColumnMap   map[string]*ResultItem
	PropertyMap map[string]*ResultItem
}

func (in *ResultMap) GenerateFile(dir, pkg string) error {
	sname := GetShortName(in.TypeName)
	filename := filepath.Join(dir, fmt.Sprintf("%s.go", sname))
	log.Info("generate file %v", filename)
	bts := in.generateContent(pkg)
	return ioutil.WriteFile(filename, bts, 0640)
}
func (in *ResultMap) generateContent(pkg string) []byte {
	var buf bytes.Buffer
	sname := GetShortName(in.TypeName)
	buf.WriteString(fmt.Sprintf("package %s\n\n", pkg))
	buf.WriteString("import(\n")
	buf.WriteString("\t\"github.com/bnulwh/mybatis-go/orm\"\n")
	if in.hasTimeItem() {
		buf.WriteString("\t\"time\"\n")
	}
	buf.WriteString(")\n\n")
	buf.WriteString(fmt.Sprintf("type %s struct{\n", sname))
	for _, item := range in.Results {
		buf.WriteString(fmt.Sprintf("\t%s \t%s\n", UpperFirst(item.Property), item.Type.String()))
	}
	buf.WriteString("}\n\n")
	buf.WriteString("func init(){\n")
	buf.WriteString(fmt.Sprintf("\torm.RegisterModel(new(%s))\n", sname))
	buf.WriteString("}\n\n")
	return buf.Bytes()
}
func (in *ResultMap) hasTimeItem() bool {
	for _, item := range in.Results {
		switch item.Type.String() {
		case "time.Time":
			return true
		}
	}
	return false
}

func parseResultMapFromXmlNode(node xmlNode) *ResultMap {
	id := node.Id
	tn := node.Attrs["type"].Value
	var arr []*ResultItem
	for _, elem := range node.Elements {
		arr = append(arr, parseResultItemFromXmlNode(elem))
	}
	cmp := makeColumnMap(arr)
	pmp := makePropertyMap(arr)
	return &ResultMap{
		Id:          id,
		TypeName:    tn,
		Results:     arr,
		ColumnMap:   cmp,
		PropertyMap: pmp,
	}
}
func makeColumnMap(items []*ResultItem) map[string]*ResultItem {
	mp := map[string]*ResultItem{}
	for _, item := range items {
		mp[item.Column] = item
		mp[strings.ToLower(item.Column)] = item
	}
	return mp
}
func makePropertyMap(items []*ResultItem) map[string]*ResultItem {
	mp := map[string]*ResultItem{}
	for _, item := range items {
		mp[item.Property] = item
		mp[strings.ToLower(item.Property)] = item
	}
	return mp
}
