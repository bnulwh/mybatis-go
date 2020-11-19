package types

import(
	"bytes"
	"encoding/xml"
	"fmt"
	log "github.com/astaxie/beego/logs"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type SqlFunction struct{
	Id string
	Type SqlFunctionType
	Param SqlParam
	Result SqlResult
	Items []*SqlText
}

func (in *SqlFunction) GenerateSQL(mapper *SqlMapper,m interface{}) string{
	if !in.Param.Need{
		return in.generateSqlWithoutParam(mapper)
	}
	err := validParams(m)
	if err !=nil{
		log.Error("valid param %v failed: %v",m,err)
		return ""
	}
	switch in.Param.Type.Kind(){
	case reflect.String:
		switch m.(type){
		case string:
			return in.generateSqlWithString(mapper,m.(string))
		}
		break
	case reflect.Int,reflect.Int8,reflect.Int16,reflect.Int32,reflect.Int64:
		switch m.(type){
		case int:
			return in.generateSqlWithInt(mapper,m.(int))
		case int8:
			return in.generateSqlWithInt(mapper,int(m.(int8)))
		case int16:
			return in.generateSqlWithInt(mapper,int(m.(int16)))
		case int32:
			return in.generateSqlWithInt(mapper,int(m.(int32)))
		case int64:
			return in.generateSqlWithInt(mapper,int(m.(int64)))						
		}
		break
	}
} 