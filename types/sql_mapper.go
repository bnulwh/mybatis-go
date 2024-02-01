package types

import (
	"bytes"
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type SqlMapper struct {
	Filename       string
	Namespace      string
	Maps           []*ResultMap
	SqlNodes       []*SqlElement
	Functions      []*SqlFunction
	NamedMaps      map[string]*ResultMap
	NamedSqls      map[string]*SqlElement
	NamedFunctions map[string]*SqlFunction
}

func (in *SqlMapper) GenerateFiles(dir, pkg string) {
	for _, mp := range in.Maps {
		err := mp.GenerateFile(dir, pkg)
		if err != nil {
			log.Warnf("result map %v generate file failed: %v", mp.TypeName, err)
		}
	}
	err := in.generateMapperFile(dir, pkg)
	if err != nil {
		log.Warnf("mapper %v generate mapper file failed: %v", in.Namespace, err)
	}
}

func (in *SqlMapper) generateMapperFile(dir, pkg string) error {
	sname := GetShortName(in.Namespace)
	filename := filepath.Join(dir, fmt.Sprintf("%s.go", sname))
	log.Infof("generate mapper file: %v", filename)
	bts := in.generateContent(pkg)
	return ioutil.WriteFile(filename, bts, 0640)
}

func (in *SqlMapper) generateContent(pkg string) []byte {
	var buf bytes.Buffer
	sname := GetShortName(in.Namespace)
	buf.WriteString(fmt.Sprintf("package %s\n\n", pkg))
	buf.WriteString("import (\n")
	buf.WriteString("\t\"github.com/bnulwh/mybatis-go/orm\"\n")
	buf.WriteString("\t\"sync\"\n")
	buf.WriteString(") \n\n")
	buf.WriteString(fmt.Sprintf("type %s struct {\n", sname))
	buf.WriteString("\torm.BaseMapper\n")
	for _, item := range in.Functions {
		buf.WriteString(item.generateDefine())
	}
	buf.WriteString("}\n\n")
	buf.WriteString("var (\n")
	buf.WriteString(fmt.Sprintf("\tg%s  *%s\n", sname, sname))
	buf.WriteString(fmt.Sprintf("\tg%sOnce  sync.Once\n", sname))
	buf.WriteString(")\n\n")
	buf.WriteString("func init() {\n")
	buf.WriteString(fmt.Sprintf("\torm.RegisterMapper(new(%s))\n", sname))
	buf.WriteString("}\n\n")
	buf.WriteString(fmt.Sprintf("func Get%s() *%s{\n", sname, sname))
	buf.WriteString(fmt.Sprintf("\tg%sOnce.Do(func() {\n", sname))
	buf.WriteString(fmt.Sprintf("\t\tg%s = orm.NewMapperPtr(\"%s\").(*%s)\n", sname, sname, sname))
	buf.WriteString(fmt.Sprintf("\t})\n"))
	buf.WriteString(fmt.Sprintf("\treturn g%s\n", sname))
	buf.WriteString("}\n\n")
	return buf.Bytes()
}

func loadMapper(filename string) *SqlMapper {
	log.Debugf("--------------------------------------------------")
	log.Debugf("begin load mapper from %v", filename)
	defer log.Debugf("finish load mapper from %v", filename)
	node, err := parseXmlFile(filename)
	if err != nil {
		log.Errorf("parse xml file %v failed: %v", filename, err)
		return nil
	}
	if node == nil {
		log.Warnf("parse xml file %v failed", filename)
		return nil
	}
	namespace := node.Attrs["namespace"]
	mps := filterResultMap(node.Elements)
	nms := makeNamedMap(mps)
	sns := filterSqlElement(node.Elements)
	nss := makeNamedSql(sns)
	items := filterSqlFunction(node.Elements, nms, nss, namespace)
	nis := makeNamedFuntion(items)
	return &SqlMapper{
		Filename:       filename,
		Namespace:      namespace,
		Maps:           mps,
		SqlNodes:       sns,
		Functions:      items,
		NamedMaps:      nms,
		NamedSqls:      nss,
		NamedFunctions: nis,
	}
}

func filterResultMap(elems []xmlElement) []*ResultMap {
	var mps []*ResultMap
	for _, elem := range elems {
		switch elem.ElementType {
		case xmlNodeElem:
			xn := elem.Val.(xmlNode)
			switch strings.ToLower(xn.Name) {
			case "resultmap":
				mps = append(mps, parseResultMapFromXmlNode(xn))
			}
		}
	}
	return mps
}
func filterSqlElement(elems []xmlElement) []*SqlElement {
	var ses []*SqlElement
	for _, elem := range elems {
		switch elem.ElementType {
		case xmlNodeElem:
			xn := elem.Val.(xmlNode)
			switch strings.ToLower(xn.Name) {
			case "sql":
				ses = append(ses, parseSqlElementFromXmlNode(xn))
			}
		}
	}
	return ses
}
func filterSqlFunction(elems []xmlElement, rms map[string]*ResultMap, sns map[string]*SqlElement, owner string) []*SqlFunction {
	var sfs []*SqlFunction
	for _, elem := range elems {
		switch elem.ElementType {
		case xmlNodeElem:
			xn := elem.Val.(xmlNode)
			switch strings.ToLower(xn.Name) {
			case "select", "insert", "delete", "update":
				sfs = append(sfs, parseSqlFunctionFromXmlNode(xn, rms, sns, owner))
			}
		}
	}
	return sfs
}

func makeNamedMap(mps []*ResultMap) map[string]*ResultMap {
	rmp := map[string]*ResultMap{}
	for _, m := range mps {
		rmp[m.Id] = m
		rmp[buildKey(m.Id)] = m
		rmp[strings.ToLower(m.Id)] = m
	}
	return rmp
}

func makeNamedSql(ses []*SqlElement) map[string]*SqlElement {
	nss := map[string]*SqlElement{}
	for _, se := range ses {
		nss[se.Id] = se
		nss[buildKey(se.Id)] = se
		nss[strings.ToLower(se.Id)] = se
	}
	return nss
}

func makeNamedFuntion(fs []*SqlFunction) map[string]*SqlFunction {
	fmp := map[string]*SqlFunction{}
	for _, f := range fs {
		fmp[f.Id] = f
		fmp[buildKey(f.Id)] = f
		fmp[strings.ToLower(f.Id)] = f
	}
	return fmp
}
