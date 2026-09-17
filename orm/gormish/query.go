package gormish

import (
	"database/sql"
	"fmt"
	"reflect"
)

// ---------- 链式子句 ----------

// Select 指定查询列（支持 Select("a", "b") 与 Select("a, b") 两种写法），返回克隆。
func (d *DB) Select(cols ...string) *DB {
	nd := d.clone()
	nd.stmt.selectCols = splitCols(cols)
	return nd
}

// Distinct 指定 DISTINCT 查询列（同时作为查询列），返回克隆。
func (d *DB) Distinct(cols ...string) *DB {
	nd := d.clone()
	nd.stmt.selectCols = splitCols(cols)
	nd.stmt.distinct = true
	return nd
}

// Where 追加 AND 条件：query 支持 string（? 占位）与 map[string]interface{}（等值），
// slice 参数配合 "IN ?" 自动展开为 (?, ?, ...)，返回克隆。
func (d *DB) Where(query interface{}, args ...interface{}) *DB {
	nd := d.clone()
	nd.stmt.addCondition(query, args, false)
	return nd
}

// Or 追加 OR 条件（参数约定同 Where），返回克隆。
func (d *DB) Or(query interface{}, args ...interface{}) *DB {
	nd := d.clone()
	nd.stmt.addCondition(query, args, true)
	return nd
}

// Order 追加排序列（多次调用以逗号连接），返回克隆。
func (d *DB) Order(cols ...string) *DB {
	nd := d.clone()
	nd.stmt.order = append(nd.stmt.order, splitCols(cols)...)
	return nd
}

// Limit 限制返回行数（<=0 视为未设置），返回克隆。
func (d *DB) Limit(n int) *DB {
	nd := d.clone()
	nd.stmt.limit = n
	return nd
}

// Offset 跳过行数（需配合 Limit 使用），返回克隆。
func (d *DB) Offset(n int) *DB {
	nd := d.clone()
	nd.stmt.offset = n
	return nd
}

// Group 指定 GROUP BY 列，返回克隆。
func (d *DB) Group(cols ...string) *DB {
	nd := d.clone()
	nd.stmt.group = splitCols(cols)
	return nd
}

// Having 指定 HAVING 条件（? 占位 + 参数绑定），返回克隆。
func (d *DB) Having(query string, args ...interface{}) *DB {
	nd := d.clone()
	nd.stmt.having = query
	nd.stmt.havingArgs = append([]interface{}(nil), args...)
	return nd
}

// Raw 指定原生 SQL（? 占位 + 参数绑定），后续 Scan/Find 直接执行该 SQL；
// Raw 不可再叠加链式条件，返回克隆。
func (d *DB) Raw(sqlStr string, args ...interface{}) *DB {
	nd := d.clone()
	nd.stmt.rawSQL = sqlStr
	nd.stmt.rawArgs = append([]interface{}(nil), args...)
	return nd
}

// ---------- finisher ----------

// Find 执行查询并把全部行扫描到 dst（*[]T / *[]*T，T 为 struct，遵循 DefaultRowLimit）。
// 链上未指定 Table/Model 时按 dst 元素类型推导表名；Raw 优先直接执行。
func (d *DB) Find(dst interface{}) error {
	return d.queryRows(dst, true)
}

// Scan 执行查询并扫描到 dst；与 Find 的区别是不按 dst 推导表名
// （需显式 Table/Model 或 Raw），适合列裁剪扫描到精简 struct。
func (d *DB) Scan(dst interface{}) error {
	return d.queryRows(dst, false)
}

// queryRows 查询执行核心（G1）：Raw 优先；否则推导表名 → 组装 SELECT → 分页 → QueryToContext。
func (d *DB) queryRows(dst interface{}, deriveTable bool) error {
	if err := d.checkFinish(); err != nil {
		return err
	}
	nd := d.clone()
	if nd.stmt.rawSQL != "" {
		nd.logSQL(nd.stmt.rawSQL, nd.stmt.rawArgs)
		return d.db.QueryToContext(nd.context(), dst, nd.stmt.rawSQL, nd.stmt.rawArgs...)
	}
	if nd.stmt.table == "" && deriveTable {
		table, err := deriveTableFromDst(dst)
		if err != nil {
			return err
		}
		nd.stmt.table = table
	}
	if nd.stmt.table == "" {
		return fmt.Errorf("gormish: table name required, use Table/Model or Raw")
	}
	sqlStr, args := nd.stmt.buildSelect()
	sqlStr = nd.applyPagination(sqlStr)
	nd.logSQL(sqlStr, args)
	return d.db.QueryToContext(nd.context(), dst, sqlStr, args...)
}

