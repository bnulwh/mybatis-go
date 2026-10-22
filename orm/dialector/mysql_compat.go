package dialector

import (
	"strings"

	"github.com/bnulwh/mybatis-go/log"
)

// RewriteMySqlCompat 将 MySQL 方言的函数/标识符改写为 PG 族等价形式（F1）：
//
//	ifnull(a, b)             → coalesce(a, b)               （仅改函数名，参数原样）
//	find_in_set(x, csv)      → position(','||x||',' in ','||csv||',')
//	date_format(d, 'fmt')    → to_char(d, 'fmt2')            （MySQL 格式符按映射表转 PG 格式）
//	`ident`                  → "ident"                        （反引号标识符 → PG 双引号标识符）
//
// 特性：
//   - 引号（'/"，含 ”/""/“ 转义）与注释（-- 行注释、/* */ 块注释）内不改写；
//   - 函数名大小写不敏感、要求独立词边界（不误伤 ifnull_x 之类标识符）；
//   - find_in_set/date_format 参数支持嵌套（参数先递归改写再嵌入）；
//   - 改写可重入：已改写结果再跑一遍不变（ifnull→coalesce 后不再命中）。
//
// 已知边界（保守取舍）：
//   - find_in_set 返回值不等于 MySQL 的 1 基序号（position 命中位置含前导逗号偏移），
//     适用于 `> 0` 存在性判断（共享 XML 的常见用法），不做序号语义对齐；
//   - date_format 第二参数必须是字符串字面量才改写（绑定参数/表达式跳过并告警）；
//   - `col = 0`（varchar 列）与 `col = '0'` 的类型差异不做推断改写。
func RewriteMySqlCompat(sql string) string {
	if !needsMySqlCompatRewrite(sql) {
		return sql
	}
	var b strings.Builder
	b.Grow(len(sql))
	i := 0
	for i < len(sql) {
		c := sql[i]
		switch c {
		case '\'':
			copyQuoted(&b, sql, &i, '\'') // 字符串字面量（'' 转义），原样保留
		case '"':
			copyQuoted(&b, sql, &i, '"') // 双引号（MySQL 字符串/PG 标识符），原样保留
		case '`':
			i = rewriteBacktickIdent(&b, sql, i)
		case '-':
			if i+1 < len(sql) && sql[i+1] == '-' {
				copyLineComment(&b, sql, &i)
			} else {
				b.WriteByte(c)
				i++
			}
		case '/':
			if i+1 < len(sql) && sql[i+1] == '*' {
				copyBlockComment(&b, sql, &i)
			} else {
				b.WriteByte(c)
				i++
			}
		default:
			if isWordByte(c) && !isWordByte(prevByte(sql, i)) {
				j := i
				for j < len(sql) && isWordByte(sql[j]) {
					j++
				}
				i = rewriteWordAt(&b, sql, i, j)
			} else {
				b.WriteByte(c)
				i++
			}
		}
	}
	return b.String()
}

// needsMySqlCompatRewrite 快速预检：不含反引号且不含任一目标函数名（大小写不敏感）时跳过。
func needsMySqlCompatRewrite(sql string) bool {
	if strings.IndexByte(sql, '`') >= 0 {
		return true
	}
	lower := strings.ToLower(sql)
	return strings.Contains(lower, "ifnull") ||
		strings.Contains(lower, "find_in_set") ||
		strings.Contains(lower, "date_format")
}

// rewriteWordAt 处理位于 [i, j) 的独立单词：命中目标函数则改写并返回新扫描位置，否则原样拷贝。
func rewriteWordAt(b *strings.Builder, sql string, i, j int) int {
	word := strings.ToLower(sql[i:j])
	switch word {
	case "ifnull":
		if lp, ok := peekParen(sql, j); ok {
			b.WriteString("coalesce")
			return lp // 括号体交回主循环，参数内嵌套改写继续
		}
	case "find_in_set":
		if args, end, ok := parseCallArgs(sql, j); ok && len(args) == 2 {
			a0, a1 := RewriteMySqlCompat(args[0]), RewriteMySqlCompat(args[1])
			b.WriteString("position(','||" + a0 + "||',' in ','||" + a1 + "||',')")
			return end
		}
	case "date_format":
		if args, end, ok := parseCallArgs(sql, j); ok && len(args) == 2 {
			if fmt2, ok := convertDateFormatLiteral(args[1]); ok {
				a0 := RewriteMySqlCompat(args[0])
				b.WriteString("to_char(" + a0 + ", " + fmt2 + ")")
				return end
			}
			log.Warnf("mysql-compat: date_format second arg is not a string literal, keep as-is: %s", sql[i:end])
		}
	}
	b.WriteString(sql[i:j])
	return j
}

// peekParen 跳过空白后判断 j 起是否为 '('，是则返回其下标。
func peekParen(sql string, j int) (int, bool) {
	for j < len(sql) && isSQLSpace(sql[j]) {
		j++
	}
	if j < len(sql) && sql[j] == '(' {
		return j, true
	}
	return j, false
}

