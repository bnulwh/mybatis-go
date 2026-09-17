package gormish_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite" // SQLite 驱动（与 orm 包测试相同的驱动注册）

	"github.com/bnulwh/mybatis-go/orm"
	"github.com/bnulwh/mybatis-go/orm/gormish"
)

// G1 gormish 链式 API 端到端测试（SQLite，无外部依赖）。
// 覆盖：链式查询各子句 / First/Last/Count / Create 主键回填与批量 /
// Update/Updates/Delete 零条件保护 / Transaction 提交回滚嵌套 /
// Raw+Scan+Exec / WithContext 参与 WithTx / 克隆隔离 / 无 tag 模型回退。

type G1User struct {
	Id    int64  `db:"id,pk"`
	Name  string `db:"name"`
	Age   int    `db:"age"`
	Email string `db:"email"`
}

func (G1User) TableName() string { return "g1_user" }

// G1PlainRow 无 db tag 结构体：走 gormish 本地元数据回退（TableName 方法 + ID 主键约定）。
type G1PlainRow struct {
	Id   int64
	Name string
}

func (G1PlainRow) TableName() string { return "g1_plain" }

// G1CatModel 无 db tag 且无 TableName 方法：表名按「短名去 Model 后缀 snake_case」推导。
type G1CatModel struct {
	Id   int64
	Name string
}

type G1AgeStat struct {
	Age int   `db:"age"`
	Cnt int64 `db:"cnt"`
}

// initG1Test 初始化 SQLite 数据源 + 测试表，返回 gormish 句柄（失败返回 nil）。
func initG1Test(t *testing.T) *gormish.DB {
	t.Helper()
	dir := t.TempDir()
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + filepath.Join(dir, "g1.db"),
		"mybatis.mapper-locations": "",
	}
	if err := orm.InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return nil
	}
	gdb, err := gormish.Open()
	if err != nil {
		t.Errorf("gormish open failed: %v", err)
		return nil
	}
	for _, ddl := range []string{
		`CREATE TABLE g1_user (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, age INTEGER, email TEXT)`,
		`CREATE TABLE g1_plain (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`,
		`CREATE TABLE g1_cat (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`,
	} {
		if _, err := gdb.Exec(ddl); err != nil {
			t.Errorf("create table failed: %v", err)
			return nil
		}
	}
	return gdb
}

// seedUsers 插入 5 个用户并返回句柄。
func seedUsers(t *testing.T, gdb *gormish.DB) {
	t.Helper()
	for i, name := range []string{"alice", "bob", "carol", "dave", "erin"} {
		user := G1User{Name: name, Age: 18 + i*7, Email: name + "@test.com"}
		if err := gdb.Create(&user); err != nil {
			t.Errorf("seed user %s failed: %v", name, err)
			return
		}
		if user.Id == 0 {
			t.Errorf("seed user %s: auto-increment pk not backfilled", name)
		}
	}
}

// ---------- 会话构造 ----------

func Test_OpenAndUse(t *testing.T) {
	if _, err := gormish.Open(); err == nil {
		t.Error("Open before orm init should fail")
	}
	if _, err := gormish.Use("notfound"); err == nil {
		t.Error("Use unknown datasource should fail")
	}
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()
	if _, err := gormish.Open(); err != nil {
		t.Errorf("Open after init failed: %v", err)
	}
	if _, err := gormish.Use("default"); err != nil {
		t.Errorf("Use default datasource failed: %v", err)
	}
}

// ---------- 链式查询 ----------

func Test_FindChain(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()
	seedUsers(t, gdb)

	var users []G1User
	err := gdb.Table("g1_user").
		Select("id", "name", "age").
		Where("age > ?", 20).
		Order("age ASC").
		Limit(2).
		Find(&users)
	if err != nil {
		t.Errorf("Find chain failed: %v", err)
		return
	}
	if len(users) != 2 {
		t.Errorf("Find chain rows = %d, want 2", len(users))
		return
	}
	if users[0].Name != "bob" || users[1].Name != "carol" {
		t.Errorf("Find chain order wrong: %s, %s", users[0].Name, users[1].Name)
	}
}

