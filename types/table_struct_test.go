package types

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// newMPTestTableStructure 构造一张含主键、普通列与逻辑删除列的表结构。
func newMPTestTableStructure() *TableStructure {
	cols := []*ColumnStructure{
		{Name: "id", Type: reflect.TypeOf(int64(0)), DbType: "bigint", Primary: true},
		{Name: "user_name", Type: reflect.TypeOf(""), DbType: "varchar(64)"},
		{Name: "deleted", Type: reflect.TypeOf(true), DbType: "boolean"},
		{Name: "delete_time", Type: reflect.TypeOf(time.Now()), DbType: "timestamp"},
	}
	cm := map[string]*ColumnStructure{}
	for _, c := range cols {
		cm[c.Name] = c
	}
	return &TableStructure{
		Columns:       cols,
		ColumnMap:     cm,
		Table:         "sys_user",
		PrimaryColumn: cols[0],
	}
}

// Test_TableStruct_SaveMPToFile 生成的 MyBatis-Plus 风格 XML 可被 mybatis-go 正常加载，
// 且 10 个 BaseMapper 内置方法齐全（M 系列 / P16：无需手写 GoExtraMapper）。
func Test_TableStruct_SaveMPToFile(t *testing.T) {
	ts := newMPTestTableStructure()
	dir := t.TempDir()
	filename := filepath.Join(dir, "SysUserMapper.xml")
	if err := ts.SaveMPToFile(filename, ""); err != nil {
		t.Error("SaveMPToFile failed:", err)
		return
	}
	if _, err := os.Stat(filename); err != nil {
		t.Error("mapper file not written:", err)
		return
	}
	mps := NewSqlMappers(dir)
	if mps == nil || len(mps.Mappers) == 0 {
		t.Error("load generated MP mapper failed")
		return
	}
	m := mps.Mappers[0]
	if m.Namespace != "SysUserMapper" {
		t.Error("namespace =", m.Namespace, "want SysUserMapper")
	}
	ids := []string{MPInsertID, MPDeleteByIDID, MPUpdateByIDID, MPSelectByIDID,
		MPSelectOneID, MPSelectListID, MPSelectPageID, MPSelectCountID,
		MPSelectBatchIDsID, MPDeleteBatchIDsID}
	for _, id := range ids {
		if m.NamedFunctions[id] == nil {
			t.Errorf("MP built-in function %q missing", id)
		}
	}
}

// Test_MPGeneratedSQL 逐个验证内置方法生成的 SQL（含逻辑删除语义）。
func Test_MPGeneratedSQL(t *testing.T) {
	ts := newMPTestTableStructure()
	dir := t.TempDir()
	filename := filepath.Join(dir, "SysUserMapper.xml")
	if err := ts.SaveMPToFile(filename, ""); err != nil {
		t.Error("SaveMPToFile failed:", err)
		return
	}
	mps := NewSqlMappers(dir)
	if mps == nil || len(mps.Mappers) == 0 {
		t.Error("load generated MP mapper failed")
		return
	}
	m := mps.Mappers[0]
	getSQL := func(t *testing.T, id string, args ...interface{}) string {
		fn := m.NamedFunctions[id]
		if fn == nil {
			t.Errorf("function %q not found", id)
			return ""
		}
		sql, _, err := fn.GenerateSQL(args...)
		if err != nil {
			t.Errorf("%s GenerateSQL(%v) failed: %v", id, args, err)
			return ""
		}
		return collapseSpace(sql)
	}
	// insert：包含全部业务列
	if sql := getSQL(t, MPInsertID, map[string]interface{}{"id": int64(1), "userName": "admin"}); sql != "" {
		if !strings.Contains(sql, "insert into sys_user") || !strings.Contains(sql, "user_name") || !strings.Contains(sql, "admin") {
			t.Error("insert sql unexpected:", sql)
		}
	}
	// deleteById：有 deleted 列 → 逻辑删除
	if sql := getSQL(t, MPDeleteByIDID, int64(1)); sql != "" {
		if !strings.Contains(sql, "update sys_user") || !strings.Contains(sql, "deleted=true") || !strings.Contains(sql, "id=1") {
			t.Error("deleteById sql unexpected:", sql)
		}
	}
	// updateById
	if sql := getSQL(t, MPUpdateByIDID, map[string]interface{}{"id": int64(1), "userName": "x"}); sql != "" {
		if !strings.Contains(sql, "update sys_user") || !strings.Contains(sql, "user_name=") || !strings.Contains(sql, "id=1") {
			t.Error("updateById sql unexpected:", sql)
		}
	}
	// selectById：逻辑删除过滤
	if sql := getSQL(t, MPSelectByIDID, int64(1)); sql != "" {
		if !strings.Contains(sql, "select id, user_name from sys_user where id=1") || !strings.Contains(sql, "deleted = false") {
			t.Error("selectById sql unexpected:", sql)
		}
	}
	// selectList / selectOne / selectPage：全表 + 逻辑删除过滤
	for _, id := range []string{MPSelectListID, MPSelectOneID, MPSelectPageID} {
		if sql := getSQL(t, id); sql != "" {
			if !strings.Contains(sql, "select id, user_name from sys_user where deleted = false") {
				t.Errorf("%s sql unexpected: %s", id, sql)
			}
		}
	}
	// selectCount
	if sql := getSQL(t, MPSelectCountID); sql != "" {
		if !strings.Contains(sql, "select count(*) from sys_user where deleted = false") {
			t.Error("selectCount sql unexpected:", sql)
		}
	}
	// selectBatchIds：in 子句 + 逻辑删除过滤
	if sql := getSQL(t, MPSelectBatchIDsID, []int64{1, 2}); sql != "" {
		if !strings.Contains(sql, "select id, user_name from sys_user where id in (") || !strings.Contains(sql, "1, 2") || !strings.Contains(sql, "deleted = false") {
			t.Error("selectBatchIds sql unexpected:", sql)
		}
	}
	// deleteBatchIds：逻辑删除
	if sql := getSQL(t, MPDeleteBatchIDsID, []int64{1, 2}); sql != "" {
		if !strings.Contains(sql, "update sys_user set deleted=true,delete_time=now() where id in (") || !strings.Contains(sql, "1, 2") {
			t.Error("deleteBatchIds sql unexpected:", sql)
		}
	}
	// PrepareSQL 冒烟：selectById 参数化
	fn := m.NamedFunctions[MPSelectByIDID]
	if fn != nil {
		if psql, pargs, err := fn.PrepareSQL(int64(1)); err != nil {
			t.Error("selectById PrepareSQL failed:", err)
		} else if !strings.Contains(psql, "id=?") || len(pargs) != 1 {
			t.Error("selectById PrepareSQL unexpected:", psql, pargs)
		}
	}
}

