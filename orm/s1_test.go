package orm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/bnulwh/mybatis-go/types"
	"github.com/bnulwh/mybatis-go/types/sqlfragment"
)

// S1（sqlc 方案 A）端到端测试：XML 存量静态 select 抽取 → 生成的 Querier 函数体
// （(db *DB) QueryToContext + ? 占位参数）在 SQLite 上真实执行。
// 覆盖：无参 resultMap 查询 / parameterType=Long 单参 / 无 parameterType 多占位符
// 按位绑定 / resultType 标量 / 动态语句不可提取 / WithTx 事务内执行。

type S1User struct {
	Id       int64  `db:"id,pk"`
	UserName string `db:"user_name"`
	Age      int    `db:"age"`
}

const s1MapperXML = `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="S1UserMapper">
  <resultMap id="BaseResultMap" type="S1User">
    <id column="id" jdbcType="BIGINT" property="id" />
    <result column="user_name" jdbcType="VARCHAR" property="userName" />
    <result column="age" jdbcType="INTEGER" property="age" />
  </resultMap>
  <sql id="base_columns">id, user_name, age</sql>
  <select id="selectAll" resultMap="BaseResultMap">
    select <include refid="base_columns"/>
    from s1_user
    order by id
  </select>
  <select id="selectById" parameterType="Long" resultMap="BaseResultMap">
    select <include refid="base_columns"/>
    from s1_user
    where id = #{id,jdbcType=BIGINT}
  </select>
  <select id="selectByAgeRange">
    select id from s1_user
    where age &gt;= #{minAge,jdbcType=INTEGER} and age &lt;= #{maxAge,jdbcType=INTEGER}
  </select>
  <select id="countAll" resultType="int">
    select count(*) from s1_user
  </select>
  <select id="selectDynamic" parameterType="S1User" resultMap="BaseResultMap">
    select <include refid="base_columns"/>
    from s1_user
    <where>
      <if test="userName != null and userName != ''">
        and user_name = #{userName}
      </if>
    </where>
  </select>
</mapper>
`