func Test_FindLimitOffset(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()
	seedUsers(t, gdb)

	var users []G1User
	err := gdb.Table("g1_user").Order("id").Limit(2).Offset(2).Find(&users)
	if err != nil {
		t.Errorf("Find limit/offset failed: %v", err)
		return
	}
	if len(users) != 2 {
		t.Errorf("limit/offset rows = %d, want 2", len(users))
		return
	}
	if users[0].Name != "carol" || users[1].Name != "dave" {
		t.Errorf("limit/offset rows = %s, %s, want carol/dave", users[0].Name, users[1].Name)
	}
}

func Test_FindDeriveTable(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()
	seedUsers(t, gdb)

	var users []G1User
	if err := gdb.Find(&users); err != nil {
		t.Errorf("Find with derived table failed: %v", err)
		return
	}
	if len(users) != 5 {
		t.Errorf("Find derived rows = %d, want 5", len(users))
	}
}

func Test_FirstLast(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()
	seedUsers(t, gdb)

	var first G1User
	if err := gdb.Table("g1_user").First(&first); err != nil {
		t.Errorf("First failed: %v", err)
		return
	}
	if first.Id != 1 || first.Name != "alice" {
		t.Errorf("First = id %d %s, want id 1 alice", first.Id, first.Name)
	}
	var last G1User
	if err := gdb.Table("g1_user").Last(&last); err != nil {
		t.Errorf("Last failed: %v", err)
		return
	}
	if last.Id != 5 || last.Name != "erin" {
		t.Errorf("Last = id %d %s, want id 5 erin", last.Id, last.Name)
	}
	// First 保留链上显式排序
	var youngest G1User
	if err := gdb.Table("g1_user").Order("age DESC").First(&youngest); err != nil {
		t.Errorf("First with explicit order failed: %v", err)
		return
	}
	if youngest.Name != "erin" {
		t.Errorf("First with explicit order = %s, want erin", youngest.Name)
	}
}

func Test_FirstNoRows(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()

	var user G1User
	err := gdb.Table("g1_user").Where("age > ?", 100).First(&user)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("First no-row err = %v, want sql.ErrNoRows", err)
	}
}

func Test_Count(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()
	seedUsers(t, gdb)

	var n int64
	if err := gdb.Table("g1_user").Where("age >= ?", 25).Count(&n); err != nil {
		t.Errorf("Count failed: %v", err)
		return
	}
	if n != 4 {
		t.Errorf("Count = %d, want 4", n)
	}
	var distinctAge int64
	if err := gdb.Table("g1_user").Distinct("age").Count(&distinctAge); err != nil {
		t.Errorf("Distinct Count failed: %v", err)
		return
	}
	if distinctAge != 5 {
		t.Errorf("Distinct Count = %d, want 5", distinctAge)
	}
}

func Test_WhereVariants(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()
	seedUsers(t, gdb)

	// map 等值条件
	var users []G1User
	if err := gdb.Table("g1_user").Where(map[string]interface{}{"name": "alice"}).Find(&users); err != nil {
		t.Errorf("Where map failed: %v", err)
		return
	}
	if len(users) != 1 || users[0].Name != "alice" {
		t.Errorf("Where map rows = %d", len(users))
	}
	// IN 展开
	var inUsers []G1User
	if err := gdb.Table("g1_user").Where("name IN ?", []string{"alice", "bob", "none"}).Find(&inUsers); err != nil {
		t.Errorf("Where IN failed: %v", err)
		return
	}
	if len(inUsers) != 2 {
		t.Errorf("Where IN rows = %d, want 2", len(inUsers))
	}
	// OR 条件
	var orUsers []G1User
	if err := gdb.Table("g1_user").Where("age < ?", 20).Or("age > ?", 40).Find(&orUsers); err != nil {
		t.Errorf("Where Or failed: %v", err)
		return
	}
	if len(orUsers) != 2 {
		t.Errorf("Where Or rows = %d, want 2", len(orUsers))
	}
	// 不支持的条件的类型：链上聚合报错
	if err := gdb.Table("g1_user").Where(123).Find(&users); err == nil {
		t.Error("Where with unsupported type should fail")
	}
}

