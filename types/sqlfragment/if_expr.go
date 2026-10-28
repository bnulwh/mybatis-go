package sqlfragment

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// `<if test>` 表达式 AST（P0-2）。
//
// 旧实现用正则把 test 文本按 "and" 切分后匹配 5 类固定模式，无法表达
// `a == null`、`a != null or b != null`、括号、取反与算术；更危险的是
// `a != null or b != null` 会被静默截断成单个条件（生成错 SQL，且不进 A3 未识别清单）。
//
// 本文件实现一个轻量表达式编译器（加载期一次编译、求值期只做取值）：
//
//	orExpr    := andExpr (('or' | '||') andExpr)*
//	andExpr   := unary  (('and' | '&&') unary)*
//	unary     := ('not' | '!') unary | '-' unary | comparison
//	comparison:= additive (('==' | '!=' | '>' | '>=' | '<' | '<=' | eq/neq/lt/lte/gt/gte) additive)?
//	additive  := multiplicative (('+' | '-') multiplicative)*
//	multiplicative := primary (('*' | '/' | '%') primary)*
//	primary   := number | string | null | true | false | ident(.ident)* | '(' orExpr ')'
//
// 编译失败（方法调用、三元、单等号等）时调用方回退旧正则解析，A3 语义保持不变。

type ifTokKind int

const (
	ifTokEOF ifTokKind = iota
	ifTokIdent
	ifTokNumber
	ifTokString
	ifTokOp
	ifTokLParen
	ifTokRParen
)

type ifToken struct {
	kind ifTokKind
	text string
	num  float64
	pos  int
}

// ifExpr 表达式节点。
type ifExpr interface {
	eval(mp map[string]interface{}) (interface{}, error)
	fields() []string
}

// parseIfExpr 编译 test 表达式；失败返回 error（调用方回退 legacy 正则解析）。
func parseIfExpr(src string) (ifExpr, error) {
	toks, err := tokenizeIfExpr(src)
	if err != nil {
		return nil, err
	}
	p := &ifParser{toks: toks}
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != ifTokEOF {
		return nil, fmt.Errorf("unexpected token %q", p.peek().text)
	}
	return expr, nil
}

// isBareIfTest 判断 test 是否为「单标识符/点号路径」（S-07 兼容：
// 裸标识符按布尔求值，非 bool 一律不通过，而不是 OGNL 真值语义）。
func isBareIfTest(src string) bool {
	src = strings.TrimSpace(src)
	if src == "" {
		return false
	}
	for i := 0; i < len(src); i++ {
		c := src[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '.') {
			return false
		}
	}
	switch strings.ToLower(src) {
	case "null", "true", "false":
		return false
	}
	return true
}

func tokenizeIfExpr(src string) ([]ifToken, error) {
	var toks []ifToken
	i := 0
	for i < len(src) {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '(':
			toks = append(toks, ifToken{kind: ifTokLParen, text: "(", pos: i})
			i++
		case c == ')':
			toks = append(toks, ifToken{kind: ifTokRParen, text: ")", pos: i})
			i++
		case c == '\'' || c == '"':
			end := scanIfString(src, i, c)
			if end < 0 {
				return nil, fmt.Errorf("unclosed string literal")
			}
			body := src[i+1 : end-1]
			q := string(c)
			toks = append(toks, ifToken{kind: ifTokString, text: strings.ReplaceAll(body, q+q, q), pos: i})
			i = end
		case c >= '0' && c <= '9':
			j := i
			for j < len(src) && (src[j] >= '0' && src[j] <= '9' || src[j] == '.') {
				j++
			}
			f, err := strconv.ParseFloat(src[i:j], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid number %q", src[i:j])
			}
			toks = append(toks, ifToken{kind: ifTokNumber, text: src[i:j], num: f, pos: i})
			i = j
		case isIfIdentStart(c):
			j := i + 1
			for j < len(src) && isIfIdentPart(src[j]) {
				j++
			}
			word := src[i:j]
			if op, ok := ifWordOperator(word); ok {
				toks = append(toks, ifToken{kind: ifTokOp, text: op, pos: i})
			} else {
				toks = append(toks, ifToken{kind: ifTokIdent, text: word, pos: i})
			}
			i = j
		default:
			op, n, err := scanIfOperator(src, i)
			if err != nil {
				return nil, err
			}
			toks = append(toks, ifToken{kind: ifTokOp, text: op, pos: i})
			i += n
		}
	}
	toks = append(toks, ifToken{kind: ifTokEOF, pos: len(src)})
	return toks, nil
}

