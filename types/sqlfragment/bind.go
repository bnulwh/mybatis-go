package sqlfragment

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/bnulwh/mybatis-go/log"
)

// <bind name="pattern" value="'%' + keyword + '%'"/>（P0-3）
//
// Java MyBatis 的 <bind> 在动态 SQL 上下文里求值一个 OGNL 表达式，并把结果
// 绑定为语句级变量，供后续 #{name} / <if test> 引用。本项目此前既不支持该标签，
// 也只在解析时打一条无上下文的 error 日志后丢弃，导致 `#{pattern}` 原样发到数据库。
//
// 支持范围（表达式子集，加载期编译、求值期只做取值/拼接）：
//   - 单/双引号字符串字面量（'' 转义）、数字字面量、null/true/false；
//   - 参数引用（a、a.b 点号路径，大小写不敏感，与 #{} 取值规则一致）；
//   - 用 + / || 连接多项：任一项为字符串即字符串拼接，全数字则求和。
//
// 超出子集（方法调用、三元、*、/、- 等）在加载期报错，避免运行期静默失效。

type bindTermKind int

const (
	bindTermString bindTermKind = iota
	bindTermNumber
	bindTermRef
	bindTermNull
	bindTermBool
)

type bindTerm struct {
	kind     bindTermKind
	text     string  // 字符串字面量值 / 引用路径
	num      float64 // 数字字面量
	integral bool    // 数字字面量是否为整数（决定求和结果类型）
	b        bool    // 布尔字面量
}

// sqlBind 对应 <bind>：渲染期把求值结果写入参数 map（不产出 SQL 文本）。
type sqlBind struct {
	Name  string
	Value string
	Terms []bindTerm

	scalarWarn sync.Once
}

