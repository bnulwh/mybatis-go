package gormish

import (
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/bnulwh/mybatis-go/orm"
)

// ErrMissingWhere 全表 Update/Delete 保护：无 WHERE 条件时拒绝执行（GORM 语义对齐）。
var ErrMissingWhere = fmt.Errorf("gormish: WHERE conditions required for update/delete (global operation blocked)")

// Create 插入记录（G1）：
//   - value 支持 struct 指针（单条）与 struct 切片（批量，不回填主键）；
//   - 零值主键视为自增列跳过；单条插入回填自增主键（LastInsertId，
//     PG 族等 NeedsReturning 方言经 RETURNING 取回）；
//   - 逻辑删除列与 db:"-" 字段不参与插入（与 MP 内置 CRUD 生成一致）；
//   - fill 自动填充 / hooks 归 G2。
func (d *DB) Create(value interface{}) error {
	if err := d.checkFinish(); err != nil {
		return err
	}
	if value == nil {
		return fmt.Errorf("gormish: Create value is nil")
	}
	rv := reflect.Indirect(reflect.ValueOf(value))
	switch rv.Kind() {
	case reflect.Struct:
		return d.createOne(value, rv)
	case reflect.Slice:
		return d.createBatch(rv)
	default:
		return fmt.Errorf("gormish: Create value must be struct or struct slice, got %T", value)
	}
}

// createOne 单条插入 + 自增主键回填（G1）。
func (d *DB) createOne(value interface{}, rv reflect.Value) error {
	nd := d.deriveModel(value)
	if err := nd.stmt.err(); err != nil {
		return err
	}
	cols, args, pkField := nd.stmt.insertColumns(rv)
	if len(cols) == 0 {
		return fmt.Errorf("gormish: no insertable columns for %v", rv.Type())
	}
	sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		nd.stmt.table, strings.Join(cols, ", "), placeholders(len(args)))
	// 自增主键回填：NeedsReturning 方言经 RETURNING 取回，其余走 LastInsertId
	if pkField != nil && nd.db.Dialector != nil && nd.db.Dialector.NeedsReturning() {
		sqlStr += " RETURNING " + pkField.Column
		nd.logSQL(sqlStr, args)
		row := nd.db.QueryRowContext(nd.context(), sqlStr, args...)
		return row.Scan(rv.FieldByName(pkField.Name).Addr().Interface())
	}
	nd.logSQL(sqlStr, args)
	result, err := nd.db.ExecContext(nd.context(), sqlStr, args...)
	if err != nil {
		return err
	}
	if pkField == nil {
		return nil
	}
	return backfillPK(rv, pkField, result)
}

// createBatch 批量插入（多行 VALUES，不回填主键，G1）。
func (d *DB) createBatch(rv reflect.Value) error {
	n := rv.Len()
	if n == 0 {
		return fmt.Errorf("gormish: Create batch is empty")
	}
	elemType := rv.Type().Elem()
	isPtr := elemType.Kind() == reflect.Ptr
	modelType := elemType
	if isPtr {
		modelType = elemType.Elem()
	}
	if modelType.Kind() != reflect.Struct {
		return fmt.Errorf("gormish: Create slice element must be struct or *struct, got %v", rv.Type())
	}
	first := rv.Index(0)
	if isPtr {
		if first.IsNil() {
			return fmt.Errorf("gormish: Create batch item 0 is nil")
		}
		first = first.Elem()
	}
	nd := d.deriveModel(reflect.New(modelType).Interface())
	if err := nd.stmt.err(); err != nil {
		return err
	}
	cols, _, _ := nd.stmt.insertColumns(first)
	if len(cols) == 0 {
		return fmt.Errorf("gormish: no insertable columns for %v", modelType)
	}
	rows := make([]string, 0, n)
	var args []interface{}
	for i := 0; i < n; i++ {
		item := rv.Index(i)
		if isPtr {
			if item.IsNil() {
				return fmt.Errorf("gormish: Create batch item %d is nil", i)
			}
			item = item.Elem()
		}
		itemArgs := nd.stmt.insertColumnsFor(item, cols)
		if itemArgs == nil {
			return fmt.Errorf("gormish: Create batch item %d column mismatch (batch rows must share zero/non-zero pk)", i)
		}
		rows = append(rows, "("+placeholders(len(itemArgs))+")")
		args = append(args, itemArgs...)
	}
	sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		nd.stmt.table, strings.Join(cols, ", "), strings.Join(rows, ", "))
	nd.logSQL(sqlStr, args)
	_, err := nd.db.ExecContext(nd.context(), sqlStr, args...)
	return err
}

// deriveModel 补齐链上模型：无 Table/Model 时按 value 推导（克隆，不改原句柄）。
func (d *DB) deriveModel(value interface{}) *DB {
	nd := d.clone()
	if nd.stmt.table == "" {
		info := resolveModelInfo(value)
		if info == nil {
			nd.stmt.addErr(fmt.Errorf("cannot derive model from %T", value))
			return nd
		}
		nd.stmt.model = info
		nd.stmt.table = info.TableName
	}
	return nd
}

// backfillPK 经 LastInsertId 回填自增主键（驱动不支持时忽略）。
func backfillPK(rv reflect.Value, pk *orm.FieldInfo, result sql.Result) error {
	field := rv.FieldByName(pk.Name)
	if !field.CanSet() || !isIntegerKind(field.Kind()) {
		return nil
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil
	}
	field.SetInt(id)
	return nil
}

func isIntegerKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	}
	return false
}

// Update 单列更新：UPDATE table SET column = ? WHERE ...（无 WHERE 条件报错）。
func (d *DB) Update(column string, value interface{}) error {
	if err := d.checkFinish(); err != nil {
		return err
	}
	nd := d.clone()
	if nd.stmt.table == "" {
		return fmt.Errorf("gormish: table name required, use Table/Model")
	}
	if column == "" {
		return fmt.Errorf("gormish: Update column is empty")
	}
	if !nd.stmt.hasCondition() {
		return ErrMissingWhere
	}
	sqlStr := "UPDATE " + nd.stmt.table + " SET " + column + " = ?"
	whereSQL, args := nd.stmt.buildWhere()
	sqlStr += whereSQL
	args = append([]interface{}{value}, args...)
	nd.logSQL(sqlStr, args)
	_, err := nd.db.ExecContext(nd.context(), sqlStr, args...)
	return err
}

// Updates 多列更新（G1）：
//   - map[string]interface{}：全量键值对（按列名排序，确定性生成）；
//   - struct：跳过零值字段（GORM 语义），主键 / 逻辑删除 / 乐观锁列不参与 SET；
//   - 无 WHERE 条件报错（全表更新保护）。
func (d *DB) Updates(values interface{}) error {
	if err := d.checkFinish(); err != nil {
		return err
	}
	nd := d.deriveModel(values)
	if err := nd.stmt.err(); err != nil {
		return err
	}
	if nd.stmt.table == "" {
		return fmt.Errorf("gormish: table name required, use Table/Model")
	}
	if nd.stmt.model == nil {
		if info := resolveModelInfo(values); info != nil {
			nd.stmt.model = info
		}
	}
	if !nd.stmt.hasCondition() {
		return ErrMissingWhere
	}
	cols, args := nd.updateColumns(values)
	if len(cols) == 0 {
		return fmt.Errorf("gormish: Updates: no updatable columns")
	}
	sets := make([]string, 0, len(cols))
	for _, c := range cols {
		sets = append(sets, c+" = ?")
	}
	sqlStr := "UPDATE " + nd.stmt.table + " SET " + strings.Join(sets, ", ")
	whereSQL, whereArgs := nd.stmt.buildWhere()
	sqlStr += whereSQL
	args = append(args, whereArgs...)
	nd.logSQL(sqlStr, args)
	_, err := nd.db.ExecContext(nd.context(), sqlStr, args...)
	return err
}

// updateColumns 提取 SET 列与参数（G1）。
func (nd *DB) updateColumns(values interface{}) ([]string, []interface{}) {
	switch v := values.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		args := make([]interface{}, 0, len(keys))
		for _, k := range keys {
			args = append(args, v[k])
		}
		return keys, args
	default:
		rv := reflect.Indirect(reflect.ValueOf(values))
		if rv.Kind() != reflect.Struct || nd.stmt.model == nil {
			return nil, nil
		}
		var cols []string
		var args []interface{}
		for _, f := range nd.stmt.model.Fields {
			if f.Primary || f.Logic || f.Version {
				continue
			}
			fv := rv.FieldByName(f.Name)
			if !fv.IsValid() || fv.IsZero() {
				continue
			}
			cols = append(cols, f.Column)
			args = append(args, fv.Interface())
		}
		return cols, args
	}
}

// Delete 删除记录（物理删除，G1；软删除自动改写归 G2）：
//   - 可选传入 struct 指针：其非零主键自动转为 WHERE 主键条件；
//   - 链上无任何条件时报错（全表删除保护）。
func (d *DB) Delete(value ...interface{}) error {
	if err := d.checkFinish(); err != nil {
		return err
	}
	nd := d.clone()
	if len(value) > 0 && value[0] != nil {
		rv := reflect.Indirect(reflect.ValueOf(value[0]))
		if rv.Kind() != reflect.Struct {
			return fmt.Errorf("gormish: Delete value must be struct ptr, got %T", value[0])
		}
		nd = nd.deriveModel(value[0])
		if err := nd.stmt.err(); err != nil {
			return err
		}
		if pk := nd.stmt.primaryColumn(); pk != "" && nd.stmt.model != nil {
			for _, f := range nd.stmt.model.Fields {
				if f.Column != pk {
					continue
				}
				if fv := rv.FieldByName(f.Name); fv.IsValid() && !fv.IsZero() {
					nd.stmt.addCondition(pk+" = ?", []interface{}{fv.Interface()}, false)
				}
				break
			}
		}
	}
	if nd.stmt.table == "" {
		return fmt.Errorf("gormish: table name required, use Table/Model")
	}
	if !nd.stmt.hasCondition() {
		return ErrMissingWhere
	}
	sqlStr := "DELETE FROM " + nd.stmt.table
	whereSQL, args := nd.stmt.buildWhere()
	sqlStr += whereSQL
	nd.logSQL(sqlStr, args)
	_, err := nd.db.ExecContext(nd.context(), sqlStr, args...)
	return err
}

// Exec 直接执行原生 SQL（? 占位 + 参数绑定；DDL 自动失效表结构缓存）。
func (d *DB) Exec(sqlStr string, args ...interface{}) (sql.Result, error) {
	if err := d.checkFinish(); err != nil {
		return nil, err
	}
	d.logSQL(sqlStr, args)
	return d.db.ExecContext(d.context(), sqlStr, args...)
}

// placeholders 生成 n 个 ? 占位符。
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}
