package types

import (
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"reflect"
	"strings"
)

type sqlFragmentType string

const (
	simpleSqlFragment  sqlFragmentType = "sql"
	ifTestSqlFragment  sqlFragmentType = "if"
	forLoopSqlFragment sqlFragmentType = "for"
	includeSqlFragment sqlFragmentType = "include"
	chooseSqlFragment  sqlFragmentType = "choose"
)

type sqlFragment struct {
	Sql     *simpleSql
	Include *sqlInclude
	IfTest  *sqlIfTest
	ForLoop *sqlForLoop
	Choose  *sqlChoose
	Type    sqlFragmentType
}

func (in *sqlFragment) prepareSqlWithSlice(m []interface{}, depth int) (string, []string) {
	log.Debugf("sql fragment [%v] generate sql with slice : %v  depth: %v", in.Type, m, depth)
	switch in.Type {
	case simpleSqlFragment:
		if in.Sql != nil {
			if len(in.Sql.Params) == 0 {
				return in.Sql.Sql, []string{}
			}
			panic("simple sql has param not replaced!!!!")
		}
	case includeSqlFragment:
		if in.Include != nil {
			return in.Include.Sql, []string{}
		}
	case ifTestSqlFragment:
		if in.IfTest != nil {
			return in.IfTest.prepareSqlWithSlice(m, depth+1)
		}
	case forLoopSqlFragment:
		if in.ForLoop != nil {
			return in.ForLoop.prepareSql(map[string]interface{}{}, m, depth+1)
		}
	}
	return "", []string{}
}

func (in *sqlFragment) generateSqlWithSlice(m []interface{}, depth int) string {
	log.Debugf("sql fragment [%v] generate sql with slice : %v  depth: %v", in.Type, m, depth)
	switch in.Type {
	case simpleSqlFragment:
		if in.Sql != nil {
			if len(in.Sql.Params) == 0 {
				return in.Sql.Sql
			}
			panic("simple sql has param not replaced!!!!")
		}
	case includeSqlFragment:
		if in.Include != nil {
			return in.Include.Sql
		}
	case ifTestSqlFragment:
		if in.IfTest != nil {
			return in.IfTest.generateSqlWithSlice(m, depth+1)
		}
	case forLoopSqlFragment:
		if in.ForLoop != nil {
			return in.ForLoop.generateSql(map[string]interface{}{}, m, depth+1)
		}
	}
	return ""
}
func (in *sqlFragment) prepareSqlWithMap(m map[string]interface{}, depth int) (string, []string) {
	log.Debugf("sql fragment [%v] generate sql with map : %v  depth: %v", in.Type, m, depth)
	switch in.Type {
	case simpleSqlFragment:
		if in.Sql != nil {
			return in.Sql.prepareSqlWithMap(m, depth+1)
		}
	case includeSqlFragment:
		if in.Include != nil {
			return in.Include.Sql, []string{}
		}
	case ifTestSqlFragment:
		if in.IfTest != nil {
			return in.IfTest.prepareSqlWithMap(m, depth+1)
		}
	case forLoopSqlFragment:
		if in.ForLoop != nil {
			val, ok := m[buildKey(in.ForLoop.Collection)]
			if !ok {
				return "", []string{}
			}
			if reflect.TypeOf(val).Kind() == reflect.Slice {
				sval := convert2Slice(reflect.ValueOf(val))
				return in.ForLoop.prepareSql(m, sval, depth+1)
			}
		}
	case chooseSqlFragment:
		if in.Choose != nil {
			return in.Choose.prepareSqlWithMap(m, depth+1)
		}
	}
	return "", []string{}
}