var (
	bindNumberRe = regexp.MustCompile(`^[-+]?\d+(?:\.\d+)?$`)
	bindRefRe    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$`)
)

func init() {
	RegisterNodeParser("bind", func(node XmlNode, sns map[string]*SqlElement) (Node, error) {
		return newBindNodeFromXml(node)
	})
}

func newBindNodeFromXml(node XmlNode) (Node, error) {
	name := strings.TrimSpace(node.Attrs["name"])
	if name == "" {
		return nil, fmt.Errorf("<bind> requires a non-empty name attribute")
	}
	if !bindRefRe.MatchString(name) {
		return nil, fmt.Errorf("<bind> name %q is not a valid identifier", name)
	}
	value := strings.TrimSpace(node.Attrs["value"])
	if value == "" {
		return nil, fmt.Errorf("<bind name=%q> requires a non-empty value attribute", name)
	}
	terms, err := parseBindExpr(value)
	if err != nil {
		return nil, fmt.Errorf("<bind name=%q value=%q>: %v", name, value, err)
	}
	return &sqlBind{Name: name, Value: value, Terms: terms}, nil
}

// parseBindExpr 解析 <bind value> 表达式子集（+ / || 连接的项序列）。
func parseBindExpr(expr string) ([]bindTerm, error) {
	raw, err := splitBindTerms(expr)
	if err != nil {
		return nil, err
	}
	terms := make([]bindTerm, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			return nil, fmt.Errorf("empty operand in expression")
		}
		if t, ok, err := parseBindLiteral(item); err != nil {
			return nil, err
		} else if ok {
			terms = append(terms, t)
			continue
		}
		if !bindRefRe.MatchString(item) {
			return nil, fmt.Errorf("unsupported operand %q (only literal, parameter reference and + / || concatenation are supported)", item)
		}
		terms = append(terms, bindTerm{kind: bindTermRef, text: item})
	}
	if len(terms) == 0 {
		return nil, fmt.Errorf("empty expression")
	}
	return terms, nil
}

// splitBindTerms 按顶层 + / || 拆分操作数（引号内的符号不拆分）。
func splitBindTerms(expr string) ([]string, error) {
	var out []string
	var buf strings.Builder
	i := 0
	for i < len(expr) {
		c := expr[i]
		switch c {
		case '\'', '"':
			end := scanBindQuote(expr, i, c)
			if end < 0 {
				return nil, fmt.Errorf("unclosed string literal")
			}
			buf.WriteString(expr[i:end])
			i = end
		case '+':
			out = append(out, buf.String())
			buf.Reset()
			i++
		case '|':
			if i+1 < len(expr) && expr[i+1] == '|' {
				out = append(out, buf.String())
				buf.Reset()
				i += 2
				continue
			}
			return nil, fmt.Errorf("unsupported operator |")
		case '(', ')', '*', '/', '%', '?', ':', '!', '<', '>', '=', '&':
			return nil, fmt.Errorf("unsupported operator %q", string(c))
		default:
			buf.WriteByte(c)
			i++
		}
	}
	out = append(out, buf.String())
	return out, nil
}

// scanBindQuote 返回引号字面量结束位置（不含）；未闭合返回 -1。
func scanBindQuote(expr string, i int, quote byte) int {
	j := i + 1
	for j < len(expr) {
		if expr[j] == quote {
			if j+1 < len(expr) && expr[j+1] == quote {
				j += 2
				continue
			}
			return j + 1
		}
		j++
	}
	return -1
}

func parseBindLiteral(item string) (bindTerm, bool, error) {
	if len(item) >= 2 && (item[0] == '\'' || item[0] == '"') {
		if item[len(item)-1] != item[0] {
			return bindTerm{}, false, fmt.Errorf("malformed string literal %q", item)
		}
		body := item[1 : len(item)-1]
		q := string(item[0])
		body = strings.ReplaceAll(body, q+q, q)
		return bindTerm{kind: bindTermString, text: body}, true, nil
	}
	switch strings.ToLower(item) {
	case "null":
		return bindTerm{kind: bindTermNull}, true, nil
	case "true":
		return bindTerm{kind: bindTermBool, b: true}, true, nil
	case "false":
		return bindTerm{kind: bindTermBool, b: false}, true, nil
	}
	if bindNumberRe.MatchString(item) {
		f, err := strconv.ParseFloat(item, 64)
		if err != nil {
			return bindTerm{}, false, fmt.Errorf("invalid number %q", item)
		}
		return bindTerm{kind: bindTermNumber, num: f, integral: !strings.Contains(item, ".")}, true, nil
	}
	return bindTerm{}, false, nil
}

// eval 求值：返回绑定值与是否成功（引用缺失按 nil/空串处理，不中断语句）。
func (in *sqlBind) eval(mp map[string]interface{}) (interface{}, bool) {
	values := make([]interface{}, 0, len(in.Terms))
	allNumber := true
	integral := true
	var sum float64
	for _, t := range in.Terms {
		switch t.kind {
		case bindTermString, bindTermNull, bindTermBool:
			allNumber = false
			if t.kind == bindTermString {
				values = append(values, t.text)
			} else if t.kind == bindTermBool {
				values = append(values, t.b)
			} else {
				values = append(values, nil)
			}
		case bindTermNumber:
			sum += t.num
			if !t.integral {
				integral = false
			}
			values = append(values, t.num)
		case bindTermRef:
			v, ok := lookupParam(mp, t.text)
			if !ok {
				log.Debugf("<bind name=%q>: reference %q not found in params", in.Name, t.text)
				v = nil
			}
			values = append(values, v)
			if f, isNum := toFloat(v); isNum {
				if f != float64(int64(f)) {
					integral = false
				}
				sum += f
			} else {
				allNumber = false
			}
		}
	}
	if len(in.Terms) == 1 {
		// 单项：保留原值类型（数字/字符串/布尔/时间等）
		if in.Terms[0].kind == bindTermNumber {
			if integral {
				return int64(sum), true
			}
			return sum, true
		}
		return values[0], true
	}
	if allNumber {
		if integral {
			return int64(sum), true
		}
		return sum, true
	}
	var b strings.Builder
	for _, v := range values {
		if v == nil {
			continue
		}
		b.WriteString(fmt.Sprint(v))
	}
	return b.String(), true
}

// toFloat 把数字类参数转 float64。
func toFloat(v interface{}) (float64, bool) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return 0, false
	}
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	}
	return 0, false
}

// bind 求值并写入参数 map（一次渲染内有效）。
func (in *sqlBind) bind(mp map[string]interface{}) {
	if mp == nil {
		return
	}
	v, _ := in.eval(mp)
	mp[buildKey(in.Name)] = v
	if in.Name != "" {
		mp[in.Name] = v
	}
}

func (in *sqlBind) PrepareSqlWithMap(mp map[string]interface{}, depth int) (string, []string) {
	in.bind(mp)
	return "", nil
}

func (in *sqlBind) GenerateSqlWithMap(mp map[string]interface{}, depth int) string {
	in.bind(mp)
	return ""
}

func (in *sqlBind) PrepareSqlWithParam(m interface{}) (string, []string) {
	in.bindParam(m)
	return "", nil
}

func (in *sqlBind) GenerateSqlWithParam(m interface{}) string {
	in.bindParam(m)
	return ""
}

func (in *sqlBind) PrepareSqlWithSlice(m []interface{}, depth int) (string, []string) {
	// 顶层切片参数：<bind> 只在 <foreach> 体内的逐项 map 上下文生效
	return "", nil
}

func (in *sqlBind) GenerateSqlWithSlice(m []interface{}, depth int) string {
	return ""
}

func (in *sqlBind) GenerateSqlWithoutParam() string {
	return ""
}

func (in *sqlBind) Children() []Node { return nil }

func (in *sqlBind) CollectSlots() []string { return nil }

func (in *sqlBind) ContainsForEach() bool { return false }

func (in *sqlBind) CollectIfTestFields() []string { return nil }

// UnsupportedIfTests 实现 A3 的结构化收集接口；<bind> 自身不携带 if 条件。
func (in *sqlBind) UnsupportedIfTests() []string { return nil }

// BoundName 返回本节点绑定的变量名。
func (in *sqlBind) BoundName() string { return in.Name }

// BindRefs 返回表达式引用的参数路径（去重、按首现序）。
func (in *sqlBind) BindRefs() []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range in.Terms {
		if t.kind != bindTermRef {
			continue
		}
		k := buildKey(t.text)
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, t.text)
	}
	return out
}

// bindParam 标量/结构体参数渲染路径：仅 map 可直接写入，其余给出一次告警。
func (in *sqlBind) bindParam(m interface{}) {
	if mp, ok := m.(map[string]interface{}); ok {
		in.bind(mp)
		return
	}
	in.scalarWarn.Do(func() {
		log.Warnf("<bind name=%q>: statement is rendered with non-map parameter %T; bind variable not injected", in.Name, m)
	})
}
