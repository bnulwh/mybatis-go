package dialector

import "strings"

// 括号/引号/注释深度感知的「顶层子句」扫描器（B1/B2 修复）。
//
// 背景：buildCountSQL / ApplyPagination 此前用全局正则清洗 LIMIT/OFFSET/ORDER BY，
// 会误伤子查询内层（如 `... (SELECT ... LIMIT 1) AS orgName` 的 LIMIT 1 被删）。
// 本扫描器只识别「括号深度 0、引号与注释之外」的关键字，子查询/窗口函数/
// 字符串字面量里的同名关键字不受影响。
//
// 关键字在顶层（深度 0）的合法出现位置只有语句尾部子句链（ORDER BY / LIMIT /
// OFFSET / RETURNING / FOR UPDATE 等），因此「最左顶层命中点」即尾部子句链起点。

// TopLevelClausePos 返回 sql 中第一个位于顶层的关键字起始下标；未命中返回 -1。
//
// keywords 传小写短语，可含空格（如 "order by"），短语内的空格匹配 sql 中任意
// 长度空白（含换行）；匹配大小写不敏感，要求前后均为非单词字符（[0-9A-Za-z_$]），
// 避免命中 limit_x、order_by 之类标识符。
func TopLevelClausePos(sql string, keywords ...string) int {
	if len(sql) == 0 || len(keywords) == 0 {
		return -1
	}
	kws := make([]string, 0, len(keywords))
	for _, k := range keywords {
		k = strings.ToLower(strings.TrimSpace(k))
		if k != "" {
			kws = append(kws, k)
		}
	}
	if len(kws) == 0 {
		return -1
	}
	depth := 0
	for i := 0; i < len(sql); {
		switch c := sql[i]; c {
		case '\'': // 字符串字面量（'' 转义）
			i++
			for i < len(sql) {
				if sql[i] == '\'' {
					if i+1 < len(sql) && sql[i+1] == '\'' {
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
		case '"', '`': // 引号标识符（"" / `` 转义）
			i++
			for i < len(sql) {
				if sql[i] == c {
					if i+1 < len(sql) && sql[i+1] == c {
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
		case '[': // SQL Server 方括号标识符
			i++
			for i < len(sql) && sql[i] != ']' {
				i++
			}
			if i < len(sql) {
				i++
			}
		case '(':
			depth++
			i++
		case ')':
			if depth > 0 {
				depth--
			}
			i++
		case '-':
			if i+1 < len(sql) && sql[i+1] == '-' { // 行注释
				for i < len(sql) && sql[i] != '\n' {
					i++
				}
			} else {
				i++
			}
		case '/':
			if i+1 < len(sql) && sql[i+1] == '*' { // 块注释
				i += 2
				for i+1 < len(sql) && !(sql[i] == '*' && sql[i+1] == '/') {
					i++
				}
				if i+1 < len(sql) {
					i += 2
				} else {
					i = len(sql)
				}
			} else {
				i++
			}
		default:
			if depth == 0 && isWordByte(c) {
				if !isWordByte(prevByte(sql, i)) { // 单词起点才尝试匹配关键字
					for _, k := range kws {
						if matchKeywordAt(sql, i, k) {
							return i
						}
					}
				}
				for i < len(sql) && isWordByte(sql[i]) { // 跳过整个标识符
					i++
				}
			} else {
				i++
			}
		}
	}
	return -1
}

// HasTopLevelClause 判断 sql 顶层是否出现任一关键字
// （B4：PageParam 与手写 LIMIT/OFFSET 共存诊断）。
func HasTopLevelClause(sql string, keywords ...string) bool {
	return TopLevelClausePos(sql, keywords...) >= 0
}

// ContainsConcatOp 报告 sql 在引号（'/"、含成对转义）与注释（--、/* */）之外
// 是否包含 || 操作符（F2：跨模式语义差异检测）。
func ContainsConcatOp(sql string) bool {
	n := len(sql)
	for i := 0; i < n; {
		c := sql[i]
		switch {
		case c == '\'' || c == '"' || c == '`':
			q := c
			i++
			for i < n {
				if sql[i] == q {
					if i+1 < n && sql[i+1] == q { // '' "" `` 转义
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
		case c == '-' && i+1 < n && sql[i+1] == '-':
			for i < n && sql[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < n && sql[i+1] == '*':
			i += 2
			for i+1 < n && !(sql[i] == '*' && sql[i+1] == '/') {
				i++
			}
			if i+1 < n {
				i += 2
			} else {
				i = n
			}
		case c == '|' && i+1 < n && sql[i+1] == '|':
			return true
		default:
			i++
		}
	}
	return false
}

// StripTopLevelTailClauses 从第一个顶层关键字处截断 sql（含其后全部内容），
// 并去掉截断点前的尾部空白与分号。
func StripTopLevelTailClauses(sql string, keywords ...string) string {
	pos := TopLevelClausePos(sql, keywords...)
	if pos < 0 {
		return sql
	}
	return strings.TrimRight(sql[:pos], " \t\n\r;")
}

// TopLevelKeywordEnd 返回顶层第一个 keyword 结束后再跳过一段空白的位置；
// 未命中返回 -1。供 buildCountSQL 定位 SELECT 列表替换边界使用。
func TopLevelKeywordEnd(sql string, keyword string) int {
	pos := TopLevelClausePos(sql, keyword)
	if pos < 0 {
		return -1
	}
	i := pos + len(keyword)
	for i < len(sql) && isSQLSpace(sql[i]) {
		i++
	}
	return i
}

// IsSQLSpace 导出空白判断（orm 包截 FROM 前空白用）。
func IsSQLSpace(b byte) bool {
	return isSQLSpace(b)
}

func isSQLSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func isWordByte(b byte) bool {
	return b == '_' || b == '$' ||
		(b >= '0' && b <= '9') ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z')
}

func prevByte(sql string, i int) byte {
	if i == 0 {
		return 0
	}
	return sql[i-1]
}

// matchKeywordAt 判断 sql 自 i 起是否匹配关键字 kw（已小写）：
// 关键字内的空格对应 sql 中任意长度空白；匹配结束后要求非单词边界。
func matchKeywordAt(sql string, i int, kw string) bool {
	j := i
	for k := 0; k < len(kw); {
		c := kw[k]
		if c == ' ' {
			if j >= len(sql) || !isSQLSpace(sql[j]) {
				return false
			}
			for j < len(sql) && isSQLSpace(sql[j]) {
				j++
			}
			k++
			continue
		}
		if j >= len(sql) || toLowerASCII(sql[j]) != c {
			return false
		}
		j++
		k++
	}
	return j >= len(sql) || !isWordByte(sql[j])
}

func toLowerASCII(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}
