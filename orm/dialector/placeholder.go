package dialector

import (
	"fmt"
	"strings"
)

// 字面量 / 注释感知的 ? 占位符改写（P0-1）。
//
// 背景：旧实现（formatDollarPlaceholders / formatColonPlaceholders /
// formatAtPPlaceholders）直接 strings.Split(src, "?") 后按出现次序编号，
// 把字符串字面量、引号标识符、行/块注释、Postgres 美元引用字符串里的 `?`
// 也当成占位符，也会把 jsonb 的 `?` / `?|` / `?&` 操作符吃掉：
//
//	SELECT * FROM t WHERE c = '?' AND id = ?   -- 旧行为：c = '$1' and id = $2（错位）
//	SELECT * FROM t WHERE payload ? 'k' AND id = ? -- 旧行为：payload $1 'k'（语法错）
//	SELECT * FROM t WHERE a = ? -- 这行注释里的 ? 也计数
//
// 本实现逐字符扫描，只在「代码区」把 `?` 替换为方言占位符（$n / :n / @pn），
// 字面量与注释原样复制；代码区连续空白折叠为单个空格（保证预编译语句缓存键稳定，
// 且不再修改字面量内部的空白/换行）。

// rewritePlaceholders 按方言风格改写 ? 占位符（PlaceholderQuestion 直接返回空白归一后的 SQL）。
func rewritePlaceholders(src string, style PlaceholderStyle) string {
	if len(src) == 0 {
		return src
	}
	var b strings.Builder
	b.Grow(len(src) + 8)
	n := 0 // 已生成的占位符序号（从 1 开始）
	i := 0
	for i < len(src) {
		c := src[i]
		switch {
		case isSQLSpace(c):
			for i < len(src) && isSQLSpace(src[i]) {
				i++
			}
			b.WriteByte(' ')
		case c == '\'' || c == '"' || c == '`':
			end := scanQuotedLiteral(src, i, c)
			b.WriteString(src[i:end])
			i = end
		case c == '[':
			// MSSQL / DB2 方括号标识符
			end := i + 1
			for end < len(src) && src[end] != ']' {
				end++
			}
			if end < len(src) {
				end++
			}
			b.WriteString(src[i:end])
			i = end
		case c == '-' && i+1 < len(src) && src[i+1] == '-':
			// 行注释：-- 到行尾（连同行尾换行一起复制，避免归一化后注释吞掉后续 SQL）
			end := i
			for end < len(src) && src[end] != '\n' {
				end++
			}
			if end < len(src) {
				end++
			}
			b.WriteString(src[i:end])
			i = end
		case c == '#' && style == PlaceholderQuestion:
			// MySQL 行注释：# 到行尾（仅 MySQL 族；PG 的 #> / #>> jsonb 操作符不受影响）
			end := i
			for end < len(src) && src[end] != '\n' && src[end] != '\r' {
				end++
			}
			if end < len(src) {
				end++
			}
			b.WriteString(src[i:end])
			i = end
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			// 块注释 / Oracle hint：原样保留（hint 有语义，不能删）
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				b.WriteString(src[i:])
				i = len(src)
			} else {
				end = i + 2 + end + 2
				b.WriteString(src[i:end])
				i = end
			}
		case c == '$':
			if end, ok := scanDollarQuoted(src, i); ok {
				// Postgres 美元引用字符串：$$...$$ / $tag$...$tag$
				b.WriteString(src[i:end])
				i = end
			} else {
				b.WriteByte(c)
				i++
			}
		case c == '?':
			if isJSONQuestionOp(src, i) {
				// jsonb 操作符：? / ?| / ?&，不是占位符
				b.WriteByte(c)
				i++
				continue
			}
			n++
			b.WriteString(placeholderAt(style, n))
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// placeholderAt 返回第 n 个占位符的方言写法。
func placeholderAt(style PlaceholderStyle, n int) string {
	switch style {
	case PlaceholderDollar:
		return fmt.Sprintf("$%d", n)
	case PlaceholderColon:
		return fmt.Sprintf(":%d", n)
	case PlaceholderAtP:
		return fmt.Sprintf("@p%d", n)
	default:
		return "?"
	}
}

// scanQuotedLiteral 返回从 i（src[i] 为引号）开始的字面量/引号标识符结束位置（不含）。
// 支持成对引号转义（单引号/双引号/反引号两两重复时表示转义）；E'...' 形式额外支持反斜杠转义。
func scanQuotedLiteral(src string, i int, quote byte) int {
	backslashEscape := quote == '\'' && i > 0 && (src[i-1] == 'E' || src[i-1] == 'e') &&
		(i == 1 || !isWordByte(src[i-2]))
	j := i + 1
	for j < len(src) {
		c := src[j]
		if backslashEscape && c == '\\' && j+1 < len(src) {
			j += 2
			continue
		}
		if c == quote {
			if j+1 < len(src) && src[j+1] == quote {
				j += 2
				continue
			}
			return j + 1
		}
		j++
	}
	return len(src)
}

// scanDollarQuoted 检测 i 处的 Postgres 美元引用字符串（$$...$$ / $tag$...$tag$）。
// 返回结束位置（不含）与是否命中。
func scanDollarQuoted(src string, i int) (int, bool) {
	if i+1 >= len(src) {
		return 0, false
	}
	if src[i+1] == '$' {
		end := strings.Index(src[i+2:], "$$")
		if end < 0 {
			return 0, false
		}
		return i + 2 + end + 2, true
	}
	j := i + 1
	if j >= len(src) || !isDollarTagByte(src[j]) || (src[j] >= '0' && src[j] <= '9') {
		return 0, false
	}
	for j < len(src) && isDollarTagByte(src[j]) {
		j++
	}
	if j >= len(src) || src[j] != '$' {
		return 0, false
	}
	closer := src[i : j+1]
	pos := strings.Index(src[j+1:], closer)
	if pos < 0 {
		return 0, false
	}
	return j + 1 + pos + len(closer), true
}

// isDollarTagByte 美元引用标签字符（字母/数字/下划线）。
func isDollarTagByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

// isJSONQuestionOp 判定 i 处的 ? 是 Postgres jsonb 操作符（? / ?| / ?&）而非占位符：
//   - 紧跟 '|' 或 '&' → 组合操作符，无需再看上下文；
//   - 左右相邻（跳空白）均为操作数（标识符字符 / ) / ] / 引号）→ 中缀操作符。
//
// 真实占位符左侧一定是运算符或括号、右侧是表达式分隔符，不会被误判。
func isJSONQuestionOp(src string, i int) bool {
	if i+1 < len(src) && (src[i+1] == '|' || src[i+1] == '&') {
		return true
	}
	prev := prevNonSpaceByte(src, i-1)
	next := nextNonSpaceByte(src, i+1)
	if prev == 0 || next == 0 {
		return false
	}
	prevOperand := isWordByte(prev) || prev == ')' || prev == ']' || prev == '\'' || prev == '"' || prev == '`'
	if !prevOperand {
		return false
	}
	return isWordByte(next) || next == '\'' || next == '"' || next == '$'
}

func prevNonSpaceByte(src string, i int) byte {
	for ; i >= 0; i-- {
		if !isSQLSpace(src[i]) {
			return src[i]
		}
	}
	return 0
}

func nextNonSpaceByte(src string, i int) byte {
	for ; i < len(src); i++ {
		if !isSQLSpace(src[i]) {
			return src[i]
		}
	}
	return 0
}

// normalizeSQLSpace 只做代码区的空白归一（字面量/注释原样保留）。
func normalizeSQLSpace(src string) string {
	return rewritePlaceholders(src, PlaceholderQuestion)
}