// ifWordOperator 识别 OGNL 字母算子（and/or/not 与 eq/neq/lt/gt/lte/gte 别名）。
func ifWordOperator(word string) (string, bool) {
	switch strings.ToLower(word) {
	case "and":
		return "and", true
	case "or":
		return "or", true
	case "not":
		return "not", true
	case "eq":
		return "eq", true
	case "neq", "ne":
		return "neq", true
	case "lt":
		return "lt", true
	case "lte", "le":
		return "lte", true
	case "gt":
		return "gt", true
	case "gte", "ge":
		return "gte", true
	}
	return "", false
}

func isIfIdentStart(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_'
}

func isIfIdentPart(c byte) bool {
	return isIfIdentStart(c) || c >= '0' && c <= '9' || c == '.'
}

func scanIfString(src string, i int, quote byte) int {
	j := i + 1
	for j < len(src) {
		if src[j] == quote {
			if j+1 < len(src) && src[j+1] == quote {
				j += 2
				continue
			}
			return j + 1
		}
		j++
	}
	return -1
}

// scanIfOperator 识别比较/逻辑/算术操作符（含 OGNL 字母别名）。
func scanIfOperator(src string, i int) (string, int, error) {
	two := ""
	if i+1 < len(src) {
		two = src[i : i+2]
	}
	switch two {
	case "==", "!=", ">=", "<=", "&&", "||":
		return two, 2, nil
	}
	switch src[i] {
	case '>', '<', '!', '+', '-', '*', '/', '%':
		return string(src[i]), 1, nil
	}
	return "", 0, fmt.Errorf("unsupported character %q", string(src[i]))
}

type ifParser struct {
	toks []ifToken
	pos  int
}

func (p *ifParser) peek() ifToken { return p.toks[p.pos] }
func (p *ifParser) next() ifToken {
	t := p.toks[p.pos]
	if t.kind != ifTokEOF {
		p.pos++
	}
	return t
}

func (p *ifParser) acceptOp(ops ...string) (string, bool) {
	t := p.peek()
	if t.kind != ifTokOp {
		return "", false
	}
	for _, op := range ops {
		if t.text == op {
			return p.next().text, true
		}
	}
	return "", false
}

func (p *ifParser) parseOr() (ifExpr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for {
		if _, ok := p.acceptOp("or", "||"); !ok {
			return left, nil
		}
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &exprLogical{op: "or", left: left, right: right}
	}
}

func (p *ifParser) parseAnd() (ifExpr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		if _, ok := p.acceptOp("and", "&&"); !ok {
			return left, nil
		}
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &exprLogical{op: "and", left: left, right: right}
	}
}

func (p *ifParser) parseUnary() (ifExpr, error) {
	if _, ok := p.acceptOp("not", "!"); ok {
		inner, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &exprNot{inner: inner}, nil
	}
	return p.parseComparison()
}

func (p *ifParser) parseComparison() (ifExpr, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	t := p.peek()
	if t.kind != ifTokOp {
		return left, nil
	}
	op := ""
	switch strings.ToLower(t.text) {
	case "==", "eq":
		op = "=="
	case "!=", "neq", "ne":
		op = "!="
	case ">", "gt":
		op = ">"
	case ">=", "gte", "ge":
		op = ">="
	case "<", "lt":
		op = "<"
	case "<=", "lte", "le":
		op = "<="
	default:
		return left, nil
	}
	p.next()
	right, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	return &exprCompare{op: op, left: left, right: right}, nil
}

func (p *ifParser) parseAdditive() (ifExpr, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}
	for {
		op, ok := p.acceptOp("+", "-")
		if !ok {
			return left, nil
		}
		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		left = &exprArith{op: op, left: left, right: right}
	}
}

func (p *ifParser) parseMultiplicative() (ifExpr, error) {
	left, err := p.parseNeg()
	if err != nil {
		return nil, err
	}
	for {
		op, ok := p.acceptOp("*", "/", "%")
		if !ok {
			return left, nil
		}
		right, err := p.parseNeg()
		if err != nil {
			return nil, err
		}
		left = &exprArith{op: op, left: left, right: right}
	}
}

// parseNeg 一元负号（只作用于 primary，保证 -1 < count 解析为 (-1) < count）。
func (p *ifParser) parseNeg() (ifExpr, error) {
	if _, ok := p.acceptOp("-"); ok {
		inner, err := p.parseNeg()
		if err != nil {
			return nil, err
		}
		return &exprNeg{inner: inner}, nil
	}
	return p.parsePrimary()
}