func Test_GroupHaving(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()
	if _, err := gdb.Exec(`INSERT INTO g1_user (name, age, email) VALUES ('a1', 20, 'a@x.com'), ('a2', 20, 'b@x.com'), ('a3', 30, 'c@x.com')`); err != nil {
		t.Errorf("seed failed: %v", err)
		return
	}

	var stats []G1AgeStat
	err := gdb.Table("g1_user").
		Select("age, count(*) as cnt").
		Group("age").
		Having("count(*) > ?", 1).
		Find(&stats)
	if err != nil {
		t.Errorf("Group/Having failed: %v", err)
		return
	}
	if len(stats) != 1 || stats[0].Age != 20 || stats[0].Cnt != 2 {
		t.Errorf("Group/Having result = %+v, want age 20 cnt 2", stats)
	}
}

// ---------- CRUD ----------

func Test_CreateBackfillAndExplicitPK(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()

	user := G1User{Name: "tom", Age: 22, Email: "tom@x.com"}
	if err := gdb.Create(&user); err != nil {
		t.Errorf("Create failed: %v", err)
		return
	}
	if user.Id != 1 {
		t.Errorf("Create pk backfill = %d, want 1", user.Id)
	}
	// 显式非零主键参与插入
	explicit := G1User{Id: 100, Name: "jerry", Age: 20, Email: "j@x.com"}
	if err := gdb.Create(&explicit); err != nil {
		t.Errorf("Create explicit pk failed: %v", err)
		return
	}
	var got G1User
	if err := gdb.Table("g1_user").Where("id = ?", 100).First(&got); err != nil {
		t.Errorf("query explicit pk failed: %v", err)
		return
	}
	if got.Name != "jerry" {
		t.Errorf("explicit pk row = %s, want jerry", got.Name)
	}
}

func Test_CreateBatch(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()

	batch := []G1User{
		{Name: "b1", Age: 21, Email: "b1@x.com"},
		{Name: "b2", Age: 22, Email: "b2@x.com"},
		{Name: "b3", Age: 23, Email: "b3@x.com"},
	}
	if err := gdb.Create(batch); err != nil {
		t.Errorf("Create batch failed: %v", err)
		return
	}
	var n int64
	if err := gdb.Table("g1_user").Count(&n); err != nil {
		t.Errorf("Count after batch failed: %v", err)
		return
	}
	if n != 3 {
		t.Errorf("Count after batch = %d, want 3", n)
	}
	// 指针切片批量
	ptrBatch := []*G1User{
		{Name: "p1", Age: 24, Email: "p1@x.com"},
		{Name: "p2", Age: 25, Email: "p2@x.com"},
	}
	if err := gdb.Create(ptrBatch); err != nil {
		t.Errorf("Create ptr batch failed: %v", err)
		return
	}
	if err := gdb.Table("g1_user").Count(&n); err != nil {
		t.Errorf("Count after ptr batch failed: %v", err)
		return
	}
	if n != 5 {
		t.Errorf("Count after ptr batch = %d, want 5", n)
	}
	// 空批量报错
	if err := gdb.Create([]G1User{}); err == nil {
		t.Error("Create empty batch should fail")
	}
}

