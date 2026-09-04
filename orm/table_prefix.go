package orm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

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

// defaultTablePrefixMap 全局默认前缀映射；SetTablePrefixMap 设置，DB 未单独配置时生效。
var defaultTablePrefixMap map[string]string

// SetTablePrefix 设置全局数据表名前缀（前缀直接拼接到表名之前，如 "test_"）。
// 传空字符串可关闭前缀。对已初始化的数据源立即生效（该数据源未在配置中单独指定前缀时）。
func SetTablePrefix(prefix string) {
	defaultTablePrefix = strings.TrimSpace(prefix)
	log.Infof("set default table prefix: %q", defaultTablePrefix)
}

// SetTablePrefixMap 设置全局前缀映射（2.1）：oldprefix→newprefix，空值表示移除前缀。
// 传空 map / nil 可关闭映射。对已初始化的数据源立即生效（未单独配置时）。
func SetTablePrefixMap(pm map[string]string) {
	defaultTablePrefixMap = pm
	if len(pm) > 0 {
		log.Infof("set default table prefix map: %v", pm)
	}
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

// tablePrefixMap 返回当前 DB 生效的前缀映射：DB 自己配置优先，否则回退全局映射（2.1）。
func (db *DB) tablePrefixMap() map[string]string {
	if db != nil && db.Config != nil && len(db.Setting.TablePrefixMap) > 0 {
		return db.Setting.TablePrefixMap
	}
	return defaultTablePrefixMap
}

// applyTablePrefix 对 SQL 执行前的最终语句做表名前缀改写；未配置前缀与映射时原样返回。
// 改写前先取当前数据源指定 schema 的真实表名集合：若 SQL 引用的表名（无前缀）在库中
// 真实存在，则保持原样不改写，避免「改了前缀但物理表未改名」导致 SQL 指向不存在的表。
func (db *DB) applyTablePrefix(query string) string {
	prefix := db.tablePrefix()
	pMap := db.tablePrefixMap()
	if prefix == "" && len(pMap) == 0 {
		return query
	}
	return rewriteSQLTablesWithMap(query, prefix, pMap, db.tableNameSet())
}

// tableNamesCacheKey 表名集合缓存在 cacheStore 中的键。
const tableNamesCacheKey = "tableNames"

// tableNamesCache 数据源真实表名集合的缓存（键为小写物理表名）。
// done 为 true 表示已获取（成功或失败降级）；失败时缓存空集合并视为已获取，
// 退化为纯前缀匹配（与历史行为一致），后续 DDL 成功会使缓存失效并重新获取。
// ttl > 0 时按时间戳周期刷新（2.4）：过期后下次访问重取，降低外部会话改表的陈旧窗口。
type tableNamesCache struct {
	mu        sync.RWMutex
	names     map[string]struct{}
	done      bool
	fetchedAt time.Time
	ttl       time.Duration
}

func (c *tableNamesCache) get() (map[string]struct{}, bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.getLocked()
}

// getLocked 持有锁时读取：返回 (names, done, stale)。
func (c *tableNamesCache) getLocked() (map[string]struct{}, bool, bool) {
	stale := c.ttl > 0 && !c.fetchedAt.IsZero() && time.Since(c.fetchedAt) > c.ttl
	return c.names, c.done, stale
}

// markDone 记录获取完成时间（读锁外调用，需已持有写锁）。
func (c *tableNamesCache) markDone() {
	c.done = true
	c.fetchedAt = time.Now()
}

// tablePrefixSetTTL 返回当前 DB 的表集合缓存 TTL（0=永不过期）。
func (db *DB) tablePrefixSetTTL() time.Duration {
	if db != nil && db.Config != nil && db.Setting.TablePrefixSetTTL > 0 {
		return db.Setting.TablePrefixSetTTL
	}
	return 0
}

// tableNameSet 返回当前数据源指定 schema 的真实表名集合（小写键），首次调用时惰性
// 查询数据库并缓存；TTL 过期后下次访问重取（失败时保留旧集合）。
// 查询失败时记录告警并返回空集（退化为纯前缀匹配）。
// 集合查询直接走 ConnPool，不会再次触发 applyTablePrefix，避免循环依赖。
func (db *DB) tableNameSet() map[string]struct{} {
	if db == nil || db.cacheStore == nil {
		return nil
	}
	v, ok := db.cacheStore.Load(tableNamesCacheKey)
	if !ok {
		tc := &tableNamesCache{ttl: db.tablePrefixSetTTL()}
		actual, _ := db.cacheStore.LoadOrStore(tableNamesCacheKey, tc)
		v = actual
	}
	cache := v.(*tableNamesCache)
	names, done, stale := cache.get()
	if done && !stale {
		return names
	}
	// 首次获取或 TTL 过期：持写锁单飞重取；失败时保留旧集合（若已有），否则降级空集
	cache.mu.Lock()
	defer cache.mu.Unlock()
	names, done, stale = cache.getLocked()
	if done && !stale {
		return names
	}
	fresh, err := db.fetchTableNames()
	if err != nil {
		if done {
			log.Warnf("refresh table names failed, keep stale set: %v", err)
			return names
		}
		log.Warnf("fetch table names from schema failed, fallback to prefix-only rewrite: %v", err)
		cache.names = map[string]struct{}{}
		cache.markDone()
		return cache.names
	}
	cache.names = fresh
	cache.markDone()
	return cache.names
}

// invalidateTableNames 使表名集合缓存失效（DDL 成功后调用），下一条 SQL 重新获取真实表集合。
func (db *DB) invalidateTableNames() {
	if db == nil || db.cacheStore == nil {
		return
	}
	db.cacheStore.Delete(tableNamesCacheKey)
}

// fetchTableNames 从当前数据源查询真实表名集合（小写键）。
// 使用底层 ConnPool 直接执行，绕过 applyTablePrefix（集合查询本身只访问
// information_schema / sqlite_master / pg_class 等系统表，无需改写）。
func (db *DB) fetchTableNames() (map[string]struct{}, error) {
	if db == nil || db.ConnPool == nil {
		return nil, nil
	}
	sqlStr := tableListSQL(db.Setting, db.Setting.Name)
	if sqlStr == "" {
		return map[string]struct{}{}, nil
	}
	ctx, cancel := withExecTimeout(context.Background())
	defer cancel()
	rows, err := db.ConnPool.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := map[string]struct{}{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names[strings.ToLower(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return names, nil
}

// tableListSQL 返回列出指定 schema 真实表名的 SQL（按方言区分）；
// 与框架内建的表结构探查（database_structure.go）共用同一套查询。
func tableListSQL(setting MyBatisSetting, dbName string) string {
	switch setting.Type {
	case MySqlDb:
		// MySQL 下 schema 即数据库名；显式配置 spring.datasource.schema 时以它为准
		return fmt.Sprintf("select DISTINCT TABLE_NAME as table_name from information_schema.COLUMNS WHERE TABLE_SCHEMA='%s'", setting.effectiveSchema(dbName))
	case PostgresDb, KingbaseDb:
		// 配置了 schema 时仅列该 schema 的表；未配置保持历史行为（列出所有 schema）
		if schema := setting.Schema; schema != "" {
			return fmt.Sprintf("select relname as TABLE_NAME from pg_class c join pg_namespace n on n.oid = c.relnamespace where c.relkind = 'r' and n.nspname = '%s' and c.relname not like 'pg_%%' and c.relname not like 'sql_%%'", schema)
		}
		return "select relname as TABLE_NAME from pg_class where  relkind = 'r' and relname not like 'pg_%' and relname not like 'sql_%'"
	case SqliteDb:
		return "select name as table_name from sqlite_master where type='table' and name not like 'sqlite_%'"
	}
	log.Errorf("unsupport database type %v to get table list", setting.Type)
	return ""
}

// isDDLStatement 判断 SQL 是否属于可能改变表集合的 DDL（CREATE/DROP/ALTER/RENAME/TRUNCATE）。
// 命中时在 SQL 执行成功后使表名集合缓存失效，保证后续改写基于最新表集合。
func isDDLStatement(query string) bool {
	q := strings.TrimLeft(query, " \t\r\n")
	if q == "" {
		return false
	}
	word := q
	if i := strings.IndexAny(q, " \t\r\n("); i >= 0 {
		word = q[:i]
	}
	switch strings.ToLower(word) {
	case "create", "drop", "alter", "rename", "truncate":
		return true
	}
	return false
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

// prefixRequired 在纯前缀匹配基础上，结合指定 schema 的真实表名集合做精确匹配，
// 提高「改了前缀后实际表不在」场景下的准确率：
//  1. 系统表 / 已带前缀的表 → 不改写（保持原样，防叠加）；
//  2. 带前缀的表名在库中真实存在 → 按配置前缀改写（配置意图优先）；
//  3. SQL 引用的表名（无前缀）在库中真实存在 → 保持原样（它就是物理表，改写反而出错）；
//  4. 两者均未收录（如 CREATE TABLE 新建表、目标表确实不存在）→ 沿用前缀匹配兜底。
//
// tableSet 为 nil 或空时退化为旧的纯前缀匹配行为。
func prefixRequired(tok sqlToken, prefix string, tableSet map[string]struct{}) bool {
	name := identContent(tok)
	if name == "" || isSystemTable(name) {
		return false
	}
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, strings.ToLower(prefix)) {
		// 已带前缀不叠加
		return false
	}
	if len(tableSet) > 0 {
		// 带前缀的表名在库中真实存在 → 按配置前缀改写（配置意图优先）
		if _, ok := tableSet[strings.ToLower(prefix)+lower]; ok {
			return true
		}
		// SQL 引用的表名（无前缀）在库中真实存在 → 保持原样
		if _, ok := tableSet[lower]; ok {
			return false
		}
		// 库中也未收录（建表 DDL / 表确实不存在）→ 沿用前缀匹配兜底
		log.Debugf("table %q not found in schema table set, rewrite with prefix %q", name, prefix)
	}
	return true
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
// 纯前缀匹配（不携带真实表名集合），保持纯函数语义供单元测试使用。
func rewriteSQLTables(query, prefix string) string {
	return rewriteSQLTablesWithMap(query, prefix, nil, nil)
}

// rewriteSQLTablesWithSet 同 rewriteSQLTables，但携带数据源指定 schema 的真实表名集合：
// 无前缀表名在库中真实存在时不改写（修复改了前缀但物理表未改名导致访问错误表的问题）。
func rewriteSQLTablesWithSet(query, prefix string, tableSet map[string]struct{}) string {
	return rewriteSQLTablesWithMap(query, prefix, nil, tableSet)
}

// mappedPrefix 返回标识符命中的前缀映射（新前缀, 命中的旧前缀, 是否命中）。
// 命中规则：标识符以旧前缀开头（大小写不敏感）；映射优先级高于表集合判定（2.1）。
func mappedPrefix(name string, prefixMap map[string]string) (string, string, bool) {
	lower := strings.ToLower(name)
	for oldp, newp := range prefixMap {
		ol := strings.ToLower(oldp)
		if ol != "" && strings.HasPrefix(lower, ol) {
			return newp, oldp, true
		}
	}
	return "", "", false
}

// mappedHit 判断标识符是否命中任一前缀映射键（用于非 public/main schema 限定名也走映射）。
func mappedHit(name string, prefixMap map[string]string) bool {
	_, _, hit := mappedPrefix(name, prefixMap)
	return hit
}

// stripPrefix 在标识符 token 上移除开头前缀（引号标识符在开引号之后切）。
func stripPrefix(tok *sqlToken, oldp string) {
	if oldp == "" {
		return
	}
	if tok.typ == tkQuoted && len(tok.text) >= 2 {
		inner := tok.text[1 : len(tok.text)-1]
		if strings.HasPrefix(strings.ToLower(inner), strings.ToLower(oldp)) {
			tok.text = tok.text[:1] + inner[len(oldp):] + tok.text[len(tok.text)-1:]
		}
		return
	}
	if strings.HasPrefix(strings.ToLower(tok.text), strings.ToLower(oldp)) {
		tok.text = tok.text[len(oldp):]
	}
}

// rewriteSQLTablesWithMap 同 rewriteSQLTablesWithSet，但叠加前缀映射（2.1）：
// 命中映射（表名以旧前缀开头）→ 映射优先于表集合：新前缀非空则替换（剥旧套新）；
// 新前缀为空（移除）→ 表集合判定：库中无前缀表存在则保持、带前缀表存在则保留原样、
// 其余（含集合缺失）按配置剥离。未命中映射时保持原有的 prefixRequired + applyPrefix 逻辑，
// 因此 nil/空映射时输出与 rewriteSQLTablesWithSet 完全一致（零回归）。
func rewriteSQLTablesWithMap(query, prefix string, prefixMap map[string]string, tableSet map[string]struct{}) string {
	if query == "" {
		return query
	}
	if prefix == "" && len(prefixMap) == 0 {
		return query
	}
	tokens := tokenizeSQL(query)
	expectTable := false // 下一个标识符处于表位置（FROM/JOIN/INTO/UPDATE/TABLE 之后）
	commaList := false   // 处于 FROM/JOIN/UPDATE 逗号分隔的多表列表中
	cteNames := map[string]bool{}
	// 2.2 扩展表位置关键字上下文
	pendingIndex  := false // CREATE INDEX ... ON 的 index→on 上下文
	renameMode    := false // RENAME TABLE 上下文
	expectToTable := false // rename 的首表已处理，期待 TO + 目标表
	consumedTo    := false // 本次表位置由 TO 触发（处理完重置 rename 上下文）

	// applyToken 处理处于表位置的单个标识符：先走前缀映射通道，未命中再走原逻辑。
	// allowPrefix=false 时禁止回落前缀匹配（用于非 public/main schema 限定名仅映射生效）。
	applyToken := func(t *sqlToken, allowPrefix bool) {
		name := identContent(*t)
		if name == "" {
			return
		}
		if newp, oldp, hit := mappedPrefix(name, prefixMap); hit {
			if newp == "" {
				// 移除：带前缀表物理存在 → 保留原样（避免指向错误表）；否则剥离
				if len(tableSet) > 0 {
					lower := strings.ToLower(name)
					if _, ok := tableSet[lower]; ok {
						return // 无前缀表已存在，保持
					}
					if _, ok := tableSet[strings.ToLower(prefix)+lower]; ok {
						return // 带前缀表存在，不剥
					}
				}
				stripPrefix(t, oldp)
				return
			}
			// 替换：剥旧前缀再套新前缀
			stripPrefix(t, oldp)
			applyPrefix(t, newp)
			return
		}
		if allowPrefix && !cteNames[strings.ToLower(name)] && prefixRequired(*t, prefix, tableSet) {
			applyPrefix(t, prefix)
		}
	}

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
				// schema 限定名 schema.table：public/main 走原逻辑；非 public/main 仅映射生效
				j := nextSig(tokens, i)
				if j >= 0 && tokens[j].typ == tkPunct && tokens[j].text == "." {
					k := nextSig(tokens, j)
					if k >= 0 && (tokens[k].typ == tkWord || tokens[k].typ == tkQuoted) {
						tblName := identContent(tokens[k])
						if isPrefixableSchema(identContent(*t)) {
							applyToken(&tokens[k], true)
						} else if mappedHit(tblName, prefixMap) {
							applyToken(&tokens[k], false)
						}
						i = k
					}
				} else {
					applyToken(t, true)
				}
				expectTable = false
				commaList = true
				if renameMode && !expectToTable {
					expectToTable = true // rename: 首表已处理，等待 TO
				}
				if consumedTo {
					renameMode = false
					consumedTo = false
					expectToTable = false
				}
				continue
			}
			switch lower {
			case "from", "join", "into", "update", "table", "using":
				expectTable = true
				commaList = false
			case "index":
				// CREATE INDEX ... ON t：index 后期待 on 触发表位置
				pendingIndex = true
				commaList = false
			case "on":
				if pendingIndex {
					expectTable = true
					pendingIndex = false
				} else {
					commaList = false
				}
			case "rename":
				renameMode = true
				commaList = false
			case "to":
				if expectToTable {
					expectTable = true
					expectToTable = false
					consumedTo = true
				} else {
					commaList = false
				}
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
						tblName := identContent(tokens[k])
						if isPrefixableSchema(identContent(*t)) {
							applyToken(&tokens[k], true)
						} else if mappedHit(tblName, prefixMap) {
							applyToken(&tokens[k], false)
						}
						i = k
					}
				} else {
					applyToken(t, true)
				}
				expectTable = false
				commaList = true
				if renameMode && !expectToTable {
					expectToTable = true // rename: 首表已处理，等待 TO
				}
				if consumedTo {
					renameMode = false
					consumedTo = false
					expectToTable = false
				}
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