func (p *ifParser) parsePrimary() (ifExpr, error) {
	t := p.next()
	switch t.kind {
	case ifTokNumber:
		return &exprConst{v: t.num}, nil
	case ifTokString:
		return &exprConst{v: t.text}, nil
	case ifTokLParen:
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != ifTokRParen {
			return nil, fmt.Errorf("missing closing parenthesis")
		}
		p.next()
		return inner, nil
	case ifTokIdent:
		switch strings.ToLower(t.text) {
		case "null", "nil":
			return &exprConst{v: nil}, nil
		case "true":
			return &exprConst{v: true}, nil
		case "false":
			return &exprConst{v: false}, nil
		case "and", "or", "not", "eq", "neq", "ne", "lt", "gt", "lte", "gte", "ge", "le":
			return nil, fmt.Errorf("unexpected keyword %q", t.text)
		}
		path := t.text
		length := false
		if strings.HasSuffix(path, ".length") {
			path = strings.TrimSuffix(path, ".length")
			length = true
		} else if strings.HasSuffix(path, ".size") {
			path = strings.TrimSuffix(path, ".size")
			length = true
		}
		if path == "" {
			return nil, fmt.Errorf("invalid reference %q", t.text)
		}
		return &exprRef{path: path, length: length}, nil
	case ifTokEOF:
		return nil, fmt.Errorf("unexpected end of expression")
	}
	return nil, fmt.Errorf("unexpected token %q", t.text)
}

// --- AST 节点 ---

type exprConst struct{ v interface{} }

func (n *exprConst) eval(map[string]interface{}) (interface{}, error) { return n.v, nil }
func (n *exprConst) fields() []string                                 { return nil }

type exprRef struct {
	path   string
	length bool
}

func (n *exprRef) fields() []string { return []string{n.path} }

func (n *exprRef) eval(mp map[string]interface{}) (interface{}, error) {
	v, _ := lookupParam(mp, n.path) // 缺失参数按 null 处理（与 #{x} 取值一致）
	if !n.length {
		return v, nil
	}
	if v == nil {
		return int64(0), nil
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return int64(0), nil
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
		return int64(rv.Len()), nil
	}
	return int64(0), nil
}

type exprNot struct{ inner ifExpr }

func (n *exprNot) fields() []string { return n.inner.fields() }
func (n *exprNot) eval(mp map[string]interface{}) (interface{}, error) {
	v, err := n.inner.eval(mp)
	if err != nil {
		return nil, err
	}
	return !ifTruthy(v), nil
}

type exprNeg struct{ inner ifExpr }

func (n *exprNeg) fields() []string { return n.inner.fields() }
func (n *exprNeg) eval(mp map[string]interface{}) (interface{}, error) {
	v, err := n.inner.eval(mp)
	if err != nil {
		return nil, err
	}
	f, ok := ifNumeric(v)
	if !ok {
		return nil, fmt.Errorf("unary '-' requires a numeric operand, got %T", v)
	}
	return -f, nil
}

type exprLogical struct {
	op          string // and / or
	left, right ifExpr
}

func (n *exprLogical) fields() []string {
	return append(append([]string{}, n.left.fields()...), n.right.fields()...)
}

func (n *exprLogical) eval(mp map[string]interface{}) (interface{}, error) {
	l, err := n.left.eval(mp)
	if err != nil {
		return nil, err
	}
	lb := ifTruthy(l)
	if n.op == "and" && !lb {
		return false, nil
	}
	if n.op == "or" && lb {
		return true, nil
	}
	r, err := n.right.eval(mp)
	if err != nil {
		return nil, err
	}
	return ifTruthy(r), nil
}

type exprCompare struct {
	op          string
	left, right ifExpr
}

func (n *exprCompare) fields() []string {
	return append(append([]string{}, n.left.fields()...), n.right.fields()...)
}

func (n *exprCompare) eval(mp map[string]interface{}) (interface{}, error) {
	l, err := n.left.eval(mp)
	if err != nil {
		return nil, err
	}
	r, err := n.right.eval(mp)
	if err != nil {
		return nil, err
	}
	// null / 空串字面量保持 legacy「有效值」语义（存量行为不变）：
	//   x != null / x != ''  ⇔  validValue(x)   （空串、空集合、nil 均为 false）
	//   x == null / x == ''  ⇔  !validValue(x)  （P0-2 新增，语义与 != 对称）
	if n.op == "==" || n.op == "!=" {
		if isNullishConst(n.right) {
			return nullishCompare(l, n.op), nil
		}
		if isNullishConst(n.left) {
			return nullishCompare(r, n.op), nil
		}
	}
	return ifCompare(n.op, l, r)
}

// isNullishConst 判断操作数是否为 null / 空字符串字面量。
func isNullishConst(e ifExpr) bool {
	c, ok := e.(*exprConst)
	if !ok {
		return false
	}
	if c.v == nil {
		return true
	}
	if s, ok := c.v.(string); ok {
		return s == ""
	}
	return false
}

