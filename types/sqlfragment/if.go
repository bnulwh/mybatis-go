package sqlfragment

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/bnulwh/mybatis-go/log"
)

type checkConditionType string

const (
	nullCheckCond    checkConditionType = "null"
	emptyCheckCond   checkConditionType = "empty"
	boolCheckCond    checkConditionType = "bool"
	compareCheckCond checkConditionType = "compare"
)

type ifCondition struct {
	CheckName string
	CheckType checkConditionType
	Operator  string // compareCheckCond：== / != / > / >= / < / <=
	Literal   string // compareCheckCond：数值字面量（如 0、18、-1）
}

// sqlIfTest 对应 MyBatis <if test="...">：条件全部满足时渲染子片段。
type sqlIfTest struct {
	Sql        []Node
	Test       string
	Conditions []ifCondition
}

func init() {
	RegisterNodeParser("if", func(node XmlNode, sns map[string]*SqlElement) (Node, error) {
		return parseSqlIfTestFromXmlNode(node.Attrs, node.Elements, sns)
	})
}

func (in *sqlIfTest) PrepareSqlWithSlice(m []interface{}, depth int) (string, []string) {
	log.Debugf("sql if test prepare sql with slice : %v  depth: %v", m, depth)
	if len(m) < 1 {
		return "", []string{}
	}
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Sql {
		buf.WriteString(" ")
		switch it := item.(type) {
		case *simpleSql:
			buf.WriteString(it.Sql)
		case *sqlInclude:
			sqlstr, items := it.PrepareSqlWithSlice(m, depth+1)
			buf.WriteString(sqlstr)
			results = append(results, items...)
		case *sqlForLoop:
			sqlstr, items := it.prepareSql(map[string]interface{}{}, m, depth+1)
			buf.WriteString(sqlstr)
			results = append(results, items...)
		default:
			log.Warnf("unsupport if test type %T", item)
		}
	}
	return buf.String(), results
}

func (in *sqlIfTest) GenerateSqlWithSlice(m []interface{}, depth int) string {
	log.Debugf("sql if test generate sql with slice : %v  depth: %v", m, depth)
	if len(m) < 1 {
		return ""
	}
	var buf bytes.Buffer
	for _, item := range in.Sql {
		buf.WriteString(" ")
		switch it := item.(type) {
		case *simpleSql:
			buf.WriteString(it.Sql)
		case *sqlInclude:
			buf.WriteString(it.GenerateSqlWithSlice(m, depth+1))
		case *sqlForLoop:
			buf.WriteString(it.generateSql(map[string]interface{}{}, m, depth+1))
		default:
			log.Warnf("unsupport if test type %T", item)
		}
	}
	return buf.String()
}

func (in *sqlIfTest) PrepareSqlWithMap(mp map[string]interface{}, depth int) (string, []string) {
	log.Debugf("sql if test prepare sql with map : %v depth: %v", mp, depth)
	bv := in.checkConditions(mp)
	if !bv {
		return "", []string{}
	}
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Sql {
		buf.WriteString(" ")
		sqlstr, items := item.PrepareSqlWithMap(mp, depth+1)
		buf.WriteString(sqlstr)
		results = append(results, items...)
	}
	return buf.String(), results
}

func (in *sqlIfTest) GenerateSqlWithMap(mp map[string]interface{}, depth int) string {
	log.Debugf("sql if test generate sql with map : %v depth: %v", mp, depth)
	bv := in.checkConditions(mp)
	if !bv {
		return ""
	}
	var buf bytes.Buffer
	for _, item := range in.Sql {
		buf.WriteString(" ")
		buf.WriteString(item.GenerateSqlWithMap(mp, depth+1))
	}
	return buf.String()
}

func (in *sqlIfTest) PrepareSqlWithParam(m interface{}) (string, []string) {
	log.Debugf("sql if test prepare sql with param: %v", m)
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Sql {
		buf.WriteString(" ")
		sqlstr, items := item.PrepareSqlWithParam(m)
		buf.WriteString(sqlstr)
		results = append(results, items...)
	}
	return buf.String(), results
}

