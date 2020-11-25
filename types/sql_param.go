package types

import (
	"encoding/xml"
	"fmt"
	log "github.com/astaxie/beego/logs"
	"reflect"
	"strings"
)

type SqlParamType string

const (
	BaseParam   SqlParamType = "base"
	SliceParam  SqlParamType = "slice"
	MapParam    SqlParamType = "map"
	StructParam SqlParamType = "struct"
)

type SqlParam struct {
	Name     string
	TypeName string
	Type     SqlParamType
	Need     bool
}

func parseSqlParamFromXmlAttrs(attrs map[string]xml.Attr) SqlParam {
	log.Info("parse sql param from: %v", attrs)
	attr, ok := attrs["parameterType"]
	if !ok {
		return SqlParam{Need: false}
	}
	val := attr.Value
	tn := GetShortName(val)
	return SqlParam{
		Name:     val,
		TypeName: tn,
		Type:     parseSqlParamTypeFrom(tn),
		Need:     true,
	}
}

func (in *SqlParam) validParam(args []interface{}) error {
	if !in.Need {
		return nil
	}
	switch in.Type {
	case BaseParam:
		if len(args) == 0 {
			return fmt.Errorf("need param, type: %v", in.TypeName)
		}
		val := reflect.ValueOf(args[0])
		typ := reflect.Indirect(val).Type()
		switch typ.String() {
		case "string",
			"bool",
			"int", "int8", "int16", "int32",
			"uint", "uint8", "uint16", "uint32",
			"int64", "uint64",
			"float32", "float64",
			"time.Time":
			return nil
		}
		return fmt.Errorf("not support param type: %v ,need type: %v", typ, in.TypeName)
	case SliceParam:
		return nil
	case MapParam, StructParam:
		if len(args) == 0 {
			return fmt.Errorf("need param, type: %v", in.TypeName)
		}
		val := reflect.ValueOf(args[0])
		typ := reflect.Indirect(val).Type()
		switch typ.String() {
		case "string",
			"bool",
			"int", "int8", "int16", "int32",
			"uint", "uint8", "uint16", "uint32",
			"int64", "uint64",
			"float32", "float64",
			"time.Time":
			return fmt.Errorf("not support param type: %v ,need type: %v", typ, in.TypeName)
		}
		if typ.Kind() == reflect.Ptr {
			return fmt.Errorf("use two references to the struce %v", typ)
		}
	}
	return nil
}

func parseSqlParamTypeFrom(tn string) SqlParamType {
	switch strings.ToUpper(GetShortName(tn)) {
	case "STRING", "VARCHAR",
	    "BOOLEAN","BOOL",
		"INT", "INTEGER", "INT8", "INT16", "INT32", "INT64",
		"UINT", "UINT8", "UINT16", "UINT32", "UINT64",
		"FLOAT", "FLOAT32", "FLOAT64", "DOUBLE",
		"TIME", "TIMESTAMP":
		return BaseParam
	case "LIST", "ARRAY", "ARRAYLIST", "SLICE":
		return SliceParam
	case "MAP", "HASHMAP", "TREEMAP":
		return MapParam
	}
	return StructParam
}