// First 查询首行（主键升序 LIMIT 1）扫描到 dst（*struct）；无行返回 sql.ErrNoRows。
func (d *DB) First(dst interface{}) error {
	return d.firstOrLast(dst, false)
}

// Last 查询末行（主键降序 LIMIT 1）扫描到 dst（*struct）；无行返回 sql.ErrNoRows。
func (d *DB) Last(dst interface{}) error {
	return d.firstOrLast(dst, true)
}

// firstOrLast First/Last 实现：未显式指定排序时按主键排序，LIMIT 1 后扫描首行。
func (d *DB) firstOrLast(dst interface{}, desc bool) error {
	if err := d.checkFinish(); err != nil {
		return err
	}
	nd := d.clone()
	if nd.stmt.rawSQL != "" {
		return fmt.Errorf("gormish: First/Last does not support Raw, use Scan")
	}
	// Table() 会清空 model：First/Last 需按 dst 补推模型元数据，以获得主键列做默认排序
	if nd.stmt.model == nil {
		if info := resolveModelInfo(dst); info != nil {
			nd.stmt.model = info
			if nd.stmt.table == "" {
				nd.stmt.table = info.TableName
			}
		}
	}
	if nd.stmt.table == "" {
		return fmt.Errorf("gormish: table name required, use Table/Model")
	}
	if len(nd.stmt.order) == 0 {
		if pk := nd.stmt.primaryColumn(); pk != "" {
			nd.stmt.order = []string{pk + orderDir(desc)}
		}
	}
	nd.stmt.limit = 1
	sqlStr, args := nd.stmt.buildSelect()
	sqlStr = nd.applyPagination(sqlStr)
	nd.logSQL(sqlStr, args)
	return d.db.QueryToContext(nd.context(), dst, sqlStr, args...)
}

func orderDir(desc bool) string {
	if desc {
		return " DESC"
	}
	return " ASC"
}

// Count 统计行数（应用 WHERE 条件；DISTINCT 单列时统计去重数）。
func (d *DB) Count(count *int64) error {
	if err := d.checkFinish(); err != nil {
		return err
	}
	if count == nil {
		return fmt.Errorf("gormish: Count dst must be *int64")
	}
	nd := d.clone()
	if nd.stmt.rawSQL != "" {
		return fmt.Errorf("gormish: Count does not support Raw")
	}
	if nd.stmt.table == "" {
		return fmt.Errorf("gormish: table name required, use Table/Model")
	}
	sqlStr, args := nd.stmt.buildCount()
	nd.logSQL(sqlStr, args)
	return d.db.QueryToContext(nd.context(), count, sqlStr, args...)
}

// applyPagination 经方言 ApplyPagination 追加 LIMIT/OFFSET 子句（G1）。
func (d *DB) applyPagination(sqlStr string) string {
	if d.stmt.limit <= 0 || d.db == nil || d.db.Dialector == nil {
		return sqlStr
	}
	return d.db.Dialector.ApplyPagination(sqlStr, d.stmt.limit, d.stmt.offset)
}

// deriveTableFromDst 按 dst 类型推导表名（Find 无 Table/Model 时的回退，G1）。
func deriveTableFromDst(dst interface{}) (string, error) {
	if dst == nil {
		return "", fmt.Errorf("gormish: dst is nil")
	}
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return "", fmt.Errorf("gormish: dst must be a non-nil pointer, got %T", dst)
	}
	elem := rv.Elem()
	typ := elem.Type()
	switch typ.Kind() {
	case reflect.Struct:
		info := resolveModelInfo(dst)
		if info == nil {
			return "", fmt.Errorf("gormish: cannot derive table from %T", dst)
		}
		return info.TableName, nil
	case reflect.Slice:
		modelType := typ.Elem()
		if modelType.Kind() == reflect.Ptr {
			modelType = modelType.Elem()
		}
		if modelType.Kind() != reflect.Struct {
			return "", fmt.Errorf("gormish: slice element must be struct or *struct, got %v", typ)
		}
		info := resolveModelInfo(reflect.New(modelType).Interface())
		if info == nil {
			return "", fmt.Errorf("gormish: cannot derive table from %v", typ)
		}
		return info.TableName, nil
	default:
		return "", fmt.Errorf("gormish: dst must point to struct or struct slice, got %T", dst)
	}
}

// ErrNoRows 与 sql.ErrNoRows 一致（First/Last 无行时返回）。
var ErrNoRows = sql.ErrNoRows
