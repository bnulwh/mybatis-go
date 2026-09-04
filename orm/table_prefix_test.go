package orm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite" // SQLite 驱动（与 sqlite_test.go 相同的驱动注册）
)

// ---------------------------------------------------------------------------
// rewriteSQLTables 单元测试（纯函数，不依赖数据库）

func Test_rewriteSQLTables_basic(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"select", "SELECT * FROM sys_user", "SELECT * FROM test_sys_user"},
		{"insert", "INSERT INTO sys_user (name) VALUES (?)", "INSERT INTO test_sys_user (name) VALUES (?)"},
		{"update", "UPDATE sys_user SET name = ? WHERE id = ?", "UPDATE test_sys_user SET name = ? WHERE id = ?"},
		{"delete", "DELETE FROM sys_user WHERE id = ?", "DELETE FROM test_sys_user WHERE id = ?"},
		{"join", "select * FROM t1 a JOIN t2 b ON a.id=b.id",
			"select * FROM test_t1 a JOIN test_t2 b ON a.id=b.id"},
		{"comma", "select * FROM t1 a, t2 b, t3 c WHERE a.id=b.id",
			"select * FROM test_t1 a, test_t2 b, test_t3 c WHERE a.id=b.id"},
		{"update-multi-table", "UPDATE t1, t2 SET t1.a = t2.b WHERE t1.id = t2.id",
			"UPDATE test_t1, test_t2 SET t1.a = t2.b WHERE t1.id = t2.id"},
		{"create-if-not-exists", "CREATE TABLE IF NOT EXISTS t_sqlite (id int)",
			"CREATE TABLE IF NOT EXISTS test_t_sqlite (id int)"},
		{"alter-table", "ALTER TABLE t_sqlite ADD COLUMN c text",
			"ALTER TABLE test_t_sqlite ADD COLUMN c text"},
		{"truncate", "TRUNCATE TABLE t1, t2", "TRUNCATE TABLE test_t1, test_t2"},
		{"create-as-select", "CREATE TABLE t1 AS SELECT id FROM t2",
			"CREATE TABLE test_t1 AS SELECT id FROM test_t2"},
		{"insert-select", "INSERT INTO t1 SELECT id FROM t2", "INSERT INTO test_t1 SELECT id FROM test_t2"},
		{"schema-public", "SELECT * FROM public.sys_user", "SELECT * FROM public.test_sys_user"},
		{"schema-main", "SELECT * FROM main.t_sqlite", "SELECT * FROM main.test_t_sqlite"},
		{"schema-other-skip", "SELECT * FROM app.sys_user", "SELECT * FROM app.sys_user"},
		{"uppercase-keep-case", "SELECT * FROM SYS_USER", "SELECT * FROM test_SYS_USER"},
		{"quoted-backtick", "SELECT * FROM `sys_user`", "SELECT * FROM `test_sys_user`"},
		{"quoted-double", "SELECT * FROM \"sys_user\"", "SELECT * FROM \"test_sys_user\""},
		{"quoted-bracket", "SELECT * FROM [sys_user]", "SELECT * FROM [test_sys_user]"},
		{"cte", "WITH cte AS (SELECT id FROM sys_user) SELECT * FROM cte c JOIN sys_role r ON c.id = r.id",
			"WITH cte AS (SELECT id FROM test_sys_user) SELECT * FROM cte c JOIN test_sys_role r ON c.id = r.id"},
	}
	for _, c := range cases {
		got := rewriteSQLTables(c.in, "test_")
		if got != c.want {
			t.Errorf("%s: rewriteSQLTables(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// 表位置之外的列名/字面量/别名/系统表/占位符不应被改写
func Test_rewriteSQLTables_noFalsePositive(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"column-named-like-table", "SELECT sys_user, name FROM sys_user WHERE sys_user = 1",
			"SELECT sys_user, name FROM test_sys_user WHERE sys_user = 1"},
		{"string-literal", "SELECT * FROM sys_user WHERE name = 'sys_user'",
			"SELECT * FROM test_sys_user WHERE name = 'sys_user'"},
		{"placeholder-dollar", "SELECT * FROM sys_user WHERE id = $1",
			"SELECT * FROM test_sys_user WHERE id = $1"},
		{"limit-and-order", "SELECT a, b FROM t GROUP BY a, b ORDER BY a, b LIMIT 10",
			"SELECT a, b FROM test_t GROUP BY a, b ORDER BY a, b LIMIT 10"},
		{"in-list", "SELECT * FROM t WHERE id IN (1, 2, 3)", "SELECT * FROM test_t WHERE id IN (1, 2, 3)"},
		{"case-when", "UPDATE t SET a = CASE WHEN x THEN 1 ELSE 2 END, b = 3 WHERE id = 1",
			"UPDATE test_t SET a = CASE WHEN x THEN 1 ELSE 2 END, b = 3 WHERE id = 1"},
		{"already-prefixed", "SELECT * FROM test_sys_user WHERE id = 1", "SELECT * FROM test_sys_user WHERE id = 1"},
		{"already-prefixed-case", "SELECT * FROM TEST_nope WHERE 1 = 1", "SELECT * FROM TEST_nope WHERE 1 = 1"},
		{"line-comment", "SELECT * FROM sys_user -- keep\nWHERE id = 1",
			"SELECT * FROM test_sys_user -- keep\nWHERE id = 1"},
		{"block-comment", "SELECT * FROM /* keep */ sys_user", "SELECT * FROM /* keep */ test_sys_user"},
		{"hash-comment", "SELECT * FROM t # mysql comment\nWHERE id = 1",
			"SELECT * FROM test_t # mysql comment\nWHERE id = 1"},
		{"dollar-string", "SELECT $tag$from sys_user$tag$ AS s, * FROM t",
			"SELECT $tag$from sys_user$tag$ AS s, * FROM test_t"},
		{"values-multirow", "INSERT INTO t (a, b) VALUES (1, 2), (3, 4)",
			"INSERT INTO test_t (a, b) VALUES (1, 2), (3, 4)"},
		{"update-from-clause", "UPDATE t1 SET v = 1 FROM t2 WHERE t1.id = t2.id",
			"UPDATE test_t1 SET v = 1 FROM test_t2 WHERE t1.id = t2.id"},
		{"cte-multi", "WITH a AS (SELECT 1 AS id), b AS (SELECT 2 AS id) SELECT * FROM a JOIN b ON a.id = b.id",
			"WITH a AS (SELECT 1 AS id), b AS (SELECT 2 AS id) SELECT * FROM a JOIN b ON a.id = b.id"},
		{"subquery-in-paren", "SELECT * FROM (SELECT id FROM sys_user) x",
			"SELECT * FROM (SELECT id FROM test_sys_user) x"},
		{"scalar-subquery", "SELECT (SELECT max(id) FROM t2) AS m FROM t1",
			"SELECT (SELECT max(id) FROM test_t2) AS m FROM test_t1"},
		{"information-schema", "SELECT * FROM information_schema.columns WHERE table_name = 'sys_user'",
			"SELECT * FROM information_schema.columns WHERE table_name = 'sys_user'"},
		{"pg-attribute", "SELECT col_description(attrelid, attnum) AS c FROM pg_attribute",
			"SELECT col_description(attrelid, attnum) AS c FROM pg_attribute"},
		{"sqlite-master", "select name from sqlite_master where type = 'table'",
			"select name from sqlite_master where type = 'table'"},
		{"pragma", "SELECT * FROM pragma_table_info('t_sqlite')", "SELECT * FROM pragma_table_info('t_sqlite')"},
		// 引号限定名：schema 非 public/main 时不加前缀
		{"quoted-schema-skip", "SELECT * FROM `dbname`.`sys_user`", "SELECT * FROM `dbname`.`sys_user`"},
		{"quoted-schema-public", "SELECT * FROM \"public\".\"sys_user\"", "SELECT * FROM \"public\".\"test_sys_user\""},
	}
	for _, c := range cases {
		got := rewriteSQLTables(c.in, "test_")
		if got != c.want {
			t.Errorf("%s: rewriteSQLTables(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

func Test_rewriteSQLTables_emptyPrefix(t *testing.T) {
	if got := rewriteSQLTables("SELECT * FROM sys_user", ""); got != "SELECT * FROM sys_user" {
		t.Errorf("empty prefix should not rewrite, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// 配置解析

func Test_parseTablePrefix(t *testing.T) {
	// mybatis.table-prefix 优先于 MP 风格 key
	got := parseTablePrefix(map[string]string{
		"mybatis.table-prefix":                              "test_",
		"mybatis-plus.global-config.db-config.table-prefix": "mp_",
	})
	if got != "test_" {
		t.Errorf("priority failed, got %q", got)
	}
	// MP 风格 key
	got = parseTablePrefix(map[string]string{
		"mybatis-plus.global-config.db-config.table-prefix": "mp_",
	})
	if got != "mp_" {
		t.Errorf("mp key failed, got %q", got)
	}
	// 未配置 → 空
	if got := parseTablePrefix(map[string]string{}); got != "" {
		t.Errorf("empty case failed, got %q", got)
	}
}

func Test_parseDatabaseConfig_tablePrefix(t *testing.T) {
	cfg := parseDatabaseConfig(map[string]string{
		"spring.datasource.url": "jdbc:sqlite::memory:",
		"mybatis.table-prefix":  "test_",
	})
	if cfg.Setting.TablePrefix != "test_" {
		t.Errorf("parseDatabaseConfig table-prefix failed, got %q", cfg.Setting.TablePrefix)
	}
}

func Test_parseMultiDatabaseConfig_tablePrefixInherit(t *testing.T) {
	m := map[string]string{
		"spring.datasource.url":     "jdbc:sqlite::memory:",
		"mybatis.table-prefix":      "test_",
		"mybatis.datasources":       "bak",
		"spring.datasource.bak.url": "jdbc:sqlite:bak.db",
	}
	configs := parseMultiDatabaseConfig(m)
	if configs[defaultDataSourceName].Setting.TablePrefix != "test_" {
		t.Errorf("default source prefix failed, got %q", configs[defaultDataSourceName].Setting.TablePrefix)
	}
	if configs["bak"] == nil {
		t.Error("bak datasource not parsed")
		return
	}
	if configs["bak"].Setting.TablePrefix != "test_" {
		t.Errorf("bak source should inherit default prefix, got %q", configs["bak"].Setting.TablePrefix)
	}
}

// ---------------------------------------------------------------------------
// 端到端：SQLite 真实执行（物理表带 test_ 前缀，Mapper XML / SQL 保持不带前缀）

type TablePrefixModel struct {
	Id   int
	Name string
}

type TablePrefixMapper struct {
	BaseMapper
	Insert    func(model *TablePrefixModel) (int64, error)
	SelectAll func() ([]TablePrefixModel, error)
	CountAll  func() (int, error)
}

func initTablePrefixSqlite(t *testing.T) string {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="TablePrefixMapper">
  <resultMap id="BaseResultMap" type="TablePrefixModel">
    <id column="id" jdbcType="INTEGER" property="id" />
    <result column="name" jdbcType="VARCHAR" property="name" />
  </resultMap>
  <insert id="insert" parameterType="TablePrefixModel" useGeneratedKeys="true" keyProperty="id">
    insert into t_sqlite (name) values (#{name})
  </insert>
  <select id="selectAll" resultMap="BaseResultMap">
    select id, name from t_sqlite order by id
  </select>
  <select id="countAll" resultType="int">
    select count(*) from t_sqlite
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "TablePrefixMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	dbPath := filepath.Join(dir, "test.db")
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
		"mybatis.table-prefix":     "test_",
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return ""
	}
	return dir
}

func Test_TablePrefix_SqliteQuery(t *testing.T) {
	dir := initTablePrefixSqlite(t)
	if dir == "" {
		return
	}
	defer Close()
	// 物理表必须带 test_ 前缀；SQL 语句保持写 t_sqlite（不写前缀）
	if _, err := Execute(`CREATE TABLE IF NOT EXISTS t_sqlite (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	if _, err := Execute(`INSERT INTO t_sqlite (name) VALUES (?)`, "hello"); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	res, err := Query(`SELECT id, name FROM t_sqlite`)
	if err != nil {
		t.Errorf("query failed: %v", err)
		return
	}
	if len(res) != 1 || res[0]["name"] != "hello" {
		t.Errorf("query result failed, got %v", res)
	}
	// 不带前缀的物理表应不存在（证明改写已生效，而非碰巧存在同名表）
	if _, err := Query(`SELECT * FROM nope_t_never_exists`); err == nil {
		t.Errorf("unprefixed physical table nope_t_never_exists should not exist")
	}
}

func Test_TablePrefix_SqliteMapper(t *testing.T) {
	dir := initTablePrefixSqlite(t)
	if dir == "" {
		return
	}
	defer Close()
	if _, err := Execute(`CREATE TABLE t_sqlite (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	RegisterModel(new(TablePrefixModel))
	if err := RegisterMapper(new(TablePrefixMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("TablePrefixMapper").(TablePrefixMapper)
	m := TablePrefixModel{Name: "prefix_user"}
	if _, err := mp.Insert(&m); err != nil {
		t.Errorf("mapper insert failed: %v", err)
		return
	}
	if m.Id <= 0 {
		t.Errorf("generated key backfill failed, got %d", m.Id)
	}
	rs, err := mp.SelectAll()
	if err != nil {
		t.Errorf("mapper select failed: %v", err)
		return
	}
	if len(rs) != 1 || rs[0].Name != "prefix_user" {
		t.Errorf("mapper select failed, got %v", rs)
	}
	n, err := mp.CountAll()
	if err != nil {
		t.Errorf("mapper count failed: %v", err)
		return
	}
	if n != 1 {
		t.Errorf("mapper count failed, got %d want 1", n)
	}
}

func Test_TablePrefix_GlobalSetter(t *testing.T) {
	if err := InitializeDatabase("sqlite", "", 0, "", "", filepath.Join(t.TempDir(), "g.db")); err != nil {
		t.Errorf("init failed: %v", err)
		return
	}
	defer Close()
	SetTablePrefix("prod_")
	defer SetTablePrefix("")
	// SQL 不带前缀，物理表 prod_t_sqlite
	if _, err := Execute(`CREATE TABLE t_sqlite (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Errorf("create table (should rewrite to prod_t_sqlite) failed: %v", err)
		return
	}
	if _, err := Query(`SELECT * FROM t_sqlite`); err != nil {
		t.Errorf("query prefixed table failed: %v", err)
	}
}

func Test_TablePrefix_GetAndTransactions(t *testing.T) {
	dir := initTablePrefixSqlite(t)
	if dir == "" {
		return
	}
	defer Close()
	if GetTablePrefix() != "test_" {
		t.Errorf("GetTablePrefix failed, got %q", GetTablePrefix())
	}
	if _, err := Execute(`CREATE TABLE t_sqlite (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	tx, err := Begin()
	if err != nil {
		t.Errorf("begin failed: %v", err)
		return
	}
	if _, err := tx.Exec(`INSERT INTO t_sqlite (name) VALUES (?)`, "tx_user"); err != nil {
		t.Errorf("tx exec failed: %v", err)
		return
	}
	rows, err := tx.Query(`SELECT name FROM t_sqlite`)
	if err != nil {
		t.Errorf("tx query failed: %v", err)
		return
	}
	_ = rows.Close()
	if err := tx.Rollback(); err != nil {
		t.Errorf("rollback failed: %v", err)
	}
	// 回滚后查不到（事务内 SQL 同样走了前缀改写，指向同一物理表）
	res, err := Query(`SELECT count(*) AS c FROM t_sqlite`)
	if err != nil {
		t.Errorf("query count failed: %v", err)
		return
	}
	if len(res) == 0 {
		t.Errorf("count query empty")
		return
	}
	if fmt.Sprint(res[0]["c"]) != "0" {
		t.Errorf("rolled back data should be invisible, got %v", res[0])
	}
}

// 带前缀时，框架内置的表结构探查（sqlite_master / pragma_table_info）不受影响
func Test_TablePrefix_SqliteTableStructure(t *testing.T) {
	dir := initTablePrefixSqlite(t)
	if dir == "" {
		return
	}
	defer Close()
	if _, err := Execute(`CREATE TABLE t_sqlite (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	tns, err := fetchTables("")
	if err != nil {
		t.Errorf("fetchTables failed: %v", err)
		return
	}
	found := false
	for _, tn := range tns {
		if tn == "test_t_sqlite" {
			found = true
		}
	}
	if !found {
		t.Errorf("fetchTables should return physical test_t_sqlite, got %v", tns)
		return
	}
	if _, err := newTableStruct("", "test_t_sqlite"); err != nil {
		t.Errorf("newTableStruct with prefixed physical table failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 真实表集合匹配（rewriteSQLTablesWithSet）单元测试：纯函数，不依赖数据库

// tableSetOf 构造小写键的真实表名集合（与 fetchTableNames 返回格式一致）。
func tableSetOf(names ...string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, n := range names {
		set[strings.ToLower(n)] = struct{}{}
	}
	return set
}

func Test_rewriteSQLTables_tableSet(t *testing.T) {
	// ① 物理表带前缀：库中只有 prefixed 表 → 照常改写（与纯前缀匹配一致）
	set1 := tableSetOf("test_sys_user", "test_sys_role")
	got := rewriteSQLTablesWithSet("SELECT * FROM sys_user JOIN sys_role r ON 1=1", "test_", set1)
	if got != "SELECT * FROM test_sys_user JOIN test_sys_role r ON 1=1" {
		t.Errorf("prefixed-physical: got %q", got)
	}

	// ② 核心修复场景：改了前缀但物理表未改名（库中只有无前缀表）→ 保持原样
	set2 := tableSetOf("sys_user", "test_other")
	cases := []struct {
		in, want string
	}{
		{"SELECT * FROM sys_user", "SELECT * FROM sys_user"},
		{"INSERT INTO sys_user (name) VALUES (?)", "INSERT INTO sys_user (name) VALUES (?)"},
		{"UPDATE sys_user SET name = ? WHERE id = ?", "UPDATE sys_user SET name = ? WHERE id = ?"},
		{"DELETE FROM sys_user WHERE id = ?", "DELETE FROM sys_user WHERE id = ?"},
		// 库中不存在的表照常加前缀（如 test_other 不存在，回退前缀匹配）
		{"SELECT * FROM other", "SELECT * FROM test_other"},
		{"SELECT * FROM public.sys_user", "SELECT * FROM public.sys_user"},
	}
	for _, c := range cases {
		got := rewriteSQLTablesWithSet(c.in, "test_", set2)
		if got != c.want {
			t.Errorf("unprefixed-physical %q: rewriteSQLTablesWithSet = %q, want %q", c.in, got, c.want)
		}
	}

	// ③ 带前缀与无前缀表并存：配置意图优先 → 改写为带前缀表
	set3 := tableSetOf("sys_user", "test_sys_user")
	got = rewriteSQLTablesWithSet("SELECT * FROM sys_user", "test_", set3)
	if got != "SELECT * FROM test_sys_user" {
		t.Errorf("both-exist: got %q", got)
	}

	// ④ 库中未知（建表 DDL / 目标表确实不存在）→ 沿用前缀匹配兜底
	set4 := tableSetOf("test_sys_user")
	got = rewriteSQLTablesWithSet("CREATE TABLE IF NOT EXISTS brand_new (id int)", "test_", set4)
	if got != "CREATE TABLE IF NOT EXISTS test_brand_new (id int)" {
		t.Errorf("untracked-ddl: got %q", got)
	}

	// ⑤ 已带前缀不叠加（即使集合中有该带前缀表）
	got = rewriteSQLTablesWithSet("SELECT * FROM test_sys_user", "test_", set1)
	if got != "SELECT * FROM test_sys_user" {
		t.Errorf("already-prefixed: got %q", got)
	}

	// ⑥ 大小写不敏感：集合键与查询均归一为小写匹配
	set6 := tableSetOf("TEST_SYS_USER")
	got = rewriteSQLTablesWithSet("SELECT * FROM sys_user", "test_", set6)
	if got != "SELECT * FROM test_sys_user" {
		t.Errorf("case-insensitive prefixed: got %q", got)
	}
	got = rewriteSQLTablesWithSet("SELECT * FROM SYs_user", "test_", set2)
	if got != "SELECT * FROM SYs_user" {
		t.Errorf("case-insensitive plain: got %q", got)
	}

	// ⑦ schema 限定名 / 引号标识符同样走集合匹配
	got = rewriteSQLTablesWithSet("SELECT * FROM public.sys_user", "test_", set1)
	if got != "SELECT * FROM public.test_sys_user" {
		t.Errorf("schema-prefixed: got %q", got)
	}
	got = rewriteSQLTablesWithSet("SELECT * FROM \"public\".\"sys_user\"", "test_", set2)
	if got != "SELECT * FROM \"public\".\"sys_user\"" {
		t.Errorf("quoted-schema-unprefixed: got %q", got)
	}

	// ⑧ nil / 空集合退化为纯前缀匹配
	if got := rewriteSQLTablesWithSet("SELECT * FROM sys_user", "test_", nil); got != "SELECT * FROM test_sys_user" {
		t.Errorf("nil-set: got %q", got)
	}
	if got := rewriteSQLTablesWithSet("SELECT * FROM sys_user", "test_", map[string]struct{}{}); got != "SELECT * FROM test_sys_user" {
		t.Errorf("empty-set: got %q", got)
	}
}

// ---------------------------------------------------------------------------
// 端到端：改了前缀但物理表未改名时，SQL 引用应保持原样（命中真实表集合）

func Test_TablePrefix_SqliteExistingUnprefixed(t *testing.T) {
	if err := InitializeDatabase("sqlite", "", 0, "", "", filepath.Join(t.TempDir(), "exist.db")); err != nil {
		t.Errorf("init failed: %v", err)
		return
	}
	defer Close()
	// 先建一张无前缀物理表（模拟表未改名）
	if _, err := Execute(`CREATE TABLE t_sqlite (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Errorf("create unprefixed table failed: %v", err)
		return
	}
	// 启用 test_ 前缀：SQL 引用的 t_sqlite 在库中真实存在（无前缀），应保持原样
	SetTablePrefix("test_")
	defer SetTablePrefix("")
	if _, err := Execute(`INSERT INTO t_sqlite (name) VALUES (?)`, "keep-me"); err != nil {
		t.Errorf("insert into existing unprefixed table failed: %v", err)
		return
	}
	res, err := Query(`SELECT name FROM t_sqlite`)
	if err != nil {
		t.Errorf("query existing unprefixed table failed: %v", err)
		return
	}
	if len(res) != 1 || res[0]["name"] != "keep-me" {
		t.Errorf("query result failed, got %v", res)
	}
	// 反向：库中不存在的表仍按前缀改写（建新表走 test_ 前缀），且 DDL 后集合刷新
	if _, err := Execute(`CREATE TABLE other_table (id int)`); err != nil {
		t.Errorf("create prefixed table failed: %v", err)
		return
	}
	if _, err := Query(`SELECT * FROM other_table`); err != nil {
		t.Errorf("query prefixed table failed: %v", err)
	}
}

// 端到端：DDL 成功后表名集合缓存失效并重新获取（新建表对后续改写可见）
func Test_TablePrefix_SqliteTableSetRefresh(t *testing.T) {
	dir := initTablePrefixSqlite(t)
	if dir == "" {
		return
	}
	defer Close()
	if _, err := Execute(`CREATE TABLE t_sqlite (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Errorf("create failed: %v", err)
		return
	}
	if set := gDbConn.tableNameSet(); set == nil || set["test_t_sqlite"] != struct{}{} {
		t.Errorf("table set should contain test_t_sqlite after ddl, got %v", set)
	}
	// 再建一张表：DDL 成功使缓存失效，新表应在集合中可见
	if _, err := Execute(`CREATE TABLE t_more (id int)`); err != nil {
		t.Errorf("create second table failed: %v", err)
		return
	}
	if set := gDbConn.tableNameSet(); set == nil || set["test_t_more"] != struct{}{} {
		t.Errorf("table set should refresh with test_t_more after ddl, got %v", set)
	}
}

// ---------------------------------------------------------------------------
// 1.2 端到端：多参数（无 args 标签、无 parameterType）按位绑定，SQLite 真实执行

type MultiParamMapper struct {
	BaseMapper
	SelectByAB func(a, b string) ([]map[string]interface{}, error)
}

func initMultiParamSqlite(t *testing.T) string {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="MultiParamMapper">
  <select id="selectByAB" resultType="map">
    select a, b from pair_t where a = #{a} and b = #{b}
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "MultiParamMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	dbPath := filepath.Join(dir, "mp.db")
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return ""
	}
	return dir
}

func Test_Sqlite_MultiParamNoTag(t *testing.T) {
	dir := initMultiParamSqlite(t)
	if dir == "" {
		return
	}
	defer Close()
	if _, err := Execute(`CREATE TABLE pair_t (a TEXT, b TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	for _, row := range []string{`('x', 'y')`, `('x', 'z')`} {
		if _, err := Execute(`INSERT INTO pair_t (a, b) VALUES ` + row); err != nil {
			t.Errorf("insert failed: %v", err)
			return
		}
	}
	if err := RegisterMapper(new(MultiParamMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("MultiParamMapper").(MultiParamMapper)
	rs, err := mp.SelectByAB("x", "y")
	if err != nil {
		t.Errorf("mapper multi-param select failed: %v", err)
		return
	}
	if len(rs) != 1 {
		t.Errorf("expect 1 row for (x,y), got %d: %v", len(rs), rs)
		return
	}
	// 断言查到的确实是 b='y' 而不是 b='z'（1.2 回归：两占位符分别绑定，不再是都绑 args[0]）
	if v, ok := rs[0]["b"].(string); !ok || v != "y" {
		t.Errorf("row b = %v, want 'y' (per-slot binding failed)", rs[0]["b"])
	}
}

// ---------------------------------------------------------------------------
// 2.1 前缀映射（移除/替换）

func Test_rewriteSQLTablesWithMap(t *testing.T) {
	cases := []struct{ name, in, prefix, want string; pMap map[string]string }{
		{"remove", "select * from threedb_sys_user", "", "select * from sys_user", map[string]string{"threedb_": ""}},
		{"remove-noop", "select * from sys_user", "", "select * from sys_user", map[string]string{"threedb_": ""}},
		{"replace", "select * from threedb_sys_user", "", "select * from app_sys_user", map[string]string{"threedb_": "app_"}},
		{"map-plus-prefix", "select * from sys_user", "test_", "select * from test_sys_user", map[string]string{"threedb_": ""}},
		{"map-into-join", "select * from threedb_a join threedb_b on a.id = b.id", "", "select * from a join b on a.id = b.id", map[string]string{"threedb_": ""}},
		{"remove-schema-qualified", "select * from subsp.threedb_ods_x", "", "select * from subsp.ods_x", map[string]string{"threedb_": ""}},
		{"remove-schema-public", "select * from public.threedb_ods_x", "", "select * from public.ods_x", map[string]string{"threedb_": ""}},
		{"unmatched-keeps", "select * from app_sys_user", "", "select * from app_sys_user", map[string]string{"threedb_": ""}},
		{"remove-in-update", "update threedb_sys_user set name = 'x'", "", "update sys_user set name = 'x'", map[string]string{"threedb_": ""}},
		{"remove-in-delete", "delete from threedb_sys_user where id = 1", "", "delete from sys_user where id = 1", map[string]string{"threedb_": ""}},
		{"no-map-nil", "select * from x", "test_", "select * from test_x", nil},
		{"case-insensitive-match", "select * from THREEDB_sys_user", "", "select * from sys_user", map[string]string{"threedb_": ""}},
	}
	for _, c := range cases {
		got := rewriteSQLTablesWithMap(c.in, c.prefix, c.pMap, nil)
		if got != c.want {
			t.Errorf("%s: rewriteSQLTablesWithMap(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

func Test_rewriteSQLTablesWithMap_tableSet(t *testing.T) {
	// 库中有无前缀表 sys_user：剥前缀后能命中 → 剥
	got := rewriteSQLTablesWithMap("select * from threedb_sys_user", "", map[string]string{"threedb_": ""}, tableSetOf("sys_user"))
	if got != "select * from sys_user" {
		t.Errorf("remove when unprefixed exists: got %q", got)
	}
	// 库中只有带前缀表 threedb_sys_user：剥了反而错 → 保留原样
	got2 := rewriteSQLTablesWithMap("select * from threedb_sys_user", "", map[string]string{"threedb_": ""}, tableSetOf("threedb_sys_user"))
	if got2 != "select * from threedb_sys_user" {
		t.Errorf("keep when prefixed table exists: got %q", got2)
	}
}

func Test_parseTablePrefixMap(t *testing.T) {
	got := parseTablePrefixMap(map[string]string{"mybatis.table-prefix-map": "threedb_:,app_:subsp_"})
	if len(got) != 2 || got["threedb_"] != "" || got["app_"] != "subsp_" {
		t.Errorf("parseTablePrefixMap = %v, want {threedb_:'' app_:subsp_}", got)
	}
	if v := parseTablePrefixMap(map[string]string{}); v != nil {
		t.Errorf("no config should be nil, got %v", v)
	}
	if v := parseTablePrefixMap(map[string]string{"mybatis.table-prefix-map": ":"}); v != nil {
		t.Errorf("empty-old entries should be ignored, got %v", v)
	}
	// 无冒号条目视为移除该前缀（等价 old:）
	if v := parseTablePrefixMap(map[string]string{"mybatis.table-prefix-map": ":,bad"}); v == nil || v["bad"] != "" {
		t.Errorf("colon-less entry should mean remove prefix, got %v", v)
	}
}

// ---------------------------------------------------------------------------
// 2.1 端到端：前缀映射移除（模拟三库 prod：物理表无前缀、XML 硬编码 threedb_ 前缀）

type PrefixMapMapper struct {
	BaseMapper
	SelectPrefixed func(id int) ([]map[string]interface{}, error)
}

func initPrefixMapSqlite(t *testing.T) string {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="PrefixMapMapper">
  <select id="selectPrefixed" resultType="map">
    select id, name from threedb_sys_user where id = #{id}
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "PrefixMapMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	dbPath := filepath.Join(dir, "pm.db")
	cm := map[string]string{
		"spring.datasource.url":      "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations":   xmlDir,
		"mybatis.table-prefix-map":   "threedb_:",
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return ""
	}
	return dir
}

func Test_TablePrefixMap_SqliteRemovePrefix(t *testing.T) {
	dir := initPrefixMapSqlite(t)
	if dir == "" {
		return
	}
	defer Close()
	// prod：物理表为无前缀 sys_user
	if _, err := Execute(`CREATE TABLE sys_user (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	if _, err := Execute(`INSERT INTO sys_user (id, name) VALUES (1, 'prod_user')`); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	if err := RegisterMapper(new(PrefixMapMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("PrefixMapMapper").(PrefixMapMapper)
	rs, err := mp.SelectPrefixed(1)
	if err != nil {
		t.Errorf("mapper select prefixed failed: %v", err)
		return
	}
	if len(rs) != 1 {
		t.Errorf("expect 1 row, got %d", len(rs))
		return
	}
	if v, ok := rs[0]["name"].(string); !ok || v != "prod_user" {
		t.Errorf("row name = %v, want prod_user (prefix removed -> real table hit)", rs[0]["name"])
	}
}

// 2.3 嵌套 CTE 验证：顶层 WITH 与子查询内的 WITH 名称均不被加前缀（主循环逐 token 扫描已覆盖）
func Test_rewriteSQLTables_nestedCTE(t *testing.T) {
	q := "with outer_cte as (select id from sys_user) select * from outer_cte where id in (with inner_cte as (select id from sys_org) select id from inner_cte)"
	got := rewriteSQLTables(q, "test_")
	if strings.Contains(got, "test_inner_cte") {
		t.Errorf("nested CTE name should not be prefixed: %v", got)
	}
	if strings.Contains(got, "test_outer_cte") {
		t.Errorf("outer CTE name should not be prefixed: %v", got)
	}
	if !strings.Contains(got, "test_sys_user") || !strings.Contains(got, "test_sys_org") {
		t.Errorf("real tables should be prefixed: %v", got)
	}
}

// 2.2 表关键字边界：MERGE INTO / CREATE INDEX ... ON / RENAME TABLE ... TO
func Test_rewriteSQLTables_edgeKeywords(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"merge-into", "merge into target using src on a.id=b.id", "merge into test_target using test_src on a.id=b.id"},
		{"create-index-on", "create index idx_x on sys_user(id)", "create index idx_x on test_sys_user(id)"},
		{"rename-table", "rename table a to b", "rename table test_a to test_b"},
	}
	for _, c := range cases {
		got := rewriteSQLTables(c.in, "test_")
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

// 2.4 表集合缓存 TTL：过期后标记 stale（下次访问重取）
func Test_tableNamesCache_TTL(t *testing.T) {
	c := &tableNamesCache{names: map[string]struct{}{"a": {}}, done: true, fetchedAt: time.Now(), ttl: time.Millisecond}
	_, done, stale := c.get()
	if !done {
		t.Error("done should be true")
	}
	if stale {
		t.Error("freshly fetched should not be stale")
	}
	time.Sleep(2 * time.Millisecond)
	_, _, stale2 := c.get()
	if !stale2 {
		t.Error("ttl elapsed should mark stale")
	}
	// ttl=0（默认）永不过期
	c0 := &tableNamesCache{names: map[string]struct{}{"a": {}}, done: true, fetchedAt: time.Now().Add(-time.Hour)}
	_, _, stale0 := c0.get()
	if stale0 {
		t.Error("ttl=0 should never be stale")
	}
}

func Test_parseTablePrefixSetTTL(t *testing.T) {
	if d := parseTablePrefixSetTTL(map[string]string{}); d != 0 {
		t.Errorf("absent config should be 0, got %v", d)
	}
	if d := parseTablePrefixSetTTL(map[string]string{"mybatis.table-prefix-set-ttl": "1h"}); d != time.Hour {
		t.Errorf("1h should parse to 1h, got %v", d)
	}
	if d := parseTablePrefixSetTTL(map[string]string{"mybatis.table-prefix-set-ttl": "bad"}); d != 0 {
		t.Errorf("bad duration should be 0, got %v", d)
	}
}
