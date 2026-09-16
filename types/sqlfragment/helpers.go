package sqlfragment

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/bnulwh/mybatis-go/log"
)

// 本文件是 types 包共享小工具的私有副本（P0-3a 拆分决策）：
// sqlfragment 不 import types（依赖方向 types → sqlfragment 单向），
// 以 ~40 行重复换取零交叉 churn。types 侧原版保持不动。

// buildKey 参数键规范化：小写 + 去首尾空白。
func buildKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

// convert2Slice 将反射值展开为 interface 切片。
func convert2Slice(val reflect.Value) []interface{} {
	var ns []interface{}
	for i := 0; i < val.Len(); i++ {
		item := val.Index(i)
		ns = append(ns, reflect.Indirect(item).Interface())
	}
	return ns
}

// toJson 序列化为 JSON 字符串（失败返回空串），仅用于调试日志。
func toJson(v interface{}) string {
	dt, err := json.Marshal(v)
	if err != nil {
		fmt.Printf("to json failed: %v", err)
		return ""
	}
	return string(dt)
}

// getShortName 取点号分隔的短名。
func getShortName(name string) string {
	pos := strings.LastIndex(name, ".")
	if pos > 0 {
		return name[pos+1:]
	}
	return name
}

// getJdbcTypePart 截取 jdbcType 的类型名部分（去掉括号内的长度等修饰）。
func getJdbcTypePart(tps string) string {
	arr := strings.Split(tps, " ")
	ret := arr[0]
	idx := strings.Index(ret, "(")
	if idx > 0 {
		ret = tps[0:idx]
	}
	return strings.TrimSpace(ret)
}

// parseJdbcTypeFrom 按 jdbcType 名称推断 Go 类型（占位符 #{x,jdbcType=...} 参数类型）。
func parseJdbcTypeFrom(tps string) reflect.Type {
	tps = getJdbcTypePart(tps)
	switch strings.ToUpper(getShortName(tps)) {
	case "VARCHAR", "STRING", "LONGVARCHAR", "TEXT", "TINYTEXT", "CHAR", "MEDIUMTEXT",
		"BLOB", "LONGBLOB", "CHARACTER":
		return reflect.TypeOf("")
	case "TIMESTAMP", "TIME", "DATETIME":
		return reflect.TypeOf(time.Now())
	case "INTEGER", "INT", "TINYINT", "SMALLINT":
		return reflect.TypeOf(0)
	case "LONG", "BIGINT":
		return reflect.TypeOf(int64(0))
	case "BOOLEAN", "BIT", "BOOL", "ENUM":
		return reflect.TypeOf(true)
	case "DOUBLE", "FLOAT", "NUMERIC":
		return reflect.TypeOf(0.0)
	default:
		log.Warnf("unsupport jdbc type to parse: %v", tps)
	}
	return reflect.TypeOf("")
}

// formatString 字符串值加引号：单引号替换为双引号（值内联渲染约定）。
func formatString(ms string) string {
	var buf strings.Builder
	if len(ms) == 0 {
		return "''"
	}
	buf.WriteString("'")
	buf.WriteString(strings.ReplaceAll(ms, "'", "\""))
	buf.WriteString("'")
	return buf.String()
}

// formatValue 值内联渲染：字符串加引号、数值原样、time.Time 加引号格式化、
// nil 按 SQL null 渲染（S-09 nil 参数反射零值 panic 防御）。
func formatValue(m interface{}) string {
	if m == nil {
		// S-09：nil 参数反射零值 panic 防御，按 SQL NULL 渲染
		return "null"
	}
	typ := reflect.TypeOf(m)
	switch typ.String() {
	case "string":
		return formatString(m.(string))
	case "bool",
		"int", "int8", "int16", "int32",
		"uint", "uint8", "uint16", "uint32",
		"int64", "uint64",
		"float32", "float64":
		return fmt.Sprintf("%v", m)
	case "time.Time":
		return fmt.Sprintf("'%v'", m.(time.Time).Format("2006-01-02 15:04:05.000000000"))
	default:
		log.Warnf("not support convert type %v", typ)
	}
	return ""
}

// validValue <if test> 非空判定：字符串看长度、切片/map 看元素数、
// time.Time 看零值、数值恒真；nil 一律为假（S-09）。
func validValue(m interface{}) bool {
	if m == nil {
		// S-09：nil 参数 reflect.TypeOf 返回 nil，String() 会 panic
		return false
	}
	typ := reflect.TypeOf(m)
	switch typ.String() {
	case "string":
		ms := m.(string)
		return len(ms) > 0
	case "bool",
		"int", "int8", "int16", "int32",
		"uint", "uint8", "uint16", "uint32",
		"int64", "uint64",
		"float32", "float64":
		return true
	case "time.Time":
		return !m.(time.Time).IsZero()
	}
	switch typ.Kind() {
	case reflect.Slice:
		val := reflect.ValueOf(m)
		return val.Len() > 0
	case reflect.Map:
		val := reflect.ValueOf(m)
		return val.Len() > 0

	}
	log.Warnf("not support valid value: %v ,type: %v, kind: %v", m, typ, typ.Kind())
	return true
}

// rawFormatValue 返回 ${...} 原始替换值：字符串原样注入（不加引号），其余按 %v。
func rawFormatValue(m interface{}) string {
	if s, ok := m.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", m)
}

// lookupParam 按 buildKey 扁平键查找参数；未命中且名称含 "." 时，
// 按点号分段遍历嵌套 map / struct（含指针），支持 #{params.beginTime}、#{item.deptId} 等。
func lookupParam(m map[string]interface{}, name string) (interface{}, bool) {
	if val, ok := m[buildKey(name)]; ok {
		return val, true
	}
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return nil, false
	}
	cur, ok := m[buildKey(parts[0])]
	if !ok {
		return nil, false
	}
	for _, part := range parts[1:] {
		v := reflect.ValueOf(cur)
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return nil, false
			}
			v = v.Elem()
		}
		switch v.Kind() {
		case reflect.Map:
			if v.Type().Key().Kind() != reflect.String {
				return nil, false
			}
			if mv := v.MapIndex(reflect.ValueOf(part)); mv.IsValid() {
				cur = mv.Interface()
				continue
			}
			if mv := v.MapIndex(reflect.ValueOf(buildKey(part))); mv.IsValid() {
				cur = mv.Interface()
				continue
			}
			found := false
			for _, k := range v.MapKeys() {
				if buildKey(k.String()) == buildKey(part) {
					cur = v.MapIndex(k).Interface()
					found = true
					break
				}
			}
			if !found {
				return nil, false
			}
		case reflect.Struct:
			if fv := v.FieldByName(part); fv.IsValid() {
				cur = fv.Interface()
				continue
			}
			fv := v.FieldByNameFunc(func(s string) bool { return buildKey(s) == buildKey(part) })
			if !fv.IsValid() {
				return nil, false
			}
			cur = fv.Interface()
		default:
			return nil, false
		}
	}
	return cur, true
}
