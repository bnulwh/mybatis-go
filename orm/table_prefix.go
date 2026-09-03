package orm

import (
	"strings"

	"github.com/bnulwh/mybatis-go/log"
)

// 数据表名前缀支持。
//
// 同一套 XML Mapper（SQL 语句保持 `sys_user` 等原始表名）可在不同环境指向不同的
// 物理表：测试环境配置 `mybatis.table-prefix=test_` 即访问 `test_sys_user`，
// 生产环境不配置前缀（或配置 `prod_`）即可。前缀在 SQL 执行入口统一改写，
// 对 Mapper 代理、orm.Execute/Query、事务、流式查询全部生效。
//
// 使用方式：
//
//	① 配置文件：mybatis.table-prefix=test_（每个数据源独立生效，多数据源可继承默认源）
//	② 编程方式：orm.SetTablePrefix("test_")（全局，作用于未单独配置前缀的数据源）

// defaultTablePrefix 全局默认表名前缀；SetTablePrefix 设置，DB 未单独配置时生效。
var defaultTablePrefix string

// SetTablePrefix 设置全局数据表名前缀（前缀直接拼接到表名之前，如 "test_"）。
// 传空字符串可关闭前缀。对已初始化的数据源立即生效（该数据源未在配置中单独指定前缀时）。
func SetTablePrefix(prefix string) {
	defaultTablePrefix = strings.TrimSpace(prefix)
	log.Infof("set default table prefix: %q", defaultTablePrefix)
}

// GetTablePrefix 返回当前生效的表名前缀：
// 优先返回全局默认数据源（gDbConn）配置的前缀，其次返回全局前缀；均未配置时为空串。
func GetTablePrefix() string {
	if gDbConn != nil && gDbConn.Config != nil {
		if p := strings.TrimSpace(gDbConn.Setting.TablePrefix); p != "" {
			return p
		}
	}
	return defaultTablePrefix
}

// tablePrefix 返回当前 DB 生效的前缀：DB 自己配置的前缀优先，否则回退全局前缀。
func (db *DB) tablePrefix() string {
	if db != nil && db.Config != nil {
		if p := strings.TrimSpace(db.Setting.TablePrefix); p != "" {
			return p
		}
	}
	return defaultTablePrefix
}

// applyTablePrefix 对 SQL 执行前的最终语句做表名前缀改写；未配置前缀时原样返回。
func (db *DB) applyTablePrefix(query string) string {
	prefix := db.tablePrefix()
	if prefix == "" {
		return query
	}
	return rewriteSQLTables(query, prefix)
}

// ---------------------------------------------------------------------------
// SQL 表名改写：基于词法扫描（状态机），区分关键字/标识符/字符串字面量/注释，
// 仅在 FROM/JOIN/INTO/UPDATE/TABLE 等表位置的标识符前插入前缀，避免误伤
// 列名、字符串字面量、别名与系统表（information_schema / pg_% / sqlite_% / pragma_%）。

type sqlTokenType int

const (
	tkWord    sqlTokenType = iota // 裸标识符 / 关键字 / 数字
	tkQuoted                      // 引号标识符："x" `x` [x]
	tkString                      // 字符串字面量：'x' $$x$$ $tag$x$tag$
	tkPunct                       // 单个标点
	tkSpace                       // 空白
	tkComment                     // 注释：-- ... / # ... / /* ... */
)

type sqlToken struct {
	text string
	typ  sqlTokenType
}

func isWordStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c >= 0x80
}

func isWordChar(c byte) bool {
	return isWordStart(c) || c == '$'
}

// scanDollarQuote 检测 Postgres 美元引用字符串（$$...$$ / $tag$...$tag$）。
// 返回结束位置（不含）与是否命中；s[i] 必须为 '$'。
func scanDollarQuote(s string, i int) (int, bool) {
	if i+1 >= len(s) {
		return 0, false
	}
	if s[i+1] == '$' {
		end := strings.Index(s[i+2:], "$$")
		if end < 0 {
			return 0, false
		}
		return i + 2 + end + 2, true
	}
	j := i + 1
	if j >= len(s) || !isWordStart(s[j]) {
		return 0, false
	}
	for j < len(s) && isWordChar(s[j]) {
		j++
	}
	if j >= len(s) || s[j] != '$' {
		return 0, false
	}
	tag := s[i+1 : j]
	closer := "$" + tag + "$"
	pos := strings.Index(s[j+1:], closer)
	if pos < 0 {
		return 0, false
	}
	return j + 1 + pos + len(closer), true
}