// nullishCompare 与 legacy ifCondition 判空语义对齐（validValue 见 helpers.go）。
func nullishCompare(v interface{}, op string) bool {
	if op == "!=" {
		return validValue(v)
	}
	return !validValue(v)
}

type exprArith struct {
	op          string
	left, right ifExpr
}

func (n *exprArith) fields() []string {
	return append(append([]string{}, n.left.fields()...), n.right.fields()...)
}

func (n *exprArith) eval(mp map[string]interface{}) (interface{}, error) {
	l, err := n.left.eval(mp)
	if err != nil {
		return nil, err
	}
	r, err := n.right.eval(mp)
	if err != nil {
		return nil, err
	}
	if n.op == "+" {
		if _, lIsStr := l.(string); lIsStr {
			return ifConcat(l, r), nil
		}
		if _, rIsStr := r.(string); rIsStr {
			return ifConcat(l, r), nil
		}
	}
	lf, lok := ifNumeric(l)
	rf, rok := ifNumeric(r)
	if !lok || !rok {
		return nil, fmt.Errorf("operator %q requires numeric operands, got %T and %T", n.op, l, r)
	}
	switch n.op {
	case "+":
		return lf + rf, nil
	case "-":
		return lf - rf, nil
	case "*":
		return lf * rf, nil
	case "/":
		if rf == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return lf / rf, nil
	case "%":
		if rf == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return float64(int64(lf) % int64(rf)), nil
	}
	return nil, fmt.Errorf("unsupported operator %q", n.op)
}

// ifTruthy OGNL 风格真值：nil/零值/空串/空集合为假。
func ifTruthy(v interface{}) bool {
	if v == nil {
		return false
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return false
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Bool:
		return rv.Bool()
	case reflect.String:
		return rv.Len() > 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() != 0
	case reflect.Slice, reflect.Array, reflect.Map:
		return rv.Len() > 0
	}
	return true
}

// ifConcat 字符串拼接（nil 视为空串）。
func ifConcat(l, r interface{}) string {
	var b strings.Builder
	if l != nil {
		b.WriteString(fmt.Sprint(l))
	}
	if r != nil {
		b.WriteString(fmt.Sprint(r))
	}
	return b.String()
}

// ifNumeric 数字归一化（数值类型 + 数字字符串，与 legacy numericValue 对齐）。
func ifNumeric(v interface{}) (float64, bool) {
	if v == nil {
		return 0, false
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return 0, false
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	case reflect.String:
		f, err := strconv.ParseFloat(strings.TrimSpace(rv.String()), 64)
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}

// ifCompare 比较求值：
//   - nil：== / != 按是否同为 nil 判定，大小比较一律 false（不刷错误日志）；
//   - 双方数值（含数字字符串）→ 数值比较；
//   - 双方字符串 → 字典序（== 精确匹配）；
//   - 从字符串 vs 布尔字面量：字面量已被 tokenize 为 bool，仅支持 == / !=；
//   - 其他类型仅支持 == / !=（reflect.DeepEqual），大小比较返回 false。
func ifCompare(op string, l, r interface{}) (interface{}, error) {
	if l == nil || r == nil {
		switch op {
		case "==":
			return l == nil && r == nil, nil
		case "!=":
			return !(l == nil && r == nil), nil
		}
		return false, nil
	}
	lf, lok := ifNumeric(l)
	rf, rok := ifNumeric(r)
	_, lStr := l.(string)
	_, rStr := r.(string)
	// 双方都是字符串：走字符串比较（避免 '05' == '5' 的数值化陷阱）
	if lStr && rStr {
		ls, rs := l.(string), r.(string)
		switch op {
		case "==":
			return ls == rs, nil
		case "!=":
			return ls != rs, nil
		case ">":
			return ls > rs, nil
		case ">=":
			return ls >= rs, nil
		case "<":
			return ls < rs, nil
		case "<=":
			return ls <= rs, nil
		}
	}
	if lok && rok {
		switch op {
		case "==":
			return lf == rf, nil
		case "!=":
			return lf != rf, nil
		case ">":
			return lf > rf, nil
		case ">=":
			return lf >= rf, nil
		case "<":
			return lf < rf, nil
		case "<=":
			return lf <= rf, nil
		}
	}
	lb, lIsBool := l.(bool)
	rb, rIsBool := r.(bool)
	if lIsBool && rIsBool {
		switch op {
		case "==":
			return lb == rb, nil
		case "!=":
			return lb != rb, nil
		}
		return false, nil
	}
	switch op {
	case "==":
		return reflect.DeepEqual(l, r), nil
	case "!=":
		return !reflect.DeepEqual(l, r), nil
	}
	return false, nil
}
