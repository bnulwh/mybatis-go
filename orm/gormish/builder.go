package gormish

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/bnulwh/mybatis-go/orm"
)

// condition 单个 WHERE 条件（G1）：
// expr 非空为字符串条件（? 占位 + 参数绑定）；expr 为空时为 map 条件（cols 与 args 一一对应）。
type condition struct {
	expr string
	args []interface{}
	cols []string
	or   bool
}

// statement 链式语句状态（G1）：链式方法克隆 DB 时深拷贝本结构，互不污染。
type statement struct {
	table      string
	model      *orm.ModelInfo
	selectCols []string
	distinct   bool
	conditions []condition
	order      []string
	group      []string
	having     string
	havingArgs []interface{}
	limit      int // -1 表示未设置
	offset     int
	rawSQL     string
	rawArgs    []interface{}
	errs       []error
}

func newStatement() *statement {
	return &statement{limit: -1}
}

func (s *statement) clone() *statement {
	ns := *s
	ns.selectCols = append([]string(nil), s.selectCols...)
	ns.conditions = append([]condition(nil), s.conditions...)
	ns.order = append([]string(nil), s.order...)
	ns.group = append([]string(nil), s.group...)
	ns.havingArgs = append([]interface{}(nil), s.havingArgs...)
	ns.rawArgs = append([]interface{}(nil), s.rawArgs...)
	ns.errs = append([]error(nil), s.errs...)
	return &ns
}

func (s *statement) addErr(err error) {
	s.errs = append(s.errs, err)
}

func (s *statement) err() error {
	if len(s.errs) == 0 {
		return nil
	}
	msgs := make([]string, 0, len(s.errs))
	for _, e := range s.errs {
		msgs = append(msgs, e.Error())
	}
	return fmt.Errorf("gormish: %s", strings.Join(msgs, "; "))
}