// initS1Test 初始化 SQLite + s1_user 表（3 行种子），返回 mapper XML 目录。
func initS1Test(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "S1UserMapper.xml"), []byte(s1MapperXML), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + filepath.Join(dir, "s1.db"),
		"mybatis.mapper-locations": xmlDir,
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return ""
	}
	if _, err := Execute(`CREATE TABLE s1_user (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_name TEXT,
		age INTEGER)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return ""
	}
	for _, seed := range []string{
		`INSERT INTO s1_user (user_name, age) VALUES ('alice', 18)`,
		`INSERT INTO s1_user (user_name, age) VALUES ('bob', 25)`,
		`INSERT INTO s1_user (user_name, age) VALUES ('carol', 32)`,
	} {
		if _, err := Execute(seed); err != nil {
			t.Errorf("seed failed: %v", err)
			return ""
		}
	}
	return xmlDir
}

// extractS1Static 提取指定 select 的静态 SQL 与参数（失败返回空 SQL）。
func extractS1Static(t *testing.T, mp *types.SqlMapper, id string) (string, []sqlfragment.StaticParam) {
	t.Helper()
	fn := mp.NamedFunctions[id]
	if fn == nil {
		t.Errorf("function %v not found", id)
		return "", nil
	}
	sqlstr, params, ok := sqlfragment.ExtractStaticSQL(fn.Items)
	if !ok {
		t.Errorf("function %v should be static", id)
		return "", nil
	}
	return sqlstr, params
}

// Test_S1QuerierStaticSelect 静态 select 抽取结果经 QueryToContext 真实执行。
func Test_S1QuerierStaticSelect(t *testing.T) {
	xmlDir := initS1Test(t)
	if xmlDir == "" {
		return
	}
	defer Close()
	mps := types.NewSqlMappers(xmlDir)
	mp := mps.NamedMappers["S1UserMapper"]
	if mp == nil {
		t.Error("S1UserMapper not loaded")
		return
	}
	ctx := context.Background()
	db := GetActiveDataSource()

	// 无参 resultMap 查询（include 展开 + db tag 列名匹配）
	sqlAll, paramsAll := extractS1Static(t, mp, "selectAll")
	if sqlAll == "" {
		return
	}
	if len(paramsAll) != 0 {
		t.Error("selectAll should have no params, got", paramsAll)
	}
	var rows []S1User
	if err := db.QueryToContext(ctx, &rows, sqlAll); err != nil {
		t.Error("selectAll execute failed:", err)
		return
	}
	if len(rows) != 3 || rows[1].UserName != "bob" || rows[1].Age != 25 {
		t.Error("selectAll rows unexpected:", rows)
	}

	// parameterType=Long 单参绑定
	sqlById, paramsById := extractS1Static(t, mp, "selectById")
	if sqlById == "" {
		return
	}
	if len(paramsById) != 1 || paramsById[0].Name != "id" {
		t.Error("selectById params unexpected:", paramsById)
	}
	var rows2 []S1User
	if err := db.QueryToContext(ctx, &rows2, sqlById, int64(2)); err != nil {
		t.Error("selectById execute failed:", err)
		return
	}
	if len(rows2) != 1 || rows2[0].UserName != "bob" {
		t.Error("selectById rows unexpected:", rows2)
	}

	// 无 parameterType：多占位符按位绑定（minAge=20, maxAge=30 → 仅 bob）
	sqlRange, paramsRange := extractS1Static(t, mp, "selectByAgeRange")
	if sqlRange == "" {
		return
	}
	if len(paramsRange) != 2 || paramsRange[0].Name != "minAge" || paramsRange[1].Name != "maxAge" {
		t.Error("selectByAgeRange params unexpected:", paramsRange)
	}
	var ids []int64
	if err := db.QueryToContext(ctx, &ids, sqlRange, 20, 30); err != nil {
		t.Error("selectByAgeRange execute failed:", err)
		return
	}
	if len(ids) != 1 || ids[0] != 2 {
		t.Error("selectByAgeRange ids unexpected:", ids)
	}

	// resultType 标量
	sqlCnt, _ := extractS1Static(t, mp, "countAll")
	if sqlCnt == "" {
		return
	}
	var cnt int32
	if err := db.QueryToContext(ctx, &cnt, sqlCnt); err != nil {
		t.Error("countAll execute failed:", err)
		return
	}
	if cnt != 3 {
		t.Error("countAll unexpected:", cnt)
	}

	// 动态语句不可提取
	fnDyn := mp.NamedFunctions["selectDynamic"]
	if fnDyn == nil {
		t.Error("selectDynamic not found")
		return
	}
	if _, _, ok := sqlfragment.ExtractStaticSQL(fnDyn.Items); ok {
		t.Error("selectDynamic (with <if>) should not be static")
	}
}

// Test_S1QuerierGeneratedFile 生成的 Querier 文件内容与生成的 SQL 语义一致（语法 + 关键签名）。
func Test_S1QuerierGeneratedFile(t *testing.T) {
	xmlDir := initS1Test(t)
	if xmlDir == "" {
		return
	}
	defer Close()
	mps := types.NewSqlMappers(xmlDir)
	mp := mps.NamedMappers["S1UserMapper"]
	if mp == nil {
		t.Error("S1UserMapper not loaded")
		return
	}
	out := t.TempDir()
	stats, err := mp.GenerateQuerierFile(out, "querier", nil)
	if err != nil {
		t.Error("GenerateQuerierFile failed:", err)
		return
	}
	if stats.Functions == 0 || stats.File == "" {
		t.Error("no querier file generated:", stats)
		return
	}
	if err := types.ParseQuerierFileSyntax(stats.File); err != nil {
		t.Error("generated file syntax invalid:", err)
	}
	content, err := os.ReadFile(stats.File)
	if err != nil {
		t.Error("read generated file failed:", err)
		return
	}
	src := string(content)
	for _, want := range []string{
		"type S1UserQuerier interface {",
		"SelectAll(ctx context.Context) ([]S1User, error)",
		"SelectById(ctx context.Context, id int64) ([]S1User, error)",
		"SelectByAgeRange(ctx context.Context, minAge int, maxAge int) ([]int64, error)",
		"CountAll(ctx context.Context) ([]int32, error)",
		"q.db.QueryToContext(ctx, &rows,",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated file missing %q\n", want)
		}
	}
	if strings.Contains(src, "#{") || strings.Contains(src, "${") {
		t.Error("generated file should not contain mybatis placeholders")
	}
}

// Test_S1QuerierInTransaction WithTx 事务内执行静态 SQL（ctx 事务路由对生成代码同样生效）。
func Test_S1QuerierInTransaction(t *testing.T) {
	xmlDir := initS1Test(t)
	if xmlDir == "" {
		return
	}
	defer Close()
	mps := types.NewSqlMappers(xmlDir)
	mp := mps.NamedMappers["S1UserMapper"]
	if mp == nil {
		t.Error("S1UserMapper not loaded")
		return
	}
	sqlCnt, _ := extractS1Static(t, mp, "countAll")
	if sqlCnt == "" {
		return
	}
	err := WithTx(context.Background(), func(ctx context.Context) error {
		if _, err := ExecuteContext(ctx, `INSERT INTO s1_user (user_name, age) VALUES ('dave', 40)`); err != nil {
			return err
		}
		var cnt int32
		// 事务内经 ctx 路由读取（含未提交写入）
		if err := GetActiveDataSource().QueryToContext(ctx, &cnt, sqlCnt); err != nil {
			return err
		}
		if cnt != 4 {
			return fmt.Errorf("count mismatch in transaction, got %v", cnt)
		}
		return nil
	})
	if err != nil {
		t.Error("WithTx with static sql failed:", err)
		return
	}
	// 事务提交后可见
	var cnt int32
	if err := QueryTo(&cnt, sqlCnt); err != nil {
		t.Error("QueryTo count failed:", err)
		return
	}
	if cnt != 4 {
		t.Error("count after commit unexpected:", cnt)
	}
}
