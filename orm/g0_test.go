package orm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// G0 改造端到端测试（SQLite，无外部依赖）：
//   - 事务上下文：WithTx 回调事务 / 嵌套复用 / 不占 TCC 槽 / ctx 优先级 / Mapper ctx 参数
//   - 强类型扫描直通道：QueryTo / QueryToContext 的 struct/slice/map/scular 形态

type G0TestModel struct {
	Id   int64  `db:"id,pk"`
	Name string `db:"name"`
}

type G0TestMapper struct {
	BaseMapper
	Insert    func(ctx context.Context, model G0TestModel) (int64, error)
	SelectAll func(ctx context.Context) ([]G0TestModel, error)
}

type G0TagModel struct {
	Id       int64  `db:"id,pk"`
	UserName string `db:"user_name"`
	Nick     string `db:"nickName"` // 非蛇形改名：仅 db tag 索引可匹配
}

type G0Level string

type G0LevelModel struct {
	Id    int64    `db:"id,pk"`
	Name  string   `db:"name"`
	Level G0Level  `db:"level"`
}

// initG0Test 初始化 SQLite 数据源 + t_g0 表，返回临时目录（失败返回 ""）。
func initG0Test(t *testing.T) string {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="G0TestMapper">
  <insert id="insert" parameterType="G0TestModel">
    insert into t_g0 (name) values (#{name})
  </insert>
  <select id="selectAll" resultType="G0TestModel">
    select id, name from t_g0 order by id
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "G0TestMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	dbPath := filepath.Join(dir, "g0.db")
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return ""
	}
	if _, err := Execute(`CREATE TABLE t_g0 (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return ""
	}
	RegisterModel(new(G0TestModel))
	RegisterModel(new(G0TagModel))
	RegisterModel(new(G0LevelModel))
	if err := RegisterMapper(new(G0TestMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return ""
	}
	return dir
}

func g0Count(t *testing.T) int64 {
	res, err := Query(`SELECT count(*) as c FROM t_g0`)
	if err != nil {
		t.Errorf("query count failed: %v", err)
		return -1
	}
	var n int64
	fmt.Sscanf(fmt.Sprintf("%v", res[0]["c"]), "%d", &n)
	return n
}

// ---------- 事务上下文改造（G0 §4.1） ----------

func Test_WithTxCommit(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	err := WithTx(context.Background(), func(ctx context.Context) error {
		if _, err := ExecuteContext(ctx, `INSERT INTO t_g0 (name) VALUES (?)`, "with_tx"); err != nil {
			return err
		}
		// 事务内可见（经 ctx 查询走同一事务；Background 查询因隔离性看不到未提交数据）
		var cnt int64
		if err := QueryToContext(ctx, &cnt, `SELECT count(*) FROM t_g0`); err != nil {
			return err
		}
		if cnt != 1 {
			return fmt.Errorf("row count inside tx = %d, want 1", cnt)
		}
		return nil
	})
	if err != nil {
		t.Errorf("WithTx failed: %v", err)
		return
	}
	if n := g0Count(t); n != 1 {
		t.Errorf("row count after commit = %d, want 1", n)
	}
}

func Test_WithTxRollback(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	sentinel := errors.New("boom")
	err := WithTx(context.Background(), func(ctx context.Context) error {
		if _, err := ExecuteContext(ctx, `INSERT INTO t_g0 (name) VALUES (?)`, "rolled_back"); err != nil {
			return err
		}
		return sentinel
	})
	if err != sentinel {
		t.Errorf("WithTx should return fn's error, got %v", err)
		return
	}
	if n := g0Count(t); n != 0 {
		t.Errorf("row count after rollback = %d, want 0", n)
	}
}

func Test_WithTxPanicRollback(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				if r != "boom" {
					t.Errorf("recovered panic = %v, want boom", r)
				}
			}
		}()
		_ = WithTx(context.Background(), func(ctx context.Context) error {
			if _, err := ExecuteContext(ctx, `INSERT INTO t_g0 (name) VALUES (?)`, "panic_row"); err != nil {
				return err
			}
			panic("boom")
		})
	}()
	if !panicked {
		t.Error("panic should propagate out of WithTx")
	}
	if n := g0Count(t); n != 0 {
		t.Errorf("row count after panic rollback = %d, want 0", n)
	}
}

func Test_WithTxNested(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	// 内层返回错误 → 嵌套复用外层事务 → 外层整体回滚
	sentinel := errors.New("inner boom")
	err := WithTx(context.Background(), func(ctx context.Context) error {
		if _, err := ExecuteContext(ctx, `INSERT INTO t_g0 (name) VALUES (?)`, "outer"); err != nil {
			return err
		}
		return WithTx(ctx, func(ctx context.Context) error {
			if _, err := ExecuteContext(ctx, `INSERT INTO t_g0 (name) VALUES (?)`, "inner"); err != nil {
				return err
			}
			return sentinel
		})
	})
	if err != sentinel {
		t.Errorf("nested WithTx should propagate inner error, got %v", err)
		return
	}
	if n := g0Count(t); n != 0 {
		t.Errorf("row count after nested rollback = %d, want 0", n)
	}

	// 嵌套全部成功 → 最外层一次提交
	err = WithTx(context.Background(), func(ctx context.Context) error {
		if _, err := ExecuteContext(ctx, `INSERT INTO t_g0 (name) VALUES (?)`, "outer2"); err != nil {
			return err
		}
		return WithTx(ctx, func(ctx context.Context) error {
			if _, err := ExecuteContext(ctx, `INSERT INTO t_g0 (name) VALUES (?)`, "inner2"); err != nil {
				return err
			}
			return nil
		})
	})
	if err != nil {
		t.Errorf("nested WithTx commit failed: %v", err)
		return
	}
	if n := g0Count(t); n != 2 {
		t.Errorf("row count after nested commit = %d, want 2", n)
	}
}

func Test_WithTxNotOccupySlot(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	// WithTx 不占用 TCC 全局事务槽：回调内仍可 Begin()（无语句执行，无锁冲突）
	err := WithTx(context.Background(), func(ctx context.Context) error {
		tx2, err := Begin()
		if err != nil {
			return fmt.Errorf("Begin inside WithTx should succeed (slot not occupied), got: %v", err)
		}
		if err := tx2.Rollback(); err != nil {
			return fmt.Errorf("rollback tx2 failed: %v", err)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
		return
	}
	// WithTx 结束后全局槽仍空闲
	tx3, err := Begin()
	if err != nil {
		t.Errorf("Begin after WithTx should succeed, got: %v", err)
		return
	}
	_ = tx3.Rollback()
}

func Test_WithTxConcurrent(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	if _, err := Execute(`INSERT INTO t_g0 (name) VALUES (?)`, "seed"); err != nil {
		t.Errorf("seed insert failed: %v", err)
		return
	}
	// 多 goroutine 并发 WithTx（只读事务，SQLite 读锁兼容）：互不冲突、各自提交
	const n = 8
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = WithTx(context.Background(), func(ctx context.Context) error {
				var cnt int64
				if err := QueryToContext(ctx, &cnt, `SELECT count(*) FROM t_g0`); err != nil {
					return err
				}
				if cnt != 1 {
					return fmt.Errorf("count in concurrent tx = %d, want 1", cnt)
				}
				return nil
			})
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("concurrent WithTx #%d failed: %v", i, err)
		}
	}
}

func Test_TransactionContextPrecedence(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	// 全局槽事务 txA 开启（不执行语句，不持锁）
	txA, err := Begin()
	if err != nil {
		t.Errorf("Begin failed: %v", err)
		return
	}
	// ctx 事务 txB 写入：优先于全局槽
	err = WithTx(context.Background(), func(ctx context.Context) error {
		if _, err := ExecuteContext(ctx, `INSERT INTO t_g0 (name) VALUES (?)`, "from_ctx_tx"); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Errorf("WithTx failed: %v", err)
		return
	}
	// txA 回滚不应影响 txB 已提交的数据（证明写入走了 ctx 事务而非全局槽）
	if err := txA.Rollback(); err != nil {
		t.Errorf("txA rollback failed: %v", err)
		return
	}
	if n := g0Count(t); n != 1 {
		t.Errorf("row count after txA rollback = %d, want 1 (ctx tx should take precedence)", n)
	}
}

func Test_MapperContextTx(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	mp := NewMapper("G0TestMapper").(G0TestMapper)

	// Mapper 方法声明 context.Context 参数 → 自动加入 WithTx 事务
	err := WithTx(context.Background(), func(ctx context.Context) error {
		if _, err := mp.Insert(ctx, G0TestModel{Name: "mapper_tx"}); err != nil {
			return err
		}
		rs, err := mp.SelectAll(ctx)
		if err != nil {
			return err
		}
		if len(rs) != 1 {
			return fmt.Errorf("rows inside tx = %d, want 1", len(rs))
		}
		if rs[0].Name != "mapper_tx" {
			return fmt.Errorf("name = %q, want mapper_tx", rs[0].Name)
		}
		return errors.New("force rollback")
	})
	if err == nil || err.Error() != "force rollback" {
		t.Errorf("WithTx should propagate rollback error, got %v", err)
		return
	}
	// 回滚后不可见
	rs, err := mp.SelectAll(context.Background())
	if err != nil {
		t.Errorf("mapper select after rollback failed: %v", err)
		return
	}
	if len(rs) != 0 {
		t.Errorf("rows after rollback = %d, want 0", len(rs))
	}

	// 提交路径
	err = WithTx(context.Background(), func(ctx context.Context) error {
		_, err := mp.Insert(ctx, G0TestModel{Name: "mapper_tx_ok"})
		return err
	})
	if err != nil {
		t.Errorf("WithTx with mapper commit failed: %v", err)
		return
	}
	rs, err = mp.SelectAll(context.Background())
	if err != nil {
		t.Errorf("mapper select after commit failed: %v", err)
		return
	}
	if len(rs) != 1 || rs[0].Name != "mapper_tx_ok" {
		t.Errorf("rows after commit = %+v, want 1 row mapper_tx_ok", rs)
	}
}

// ---------- 强类型扫描直通道（G0 §4.2） ----------

func Test_QueryToStruct(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	if _, err := Execute(`INSERT INTO t_g0 (name) VALUES ('alice'), ('bob')`); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	var m G0TestModel
	if err := QueryTo(&m, `SELECT id, name FROM t_g0 WHERE name = ?`, "alice"); err != nil {
		t.Errorf("QueryTo struct failed: %v", err)
		return
	}
	if m.Id != 1 || m.Name != "alice" {
		t.Errorf("QueryTo struct = %+v, want {Id:1 Name:alice}", m)
	}

	// 无行 → sql.ErrNoRows
	var m2 G0TestModel
	if err := QueryTo(&m2, `SELECT id, name FROM t_g0 WHERE id = -1`); err != sql.ErrNoRows {
		t.Errorf("QueryTo struct with no rows should return sql.ErrNoRows, got %v", err)
	}
}

func Test_QueryToSlice(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	if _, err := Execute(`INSERT INTO t_g0 (name) VALUES ('a'), ('b'), ('c')`); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	var users []G0TestModel
	if err := QueryTo(&users, `SELECT id, name FROM t_g0 ORDER BY id`); err != nil {
		t.Errorf("QueryTo slice failed: %v", err)
		return
	}
	if len(users) != 3 {
		t.Errorf("QueryTo slice len = %d, want 3", len(users))
		return
	}
	if users[0].Name != "a" || users[2].Name != "c" {
		t.Errorf("QueryTo slice order = %v", users)
	}

	// *struct 元素切片
	var ptrs []*G0TestModel
	if err := QueryTo(&ptrs, `SELECT id, name FROM t_g0 ORDER BY id`); err != nil {
		t.Errorf("QueryTo ptr slice failed: %v", err)
		return
	}
	if len(ptrs) != 3 || ptrs[1] == nil || ptrs[1].Name != "b" {
		t.Errorf("QueryTo ptr slice = %v", ptrs)
	}

	// 空结果 → 空切片无错误
	var empty []G0TestModel
	if err := QueryTo(&empty, `SELECT id, name FROM t_g0 WHERE id > 100`); err != nil {
		t.Errorf("QueryTo empty slice should not fail, got %v", err)
		return
	}
	if len(empty) != 0 {
		t.Errorf("QueryTo empty slice len = %d, want 0", len(empty))
	}
}

func Test_QueryToMapAndScalar(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	if _, err := Execute(`INSERT INTO t_g0 (name) VALUES ('alice')`); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	var mp map[string]interface{}
	if err := QueryTo(&mp, `SELECT id, name FROM t_g0 WHERE name = 'alice'`); err != nil {
		t.Errorf("QueryTo map failed: %v", err)
		return
	}
	if mp["name"] != "alice" {
		t.Errorf("QueryTo map name = %v, want alice", mp["name"])
	}

	var n int64
	if err := QueryTo(&n, `SELECT count(*) FROM t_g0`); err != nil {
		t.Errorf("QueryTo scalar failed: %v", err)
		return
	}
	if n != 1 {
		t.Errorf("QueryTo scalar = %d, want 1", n)
	}

	// 标量多列 → 报错
	var bad int64
	if err := QueryTo(&bad, `SELECT id, name FROM t_g0`); err == nil {
		t.Error("QueryTo scalar with 2 columns should fail")
	}
}

func Test_QueryToNullZeroValue(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	if _, err := Execute(`INSERT INTO t_g0 (name) VALUES (NULL)`); err != nil {
		t.Errorf("insert null failed: %v", err)
		return
	}
	var users []G0TestModel
	if err := QueryTo(&users, `SELECT id, name FROM t_g0`); err != nil {
		t.Errorf("QueryTo with NULL failed: %v", err)
		return
	}
	if len(users) != 1 {
		t.Errorf("rows = %d, want 1", len(users))
		return
	}
	if users[0].Name != "" {
		t.Errorf("NULL should scan to zero value, got %q", users[0].Name)
	}
}

func Test_QueryToTagColumn(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	if _, err := Execute(`CREATE TABLE t_g0tag (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_name TEXT,
		nickName TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	if _, err := Execute(`INSERT INTO t_g0tag (user_name, nickName) VALUES ('alice', 'Al')`); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	var users []G0TagModel
	if err := QueryTo(&users, `SELECT id, user_name, nickName FROM t_g0tag`); err != nil {
		t.Errorf("QueryTo tag model failed: %v", err)
		return
	}
	if len(users) != 1 {
		t.Errorf("rows = %d, want 1", len(users))
		return
	}
	// user_name 蛇形：tag 与四策略均可匹配
	if users[0].UserName != "alice" {
		t.Errorf("UserName = %q, want alice", users[0].UserName)
	}
	// nickName 非蛇形改名：仅 db tag 索引可匹配
	if users[0].Nick != "Al" {
		t.Errorf("Nick = %q, want Al (db tag column index)", users[0].Nick)
	}
}

func Test_QueryToTypeHandler(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	RegisterTypeHandlerFor[G0Level](func(source interface{}) (G0Level, error) {
		s, ok := source.(string)
		if !ok {
			return "", fmt.Errorf("G0Level handler: expect string, got %T", source)
		}
		return G0Level("LV" + s), nil
	})

	if _, err := Execute(`CREATE TABLE t_g0lv (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		level TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	if _, err := Execute(`INSERT INTO t_g0lv (name, level) VALUES ('alice', '3')`); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	var users []G0LevelModel
	if err := QueryTo(&users, `SELECT id, name, level FROM t_g0lv`); err != nil {
		t.Errorf("QueryTo with TypeHandler failed: %v", err)
		return
	}
	if len(users) != 1 {
		t.Errorf("rows = %d, want 1", len(users))
		return
	}
	if users[0].Level != "LV3" {
		t.Errorf("Level = %q, want LV3 (TypeHandler applied)", users[0].Level)
	}
}

func Test_QueryToInTx(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	err := WithTx(context.Background(), func(ctx context.Context) error {
		if _, err := ExecuteContext(ctx, `INSERT INTO t_g0 (name) VALUES (?)`, "in_tx"); err != nil {
			return err
		}
		var users []G0TestModel
		if err := QueryToContext(ctx, &users, `SELECT id, name FROM t_g0`); err != nil {
			return err
		}
		if len(users) != 1 {
			return fmt.Errorf("rows in tx = %d, want 1", len(users))
		}
		return errors.New("force rollback")
	})
	if err == nil || err.Error() != "force rollback" {
		t.Errorf("WithTx should propagate rollback error, got %v", err)
		return
	}
	// 回滚后直通道查询为空
	var users []G0TestModel
	if err := QueryTo(&users, `SELECT id, name FROM t_g0`); err != nil {
		t.Errorf("QueryTo after rollback failed: %v", err)
		return
	}
	if len(users) != 0 {
		t.Errorf("rows after rollback = %d, want 0", len(users))
	}
}

func Test_QueryToInvalidDst(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	var notPtr G0TestModel
	if err := QueryTo(notPtr, `SELECT id FROM t_g0`); err == nil {
		t.Error("QueryTo with non-pointer dst should fail")
	}
	var nilPtr *G0TestModel
	if err := QueryTo(&nilPtr, `SELECT id FROM t_g0`); err == nil {
		t.Error("QueryTo with **struct dst should fail")
	}
	if _, err := Execute(`INSERT INTO t_g0 (name) VALUES ('alice')`); err != nil {
		t.Error("seed row failed:", err)
		return
	}
	var scalarSlice []int64
	if err := QueryTo(&scalarSlice, `SELECT id FROM t_g0`); err != nil {
		t.Error("QueryTo with scalar slice dst (1 column) should succeed:", err)
	}
	if len(scalarSlice) == 0 {
		t.Error("scalar slice dst should contain rows")
	}
	if err := QueryTo(&scalarSlice, `SELECT id, name FROM t_g0`); err == nil {
		t.Error("QueryTo with scalar slice dst (2 columns) should fail")
	}
}