func (in *sqlIfTest) GenerateSqlWithParam(m interface{}) string {
	log.Debugf("sql if test generate sql with param: %v", m)
	var buf bytes.Buffer
	for _, item := range in.Sql {
		buf.WriteString(" ")
		buf.WriteString(item.GenerateSqlWithParam(m))
	}
	return buf.String()
}

func (in *sqlIfTest) GenerateSqlWithoutParam() string {
	return ""
}

func (in *sqlIfTest) Children() []Node {
	return in.Sql
}

// CollectSlots 条件体内占位符不统计（条件体由运行时求值，不视为必参）。
func (in *sqlIfTest) CollectSlots() []string {
	return nil
}

func (in *sqlIfTest) ContainsForEach() bool {
	for _, item := range in.Sql {
		if item != nil && item.ContainsForEach() {
			return true
		}
	}
	return false
}

// CollectIfTestFields 自身条件引用的字段名 + 子片段递归聚合。
func (in *sqlIfTest) CollectIfTestFields() []string {
	names := ifTestFieldNames(in.Conditions)
	for _, item := range in.Sql {
		if item == nil {
			continue
		}
		names = append(names, item.CollectIfTestFields()...)
	}
	return names
}

func (in *sqlIfTest) checkConditions(m map[string]interface{}) bool {
	log.Debugf("sql if test check conditions with param: %v", m)
	for _, cond := range in.Conditions {
		bv := cond.checkValue(m)
		if !bv {
			return false
		}
	}
	return true
}

func (in *ifCondition) checkValue(m map[string]interface{}) bool {
	log.Debugf("if condition %v check value: %v", in.CheckName, m)
	// M-02：集合长度比较（businessTypes.length > 0）——切片没有 length 键，
	// 普通 lookupParam 对完整点号名必然失败，必须先按 .length 后缀取基础值求长度
	if in.CheckType == compareCheckCond && strings.HasSuffix(in.CheckName, ".length") {
		base, ok := lookupParam(m, strings.TrimSuffix(in.CheckName, ".length"))
		if !ok || base == nil {
			return false
		}
		v := reflect.ValueOf(base)
		switch v.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
			return compareNumeric(float64(v.Len()), in.Operator, in.Literal)
		default:
			log.Warnf("compare condition %v base is not a collection/string: %T", in.CheckName, base)
			return false
		}
	}
	val, ok := lookupParam(m, in.CheckName)
	if !ok {
		return false
	}
	if val == nil {
		return false
	}
	if in.CheckType == boolCheckCond {
		// 裸标识符：布尔 false / 缺失 / 非布尔一律不满足，true 才通过（S-07）
		b, ok := val.(bool)
		if !ok {
			log.Warnf("bool condition %v got non-bool value %v (%T)", in.CheckName, val, val)
			return false
		}
		return b
	}
	if in.CheckType == compareCheckCond {
		// 数值比较（M-02）：!= 0 / > 0 / == 0 等，缺失/nil 一律不满足
		n, ok := numericValue(val)
		if !ok {
			log.Warnf("compare condition %v got non-numeric value %v (%T)", in.CheckName, val, val)
			return false
		}
		return compareNumeric(n, in.Operator, in.Literal)
	}
	return validValue(val)
}

// compareNumeric 数值比较求值（M-02）：f 与字面量按 op 比较。
func compareNumeric(f float64, op, literal string) bool {
	lit, err := strconv.ParseFloat(literal, 64)
	if err != nil {
		log.Warnf("compare condition bad literal %q: %v", literal, err)
		return false
	}
	switch op {
	case "==":
		return f == lit
	case "!=":
		return f != lit
	case ">":
		return f > lit
	case ">=":
		return f >= lit
	case "<":
		return f < lit
	case "<=":
		return f <= lit
	}
	log.Warnf("unsupported compare operator %q", op)
	return false
}