// splitCols 兼容 Select("a, b") / Select("a", "b") 两种写法。
func splitCols(cols []string) []string {
	out := make([]string, 0, len(cols))
	for _, c := range cols {
		for _, p := range strings.Split(c, ",") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

// addCondition 追加 WHERE 条件（G1）：query 支持 string（? 占位）与 map[string]interface{}（等值）。
func (s *statement) addCondition(query interface{}, args []interface{}, or bool) {
	switch q := query.(type) {
	case string:
		expr, expandArgs := expandInPlaceholders(q, args)
		s.conditions = append(s.conditions, condition{expr: expr, args: expandArgs, or: or})
	case map[string]interface{}:
		c := condition{or: or}
		keys := make([]string, 0, len(q))
		for k := range q {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			c.cols = append(c.cols, k)
			c.args = append(c.args, q[k])
		}
		s.conditions = append(s.conditions, c)
	default:
		s.addErr(fmt.Errorf("unsupported where condition type %T (want string or map[string]interface{})", query))
	}
}

// hasCondition 是否存在 WHERE 条件（Update/Delete 全表操作保护依据）。
func (s *statement) hasCondition() bool {
	return len(s.conditions) > 0
}

// expandInPlaceholders 展开 IN 占位（G1）：Where("id IN ?", ids) 中 slice 参数展开为 (?, ?, ...)。
// 空 slice 展开为 (NULL) 以保持 SQL 合法；[]byte 不展开（二进制值）。
func expandInPlaceholders(expr string, args []interface{}) (string, []interface{}) {
	if !strings.Contains(expr, "?") || len(args) == 0 {
		return expr, args
	}
	var out strings.Builder
	newArgs := make([]interface{}, 0, len(args))
	argIdx := 0
	for i := 0; i < len(expr); i++ {
		if expr[i] == '?' && argIdx < len(args) {
			arg := args[argIdx]
			argIdx++
			if isExpandableSlice(arg) {
				rv := reflect.ValueOf(arg)
				n := rv.Len()
				if n == 0 {
					out.WriteString("(NULL)")
					continue
				}
				out.WriteString("(")
				for j := 0; j < n; j++ {
					if j > 0 {
						out.WriteString(",")
					}
					out.WriteString("?")
					newArgs = append(newArgs, rv.Index(j).Interface())
				}
				out.WriteString(")")
				continue
			}
			newArgs = append(newArgs, arg)
			out.WriteByte(expr[i])
			continue
		}
		out.WriteByte(expr[i])
	}
	for ; argIdx < len(args); argIdx++ {
		newArgs = append(newArgs, args[argIdx])
	}
	return out.String(), newArgs
}

func isExpandableSlice(arg interface{}) bool {
	if arg == nil {
		return false
	}
	rv := reflect.ValueOf(arg)
	kind := rv.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		return false
	}
	return rv.Type().Elem().Kind() != reflect.Uint8
}

// buildWhere 组装 WHERE 子句（G1）：条件间默认 AND，Or 条件以 OR 连接；
// map 等值条件内部以 AND 连接并整体括号包裹。
func (s *statement) buildWhere() (string, []interface{}) {
	if len(s.conditions) == 0 {
		return "", nil
	}
	frags := make([]string, 0, len(s.conditions))
	var args []interface{}
	for i, c := range s.conditions {
		f := c.expr
		if f == "" {
			parts := make([]string, 0, len(c.cols))
			for _, col := range c.cols {
				parts = append(parts, col+" = ?")
			}
			f = "(" + strings.Join(parts, " AND ") + ")"
		}
		args = append(args, c.args...)
		switch {
		case i == 0:
			frags = append(frags, f)
		case c.or:
			frags = append(frags, "OR "+f)
		default:
			frags = append(frags, "AND "+f)
		}
	}
	return " WHERE " + strings.Join(frags, " "), args
}

// buildSelect 组装 SELECT 语句（G1）：不含分页子句，分页由调用方经 ApplyPagination 追加。
func (s *statement) buildSelect() (string, []interface{}) {
	cols := "*"
	if len(s.selectCols) > 0 {
		cols = strings.Join(s.selectCols, ", ")
	}
	var b strings.Builder
	b.WriteString("SELECT ")
	if s.distinct {
		b.WriteString("DISTINCT ")
	}
	b.WriteString(cols)
	b.WriteString(" FROM ")
	b.WriteString(s.table)
	whereSQL, args := s.buildWhere()
	b.WriteString(whereSQL)
	if len(s.group) > 0 {
		b.WriteString(" GROUP BY " + strings.Join(s.group, ", "))
	}
	if s.having != "" {
		b.WriteString(" HAVING " + s.having)
		args = append(args, s.havingArgs...)
	}
	if len(s.order) > 0 {
		b.WriteString(" ORDER BY " + strings.Join(s.order, ", "))
	}
	return b.String(), args
}

// buildCount 组装 COUNT 语句（G1）：DISTINCT 单列时生成 COUNT(DISTINCT col)。
func (s *statement) buildCount() (string, []interface{}) {
	cols := "*"
	if s.distinct && len(s.selectCols) == 1 {
		cols = "DISTINCT " + s.selectCols[0]
	}
	var b strings.Builder
	b.WriteString("SELECT COUNT(" + cols + ") FROM " + s.table)
	whereSQL, args := s.buildWhere()
	b.WriteString(whereSQL)
	return b.String(), args
}

// primaryColumn 主键列名：优先链上 Model 的显式主键，回退约定字段（id）。
func (s *statement) primaryColumn() string {
	if s.model != nil && s.model.PrimaryColumn != "" {
		return s.model.PrimaryColumn
	}
	if s.model != nil {
		for _, f := range s.model.Fields {
			if f.Column == "id" {
				return f.Column
			}
		}
	}
	return ""
}

// resolveModelInfo 解析模型元数据（G1）：
// 优先 orm.GetModelInfo（db tag / RegisterModel 缓存），无 tag 结构体回退本地推导
// （snake_case 列名 + TableName() 方法 + ID 字段主键约定），不改 orm.ParseModelInfo 的
// 「无 tag 返回 nil」语义（避免误触发 MP 内置 CRUD 生成）。
func resolveModelInfo(in interface{}) *orm.ModelInfo {
	if in == nil {
		return nil
	}
	rv := reflect.Indirect(reflect.ValueOf(in))
	if rv.Kind() != reflect.Struct {
		return nil
	}
	if info := orm.GetModelInfo(in); info != nil {
		return info
	}
	return fallbackModelInfo(rv.Type())
}

// fallbackModelInfo 无 db tag 结构体的本地元数据推导（G1）。
func fallbackModelInfo(typ reflect.Type) *orm.ModelInfo {
	info := &orm.ModelInfo{GoType: typ, Fields: []*orm.FieldInfo{}}
	if m := reflect.New(typ).Elem().MethodByName("TableName"); m.IsValid() {
		mt := m.Type()
		if mt.NumIn() == 0 && mt.NumOut() == 1 && mt.Out(0).Kind() == reflect.String {
			info.TableName = m.Call(nil)[0].String()
			info.ExplicitTable = true
		}
	}
	if !info.ExplicitTable {
		info.TableName = snakeCase(trimModelSuffix(typ.Name()))
	}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.PkgPath != "" {
			continue
		}
		col := ""
		primary := false
		if tag, ok := f.Tag.Lookup("db"); ok {
			parts := strings.Split(tag, ",")
			col = strings.TrimSpace(parts[0])
			if col == "-" {
				continue
			}
			for _, opt := range parts[1:] {
				if strings.TrimSpace(opt) == "pk" {
					primary = true
				}
			}
		}
		if col == "" {
			col = snakeCase(f.Name)
		}
		fi := &orm.FieldInfo{Name: f.Name, Column: col, Type: f.Type, Primary: primary}
		if !fi.Primary && (f.Name == "ID" || f.Name == "Id") {
			fi.Primary = true
		}
		info.Fields = append(info.Fields, fi)
		if fi.Primary && info.PrimaryColumn == "" {
			info.PrimaryColumn = fi.Column
		}
	}
	return info
}

