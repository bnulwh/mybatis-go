package types

import (
	"bytes"
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"reflect"
	"sync/atomic"
	"time"
)

type SqlFunction struct {
	Id               string
	Owner            string
	Type             SqlFunctionType
	Param            SqlParam
	Result           SqlResult
	Items            []*sqlFragment
	TotalUsage       int64
	FailedUsage      int64
	TotalDuration    int64
	MaxDuration      int64
	MinDuration      int64
	GenerateCount    int64
	GenerateDuration int64
}

func (in *SqlFunction) UpdateUsage(start time.Time, success bool) {
	atomic.AddInt64(&in.TotalUsage, 1)
	if !success {
		atomic.AddInt64(&in.FailedUsage, 1)
	}
	d := time.Since(start).Milliseconds()
	atomic.AddInt64(&in.TotalDuration, d)
	if d > in.MaxDuration {
		atomic.SwapInt64(&in.MaxDuration, d)
	}
	if d < in.MinDuration {
		atomic.SwapInt64(&in.MinDuration, d)
	}
}
func (in *SqlFunction) String() string {
	return fmt.Sprintf("%v,%v,%v,%v,%v,%v,%v,%v,%v", in.Owner, in.Id,
		in.TotalUsage-in.FailedUsage, in.TotalUsage, in.MinDuration, in.MaxDuration, in.TotalDuration,
		in.GenerateCount, in.GenerateDuration)
}

func (in *SqlFunction) updateGenerate(start time.Time) {
	atomic.AddInt64(&in.GenerateCount, 1)
	atomic.AddInt64(&in.GenerateDuration, time.Since(start).Milliseconds())
}

// GenerateSQL
func (in *SqlFunction) GenerateSQL(args ...interface{}) (string, []interface{}, error) {
	log.Debugf("========================================")
	log.Debugf("sql function %v begin generate sql args: %v", in.Id, args)
	start := time.Now()
	defer log.Debugf("sql function %v finish  generate sql", in.Id)
	defer in.updateGenerate(start)
	err := in.Param.validParam(args)
	if err != nil {
		log.Warnf("valid param failed: %v", err)
		return "", []interface{}{}, err
	}
	if !in.Param.Need {
		return in.generateSqlWithoutParam(), []interface{}{}, nil
	}
	switch in.Param.Type {
	case BaseSqlParam:
		return in.generateSqlWithParam(args[0]), []interface{}{}, nil
	case SliceSqlParam:
		smp := convert2Slice(reflect.Indirect(reflect.ValueOf(args)))
		return in.generateSqlWithSlice(smp), []interface{}{}, nil
	}
	nmp := convert2Map(reflect.Indirect(reflect.ValueOf(args[0])))
	return in.generateSqlWithMap(nmp), []interface{}{}, nil
}
func (in *SqlFunction) PrepareSQL(args ...interface{}) (string, []string, error) {
	log.Debugf("========================================")
	log.Debugf("sql function %v begin prepare sql args: %v", in.Id, args)
	defer log.Debugf("sql function %v finish  prepare sql", in.Id)
	err := in.Param.validParam(args)
	if err != nil {
		log.Warnf("valid param failed: %v", err)
		return "", nil, err
	}
	if !in.Param.Need {
		return in.generateSqlWithoutParam(), []string{}, nil
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
			buf.WriteString("models.")
			buf.WriteString(GetShortName(in.Result.ResultM.TypeName))
		} else {
			buf.WriteString(toGolangType(in.Result.ResultT.String()))
		}
		buf.WriteString(",error")
	}
	buf.WriteString(")\n")
	return buf.String()
}
func (in *SqlFunction) prepareSqlWithMap(m map[string]interface{}) (string, []string) {
	log.Debugf("sql function %v generate sql with map: %v", in.Id, m)
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Items {
		buf.WriteString(" ")
		sqlstr, items := item.prepareSqlWithMap(m, 0)
		buf.WriteString(sqlstr)
		results = append(results, items...)
	}
	return buf.String(), results
}
func (in *SqlFunction) generateSqlWithMap(m map[string]interface{}) string {
	log.Debugf("sql function %v prepare sql with map: %v", in.Id, m)
	var buf bytes.Buffer
	for _, item := range in.Items {
		buf.WriteString(" ")
		buf.WriteString(item.generateSqlWithMap(m, 0))
	}
	return buf.String()
}
func (in *SqlFunction) prepareSqlWithSlice(m []interface{}) (string, []string) {
	log.Debugf("sql function %v prepare sql with slice: %v", in.Id, m)
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Items {
		buf.WriteString(" ")
		sqlstr, items := item.prepareSqlWithSlice(m, 0)
		buf.WriteString(sqlstr)
		results = append(results, items...)
	}
	return buf.String(), results
}
func (in *SqlFunction) generateSqlWithSlice(m []interface{}) string {
	log.Debugf("sql function %v generate sql with slice: %v", in.Id, m)
	var buf bytes.Buffer
	for _, item := range in.Items {
		buf.WriteString(" ")
		buf.WriteString(item.generateSqlWithSlice(m, 0))
	}
	return buf.String()
}
func (in *SqlFunction) prepareSqlWithParam(m interface{}) (string, []string) {
	log.Debugf("sql function %v generate sql with param: %v", in.Id, m)
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Items {
		buf.WriteString(" ")
		sqlstr, items := item.prepareSqlWithParam(m)
		buf.WriteString(sqlstr)
		results = append(results, items...)
	}
	return buf.String(), results
}
func (in *SqlFunction) generateSqlWithParam(m interface{}) string {
	log.Debugf("sql function %v generate sql with param: %v", in.Id, m)
	var buf bytes.Buffer
	for _, item := range in.Items {
		buf.WriteString(" ")
		buf.WriteString(item.generateSqlWithParam(m))
	}
	return buf.String()
}

func (in *SqlFunction) generateSqlWithoutParam() string {
	log.Debugf("sql function %v generate sql without param", in.Id)
	var buf bytes.Buffer
	for _, item := range in.Items {
		buf.WriteString(" ")
		buf.WriteString(item.generateSqlWithoutParam())
	}
	return buf.String()
}

func parseSqlFunctionFromXmlNode(node xmlNode, rms map[string]*ResultMap, sns map[string]*SqlElement, owner string) *SqlFunction {
	log.Debugf("begin parse sql function from %v %v", node.Id, node.Name)
	defer log.Debugf("finish parse sql function from %v %v", node.Id, node.Name)
	tp := parseSqlFunctionType(node.Name)
	return &SqlFunction{
		Id:               node.Id,
		Owner:            owner,
		Type:             tp,
		Param:            parseSqlParamFromXmlAttrs(node.Attrs),
		Result:           parseSqlResultFromXmlAttrs(node.Attrs, rms),
		Items:            parsesqlFragmentsFromXmlElements(node.Elements, sns),
		TotalDuration:    0,
		TotalUsage:       0,
		MinDuration:      60000,
		MaxDuration:      0,
		FailedUsage:      0,
		GenerateDuration: 0,
		GenerateCount:    0,
	}
}
func parsesqlFragmentsFromXmlElements(elems []xmlElement, sns map[string]*SqlElement) []*sqlFragment {
	var sts []*sqlFragment
	for _, elem := range elems {
		st, err := parsesqlFragmentFromXmlElement(elem, sns)
		if err != nil {
			log.Errorf("parse error:%v", err)
			continue
		}
		sts = append(sts, st)
	}
	return sts
}
