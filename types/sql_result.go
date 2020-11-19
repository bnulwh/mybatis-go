package types

import(
	"bytes"
	"encoding/xml"
	"fmt"
	log "github.com/astaxie/beego/logs"
	"reflect"
	"strings"
	"time"
)

type SqlResult struct{
	ResultM *ResultMap
	ResultT reflect.Type
}

func parseSqlResultFromXmlAttrs(attrs map[string]xml.Attr,rms map[string]*ResultMap) SqlResult{
	log.Info("begin parse sql result from: %v",attrs)
	attr,ok := attrs["resultMap"]
	if ok{
		return parseSqlResult0(attr.Value,rms)
	}
	attr,ok = attrs["resultType"]
	if ok{
		return parseSqlResult1(attr.Value)
	}
	return SqlResult{
		ResultM: nil,
		ResultT: reflect.TypeOf(0),
	}
}

func parseSqlResult0(val string,rms map[string]*ResultMap) SqlResult{
	r,ok := rms[buildKey(val)]
	if ok{
		return SqlResult{
			ResultM: r,
			ResultT: reflect.TypeOf(0),
		}
	}
	log.Warn("can not find result map: %v",val)
	return SqlResult{
		ResultM: nil,
		ResultT: reflect.TypeOf(map[string]interface{}{})
	}
}
func parseSqlResult1(val string{
	return SqlResult{
		ResultM: nil,
		ResultT: parseResultTypeFrom(val)
	}
}