// tokenizeSQL 将 SQL 拆成词法单元；所有文本原样保留（改写后拼接还原）。
func tokenizeSQL(s string) []sqlToken {
	tokens := make([]sqlToken, 0, len(s)/4)
	i, n := 0, len(s)
	for i < n {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			j := i
			for j < n && (s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r') {
				j++
			}
			tokens = append(tokens, sqlToken{text: s[i:j], typ: tkSpace})
			i = j
		case c == '\'':
			j := i + 1
			for j < n {
				if s[j] == '\'' {
					if j+1 < n && s[j+1] == '\'' {
						j += 2 // '' 转义
						continue
					}
					j++
					break
				}
				j++
			}
			if j > n {
				j = n
			}
			tokens = append(tokens, sqlToken{text: s[i:j], typ: tkString})
			i = j
		case c == '"' || c == '`':
			q := c
			j := i + 1
			for j < n {
				if s[j] == q {
					if j+1 < n && s[j+1] == q {
						j += 2 // 引号转义
						continue
					}
					j++
					break
				}
				j++
			}
			if j > n {
				j = n
			}
			tokens = append(tokens, sqlToken{text: s[i:j], typ: tkQuoted})
			i = j
		case c == '[':
			j := i + 1
			for j < n && s[j] != ']' {
				j++
			}
			if j < n {
				j++
			}
			tokens = append(tokens, sqlToken{text: s[i:j], typ: tkQuoted})
			i = j
		case c == '-' && i+1 < n && s[i+1] == '-':
			j := i
			for j < n && s[j] != '\n' {
				j++
			}
			tokens = append(tokens, sqlToken{text: s[i:j], typ: tkComment})
			i = j
		case c == '#':
			j := i
			for j < n && (s[j] != '\n' && s[j] != '\r') {
				j++
			}
			tokens = append(tokens, sqlToken{text: s[i:j], typ: tkComment})
			i = j
		case c == '/' && i+1 < n && s[i+1] == '*':
			end := strings.Index(s[i+2:], "*/")
			j := n
			if end >= 0 {
				j = i + 2 + end + 2
			}
			tokens = append(tokens, sqlToken{text: s[i:j], typ: tkComment})
			i = j
		case c == '$':
			if end, ok := scanDollarQuote(s, i); ok {
				tokens = append(tokens, sqlToken{text: s[i:end], typ: tkString})
				i = end
			} else {
				j := i
				for j < n && isWordChar(s[j]) {
					j++
				}
				tokens = append(tokens, sqlToken{text: s[i:j], typ: tkWord})
				i = j
			}
		case isWordStart(c):
			j := i
			for j < n && isWordChar(s[j]) {
				j++
			}
			tokens = append(tokens, sqlToken{text: s[i:j], typ: tkWord})
			i = j
		default:
			tokens = append(tokens, sqlToken{text: s[i : i+1], typ: tkPunct})
			i++
		}
	}
	return tokens
}

// nextSig 返回 i 之后第一个非空白/注释的 token 下标；不存在返回 -1。
func nextSig(tokens []sqlToken, i int) int {
	for j := i + 1; j < len(tokens); j++ {
		if tokens[j].typ == tkSpace || tokens[j].typ == tkComment {
			continue
		}
		return j
	}
	return -1
}

// isSystemTable 判断是否为系统目录/系统表（不参与前缀改写）。
func isSystemTable(name string) bool {
	n := strings.ToLower(name)
	return n == "information_schema" || n == "pg_catalog" ||
		strings.HasPrefix(n, "pg_") ||
		strings.HasPrefix(n, "sqlite_") ||
		strings.HasPrefix(n, "pragma_")
}

// prefixable 判断该标识符是否应加前缀：排除系统表与已带前缀的表名。
func prefixable(tok sqlToken, prefix string) bool {
	name := identContent(tok)
	if name == "" || isSystemTable(name) {
		return false
	}
	return !strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix))
}

// identContent 取标识符内容（引号标识符去掉引号）。
func identContent(tok sqlToken) string {
	if tok.typ == tkQuoted && len(tok.text) >= 2 {
		return tok.text[1 : len(tok.text)-1]
	}
	return tok.text
}

// applyPrefix 在标识符上插入前缀（引号标识符插在开引号之后）。
func applyPrefix(tok *sqlToken, prefix string) {
	if tok.typ == tkQuoted && len(tok.text) >= 2 {
		tok.text = tok.text[:1] + prefix + tok.text[1:]
		return
	}
	tok.text = prefix + tok.text
}

// clauseBreakWords 出现后，不再处于 FROM/JOIN/UPDATE 逗号列表上下文的关键字。
var clauseBreakWords = map[string]bool{
	"where": true, "on": true, "set": true, "select": true, "group": true,
	"having": true, "order": true, "limit": true, "values": true, "union": true,
	"and": true, "or": true, "by": true, "as": true, "distinct": true,
	"when": true, "then": true, "else": true, "end": true, "case": true,
	"in": true, "exists": true, "like": true, "not": true, "null": true,
	"is": true, "returning": true, "offset": true, "fetch": true, "for": true,
	"using": true, "left": true, "right": true, "inner": true, "outer": true,
	"cross": true, "full": true, "natural": true,
}

