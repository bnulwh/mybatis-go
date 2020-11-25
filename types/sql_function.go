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
	Items  []*SqlFragment
}

func (in *SqlFunction) GenerateSQL(mapper *SqlMapper, args []interface{}) (string, error) {
	log.Info("========================================")
	log.Info("sql function %v begin generate sql", in.Id)
	defer log.Info("sql function %v finish  generate sql", in.Id)
	err := in.Param.validParam(args)
	if err != nil {
		log.Warn("valid param failed: %v", err)
		return "", nil
	}
	if !in.Param.Need {
		return in.generateSqlWithoutParam(mapper), nil
	}
	switch in.Param.Type {
	case BaseParam:
		return in.generateSqlWithParam(mapper, args[0]), nil
	case SliceParam:
		smp := convert2Slice(reflect.Indirect(reflect.ValueOf(args)))
		return in.generateSqlWithSlice(mapper, smp), nil
	}
	nmp := convert2Map(reflect.Indirect(reflect.ValueOf(args[0])))
	return in.generateSqlWithMap(mapper, nmp), nil
}
func (in *SqlFunction) generateDefine() string {
	var buf bytes.Buffer
	buf.WriteString("\t")
	buf.WriteString(in.Id)
	buf.WriteString(" \tfunc (")
	if in.Param.Need {
		buf.WriteString(toGolangType(in.Param.TypeName))
	}
	buf.WriteString(") (")
	switch in.Type {
	case UpdateSQL, InsertSQL, DeleteSQL:
		buf.WriteString("int64,error")
	case SelectSQL:
		buf.WriteString("[]")
		if in.Result.ResultM != nil {
			buf.WriteString(GetShortName(in.Result.ResultM.Id))
		} else {
			buf.WriteString(toGolangType(in.Result.ResultT.String()))
		}
		buf.WriteString(",error")
	}
	buf.WriteString(")\n")
	return buf.String()
}
func (in *SqlFunction) generateSqlWithMap(mapper *SqlMapper, m map[string]interface{}) string {
	log.Info("sql function %v generate sql with map: %v", in.Id, m)
	var buf bytes.Buffer
	for _, item := range in.Items {
		buf.WriteString(" ")
		buf.WriteString(item.generateSqlWithMap(mapper, m, 0))
	}
	return buf.String()
}
func (in *SqlFunction) generateSqlWithSlice(mapper *SqlMapper, m []interface{}) string {
	log.Info("sql function %v generate sql with slice: %v", in.Id, m)
	var buf bytes.Buffer
	for _, item := range in.Items {
		buf.WriteString(" ")
		buf.WriteString(item.generateSqlWithSlice(mapper, m, 0))
	}
	return buf.String()
}
func (in *SqlFunction) generateSqlWithParam(mapper *SqlMapper, m interface{}) string {
	log.Info("sql function %v generate sql with param: %v", in.Id, m)
	var buf bytes.Buffer
	for _, item := range in.Items {
		buf.WriteString(" ")
		buf.WriteString(item.generateSqlWithParam(mapper, m))
	}
	return buf.String()
}

func (in *SqlFunction) generateSqlWithoutParam(mapper *SqlMapper) string {
	log.Info("sql function %v generate sql without param", in.Id)
	var buf bytes.Buffer
	for _, item := range in.Items {
		buf.WriteString(" ")
		buf.WriteString(item.generateSqlWithoutParam(mapper))
	}
	return buf.String()
}

func parseSqlFunctionFromXmlNode(node xmlNode, rms map[string]*ResultMap, sns map[string]*SqlElement) *SqlFunction {
	log.Info("--begin parse sql function from %v", node)
	defer log.Info("--finish parse sql function from %v", node)
	tp := parseSqlFunctionType(node.Name)
	return &SqlFunction{
		Type:   tp,
		Id:     node.Id,
		Param:  parseSqlParamFromXmlAttrs(node.Attrs),
		Result: parseSqlResultFromXmlAttrs(node.Attrs, rms),
		Items:  parseSqlFragmentsFromXmlElements(node.Elements, sns),
	}
}
func parseSqlFragmentsFromXmlElements(elems []xmlElement, sns map[string]*SqlElement) []*SqlFragment {
	var sts []*SqlFragment
	for _, elem := range elems {
		sts = append(sts, parseSqlFragmentFromXmlElement(elem, sns))
	}
	return sts
}
