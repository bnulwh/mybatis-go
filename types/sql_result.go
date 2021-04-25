package types

import (
	"encoding/xml"
	log "github.com/bnulwh/logrus"
	"reflect"
)

type SqlResult struct {
	ResultM *ResultMap
	ResultT reflect.Type
}

func parseSqlResultFromXmlAttrs(attrs map[string]xml.Attr, rms map[string]*ResultMap) SqlResult {
	log.Debugf("--begin parse sql result from: %v", ToJson(attrs))
	defer log.Debugf("--finish parse sql result from: %v", ToJson(attrs))
	attr, ok := attrs["resultMap"]
	if ok {
		return parseSqlResult0(attr.Value, rms)
	}
	attr, ok = attrs["resultType"]
	if ok {
		return parseSqlResult1(attr.Value)
	}
	return SqlResult{
		ResultM: nil,
		ResultT: reflect.TypeOf(int64(0)),
	}
}

func parseSqlResult0(val string, rms map[string]*ResultMap) SqlResult {
	r, ok := rms[buildKey(val)]
	if ok {
		return SqlResult{
			ResultM: r,
			ResultT: reflect.TypeOf(-1),
		}
	}
	log.Warnf("can not find result map: %v", val)
	return SqlResult{
		ResultM: nil,
		ResultT: reflect.TypeOf(map[string]interface{}{}),
	}
}
func parseSqlResult1(val string) SqlResult {
	return SqlResult{
		ResultM: nil,
		ResultT: parseResultTypeFrom(val),
	}
}