func (in *sqlFragment) generateSqlWithMap(m map[string]interface{}, depth int) string {
	log.Debugf("sql fragment [%v] generate sql with map : %v  depth: %v", in.Type, m, depth)
	switch in.Type {
	case simpleSqlFragment:
		if in.Sql != nil {
			return in.Sql.generateSqlWithMap(m, depth+1)
		}
	case includeSqlFragment:
		if in.Include != nil {
			return in.Include.Sql
		}
	case ifTestSqlFragment:
		if in.IfTest != nil {
			return in.IfTest.generateSqlWithMap(m, depth+1)
		}
	case forLoopSqlFragment:
		if in.ForLoop != nil {
			val, ok := m[buildKey(in.ForLoop.Collection)]
			if !ok {
				return ""
			}
			if reflect.TypeOf(val).Kind() == reflect.Slice {
				sval := convert2Slice(reflect.ValueOf(val))
				return in.ForLoop.generateSql(m, sval, depth+1)
			}
		}
	case chooseSqlFragment:
		if in.Choose != nil {
			return in.Choose.generateSqlWithMap(m, depth+1)
		}
	}
	return ""
}
func (in *sqlFragment) prepareSqlWithParam(m interface{}) (string, []string) {
	log.Debugf("sql fragment [%v] prepare sql with param : %v  ", in.Type, m)
	switch in.Type {
	case simpleSqlFragment:
		if in.Sql != nil {
			return in.Sql.prepareSqlWithParam(m)
		}
	case includeSqlFragment:
		if in.Include != nil {
			return in.Include.Sql, []string{}
		}
	case ifTestSqlFragment:
		if in.IfTest != nil {
			return in.IfTest.prepareSqlWithParam(m)
		}
	case forLoopSqlFragment:
		return "", []string{}
	case chooseSqlFragment:
		if in.Choose != nil {
			return in.Choose.Otherwise.prepareSqlWithParam(m)
		}
	}
	return "", []string{}
}
func (in *sqlFragment) generateSqlWithParam(m interface{}) string {
	log.Debugf("sql fragment [%v] generate sql with param : %v  ", in.Type, m)
	switch in.Type {
	case simpleSqlFragment:
		if in.Sql != nil {
			return in.Sql.generateSqlWithParam(m)
		}
	case includeSqlFragment:
		if in.Include != nil {
			return in.Include.Sql
		}
	case ifTestSqlFragment:
		if in.IfTest != nil {
			return in.IfTest.generateSqlWithParam(m)
		}
	case forLoopSqlFragment:
		return ""
	case chooseSqlFragment:
		if in.Choose != nil {
			return in.Choose.Otherwise.generateSqlWithParam(m)
		}
	}
	return ""
}

func (in *sqlFragment) generateSqlWithoutParam() string {
	log.Debugf("sql fragment [%v] generate sql without param ", in.Type)
	switch in.Type {
	case simpleSqlFragment:
		if in.Sql != nil {
			return in.Sql.Sql
		}
	case includeSqlFragment:
		if in.Include != nil {
			return in.Include.Sql
		}
	case ifTestSqlFragment, forLoopSqlFragment:
		return ""
	case chooseSqlFragment:
		if in.Choose != nil {
			return in.Choose.Otherwise.Sql
		}
	}
	return ""
}

func parsesqlFragmentFromXmlElement(elem xmlElement, sns map[string]*SqlElement) (*sqlFragment, error) {
	log.Debugf("++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
	log.Debugf("++begin parse sql fragment from element: %v", ToJson(elem))
	defer log.Debugf("++finish parse sql fragment from element: %v", ToJson(elem))
	switch elem.ElementType {
	case xmlTextElem:
		return &sqlFragment{
			Sql:     parseSimpleSqlFromText(elem.Val.(string)),
			ForLoop: nil,
			IfTest:  nil,
			Include: nil,
			Choose:  nil,
			Type:    simpleSqlFragment,
		}, nil
	case xmlNodeElem:
		xn := elem.Val.(xmlNode)
		return parsesqlFragmentFromXmlNode(xn, sns)
	}
	return nil, fmt.Errorf("wrong type of element type %v", elem.ElementType)
}

func parsesqlFragmentFromXmlNode(node xmlNode, sns map[string]*SqlElement) (*sqlFragment, error) {
	switch strings.ToLower(node.Name) {
	case "if":
		return parseSqlIfTestFromXmlNode(node.Attrs, node.Elements)
	case "include":
		return parseSqlIncludeFromXmlNode(node.Attrs, sns)
	case "for", "foreach":
		return parseSqlForLoopFromXmlNode(node.Attrs, node.Elements)
	case "choose":
		return parseSqlChooseFromXmlNode(node.Elements)
	}
	return nil, fmt.Errorf("not support sql text type %v", node.Name)
}