// numericValue 将数值（含字符串数字，如 "5"）统一转为 float64。
func numericValue(val interface{}) (float64, bool) {
	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint()), true
	case reflect.Float32, reflect.Float64:
		return v.Float(), true
	case reflect.String:
		f, err := strconv.ParseFloat(strings.TrimSpace(v.String()), 64)
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}

// ifTestFieldNames 条件字段名归一化：去除 .length 后缀、跳过 params. 前缀（通用 map 参数，无法静态校验）。
func ifTestFieldNames(conds []ifCondition) []string {
	var names []string
	for _, cond := range conds {
		name := strings.TrimSuffix(cond.CheckName, ".length")
		if strings.HasPrefix(name, "params.") {
			continue
		}
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// parseSqlIfTestFromXmlNode 解析 <if test="...">；嵌套标签经统一注册表解析
// （已注册标签均支持；未注册标签记日志跳过）。
func parseSqlIfTestFromXmlNode(attrs map[string]string, elems []XmlElement, sns map[string]*SqlElement) (*sqlIfTest, error) {
	ts, ok := attrs["test"]
	if !ok {
		return nil, fmt.Errorf("not found test attr in input")
	}
	if len(elems) < 1 {
		return nil, fmt.Errorf("wrong input for if test sql")
	}
	var sts []Node
	for _, elem := range elems {
		switch elem.ElementType {
		case XmlTextElem:
			sts = append(sts, parseSimpleSqlFromText(elem.Val.(string)))
		case XmlNodeElem:
			xn := elem.Val.(XmlNode)
			parser, ok := lookupNodeParser(xn.Name)
			if !ok {
				log.Errorf("not support sql node type %v in if test", xn.Name)
				continue
			}
			stemp, err := parser(xn, sns)
			if err != nil {
				return nil, err
			}
			sts = append(sts, stemp)
		}
	}
	return &sqlIfTest{
		Test:       ts,
		Sql:        sts,
		Conditions: parseIfConditionsFromText(ts),
	}, nil
}

func parseIfConditionsFromText(text string) []ifCondition {
	reSplit := regexp.MustCompile("[aA][nN][dD]")
	reNC := regexp.MustCompile(`[\w.]+[\s]*[!][=][\s]*null`)
	reEC := regexp.MustCompile(`[\w.]+[\s]*[!][=][\s]*[']{2}`)
	reBool := regexp.MustCompile(`^[\w.]+$`)
	reName := regexp.MustCompile(`[\w.]+`)
	// M-02：数值比较（userId != 0 / age > 18 / count >= 1 / x == 0 / y <= -1），
	// 也覆盖 OGNL 集合长度（businessTypes.length > 0，字面量为数值即可）
	reCmp := regexp.MustCompile(`([\w.]+)[\s]*(==|!=|>=|<=|>|<)[\s]*([-+]?\d+(?:\.\d+)?)`)
	var cs []ifCondition
	for _, item := range reSplit.Split(text, -1) {
		item = strings.TrimSpace(item)
		if len(item) == 0 {
			continue
		}
		matches := reName.FindStringSubmatch(item)
		if matches == nil {
			continue
		}
		if reNC.MatchString(item) {
			cs = append(cs, ifCondition{
				CheckName: matches[0],
				CheckType: nullCheckCond,
			})
		} else if reEC.MatchString(item) {
			cs = append(cs, ifCondition{
				CheckName: matches[0],
				CheckType: emptyCheckCond,
			})
		} else if cm := reCmp.FindStringSubmatch(item); cm != nil {
			// 数值比较（M-02）：不再静默丢弃，userId != 0 按真实数值求值
			cs = append(cs, ifCondition{
				CheckName: cm[1],
				CheckType: compareCheckCond,
				Operator:  cm[2],
				Literal:   cm[3],
			})
		} else if reBool.MatchString(item) {
			// 裸标识符按布尔求值（如 <if test="deptCheckStrictly">），
			// 避免恒为 true 导致 false 时也不剔除（S-07）。
			cs = append(cs, ifCondition{
				CheckName: matches[0],
				CheckType: boolCheckCond,
			})
		}
	}
	return cs
}