func Test_UpdateAndUpdates(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()
	seedUsers(t, gdb)

	// 单列更新
	if err := gdb.Table("g1_user").Where("name = ?", "alice").Update("email", "new@x.com"); err != nil {
		t.Errorf("Update failed: %v", err)
		return
	}
	var alice G1User
	if err := gdb.Table("g1_user").Where("name = ?", "alice").First(&alice); err != nil {
		t.Errorf("query alice failed: %v", err)
		return
	}
	if alice.Email != "new@x.com" {
		t.Errorf("alice email = %s, want new@x.com", alice.Email)
	}
	// map 多列更新
	if err := gdb.Table("g1_user").Where("name = ?", "bob").
		Updates(map[string]interface{}{"age": 30, "email": "bob30@x.com"}); err != nil {
		t.Errorf("Updates map failed: %v", err)
		return
	}
	var bob G1User
	if err := gdb.Table("g1_user").Where("name = ?", "bob").First(&bob); err != nil {
		t.Errorf("query bob failed: %v", err)
		return
	}
	if bob.Age != 30 || bob.Email != "bob30@x.com" {
		t.Errorf("bob = age %d email %s, want 30 bob30@x.com", bob.Age, bob.Email)
	}
	// struct 更新跳零值字段
	if err := gdb.Model(&G1User{}).Where("name = ?", "carol").
		Updates(G1User{Name: "carol2"}); err != nil {
		t.Errorf("Updates struct failed: %v", err)
		return
	}
	var carol G1User
	if err := gdb.Table("g1_user").Where("name = ?", "carol2").First(&carol); err != nil {
		t.Errorf("query carol2 failed: %v", err)
		return
	}
	if carol.Age != 32 || carol.Email != "carol@test.com" {
		t.Errorf("carol zero fields overwritten: age %d email %s", carol.Age, carol.Email)
	}
	// 零条件保护
	if err := gdb.Table("g1_user").Update("email", "x@x.com"); !errors.Is(err, gormish.ErrMissingWhere) {
		t.Errorf("Update without where err = %v, want ErrMissingWhere", err)
	}
	if err := gdb.Table("g1_user").Updates(map[string]interface{}{"age": 1}); !errors.Is(err, gormish.ErrMissingWhere) {
		t.Errorf("Updates without where err = %v, want ErrMissingWhere", err)
	}
}

func Test_Delete(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()
	seedUsers(t, gdb)

	// 条件删除
	if err := gdb.Table("g1_user").Where("age > ?", 40).Delete(); err != nil {
		t.Errorf("Delete failed: %v", err)
		return
	}
	var n int64
	if err := gdb.Table("g1_user").Count(&n); err != nil {
		t.Errorf("Count after delete failed: %v", err)
		return
	}
	if n != 4 {
		t.Errorf("Count after delete = %d, want 4", n)
	}
	// 模型主键条件删除
	victim := G1User{Id: 2}
	if err := gdb.Delete(&victim); err != nil {
		t.Errorf("Delete by model pk failed: %v", err)
		return
	}
	if err := gdb.Table("g1_user").Where("id = ?", 2).Count(&n); err != nil {
		t.Errorf("Count victim failed: %v", err)
		return
	}
	if n != 0 {
		t.Errorf("victim still exists, count = %d", n)
	}
	// 零条件保护
	if err := gdb.Table("g1_user").Delete(); !errors.Is(err, gormish.ErrMissingWhere) {
		t.Errorf("Delete without where err = %v, want ErrMissingWhere", err)
	}
}

// ---------- 事务 ----------