// insertColumns 组装插入列与参数（G1）：跳过零值自增主键、逻辑删除列、db:"-" 字段
// （与 MP 内置 CRUD 生成一致）；返回被跳过的零值主键字段（供自增回填），无则 nil。
func (s *statement) insertColumns(rv reflect.Value) ([]string, []interface{}, *orm.FieldInfo) {
	if s.model == nil {
		return nil, nil, nil
	}
	var cols []string
	var args []interface{}
	var pkField *orm.FieldInfo
	for _, f := range s.model.Fields {
		if f.Logic {
			continue
		}
		fv := rv.FieldByName(f.Name)
		if !fv.IsValid() {
			continue
		}
		if f.Primary && fv.IsZero() {
			pkField = f
			continue
		}
		cols = append(cols, f.Column)
		args = append(args, fv.Interface())
	}
	return cols, args, pkField
}

// insertColumnsFor 按给定列集提取单行参数（批量插入，G1）；
// 主键列出现零值（与批量首行非零主键不一致）时返回 nil，由调用方报错。
func (s *statement) insertColumnsFor(rv reflect.Value, cols []string) []interface{} {
	if s.model == nil {
		return nil
	}
	colMap := make(map[string]*orm.FieldInfo, len(s.model.Fields))
	for _, f := range s.model.Fields {
		colMap[f.Column] = f
	}
	args := make([]interface{}, 0, len(cols))
	for _, c := range cols {
		f, ok := colMap[c]
		if !ok {
			return nil
		}
		fv := rv.FieldByName(f.Name)
		if !fv.IsValid() {
			return nil
		}
		if f.Primary && fv.IsZero() {
			return nil
		}
		args = append(args, fv.Interface())
	}
	return args
}

func trimModelSuffix(name string) string {
	return strings.TrimSuffix(name, "Model")
}

// snakeCase CamelCase → snake_case（本地实现，与 orm.camelToSnake 语义一致，G1）。
func snakeCase(name string) string {
	runes := []rune(name)
	var b strings.Builder
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLower(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
