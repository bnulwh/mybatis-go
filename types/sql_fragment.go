package types

import(
	"fmt"
	log "github.com/astaxie/beego/logs"
	"reflect"
	"strings"
)

type SqlFragmentType string

const (
	SimpleSQL SqlFragmentType="sql"
	IfTestSQL SqlFragmentType="if"
	ForLoopSQL SqlFragmentType="for"
	IncludeSQL SqlFragmentType="include"
	ChooseSQL SqlFragmentType="choose"
)

type SqlFragment struct{
	Sql *SimpleSql
	Include *SqlInclude
	IfTest *SqlIfTest
	ForLoop *SqlForLoop
	Choose *SqlChoose
	Type SqlFragmentType
}

func (in *SqlFragment)generateSqlWithSlice(mapper* SqlMapper, m []interface{},depth int) string{
	log.Info("sql fragment generate sql with slice : %v  depth: %v",m,depth)
	switch in.Type{
	case SimpleSQL:
		if in.Sql != nil{
			if len(in.Sql.Params) ==0{
				return in.Sql.Sql
			}
			panic("simple sql has param not replaced!!!!")
		}
	case IncludeSQL:
		if in.Include != nil{
			return in.Include.Sql
		}
	case IfTestSQL:
		if in.IfTest != nil{
			return in.IfTest.generateSqlWithSlice(mapper,m,depth+1)
		}
	case ForLoopSQL:
		if in.ForLoop != nil{
			return in.ForLoop.generateSql(mapper,map[string]interface{}{},m,depth+1)
		}
	}
	return ""
}
func (in *SqlFragment)generateSqlWithMap(mapper* SqlMapper, m map[string]interface{},depth int) string{
	log.Info("sql fragment generate sql with map : %v  depth: %v",m,depth)
	switch in.Type{
	case SimpleSQL:
		if in.Sql != nil{
			return in.Sql.generateSqlWithMap(mapper,m,depth+1)
		}
	case IncludeSQL:
		if in.Include != nil{
			return in.Include.Sql
		}
	case IfTestSQL:
		if in.IfTest != nil{
			return in.IfTest.generateSqlWithMap(mapper,m,depth+1)
		}
	case ForLoopSQL:
		if in.ForLoop != nil{
			val,ok := m[buildKey(in.ForLoop.Collection)]
			if !ok{
				return ""
			}
			if reflect.TypeOf(val).Kind() == reflect.Slice{
				sval := convert2Slice(reflect.ValueOf(val))
				return in.ForLoop.generateSql(mapper,m,sval,depth+1)
			}
		}
	case ChooseSQL:
		if in.Choose!=nil{
			return in.Choose.generateSqlWithMap(mapper,m,depth+1)
		}
	}
	return ""	
}

func (in *SqlFragment)generateSqlWithParam(mapper* SqlMapper, m interface{}) string{
	log.Info("sql fragment generate sql with param : %v  ",m)
	switch in.Type{
	case SimpleSQL:
		if in.Sql != nil{
			return in.Sql.generateSqlWithParam(mapper,m)
		}
	case IncludeSQL:
		if in.Include != nil{
			return in.Include.Sql
		}
	case IfTestSQL:
		if in.IfTest != nil{
			return in.IfTest.generateSqlWithParam(mapper,m)
		}
	case ForLoopSQL:
		return ""
	case ChooseSQL:
		if in.Choose!=nil{
			return in.Choose.Otherwise.generateSqlWithParam(mapper,m)
		}
	}
	return ""	
}

func (in *SqlFragment)generateSqlWithoutParam(mapper* SqlMapper) string{
	log.Info("sql fragment generate sql without param ")
	switch in.Type{
	case SimpleSQL:
		if in.Sql != nil{
			return in.Sql.Sql
		}
	case IncludeSQL:
		if in.Include != nil{
			return in.Include.Sql
		}
	case IfTestSQL, ForLoopSQL:
		return ""
	case ChooseSQL:
		if in.Choose!=nil{
			return in.Choose.Otherwise.Sql
		}
	}
	return ""	
}

func parseSqlFragmentFromXmlElement(elem xmlElement,sns map[string]*SqlElement) *SqlFragment{
	log.Info("++begin parse sql fragment from element: %v",elem)
	defer log.Info("++finish parse sql fragment from element: %v",elem)
	switch elem.ElementType{
	case xmlTextElem:
		return &SqlFragment{
			Sql: parseSimpleSqlFromText(elem.Val.(string)),
			ForLoop: nil,
			IfTest: nil,
			Include: nil,
			Choose: nil,
			Type: SimpleSQL,
		}
	case xmlNodeElem:
		xn := elem.Val.(xmlNode)
		return parseSqlFragmentFromXmlNode(xn,sns)
	}
	panic(fmt.Sprintf("wrong type of element type %v",elem.ElementType))
}

func parseSqlFragmentFromXmlNode(node xmlNode,sns map[string]*SqlElement) *SqlFragment{
	switch strings.ToLower(node.Name){
	case "if":
		return parseSqlIfTestFromXmlNode(node.Attrs,node.Elements)
	case "include":
		return parseSqlIncludeFromXmlNode(node.Attrs,sns)
	case "for","foreach":
		return parseSqlForLoopFromXmlNode(node.Attrs,node.Elements)
	case "choose":
		return parseSqlChooseFromXmlNode(node.Attrs,node.Elements)
	}
	panic(fmt.Sprintf("not support sql text type %v",node.Name))
}