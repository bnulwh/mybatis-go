package orm

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/bnulwh/mybatis-go/types/sqlfragment"
)

// QueryWrapper 条件构造器（P0-1，MyBatis-Plus QueryWrapper 风格）：
// 字符串列名 + 链式调用；条件值内联渲染，转义走 sqlfragment.FormatValue
// （与框架 ${} 值内联行为单一事实源）。
//
// 用法一（自动注入）：mapper 方法声明 *QueryWrapper 参数（可与 *PageParam 组合），
// 框架在执行前把条件注入 SQL——已有 WHERE 追加 AND (...)，否则追加 WHERE (...)；
// 语句尾部已有 ORDER BY/LIMIT 时条件插入在其之前。
//
// 用法二（显式 ${ew}）：XML 写 `where ${ew}`，Wrapper 片段按占位符渲染。
//
// 注：列名与原生片段（Apply/Having/Last）直接拼接，调用方自行防止注入；
// 值虽经转义，仍不建议直接拼接用户输入（与框架 ${} 现状一致的已知限制）。
type QueryWrapper struct {
	conds      []wrapperCond
	selectCols []string
	groupCols  []string
	havingSQL  string
	orderBys   []string
	lastSQL    string
	nextOr     bool
}

// wrapperCond 单个条件片段；or 为 true 时与前一条件用 OR 连接（默认 AND）。
type wrapperCond struct {
	sql string
	or  bool
}

// queryWrapperType Wrapper 参数类型（NewProxyArg 提取检测用，照搬 pageParamType 模式）。
var queryWrapperType = reflect.TypeOf((*QueryWrapper)(nil))

// NewQueryWrapper 创建空 Wrapper。
func NewQueryWrapper() *QueryWrapper {
	return &QueryWrapper{}
}

func (w *QueryWrapper) addCond(sql string) *QueryWrapper {
	w.conds = append(w.conds, wrapperCond{sql: sql, or: w.nextOr})
	w.nextOr = false
	return w
}

// Or 下一个条件用 OR 连接（默认 AND）。例：w.Eq("a",1).Or().Eq("b",2) → a = 1 OR b = 2。
func (w *QueryWrapper) Or() *QueryWrapper {
	w.nextOr = true
	return w
}

// Eq 等值：col = val。
func (w *QueryWrapper) Eq(col string, val interface{}) *QueryWrapper {
	return w.addCond(col + " = " + sqlfragment.FormatValue(val))
}

// Ne 不等：col <> val。
func (w *QueryWrapper) Ne(col string, val interface{}) *QueryWrapper {
	return w.addCond(col + " <> " + sqlfragment.FormatValue(val))
}

// Gt 大于：col > val。
func (w *QueryWrapper) Gt(col string, val interface{}) *QueryWrapper {
	return w.addCond(col + " > " + sqlfragment.FormatValue(val))
}

// Ge 大于等于：col >= val。
func (w *QueryWrapper) Ge(col string, val interface{}) *QueryWrapper {
	return w.addCond(col + " >= " + sqlfragment.FormatValue(val))
}

// Lt 小于：col < val。
func (w *QueryWrapper) Lt(col string, val interface{}) *QueryWrapper {
	return w.addCond(col + " < " + sqlfragment.FormatValue(val))
}

// Le 小于等于：col <= val。
func (w *QueryWrapper) Le(col string, val interface{}) *QueryWrapper {
	return w.addCond(col + " <= " + sqlfragment.FormatValue(val))
}

// Like 模糊：col LIKE '%val%'。
func (w *QueryWrapper) Like(col string, val interface{}) *QueryWrapper {
	return w.addCond(col + " LIKE " + likePattern(val, true, true))
}

// NotLike 反向模糊：col NOT LIKE '%val%'。
func (w *QueryWrapper) NotLike(col string, val interface{}) *QueryWrapper {
	return w.addCond(col + " NOT LIKE " + likePattern(val, true, true))
}

// LikeLeft 左模糊：col LIKE '%val'。
func (w *QueryWrapper) LikeLeft(col string, val interface{}) *QueryWrapper {
	return w.addCond(col + " LIKE " + likePattern(val, true, false))
}

// LikeRight 右模糊：col LIKE 'val%'。
func (w *QueryWrapper) LikeRight(col string, val interface{}) *QueryWrapper {
	return w.addCond(col + " LIKE " + likePattern(val, false, true))
}

// Between 区间：col BETWEEN v1 AND v2。
func (w *QueryWrapper) Between(col string, v1, v2 interface{}) *QueryWrapper {
	return w.addCond(fmt.Sprintf("%s BETWEEN %s AND %s", col, sqlfragment.FormatValue(v1), sqlfragment.FormatValue(v2)))
}

// NotBetween 反向区间：col NOT BETWEEN v1 AND v2。
func (w *QueryWrapper) NotBetween(col string, v1, v2 interface{}) *QueryWrapper {
	return w.addCond(fmt.Sprintf("%s NOT BETWEEN %s AND %s", col, sqlfragment.FormatValue(v1), sqlfragment.FormatValue(v2)))
}

// IsNull 空判断：col IS NULL。
func (w *QueryWrapper) IsNull(col string) *QueryWrapper {
	return w.addCond(col + " IS NULL")
}

// IsNotNull 非空判断：col IS NOT NULL。
func (w *QueryWrapper) IsNotNull(col string) *QueryWrapper {
	return w.addCond(col + " IS NOT NULL")
}

// In 集合：col IN (v1, v2, ...)；空集合恒假（1=0）。
func (w *QueryWrapper) In(col string, vals ...interface{}) *QueryWrapper {
	if len(vals) == 0 {
		return w.addCond("1=0")
	}
	return w.addCond(col + " IN (" + joinWrapperValues(vals) + ")")
}

