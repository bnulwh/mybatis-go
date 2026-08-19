package types

import (
	"bytes"
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"reflect"
	"regexp"
	"strings"
)

type checkConditionType string

const (
	nullCheckCond  checkConditionType = "null"
	emptyCheckCond checkConditionType = "empty"
	boolCheckCond  checkConditionType = "bool"
)

type sqlFragmentParam struct {
	Name     string
	TypeName string
	Type     reflect.Type
	Origin   string
	Raw      bool // ${...} 原始替换：不入占位符参数、不加引号
}

type simpleSql struct {
	Sql    string
	Params []sqlFragmentParam
}

type ifCondition struct {
	CheckName string
	CheckType checkConditionType
}

type sqlIfTest struct {
	Sql        []*sqlFragment
	Test       string
	Conditions []ifCondition
}

type sqlForLoop struct {
	Sql        *simpleSql
	Collection string
	Item       string
	Index      string
	Separator  string
	Open       string
	Close      string
}

type sqlChoose struct {
	Otherwise *simpleSql
	When      []*sqlIfTest
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
		sqlstr, ritems := in.Sql.prepareSqlWithMap(nmp, depth+1)
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
		buf.WriteString(in.Sql.generateSqlWithMap(nmp, depth+1))
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
	ityp := reflect.Indirect(ival).Type()
	switch ityp.Kind() {
	case reflect.Struct:
		if strings.Compare(ityp.String(), "time.Time") == 0 {
			nmp[buildKey(in.Item)] = getFormatValue(item)
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
func (in *sqlIfTest) prepareSqlWithSlice(m []interface{}, depth int) (string, []string) {
	log.Debugf("sql if test prepare sql with slice : %v  depth: %v", m, depth)
	if len(m) < 1 {
		return "", []string{}
	}
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Sql {
		buf.WriteString(" ")
		switch item.Type {
		case simpleSqlFragment:
			buf.WriteString(item.Sql.Sql)
		case includeSqlFragment:
			sqlstr, items := item.Include.prepareSqlWithSlice(m, depth+1)
			buf.WriteString(sqlstr)
			results = append(results, items...)
		case forLoopSqlFragment:
			sqlstr, items := item.ForLoop.prepareSql(map[string]interface{}{}, m, depth+1)
			buf.WriteString(sqlstr)
			results = append(results, items...)
		default:
			log.Warnf("unsupport if test type %v", item.Type)
		}
	}
	return buf.String(), results
}
func (in *sqlIfTest) generateSqlWithSlice(m []interface{}, depth int) string {
	log.Debugf("sql if test generate sql with slice : %v  depth: %v", m, depth)
	if len(m) < 1 {
		return ""
	}
	var buf bytes.Buffer
	for _, item := range in.Sql {
		buf.WriteString(" ")
		switch item.Type {
		case simpleSqlFragment:
			buf.WriteString(item.Sql.Sql)
		case includeSqlFragment:
			buf.WriteString(item.Include.generateSqlWithSlice(m, depth+1))
		case forLoopSqlFragment:
			buf.WriteString(item.ForLoop.generateSql(map[string]interface{}{}, m, depth+1))
		default:
			log.Warnf("unsupport if test type %v", item.Type)
		}
	}
	return buf.String()
}
func (in *sqlIfTest) prepareSqlWithMap(mp map[string]interface{}, depth int) (string, []string) {
	log.Debugf("sql if test prepare sql with map : %v depth: %v", mp, depth)
	bv := in.checkConditions(mp)
	if !bv {
		return "", []string{}
	}
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Sql {
		buf.WriteString(" ")
		sqlstr, items := item.prepareSqlWithMap(mp, depth+1)
		buf.WriteString(sqlstr)
		results = append(results, items...)
	}
	return buf.String(), results
}
func (in *sqlIfTest) generateSqlWithMap(mp map[string]interface{}, depth int) string {
	log.Debugf("sql if test generate sql with map : %v depth: %v", mp, depth)
	bv := in.checkConditions(mp)
	if !bv {
		return ""
	}
	var buf bytes.Buffer
	for _, item := range in.Sql {
		buf.WriteString(" ")
		buf.WriteString(item.generateSqlWithMap(mp, depth+1))
	}
	return buf.String()
}
func (in *sqlIfTest) prepareSqlWithParam(m interface{}) (string, []string) {
	log.Debugf("sql if test prepare sql with param: %v", m)
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Sql {
		buf.WriteString(" ")
		sqlstr, items := item.prepareSqlWithParam(m)
		buf.WriteString(sqlstr)
		results = append(results, items...)
	}
	return buf.String(), results
}

func (in *sqlIfTest) generateSqlWithParam(m interface{}) string {
	log.Debugf("sql if test generate sql with param: %v", m)
	var buf bytes.Buffer
	for _, item := range in.Sql {
		buf.WriteString(" ")
		buf.WriteString(item.generateSqlWithParam(m))
	}
	return buf.String()
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
	return validValue(val)
}
func (in *sqlChoose) prepareSqlWithMap(mp map[string]interface{}, depth int) (string, []string) {
	log.Debugf("sql choose prepare sql with map: %v", mp)
	for _, item := range in.When {
		if item.checkConditions(mp) {
			return item.prepareSqlWithMap(mp, depth+1)
		}
	}
	return in.Otherwise.prepareSqlWithMap(mp, depth+1)
}

func (in *sqlChoose) generateSqlWithMap(mp map[string]interface{}, depth int) string {
	log.Debugf("sql choose generate sql with map: %v", mp)
	for _, item := range in.When {
		if item.checkConditions(mp) {
			return item.generateSqlWithMap(mp, depth+1)
		}
	}
	return in.Otherwise.generateSqlWithMap(mp, depth+1)
}
func (in *simpleSql) prepareSqlWithMap(mp map[string]interface{}, depth int) (string, []string) {
	log.Debugf("simple sql prepare sql with map: %v", mp)
	sqlstr := in.Sql
	var results []string
	for _, param := range in.Params {
		val, ok := lookupParam(mp, param.Name)
		if !ok {
			log.Warnf("not found %v in map", param.Name)
			continue
		}
		if param.Raw {
			// ${...} 原始注入：直接拼接值，不入占位符参数
			sqlstr = strings.ReplaceAll(sqlstr, param.Origin, rawFormatValue(val))
			continue
		}
		sqlstr = strings.ReplaceAll(sqlstr, param.Origin, "?")
		results = append(results, getFormatValue(val))
	}
	return sqlstr, results
}

func (in *simpleSql) generateSqlWithMap(mp map[string]interface{}, depth int) string {
	log.Debugf("simple sql generate sql with map: %v", mp)
	sqlstr := in.Sql
	for _, param := range in.Params {
		val, ok := lookupParam(mp, param.Name)
		if !ok {
			log.Warnf("not found %v in map", param.Name)
			continue
		}
		var valstr string
		if param.Raw {
			// ${...} 原始注入：不加引号
			valstr = rawFormatValue(val)
		} else {
			valstr = getFormatValue(val)
		}
		sqlstr = strings.ReplaceAll(sqlstr, param.Origin, valstr)
	}
	return sqlstr
}
func (in *simpleSql) prepareSqlWithParam(m interface{}) (string, []string) {
	log.Debugf("sql if test prepare sql with param: %v", m)
	sqlstr := in.Sql
	var results []string
	for _, param := range in.Params {
		if param.Raw {
			sqlstr = strings.ReplaceAll(sqlstr, param.Origin, rawFormatValue(m))
			continue
		}
		sqlstr = strings.ReplaceAll(sqlstr, param.Origin, "?")
		results = append(results, getFormatValue(m))
	}
	return sqlstr, results
}
func (in *simpleSql) generateSqlWithParam(m interface{}) string {
	log.Debugf("sql if test generate sql with param: %v", m)
	sqlstr := in.Sql
	for _, param := range in.Params {
		var valstr string
		if param.Raw {
			valstr = rawFormatValue(m)
		} else {
			valstr = getFormatValue(m)
		}
		sqlstr = strings.ReplaceAll(sqlstr, param.Origin, valstr)
	}
	return sqlstr
}

func parseSqlIfTestFromXmlNode(attrs map[string]string, elems []xmlElement) (*sqlFragment, error) {
	ts, ok := attrs["test"]
	if !ok {
		return nil, fmt.Errorf("not found test attr in input")
	}
	if len(elems) < 1 {
		return nil, fmt.Errorf("wrong input for if test sql")
	}
	var sts []*sqlFragment
	for _, elem := range elems {
		switch elem.ElementType {
		case xmlTextElem:
			sts = append(sts, &sqlFragment{
				Sql:     parseSimpleSqlFromText(elem.Val.(string)),
				Include: nil,
				IfTest:  nil,
				ForLoop: nil,
				Choose:  nil,
				Type:    simpleSqlFragment,
			})
		case xmlNodeElem:
			xn := elem.Val.(xmlNode)
			switch strings.ToLower(xn.Name) {
			case "if":
				stemp, err := parseSqlIfTestFromXmlNode(xn.Attrs, xn.Elements)
				if err != nil {
					return nil, err
				}
				sts = append(sts, stemp)

			case "foreach":
				stemp, err := parseSqlForLoopFromXmlNode(xn.Attrs, xn.Elements)
				if err != nil {
					return nil, err
				}
				sts = append(sts, stemp)

			case "where":
				stemp, err := parseSqlWhereFromXmlNode(xn.Elements, nil)
				if err != nil {
					return nil, err
				}
				sts = append(sts, stemp)
			}
		}
	}
	return &sqlFragment{
		IfTest: &sqlIfTest{
			Test:       ts,
			Sql:        sts,
			Conditions: parseIfConditionsFromText(ts),
		},
		Sql:     nil,
		ForLoop: nil,
		Include: nil,
		Choose:  nil,
		Type:    ifTestSqlFragment,
	}, nil
}

func parseSqlForLoopFromXmlNode(attrs map[string]string, elems []xmlElement) (*sqlFragment, error) {
	col, ok := attrs["collection"]
	if !ok {
		return nil, fmt.Errorf("not found  collection in input for parsing sql for loop")
	}
	if len(elems) < 1 {
		return nil, fmt.Errorf("wrong input for parsing sql for loop")
	}
	return &sqlFragment{
		ForLoop: &sqlForLoop{
			Collection: col,
			Open:       attrs["open"],
			Close:      attrs["close"],
			Index:      attrs["index"],
			Item:       attrs["item"],
			Separator:  attrs["separator"],
			Sql:        parseSimpleSqlFromText(elems[0].Val.(string)),
		},
		Sql:     nil,
		IfTest:  nil,
		Include: nil,
		Choose:  nil,
		Type:    forLoopSqlFragment,
	}, nil
}

func parseSqlChooseFromXmlNode(elems []xmlElement) (*sqlFragment, error) {
	var conds []*sqlIfTest
	var defCond []*simpleSql
	for _, elem := range elems {
		xn := elem.Val.(xmlNode)
		switch strings.ToLower(xn.Name) {
		case "when":
			st, err := parseSqlIfTestFromXmlNode(xn.Attrs, xn.Elements)
			if err != nil {
				return nil, err
			}
			conds = append(conds, st.IfTest)
		case "otherwise":
			dc := parseSimpleSqlFromText(xn.Elements[0].Val.(string))
			defCond = append(defCond, dc)
		}
	}
	if len(defCond) < 1 {
		return nil, fmt.Errorf("choose sql not contains otherwise")
	}
	return &sqlFragment{
		Choose: &sqlChoose{
			When:      conds,
			Otherwise: defCond[0],
		},
		Include: nil,
		ForLoop: nil,
		IfTest:  nil,
		Sql:     nil,
		Type:    chooseSqlFragment,
	}, nil
}

func parseIfConditionsFromText(text string) []ifCondition {
	reSplit := regexp.MustCompile("[aA][nN][dD]")
	reNC := regexp.MustCompile(`[\w.]+[\s]*[!][=][\s]*null`)
	reEC := regexp.MustCompile(`[\w.]+[\s]*[!][=][\s]*[']{2}`)
	reBool := regexp.MustCompile(`^[\w.]+$`)
	reName := regexp.MustCompile(`[\w.]+`)
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

func parseSimpleSqlFromText(text string) *simpleSql {
	return &simpleSql{
		Sql:    text,
		Params: parseSqlFragmentParamFromText(text),
	}
}

func parseSqlFragmentParamFromText(text string) []sqlFragmentParam {
	re := regexp.MustCompile(`[#$][{][\s]*([\w.]+)[\s]*(,[\s]*([\w.]+)[\s]*=[\s]*([\w.]+)[\s]*)*[}]`)
	matches := re.FindAllStringSubmatch(text, -1)
	var stps []sqlFragmentParam
	for _, match := range matches {
		raw := strings.HasPrefix(match[0], "${")
		if len(match) == 2 {
			stps = append(stps, sqlFragmentParam{
				Origin:   match[0],
				Name:     match[1],
				TypeName: "",
				Raw:      raw,
			})
		} else if len(match) == 5 {
			if len(match[4]) > 0 {
				stps = append(stps, sqlFragmentParam{
					Origin:   match[0],
					Name:     match[1],
					TypeName: match[4],
					Type:     ParseJdbcTypeFrom(match[4]),
					Raw:      raw,
				})
			} else {
				stps = append(stps, sqlFragmentParam{
					Origin:   match[0],
					Name:     match[1],
					TypeName: "",
					Raw:      raw,
				})
			}
		}
	}
	return stps
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