// rewriteSQLTables 为查询中的表名插入前缀；输入输出 SQL 除表名前缀外完全一致。
func rewriteSQLTables(query, prefix string) string {
	if prefix == "" || query == "" {
		return query
	}
	tokens := tokenizeSQL(query)
	expectTable := false // 下一个标识符处于表位置（FROM/JOIN/INTO/UPDATE/TABLE 之后）
	commaList := false   // 处于 FROM/JOIN/UPDATE 逗号分隔的多表列表中
	cteNames := map[string]bool{}

	for i := 0; i < len(tokens); i++ {
		t := &tokens[i]
		switch t.typ {
		case tkWord:
			lower := strings.ToLower(t.text)
			if expectTable {
				// CREATE/DROP/ALTER TABLE IF NOT EXISTS 等修饰词属于 DDL 语法，跳过
				if lower == "if" || lower == "not" || lower == "exists" {
					continue
				}
				// schema 限定名 schema.table：仅 public/main 模式加前缀，其余跳过
				j := nextSig(tokens, i)
				if j >= 0 && tokens[j].typ == tkPunct && tokens[j].text == "." {
					k := nextSig(tokens, j)
					if k >= 0 && (tokens[k].typ == tkWord || tokens[k].typ == tkQuoted) {
						if isPrefixableSchema(identContent(*t)) && prefixable(tokens[k], prefix) {
							applyPrefix(&tokens[k], prefix)
						}
						i = k
					}
				} else if !cteNames[lower] && prefixable(*t, prefix) {
					applyPrefix(t, prefix)
				}
				expectTable = false
				commaList = true
				continue
			}
			switch lower {
			case "from", "join", "into", "update", "table":
				expectTable = true
				commaList = false
			case "with":
				// WITH cte AS (...) 收集 CTE 名，后续 FROM cte 不参与前缀
				cteModeCollect(tokens, i, cteNames)
				commaList = false
			default:
				if clauseBreakWords[lower] {
					commaList = false
				}
			}
		case tkQuoted:
			if expectTable {
				// 引号标识符同样支持 schema.table 限定名（如 `db`.`tbl` / "public"."tbl"）
				j := nextSig(tokens, i)
				if j >= 0 && tokens[j].typ == tkPunct && tokens[j].text == "." {
					k := nextSig(tokens, j)
					if k >= 0 && (tokens[k].typ == tkWord || tokens[k].typ == tkQuoted) {
						if isPrefixableSchema(identContent(*t)) && prefixable(tokens[k], prefix) {
							applyPrefix(&tokens[k], prefix)
						}
						i = k
					}
				} else if !cteNames[strings.ToLower(identContent(*t))] && prefixable(*t, prefix) {
					applyPrefix(t, prefix)
				}
				expectTable = false
				commaList = true
			}
		case tkPunct:
			switch t.text {
			case "(", ")":
				expectTable = false
				commaList = false
			case ",":
				if commaList {
					expectTable = true
				}
			}
		}
	}
	var sb strings.Builder
	sb.Grow(len(query))
	for i := range tokens {
		sb.WriteString(tokens[i].text)
	}
	return sb.String()
}

// isPrefixableSchema 判断 schema 限定名是否应加前缀（只有默认模式 public/main）。
func isPrefixableSchema(schema string) bool {
	switch strings.ToLower(schema) {
	case "public", "main":
		return true
	}
	return false
}

// cteModeCollect 在 WITH 处收集顶层 CTE 名：`WITH [RECURSIVE] <name> AS [(MATERIALIZED|NOT MATERIALIZED)] (`，
// 多个 CTE 以逗号分隔；靠配对括号跳过各子查询体。tokens[i] 已确认是关键字 with。
func cteModeCollect(tokens []sqlToken, i int, cteNames map[string]bool) {
	j := nextSig(tokens, i)
	for j >= 0 {
		tok := tokens[j]
		if tok.typ != tkWord {
			break
		}
		lower := strings.ToLower(tok.text)
		if lower == "recursive" {
			j = nextSig(tokens, j)
			continue
		}
		// name AS [(NOT) MATERIALIZED] ( ... )
		a := nextSig(tokens, j)
		if a < 0 || !isWordToken(tokens[a], "as") {
			break
		}
		p := nextSig(tokens, a)
		for p >= 0 && (isWordToken(tokens[p], "not") || isWordToken(tokens[p], "materialized")) {
			p = nextSig(tokens, p)
		}
		if p < 0 || tokens[p].typ != tkPunct || tokens[p].text != "(" {
			break
		}
		cteNames[lower] = true
		// 跳过子查询体到配对的右括号，查看逗号继续下一个 CTE
		end := matchParenIndex(tokens, p)
		if end < 0 {
			break
		}
		q := nextSig(tokens, end)
		if q < 0 || tokens[q].typ != tkPunct || tokens[q].text != "," {
			break
		}
		j = nextSig(tokens, q)
	}
}

// matchParenIndex 返回从 index（必须是 '('）开始的配对右括号下标；未闭合返回 -1。
// 字符串/注释已被 tokenizer 折叠为单个 token，只需统计 tkPunct 括号。
func matchParenIndex(tokens []sqlToken, index int) int {
	depth := 0
	for i := index; i < len(tokens); i++ {
		if tokens[i].typ != tkPunct {
			continue
		}
		switch tokens[i].text {
		case "(":
			depth++
		case ")":
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func isWordToken(t sqlToken, lower string) bool {
	return t.typ == tkWord && strings.ToLower(t.text) == lower
}