func Test_TransactionCommitRollback(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()

	// 提交
	err := gdb.Transaction(func(tx *gormish.DB) error {
		return tx.Create(&G1User{Name: "tx1", Age: 20, Email: "tx1@x.com"})
	})
	if err != nil {
		t.Errorf("Transaction commit failed: %v", err)
		return
	}
	var n int64
	if err := gdb.Table("g1_user").Count(&n); err != nil {
		t.Errorf("Count after commit failed: %v", err)
		return
	}
	if n != 1 {
		t.Errorf("Count after commit = %d, want 1", n)
	}
	// 回滚
	sentinel := errors.New("rollback-me")
	err = gdb.Transaction(func(tx *gormish.DB) error {
		if err := tx.Create(&G1User{Name: "tx2", Age: 21, Email: "tx2@x.com"}); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("Transaction rollback err = %v, want sentinel", err)
	}
	if err := gdb.Table("g1_user").Count(&n); err != nil {
		t.Errorf("Count after rollback failed: %v", err)
		return
	}
	if n != 1 {
		t.Errorf("Count after rollback = %d, want 1", n)
	}
}

func Test_TransactionNested(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()

	err := gdb.Transaction(func(tx *gormish.DB) error {
		if err := tx.Create(&G1User{Name: "outer", Age: 20, Email: "o@x.com"}); err != nil {
			return err
		}
		// 嵌套事务：复用外层事务（无 savepoint，G2），内层错误逐层透传
		err := tx.Transaction(func(tx2 *gormish.DB) error {
			if err := tx2.Create(&G1User{Name: "inner", Age: 21, Email: "i@x.com"}); err != nil {
				return err
			}
			return errors.New("inner rollback")
		})
		if err == nil {
			return errors.New("nested Transaction should propagate inner error")
		}
		return nil
	})
	if err != nil {
		t.Errorf("nested Transaction failed: %v", err)
		return
	}
	var n int64
	if err := gdb.Table("g1_user").Count(&n); err != nil {
		t.Errorf("Count failed: %v", err)
		return
	}
	if n != 2 {
		t.Errorf("rows after nested tx = %d, want 2 (inner shares outer tx, no savepoint until G2)", n)
	}
}

func Test_WithContextJoinsWithTx(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()

	// gormish.WithContext 参与外层 orm.WithTx 事务
	err := orm.WithTx(context.Background(), func(ctx context.Context) error {
		if err := gdb.WithContext(ctx).Create(&G1User{Name: "wtx", Age: 20, Email: "w@x.com"}); err != nil {
			return err
		}
		// 事务内可见
		var n int64
		if err := gdb.WithContext(ctx).Table("g1_user").Count(&n); err != nil {
			return err
		}
		if n != 1 {
			return fmt.Errorf("rows inside tx = %d, want 1", n)
		}
		return nil
	})
	if err != nil {
		t.Errorf("WithContext join WithTx failed: %v", err)
		return
	}
	var n int64
	if err := gdb.Table("g1_user").Count(&n); err != nil {
		t.Errorf("Count after WithTx failed: %v", err)
		return
	}
	if n != 1 {
		t.Errorf("Count after WithTx = %d, want 1", n)
	}
}

// ---------- Raw / Exec ----------

func Test_RawScanExec(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()
	seedUsers(t, gdb)

	// Raw + Scan 到标量
	var n int64
	if err := gdb.Raw("SELECT count(*) FROM g1_user WHERE age > ?", 20).Scan(&n); err != nil {
		t.Errorf("Raw Scan scalar failed: %v", err)
		return
	}
	if n != 4 {
		t.Errorf("Raw Scan scalar = %d, want 4", n)
	}
	// Raw + Scan 到 struct 切片
	var rows []G1User
	if err := gdb.Raw("SELECT name, age FROM g1_user ORDER BY age LIMIT ?", 2).Scan(&rows); err != nil {
		t.Errorf("Raw Scan rows failed: %v", err)
		return
	}
	if len(rows) != 2 || rows[0].Name != "alice" || rows[1].Name != "bob" {
		t.Errorf("Raw Scan rows = %+v", rows)
	}
	// Exec 带参数 + RowsAffected
	result, err := gdb.Exec("UPDATE g1_user SET email = ? WHERE age >= ?", "mature@x.com", 32)
	if err != nil {
		t.Errorf("Exec update failed: %v", err)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		t.Errorf("RowsAffected failed: %v", err)
		return
	}
	if affected != 3 {
		t.Errorf("affected rows = %d, want 3", affected)
	}
	// Scan 缺表报错
	if err := gdb.Scan(&rows); err == nil {
		t.Error("Scan without table/raw should fail")
	}
}

// ---------- 克隆隔离 ----------

func Test_CloneIsolation(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()
	seedUsers(t, gdb)

	base := gdb.Table("g1_user")
	qYoung := base.Where("age < ?", 25)
	qOld := base.Where("age > ?", 35)

	var young, old, all []G1User
	if err := qYoung.Find(&young); err != nil {
		t.Errorf("qYoung failed: %v", err)
		return
	}
	if err := qOld.Find(&old); err != nil {
		t.Errorf("qOld failed: %v", err)
		return
	}
	if err := base.Find(&all); err != nil {
		t.Errorf("base failed: %v", err)
		return
	}
	if len(young) != 1 {
		t.Errorf("qYoung rows = %d, want 1", len(young))
	}
	if len(old) != 2 {
		t.Errorf("qOld rows = %d, want 2", len(old))
	}
	if len(all) != 5 {
		t.Errorf("base rows = %d, want 5 (clones must not pollute base)", len(all))
	}
}

// ---------- 无 tag 模型回退 ----------

func Test_FallbackModelNoTag(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()

	// TableName 方法 + ID 主键约定
	row := G1PlainRow{Name: "plain"}
	if err := gdb.Create(&row); err != nil {
		t.Errorf("Create fallback model failed: %v", err)
		return
	}
	if row.Id != 1 {
		t.Errorf("fallback pk backfill = %d, want 1", row.Id)
	}
	var rows []G1PlainRow
	if err := gdb.Find(&rows); err != nil {
		t.Errorf("Find fallback model failed: %v", err)
		return
	}
	if len(rows) != 1 || rows[0].Name != "plain" {
		t.Errorf("fallback rows = %+v", rows)
	}
	// First 默认主键排序
	var first G1PlainRow
	if err := gdb.First(&first); err != nil {
		t.Errorf("First fallback failed: %v", err)
		return
	}
	if first.Id != 1 {
		t.Errorf("fallback First id = %d, want 1", first.Id)
	}
	// 短名去 Model 后缀推导表名：G1CatModel -> g1_cat
	cat := G1CatModel{Name: "mimi"}
	if err := gdb.Create(&cat); err != nil {
		t.Errorf("Create G1CatModel failed: %v", err)
		return
	}
	var cats []G1CatModel
	if err := gdb.Find(&cats); err != nil {
		t.Errorf("Find G1CatModel failed: %v", err)
		return
	}
	if len(cats) != 1 || cats[0].Name != "mimi" {
		t.Errorf("cats = %+v", cats)
	}
}

// ---------- Model 绑定 ----------

func Test_ModelBinding(t *testing.T) {
	gdb := initG1Test(t)
	if gdb == nil {
		return
	}
	defer orm.Close()
	seedUsers(t, gdb)

	var users []G1User
	if err := gdb.Model(&G1User{}).Where("age < ?", 25).Find(&users); err != nil {
		t.Errorf("Model Find failed: %v", err)
		return
	}
	if len(users) != 1 {
		t.Errorf("Model Find rows = %d, want 1", len(users))
	}
	// Model 绑定错误类型聚合报错
	if err := gdb.Model(123).Find(&users); err == nil {
		t.Error("Model with non-struct should fail")
	}
	// Table 覆盖 Model 表名
	var byTable []G1PlainRow
	if err := gdb.Model(&G1User{}).Table("g1_plain").Find(&byTable); err != nil {
		t.Errorf("Table override failed: %v", err)
	}
	if len(byTable) != 0 {
		t.Errorf("Table override rows = %d, want 0", len(byTable))
	}
}