// NotIn 反向集合：col NOT IN (v1, v2, ...)；空集合恒真（1=1）。
func (w *QueryWrapper) NotIn(col string, vals ...interface{}) *QueryWrapper {
	if len(vals) == 0 {
		return w.addCond("1=1")
	}
	return w.addCond(col + " NOT IN (" + joinWrapperValues(vals) + ")")
}

// Nested 括号嵌套组：fn 内的条件整体加括号，作为一个条件参与 AND/OR 连接。
func (w *QueryWrapper) Nested(fn func(*QueryWrapper)) *QueryWrapper {
	sub := NewQueryWrapper()
	fn(sub)
	if seg := sub.conditionSQL(); seg != "" {
		w.addCond("(" + seg + ")")
	}
	return w
}

// Apply 原生 SQL 片段，作为一个条件 AND/OR 拼接。
func (w *QueryWrapper) Apply(rawSQL string) *QueryWrapper {
	return w.addCond(rawSQL)
}

// GroupBy 分组列。
func (w *QueryWrapper) GroupBy(cols ...string) *QueryWrapper {
	w.groupCols = append(w.groupCols, cols...)
	return w
}

// Having 分组过滤（原生片段）。
func (w *QueryWrapper) Having(rawSQL string) *QueryWrapper {
	w.havingSQL = rawSQL
	return w
}

// OrderByAsc 升序排序。
func (w *QueryWrapper) OrderByAsc(cols ...string) *QueryWrapper {
	for _, c := range cols {
		w.orderBys = append(w.orderBys, c+" ASC")
	}
	return w
}

// OrderByDesc 降序排序。
func (w *QueryWrapper) OrderByDesc(cols ...string) *QueryWrapper {
	for _, c := range cols {
		w.orderBys = append(w.orderBys, c+" DESC")
	}
	return w
}

// Last 原生片段直接拼接到 SQL 末尾（LIMIT 等），恒置最后。
func (w *QueryWrapper) Last(rawSQL string) *QueryWrapper {
	w.lastSQL = rawSQL
	return w
}

// Select 指定查询列（GetSQLSelectSegment 输出，显式 ${ew} 场景配合使用；
// 自动注入模式不改写原 SQL 的 select 列清单）。
func (w *QueryWrapper) Select(cols ...string) *QueryWrapper {
	w.selectCols = append(w.selectCols, cols...)
	return w
}

// GetSQLSelectSegment 查询列片段："c1, c2"（未指定返回空串）。
func (w *QueryWrapper) GetSQLSelectSegment() string {
	return strings.Join(w.selectCols, ", ")
}

// conditionSQL 仅条件部分（无 WHERE 前缀、无 GROUP/ORDER 等子句）。
func (w *QueryWrapper) conditionSQL() string {
	var sb strings.Builder
	for i, c := range w.conds {
		if i > 0 {
			if c.or {
				sb.WriteString(" OR ")
			} else {
				sb.WriteString(" AND ")
			}
		}
		sb.WriteString(c.sql)
	}
	return sb.String()
}

// clauseSQL GROUP BY / HAVING / ORDER BY 子句部分（不含 last）。
func (w *QueryWrapper) clauseSQL() string {
	var parts []string
	if len(w.groupCols) > 0 {
		parts = append(parts, "GROUP BY "+strings.Join(w.groupCols, ", "))
	}
	if w.havingSQL != "" {
		parts = append(parts, "HAVING "+w.havingSQL)
	}
	if len(w.orderBys) > 0 {
		parts = append(parts, "ORDER BY "+strings.Join(w.orderBys, ", "))
	}
	return strings.Join(parts, " ")
}

// GetSQLSegment 完整片段："c1 = v1 AND (c2 > v2 OR c3 IS NULL) ORDER BY ..."
// （无 WHERE 前缀）。实现 sqlfragment.SQLSegmentProvider，${ew} 渲染即此值。
//
// 值接收者：单 map 参数路径经 convert2Map/safeIndirectInterface 解引用为
// QueryWrapper 值类型，值/指针两形态均须满足 SQLSegmentProvider 接口。
func (w QueryWrapper) GetSQLSegment() string {
	var segs []string
	if c := w.conditionSQL(); c != "" {
		segs = append(segs, c)
	}
	if cl := w.clauseSQL(); cl != "" {
		segs = append(segs, cl)
	}
	if w.lastSQL != "" {
		segs = append(segs, w.lastSQL)
	}
	return strings.Join(segs, " ")
}

// GetSQLWhereSegment 带 "WHERE " 前缀版本（空 Wrapper 返回空串）。
func (w *QueryWrapper) GetSQLWhereSegment() string {
	seg := w.GetSQLSegment()
	if seg == "" {
		return ""
	}
	return "WHERE " + seg
}

// likePattern 模糊匹配值：字符串转义同 formatString（单引号→双引号），
// 按 MP 语义补 % 通配（left → 前置 %，right → 后置 %）。
func likePattern(v interface{}, left, right bool) string {
	var s string
	if str, ok := v.(string); ok {
		s = strings.ReplaceAll(str, "'", "\"")
	} else {
		s = fmt.Sprintf("%v", v)
	}
	if left {
		s = "%" + s
	}
	if right {
		s = s + "%"
	}
	return "'" + s + "'"
}

// joinWrapperValues 集合值内联渲染：v1, v2, ...
func joinWrapperValues(vals []interface{}) string {
	parts := make([]string, 0, len(vals))
	for _, v := range vals {
		parts = append(parts, sqlfragment.FormatValue(v))
	}
	return strings.Join(parts, ", ")
}