// parseCallArgs 从 j（跳过空白后为 '('）解析平衡括号内的顶层逗号分隔参数：
// 引号/注释/嵌套括号内的逗号不切分。返回参数原文（两侧空白已去）与闭括号后的下标。
func parseCallArgs(sql string, j int) ([]string, int, bool) {
	j, _ = peekParen(sql, j)
	if j >= len(sql) || sql[j] != '(' {
		return nil, j, false
	}
	depth := 0
	start := j + 1
	var args []string
	for i := j; i < len(sql); {
		switch c := sql[i]; c {
		case '\'':
			i = skipQuoted(sql, i, '\'')
		case '"':
			i = skipQuoted(sql, i, '"')
		case '`':
			i = skipQuoted(sql, i, '`')
		case '-':
			if i+1 < len(sql) && sql[i+1] == '-' {
				i = skipLineComment(sql, i)
			} else {
				i++
			}
		case '/':
			if i+1 < len(sql) && sql[i+1] == '*' {
				i = skipBlockComment(sql, i)
			} else {
				i++
			}
		case '(':
			depth++
			i++
		case ')':
			depth--
			if depth == 0 {
				args = appendArg(args, sql[start:i])
				return args, i + 1, true
			}
			i++
		case ',':
			if depth == 1 {
				args = appendArg(args, sql[start:i])
				start = i + 1
			}
			i++
		default:
			i++
		}
	}
	return nil, j, false // 未闭合，原样保留
}

func appendArg(args []string, s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return args
	}
	return append(args, s)
}

// convertDateFormatLiteral 将 date_format 第二参数（须为 '...' 字符串字面量）内的
// MySQL 格式符转为 PG to_char 格式，返回带引号的字面量；非字面量返回 false。
func convertDateFormatLiteral(arg string) (string, bool) {
	arg = strings.TrimSpace(arg)
	if len(arg) < 2 || !strings.HasPrefix(arg, "'") || !strings.HasSuffix(arg, "'") {
		return arg, false
	}
	inner := arg[1 : len(arg)-1]
	var b strings.Builder
	b.Grow(len(inner))
	i := 0
	for i < len(inner) {
		c := inner[i]
		if c == '\'' && i+1 < len(inner) && inner[i+1] == '\'' { // '' 转义原样保留
			b.WriteString("''")
			i += 2
			continue
		}
		if c != '%' {
			b.WriteByte(c)
			i++
			continue
		}
		i++
		if i >= len(inner) { // 尾部孤立 %
			b.WriteByte('%')
			break
		}
		b.WriteString(convertFormatSpecifier(inner[i]))
		i++
	}
	return "'" + b.String() + "'", true
}

// mysqlToPgFormats MySQL date_format 格式符 → PG to_char 格式（F1）。
// 复合格式（%T/%r）直接展开；%D（带序数后缀的天）无 PG 对应，原样保留由调用方兜底。
var mysqlToPgFormats = map[byte]string{
	'Y': "YYYY", 'y': "YY",
	'm': "MM", 'c': "MM",
	'd': "DD", 'e': "DD",
	'H': "HH24", 'k': "HH24",
	'h': "HH12", 'I': "HH12", 'l': "HH12",
	'i': "MI",
	's': "SS", 'S': "SS",
	'f': "US",
	'p': "AM",
	'W': "Day", 'a': "Dy",
	'M': "Month", 'b': "Mon",
	'j': "DDD",
	'T': "HH24:MI:SS",
	'r': "HH12:MI:SS AM",
	'%': "%",
}

func convertFormatSpecifier(c byte) string {
	if v, ok := mysqlToPgFormats[c]; ok {
		return v
	}
	return "%" + string(c) // 未识别的 %x 原样保留（PG to_char 对未知字符按字面拷贝）
}

// rewriteBacktickIdent 将 MySQL 反引号标识符转为 PG 双引号标识符（“ → ` 转义）；
// 未闭合时原样保留剩余部分。返回新的扫描位置。
func rewriteBacktickIdent(b *strings.Builder, sql string, i int) int {
	j := i + 1
	var name strings.Builder
	closed := false
	for j < len(sql) {
		if sql[j] == '`' {
			if j+1 < len(sql) && sql[j+1] == '`' { // `` 转义为字面反引号
				name.WriteByte('`')
				j += 2
				continue
			}
			j++
			closed = true
			break
		}
		name.WriteByte(sql[j])
		j++
	}
	if !closed {
		b.WriteString(sql[i:j])
		return j
	}
	b.WriteString(`"` + name.String() + `"`)
	return j
}

// ---- 以下为跳过/拷贝辅助（与 sql_scan.go 的扫描规则保持一致） ----

func copyQuoted(b *strings.Builder, sql string, i *int, q byte) {
	j := skipQuoted(sql, *i, q)
	b.WriteString(sql[*i:j])
	*i = j
}

func skipQuoted(sql string, i int, q byte) int {
	i++ // 起始引号
	for i < len(sql) {
		if sql[i] == q {
			if i+1 < len(sql) && sql[i+1] == q { // ''/""/`` 转义
				i += 2
				continue
			}
			return i + 1
		}
		i++
	}
	return len(sql) // 未闭合
}

func copyLineComment(b *strings.Builder, sql string, i *int) {
	j := skipLineComment(sql, *i)
	b.WriteString(sql[*i:j])
	*i = j
}

func skipLineComment(sql string, i int) int {
	for i < len(sql) && sql[i] != '\n' {
		i++
	}
	return i // 换行符留给主循环（保持行结构）
}

func copyBlockComment(b *strings.Builder, sql string, i *int) {
	j := skipBlockComment(sql, *i)
	b.WriteString(sql[*i:j])
	*i = j
}

func skipBlockComment(sql string, i int) int {
	i += 2
	for i+1 < len(sql) && !(sql[i] == '*' && sql[i+1] == '/') {
		i++
	}
	if i+1 < len(sql) {
		return i + 2
	}
	return len(sql)
}