// Test_TableStruct_SaveMPToFile_NoDeleted 无逻辑删除列时：deleteById/deleteBatchIds 走物理删除，
// select 不带 deleted 过滤。
func Test_TableStruct_SaveMPToFile_NoDeleted(t *testing.T) {
	ts := newMPTestTableStructure()
	// 移除逻辑删除列
	var cols []*ColumnStructure
	for _, c := range ts.Columns {
		if strings.EqualFold(c.Name, "deleted") || strings.EqualFold(c.Name, "delete_time") {
			continue
		}
		cols = append(cols, c)
	}
	ts.Columns = cols
	ts.ColumnMap = map[string]*ColumnStructure{}
	for _, c := range cols {
		ts.ColumnMap[c.Name] = c
	}
	dir := t.TempDir()
	filename := filepath.Join(dir, "SysUserMapper.xml")
	if err := ts.SaveMPToFile(filename, ""); err != nil {
		t.Error("SaveMPToFile failed:", err)
		return
	}
	mps := NewSqlMappers(dir)
	if mps == nil || len(mps.Mappers) == 0 {
		t.Error("load generated MP mapper failed")
		return
	}
	m := mps.Mappers[0]
	if sql, _, err := m.NamedFunctions[MPDeleteByIDID].GenerateSQL(int64(1)); err != nil {
		t.Error("deleteById GenerateSQL failed:", err)
	} else if sql = collapseSpace(sql); !strings.Contains(sql, "delete from sys_user where id=1") {
		t.Error("deleteById without deleted column should hard delete, sql:", sql)
	}
	if sql, _, err := m.NamedFunctions[MPDeleteBatchIDsID].GenerateSQL([]int64{1, 2}); err != nil {
		t.Error("deleteBatchIds GenerateSQL failed:", err)
	} else if sql = collapseSpace(sql); !strings.Contains(sql, "delete from sys_user where id in (") || !strings.Contains(sql, "1, 2") {
		t.Error("deleteBatchIds without deleted column should hard delete, sql:", sql)
	}
	if sql, _, err := m.NamedFunctions[MPSelectListID].GenerateSQL(); err != nil {
		t.Error("selectList GenerateSQL failed:", err)
	} else if sql = collapseSpace(sql); !strings.Contains(sql, "select id, user_name from sys_user") || strings.Contains(sql, "deleted") {
		t.Error("selectList without deleted column should have no filter, sql:", sql)
	}
}

// Test_TableStruct_MPCodegen 生成的 MP mapper 能产出 Go 代码（generateDefine 方法名）。
func Test_TableStruct_MPCodegen(t *testing.T) {
	ts := newMPTestTableStructure()
	dir := t.TempDir()
	filename := filepath.Join(dir, "SysUserMapper.xml")
	if err := ts.SaveMPToFile(filename, ""); err != nil {
		t.Error("SaveMPToFile failed:", err)
		return
	}
	mps := NewSqlMappers(dir)
	if mps == nil || len(mps.Mappers) == 0 {
		t.Error("load generated MP mapper failed")
		return
	}
	content := string(mps.Mappers[0].generateContent("src"))
	for _, name := range []string{"DeleteById", "UpdateById", "SelectById", "SelectList",
		"SelectOne", "SelectPage", "SelectCount", "SelectBatchIds", "DeleteBatchIds"} {
		if !strings.Contains(content, name) {
			t.Errorf("generated mapper missing method %s", name)
		}
	}
}
