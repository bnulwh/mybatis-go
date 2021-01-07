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
func (in *SqlFunction) GenerateSQL(args ...interface{}) (string,  []interface{}, error) {
	log.Debug("========================================")
	log.Debug("sql function %v begin generate sql args: %v", in.Id, args)
	defer log.Debug("sql function %v finish  generate sql", in.Id)
	err := in.Param.validParam(args)
	if err != nil {
		log.Warn("valid param failed: %v", err)
		return "",[]interface{}{}, err
	}
	if !in.Param.Need {
		return in.generateSqlWithoutParam(),[]interface{}{}, nil
	}
	switch in.Param.Type {
	case BaseSqlParam:
		return in.generateSqlWithParam(args[0]),[]interface{}{}, nil
	case SliceSqlParam:
		smp := convert2Slice(reflect.Indirect(reflect.ValueOf(args)))
		return in.generateSqlWithSlice(smp),[]interface{}{}, nil
	}
	nmp := convert2Map(reflect.Indirect(reflect.ValueOf(args[0])))
	return in.generateSqlWithMap(nmp),[]interface{}{}, nil
}
func (in *SqlFunction) PrepareSQL(args ...interface{}) (string, []interface{}, error) {
	log.Debug("========================================")
	log.Debug("sql function %v begin prepare sql args: %v", in.Id, args)
	defer log.Debug("sql function %v finish  prepare sql", in.Id)
	err := in.Param.validParam(args)
	if err != nil {
		log.Warn("valid param failed: %v", err)
		return "", nil, err
	}
	if !in.Param.Need {
		return in.generateSqlWithoutParam(), []interface{}{}, nil
	}
	switch in.Param.Type {
	case BaseSqlParam:
		sqlstr, results := in.prepareSqlWithParam(args[0])
		return sqlstr, results, nil
	case SliceSqlParam:
		smp := convert2Slice(reflect.Indirect(reflect.ValueOf(args)))
		sqlstr, results := in.prepareSqlWithSlice(smp)
		return sqlstr, results, nil
	}
	nmp := convert2Map(reflect.Indirect(reflect.ValueOf(args[0])))
	sqlstr, results := in.prepareSqlWithMap(nmp)
	return sqlstr, results, nil

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
func (in *SqlFunction) prepareSqlWithMap(m map[string]interface{}) (string, []interface{}) {
	log.Debug("sql function %v generate sql with map: %v", in.Id, m)
	var buf bytes.Buffer
	var results []interface{}
	for _, item := range in.Items {
		buf.WriteString(" ")
		sqlstr, items := item.prepareSqlWithMap(m, 0)
		buf.WriteString(sqlstr)
		results = append(results, items...)
	}
	return buf.String(), results
}
func (in *SqlFunction) generateSqlWithMap(m map[string]interface{}) string {
	log.Debug("sql function %v prepare sql with map: %v", in.Id, m)
	var buf bytes.Buffer
	for _, item := range in.Items {
		buf.WriteString(" ")
		buf.WriteString(item.generateSqlWithMap(m, 0))
	}
	return buf.String()
}
func (in *SqlFunction) prepareSqlWithSlice(m []interface{}) (string, []interface{}) {
	log.Debug("sql function %v prepare sql with slice: %v", in.Id, m)
	var buf bytes.Buffer
	var results []interface{}
	for _, item := range in.Items {
		buf.WriteString(" ")
		sqlstr, items := item.prepareSqlWithSlice(m, 0)
		buf.WriteString(sqlstr)
		results = append(results, items...)
	}
	return buf.String(), results
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
func (in *SqlFunction) prepareSqlWithParam(m interface{}) (string, []interface{}) {
	log.Debug("sql function %v generate sql with param: %v", in.Id, m)
	var buf bytes.Buffer
	var results []interface{}
	for _, item := range in.Items {
		buf.WriteString(" ")
		sqlstr, items := item.prepareSqlWithParam(m)
		buf.WriteString(sqlstr)
		results = append(results, items...)
	}
	return buf.String(), results
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
