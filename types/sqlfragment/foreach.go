package sqlfragment

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/bnulwh/mybatis-go/log"
)

// sqlForLoop 对应 MyBatis <foreach>（兼容别名 <for>）：
// 遍历集合参数，按 open/close/separator 拼接循环体。
type sqlForLoop struct {
	Sql        *simpleSql
	Collection string
	Item       string
	Index      string
	Separator  string
	Open       string
	Close      string
}

func init() {
	RegisterNodeParser("for", parseForeachNode)
	RegisterNodeParser("foreach", parseForeachNode)
}

func parseForeachNode(node XmlNode, sns map[string]*SqlElement) (Node, error) {
	return parseSqlForLoopFromXmlNode(node.Attrs, node.Elements)
}

func (in *sqlForLoop) prepareSql(mp map[string]interface{}, items []interface{}, depth int) (string, []string) {
	log.Debugf("sql for loop prepare sql params: %v %v depth: %v", mp, items, depth)
	if items == nil || len(items) == 0 {
		return "", []string{}
	}
	var buf bytes.Buffer
	var results []string
	buf.WriteString(" ")
	buf.WriteString(in.Open)
	for i, item := range items {
		buf.WriteString(" ")
		nmp := in.buildParams(i, item, mp)
		sqlstr, ritems := in.Sql.PrepareSqlWithMap(nmp, depth+1)
		buf.WriteString(sqlstr)
		results = append(results, ritems...)
		if i < len(items)-1 {
			buf.WriteString(in.Separator)
		}
	}
	buf.WriteString(in.Close)
	return buf.String(), results
}

func (in *sqlForLoop) generateSql(mp map[string]interface{}, items []interface{}, depth int) string {
	log.Debugf("sql for loop generate sql params: %v %v depth: %v", mp, items, depth)
	if items == nil || len(items) == 0 {
		return ""
	}
	var buf bytes.Buffer
	buf.WriteString(" ")
	buf.WriteString(in.Open)
	for i, item := range items {
		buf.WriteString(" ")
		nmp := in.buildParams(i, item, mp)
		buf.WriteString(in.Sql.GenerateSqlWithMap(nmp, depth+1))
		if i < len(items)-1 {
			buf.WriteString(in.Separator)
		}
	}
	buf.WriteString(in.Close)
	return buf.String()
}

func (in *sqlForLoop) buildParams(index int, item interface{}, mp map[string]interface{}) map[string]interface{} {
	nmp := map[string]interface{}{buildKey(in.Index): fmt.Sprintf("%v", index)}
	for k, v := range mp {
		nmp[k] = v
	}
	ival := reflect.ValueOf(item)
	if !ival.IsValid() {
		// S-09：nil 元素反射零值 panic 防御，直接保留 item 键
		nmp[buildKey(in.Item)] = item
		return nmp
	}
	ityp := reflect.Indirect(ival).Type()
	switch ityp.Kind() {
	case reflect.Struct:
		if strings.Compare(ityp.String(), "time.Time") == 0 {
			nmp[buildKey(in.Item)] = formatValue(item)
		} else {
			for i := 0; i < ityp.NumField(); i++ {
				field := ityp.Field(i)
				key := buildKey(fmt.Sprintf("%s.%s", in.Item, field.Name))
				nmp[key] = ival.Field(i).Interface()
			}
			// 同时保留 item 本身，支持 #{item.xxx} 按点号分段遍历
			nmp[buildKey(in.Item)] = item
		}
	case reflect.Map:
		// 保留 item 本身，支持 #{item.xxx} 按点号分段遍历 map 键
		nmp[buildKey(in.Item)] = item
	case reflect.String,
		reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		// 保留原始值，由 simpleSql 渲染统一格式化（避免二次格式化加引号）
		nmp[buildKey(in.Item)] = item
	}
	log.Debugf("build param result: %v", nmp)
	return nmp
}

// PrepareSqlWithMap 集合参数为切片时展开循环渲染；缺失或非切片返回空。
func (in *sqlForLoop) PrepareSqlWithMap(mp map[string]interface{}, depth int) (string, []string) {
	val, ok := mp[buildKey(in.Collection)]
	if !ok {
		return "", []string{}
	}
	if reflect.TypeOf(val).Kind() == reflect.Slice {
		sval := convert2Slice(reflect.ValueOf(val))
		return in.prepareSql(mp, sval, depth+1)
	}
	return "", []string{}
}

func (in *sqlForLoop) GenerateSqlWithMap(mp map[string]interface{}, depth int) string {
	val, ok := mp[buildKey(in.Collection)]
	if !ok {
		return ""
	}
	if reflect.TypeOf(val).Kind() == reflect.Slice {
		sval := convert2Slice(reflect.ValueOf(val))
		return in.generateSql(mp, sval, depth+1)
	}
	return ""
}

func (in *sqlForLoop) PrepareSqlWithParam(m interface{}) (string, []string) {
	return "", []string{}
}

func (in *sqlForLoop) GenerateSqlWithParam(m interface{}) string {
	return ""
}

func (in *sqlForLoop) PrepareSqlWithSlice(m []interface{}, depth int) (string, []string) {
	return in.prepareSql(map[string]interface{}{}, m, depth+1)
}

func (in *sqlForLoop) GenerateSqlWithSlice(m []interface{}, depth int) string {
	return in.generateSql(map[string]interface{}{}, m, depth+1)
}

func (in *sqlForLoop) GenerateSqlWithoutParam() string {
	return ""
}

func (in *sqlForLoop) Children() []Node {
	return []Node{in.Sql}
}

// CollectSlots 循环体由集合参数驱动，不入 slots。
func (in *sqlForLoop) CollectSlots() []string {
	return nil
}

func (in *sqlForLoop) ContainsForEach() bool {
	return true
}

func (in *sqlForLoop) CollectIfTestFields() []string {
	return nil
}

func parseSqlForLoopFromXmlNode(attrs map[string]string, elems []XmlElement) (*sqlForLoop, error) {
	col, ok := attrs["collection"]
	if !ok {
		return nil, fmt.Errorf("not found  collection in input for parsing sql for loop")
	}
	if len(elems) < 1 {
		return nil, fmt.Errorf("wrong input for parsing sql for loop")
	}
	return &sqlForLoop{
		Collection: col,
		Open:       attrs["open"],
		Close:      attrs["close"],
		Index:      attrs["index"],
		Item:       attrs["item"],
		Separator:  attrs["separator"],
		Sql:        parseSimpleSqlFromText(elems[0].Val.(string)),
	}, nil
}
