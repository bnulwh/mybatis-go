// Package gormish 提供 GORM 风格的链式查询与 CRUD API（G1，可行性报告 §2.4/§4.3）。
//
// 设计要点：
//   - 单向依赖 orm 包：SQL 执行走 orm.DB 的 ExecContext/QueryContext（自动表名前缀
//     改写 + 方言占位符转换），结果扫描走 G0 强类型扫描直通道 QueryToContext；
//   - 链式方法克隆 statement（写时复制切片），多克隆互不污染；
//   - 严格参数绑定：条件一律输出 `?` 占位 + 参数，与 QueryWrapper 的值内联模式分界；
//   - finisher 方法返回 (value, error) 双返回值，不采用 GORM 的 db.Error 链式错误；
//   - 事务基于 orm.WithTx（G0）：ctx 携带、不占 TCC 全局槽、嵌套自动复用外层事务；
//   - G1 不做（归 G2）：软删除自动改写、生命周期 hooks、AutoMigrate、savepoint、Save。
//
// 用法示例：
//
//	gdb, err := gormish.Open()
//	users := []User{}
//	err = gdb.Table("t_user").Where("age > ?", 18).Order("id DESC").Limit(10).Find(&users)
//	err = gdb.Transaction(func(tx *gormish.DB) error {
//	    return tx.Create(&User{Name: "tom"})
//	})
package gormish

import (
	"context"
	"fmt"

	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/orm"
)

// DB gormish 链式句柄：绑定目标数据源（*orm.DB）、context 与链式语句状态。
// 零值可用性由构造函数保证；链式方法返回克隆，原句柄状态不变。
type DB struct {
	db   *orm.DB
	ctx  context.Context
	stmt *statement
}

// New 包装指定 orm.DB 实例创建 gormish 句柄。
func New(ormDB *orm.DB) *DB {
	return &DB{db: ormDB, ctx: context.Background(), stmt: newStatement()}
}

// Open 基于当前活跃数据源（orm.GetActiveDataSource）创建句柄；未初始化返回错误。
func Open() (*DB, error) {
	ormDB := orm.GetActiveDataSource()
	if ormDB == nil {
		return nil, fmt.Errorf("gormish: connection not init, call orm.Initialize* first")
	}
	return New(ormDB), nil
}

// Use 基于命名数据源（orm.GetDataSource）创建句柄。
func Use(name string) (*DB, error) {
	ormDB, err := orm.GetDataSource(name)
	if err != nil {
		return nil, fmt.Errorf("gormish: %v", err)
	}
	return New(ormDB), nil
}

// clone 克隆句柄：statement 深拷贝（切片写时复制），ctx 与数据源共享。
func (d *DB) clone() *DB {
	return &DB{db: d.db, ctx: d.ctx, stmt: d.stmt.clone()}
}

// WithContext 绑定 context（超时控制 / WithTx 事务参与），返回克隆。
func (d *DB) WithContext(ctx context.Context) *DB {
	if ctx == nil {
		return d.clone()
	}
	nd := d.clone()
	nd.ctx = ctx
	return nd
}

// context 返回绑定的 context（nil 兜底 Background）。
func (d *DB) context() context.Context {
	if d.ctx == nil {
		return context.Background()
	}
	return d.ctx
}

// Transaction 回调式事务（G1）：基于 orm.WithTx（G0）实现。
//   - fn 返回 nil → Commit；返回 error → Rollback 并原样返回；
//   - fn panic → Rollback 后重新抛出；
//   - ctx 已携带事务（嵌套 Transaction / 外层 WithTx）→ 直接复用，内层不独立提交；
//   - 事务不占 TCC 全局事务槽，多 goroutine 可各自开事务互不冲突。
func (d *DB) Transaction(fn func(tx *DB) error) error {
	if d.db == nil {
		return fmt.Errorf("gormish: target orm.DB is nil")
	}
	if fn == nil {
		return fmt.Errorf("gormish: Transaction fn cannot be nil")
	}
	return d.db.WithTx(d.context(), func(ctx context.Context) error {
		tx := d.clone()
		tx.ctx = ctx
		return fn(tx)
	})
}

// Table 指定表名（清空链上 Model 元数据），返回克隆。
func (d *DB) Table(name string) *DB {
	nd := d.clone()
	nd.stmt.table = name
	nd.stmt.model = nil
	return nd
}

// Model 绑定模型（struct 指针或值）：推导表名与字段元数据（db tag 优先，无 tag
// 结构体回退本地约定：snake_case 列名 + ID 主键），返回克隆。
func (d *DB) Model(value interface{}) *DB {
	nd := d.clone()
	info := resolveModelInfo(value)
	if info == nil {
		nd.stmt.addErr(fmt.Errorf("Model: expect struct or struct ptr, got %T", value))
		return nd
	}
	nd.stmt.model = info
	nd.stmt.table = info.TableName
	return nd
}

// checkFinish finisher 方法入口校验：链上错误聚合 + 数据源有效性。
func (d *DB) checkFinish() error {
	if err := d.stmt.err(); err != nil {
		return err
	}
	if d.db == nil {
		return fmt.Errorf("gormish: target orm.DB is nil")
	}
	return nil
}

// logSQL 输出调试日志（与 orm 包日志风格一致）。
func (d *DB) logSQL(sqlStr string, args []interface{}) {
	log.Debugf("gormish sql: %v, args: %v", sqlStr, args)
}
