package types

import (
	"bytes"
	log "github.com/astaxie/beego/logs"
	"reflect"
)

type SqlFunction struct {
	Id     string
	Type   SqlFunctionType
	Param  SqlParam
	Result SqlResult
	Items  []*sqlFragment
}

//GenerateSQL
func (in *SqlFunction) GenerateSQL(args ...interface{}) (string, error) {
	log.Debug("========================================")
	log.Debug("sql function %v begin generate sql args: %v", in.Id, args)
	defer log.Debug("sql function %v finish  generate sql", in.Id)
	err := in.Param.validParam(args)
	if err != nil {
		log.Warn("valid param failed: %v", err)
		return "", nil
	}
	if !in.Param.Need {
		return in.generateSqlWithoutParam(), nil
	}
	switch in.Param.Type {
	case BaseSqlParam:
		return in.generateSqlWithParam(args[0]), nil
	case SliceSqlParam:
		smp := convert2Slice(reflect.Indirect(reflect.ValueOf(args)))
		return in.generateSqlWithSlice(smp), nil
	}
	nmp := convert2Map(reflect.Indirect(reflect.ValueOf(args[0])))
	return in.generateSqlWithMap(nmp), nil
}
func (in *SqlFunction) generateDefine() string {
	var buf bytes.Buffer
	buf.WriteString("\t")
	buf.WriteString(UpperFirst(in.Id))
	buf.WriteString(" \tfunc (")
	if in.Param.Need {
		buf.WriteString(toGolangType(in.Param.TypeName))
	}
	buf.WriteString(") (")
	switch in.Type {
	case UpdateFunction, InsertFunction, DeleteFunction:
		buf.WriteString("int64,error")
	case SelectFunction:
		buf.WriteString("[]")
		if in.Result.ResultM != nil {
			buf.WriteString(GetShortName(in.Result.ResultM.TypeName))
		} else {
			buf.WriteString(toGolangType(in.Result.ResultT.String()))
		}
		buf.WriteString(",error")
	}
	buf.WriteString(")\n")
	return buf.String()
}
func (in *SqlFunction) generateSqlWithMap(m map[string]interface{}) string {
	log.Debug("sql function %v generate sql with map: %v", in.Id, m)
	var buf bytes.Buffer
	for _, item := range in.Items {
		buf.WriteString(" ")
		buf.WriteString(item.generateSqlWithMap(m, 0))
	}
	return buf.String()
}
func (in *SqlFunction) generateSqlWithSlice(m []interface{}) string {
	log.Debug("sql function %v generate sql with slice: %v", in.Id, m)
	var buf bytes.Buffer
	for _, item := range in.Items {
		buf.WriteString(" ")
		buf.WriteString(item.generateSqlWithSlice(m, 0))
	}
	return buf.String()
}
func (in *SqlFunction) generateSqlWithParam(m interface{}) string {
	log.Debug("sql function %v generate sql with param: %v", in.Id, m)
	var buf bytes.Buffer
	for _, item := range in.Items {
		buf.WriteString(" ")
		buf.WriteString(item.generateSqlWithParam(m))
	}
	return buf.String()
}

func (in *SqlFunction) generateSqlWithoutParam() string {
	log.Debug("sql function %v generate sql without param", in.Id)
	var buf bytes.Buffer
	for _, item := range in.Items {
		buf.WriteString(" ")
		buf.WriteString(item.generateSqlWithoutParam())
	}
	return buf.String()
}

func parseSqlFunctionFromXmlNode(node xmlNode, rms map[string]*ResultMap, sns map[string]*SqlElement) *SqlFunction {
	log.Debug("begin parse sql function from %v %v", node.Id, node.Name)
	defer log.Debug("finish parse sql function from %v %v", node.Id, node.Name)
	tp := parseSqlFunctionType(node.Name)
	return &SqlFunction{
		Type:   tp,
		Id:     node.Id,
		Param:  parseSqlParamFromXmlAttrs(node.Attrs),
		Result: parseSqlResultFromXmlAttrs(node.Attrs, rms),
		Items:  parsesqlFragmentsFromXmlElements(node.Elements, sns),
	}
}
func parsesqlFragmentsFromXmlElements(elems []xmlElement, sns map[string]*SqlElement) []*sqlFragment {
	var sts []*sqlFragment
	for _, elem := range elems {
		sts = append(sts, parsesqlFragmentFromXmlElement(elem, sns))
	}
	return sts
}
