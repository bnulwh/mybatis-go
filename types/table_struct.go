package types

import (
	"fmt"
	"github.com/beevik/etree"
	"github.com/bnulwh/mybatis-go/log"
	"os"
	"reflect"
	"strings"
)

const (
	DefaultResultMapName = "BaseResultMap"
	DefaultBCLName       = "base_column_list"

	// MyBatis-Plus 内置方法 ID（BaseMapper 标准方法名），与 MP 生成器产出一致：
	// 生成的 XML 可直接被 mybatis-go 加载，业务层无需手写 GoExtraMapper（TODO P16）。
	MPInsertID         = "insert"
	MPDeleteByIDID     = "deleteById"
	MPUpdateByIDID     = "updateById"
	MPSelectByIDID     = "selectById"
	MPSelectOneID      = "selectOne"
	MPSelectListID     = "selectList"
	MPSelectPageID     = "selectPage"
	MPSelectCountID    = "selectCount"
	MPSelectBatchIDsID = "selectBatchIds"
	MPDeleteBatchIDsID = "deleteBatchIds"
)

func NewTableStruct(table string, res []map[string]interface{}) (*TableStructure, error) {
	ret := &TableStructure{
		Columns:       []*ColumnStructure{},
		ColumnMap:     map[string]*ColumnStructure{},
		Table:         table,
		PrimaryColumn: nil,
	}
	find := false
	for _, row := range res {
		pcs := newColumnStructure(row)
		if find {
			pcs.Primary = false
		}
		ret.Columns = append(ret.Columns, pcs)
		ret.ColumnMap[pcs.Name] = pcs
		if pcs.Primary && !find {
			ret.PrimaryColumn = pcs
			find = true
		}
	}
	if len(ret.Columns) == 0 {
		log.Errorf("get table %s structure failed", table)
		return nil, fmt.Errorf("get table %s structure failed", table)
	}
	if ret.PrimaryColumn == nil {
		log.Warnf("not found primary key in table %s", table)
	}
	return ret, nil
}

type TableStructure struct {
	Columns       []*ColumnStructure
	ColumnMap     map[string]*ColumnStructure
	Table         string
	PrimaryColumn *ColumnStructure
}

func (ts *TableStructure) SaveToFile(filename, prefix string) error {
	return ts.saveToFile(filename, prefix, false)
}

// SaveMPToFile 生成 MyBatis-Plus 风格 XML：内置 CRUD 使用 BaseMapper 标准方法名
// （insert/deleteById/updateById/selectById/selectList/selectOne/selectPage/
// selectCount/selectBatchIds/deleteBatchIds），可直接被 mybatis-go 加载。
func (ts *TableStructure) SaveMPToFile(filename, prefix string) error {
	return ts.saveToFile(filename, prefix, true)
}

func (ts *TableStructure) saveToFile(filename, prefix string, mp bool) error {
	doc := etree.NewDocument()
	ts.writeHeader(doc)
	mapper := ts.CreateMapper(doc, prefix)
	ts.writeResultMap(mapper, prefix)
	ts.writeBaseColumnList(mapper)
	if mp {
		ts.writeMPFunctions(mapper, prefix)
	} else {
		ts.writeDeleteFunction(mapper)
		ts.writeInsertFunction(mapper, prefix)
		ts.writeUpdateFunction(mapper, prefix)
		ts.writeUpdateTimeFunction(mapper, prefix)
		ts.writeSetDeletedFunction(mapper, prefix)
		ts.writeSelectFunction(mapper)
		ts.writeSelectAllFunction(mapper)
		ts.writeCountFunction(mapper)
	}
	doc.IndentTabs()
	bts, err := doc.WriteToBytes()
	if err != nil {
		return err
	}
	//fmt.Println(string(bts))
	return os.WriteFile(filename, bts, 0640)
}

// hasColumn 判断表结构是否包含指定列（大小写不敏感）。
func (ts *TableStructure) hasColumn(name string) bool {
	if ts.ColumnMap != nil {
		if _, ok := ts.ColumnMap[name]; ok {
			return true
		}
	}
	for _, c := range ts.Columns {
		if strings.EqualFold(c.Name, name) {
			return true
		}
	}
	return false
}

func (ts *TableStructure) writeHeader(doc *etree.Document) {
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)
	doc.CreateDirective(`DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd"`)
}

func (ts *TableStructure) getMapperName(prefix string) string {
	tname := ts.Table
	if len(prefix) > 0 && strings.HasPrefix(ts.Table, prefix) {
		tname = tname[len(prefix):]
	}
	arr := strings.Split(tname, "_")
	var res []string
	for _, item := range arr {
		res = append(res, UpperFirst(strings.TrimSpace(item)))
	}
	res = append(res, "Mapper")
	return strings.Join(res, "")
}
func (ts *TableStructure) getModelName(prefix string) string {
	tname := ts.Table
	if len(prefix) > 0 && strings.HasPrefix(ts.Table, prefix) {
		tname = tname[len(prefix):]
	}
	arr := strings.Split(tname, "_")
	var res []string
	for _, item := range arr {
		res = append(res, UpperFirst(strings.TrimSpace(item)))
	}
	res = append(res, "Model")
	return strings.Join(res, "")
}

func (ts *TableStructure) CreateMapper(doc *etree.Document, prefix string) *etree.Element {
	mapper := doc.CreateElement("mapper")
	mapper.CreateAttr("namespace", ts.getMapperName(prefix))
	return mapper
}

func (ts *TableStructure) writeResultMap(mapper *etree.Element, prefix string) {
	resultMap := mapper.CreateElement("resultMap")
	resultMap.CreateAttr("id", DefaultResultMapName)
	resultMap.CreateAttr("type", ts.getModelName(prefix))
	for _, column := range ts.Columns {
		if strings.Compare(strings.ToLower(column.Name), "deleted") == 0 {
			continue
		}
		if strings.Compare(strings.ToLower(column.Name), "delete_time") == 0 {
			continue
		}
		result := resultMap.CreateElement("result")
		result.CreateAttr("column", column.Name)
		result.CreateAttr("jdbcType", column.getJdbcType())
		result.CreateAttr("property", column.getPropertyName())
	}
}
func (ts *TableStructure) writeBaseColumnList(mapper *etree.Element) {
	sql := mapper.CreateElement("sql")
	sql.CreateAttr("id", DefaultBCLName)
	var cnames []string
	for _, column := range ts.Columns {
		if strings.Compare(strings.ToLower(column.Name), "deleted") == 0 {
			continue
		}
		if strings.Compare(strings.ToLower(column.Name), "delete_time") == 0 {
			continue
		}
		cnames = append(cnames, column.Name)
	}
	sql.CreateText(fmt.Sprintf("\n\t\t%s\n\t", strings.Join(cnames, ",\n\t\t")))
}
func (ts *TableStructure) getPrimaryJdbcType() string {
	if ts.PrimaryColumn != nil {
		return ToJavaType(ts.PrimaryColumn.Type)
	}
	return ToJavaType(reflect.TypeOf(""))
}
func (ts *TableStructure) generateDeleteSQL() string {
	return fmt.Sprintf("\n\t\tdelete from %s where %s=#{%s,jdbcType=%s}\n\t",
		ts.Table,
		ts.PrimaryColumn.Name,
		ts.PrimaryColumn.Name,
		ts.PrimaryColumn.getJdbcType(),
	)
}
func (ts *TableStructure) writeDeleteFunction(mapper *etree.Element) {
	de := mapper.CreateElement("delete")
	de.CreateAttr("id", "deleteByPrimaryKey")
	de.CreateAttr("parameterType", ts.getPrimaryJdbcType())
	de.CreateText(ts.generateDeleteSQL())
}
func (ts *TableStructure) generateInsertSQL() string {
	var cnames, cvalues []string
	for _, column := range ts.Columns {
		if strings.Compare(strings.ToLower(column.Name), "deleted") == 0 {
			continue
		}
		if strings.Compare(strings.ToLower(column.Name), "delete_time") == 0 {
			continue
		}
		cnames = append(cnames, column.Name)
		cvalues = append(cvalues, fmt.Sprintf("#{%s,jdbcType=%s}", column.getPropertyName(), column.getJdbcType()))
	}
	cns := strings.Join(cnames, ",\n\t\t")
	cvs := strings.Join(cvalues, ",\n\t\t")
	sql := fmt.Sprintf("\n\t\tinsert into %s \n\t\t(%s) \n\t\tvalues \n\t\t(%s)\n\t", ts.Table, cns, cvs)
	return sql
}
func (ts *TableStructure) writeInsertFunction(mapper *etree.Element, prefix string) {
	in := mapper.CreateElement("insert")
	in.CreateAttr("id", "insert")
	in.CreateAttr("parameterType", ts.getModelName(prefix))
	in.CreateText(ts.generateInsertSQL())
}
func (ts *TableStructure) generateUpdateSQL() string {
	var cvalues []string
	for _, column := range ts.Columns {
		if column.Primary {
			continue
		}
		if strings.Compare(strings.ToLower(column.Name), "deleted") == 0 {
			continue
		}
		if strings.Compare(strings.ToLower(column.Name), "delete_time") == 0 {
			continue
		}
		cvalues = append(cvalues, fmt.Sprintf("%s=#{%s,jdbcType=%s}", column.Name, column.getPropertyName(), column.getJdbcType()))
	}
	if len(ts.Columns) != len(cvalues)+1 {
		log.Warnf("check primary key for table %s", ts.Table)
	}
	cvs := strings.Join(cvalues, ",\n\t\t\t ")
	sql := fmt.Sprintf("\n\t\tupdate %s \n\t\tset %s \n\t\t where %s=#{%s,jdbcType=%s}\n\t",
		ts.Table,
		cvs,
		ts.PrimaryColumn.Name,
		ts.PrimaryColumn.getPropertyName(),
		ts.PrimaryColumn.getJdbcType(),
	)
	return sql
}
func (ts *TableStructure) generateSetDeletedSQL() string {
	sql := fmt.Sprintf("\n\t\tupdate %s \n\t\tset deleted=true,delete_time=now() \n\t\t where %s=#{%s,jdbcType=%s}\n\t",
		ts.Table,
		ts.PrimaryColumn.Name,
		ts.PrimaryColumn.getPropertyName(),
		ts.PrimaryColumn.getJdbcType(),
	)
	return sql
}
func (ts *TableStructure) generateUpdateTimeSQL() string {
	sql := fmt.Sprintf("\n\t\tupdate %s \n\t\tset update_time=now() \n\t\t where %s=#{%s,jdbcType=%s}\n\t",
		ts.Table,
		ts.PrimaryColumn.Name,
		ts.PrimaryColumn.getPropertyName(),
		ts.PrimaryColumn.getJdbcType(),
	)
	return sql
}

func (ts *TableStructure) writeUpdateFunction(mapper *etree.Element, prefix string) {
	up := mapper.CreateElement("update")
	up.CreateAttr("id", "updateByPrimaryKey")
	up.CreateAttr("parameterType", ts.getModelName(prefix))
	up.CreateText(ts.generateUpdateSQL())
}
func (ts *TableStructure) writeSelectFunction(mapper *etree.Element) {
	sf := mapper.CreateElement("select")
	sf.CreateAttr("id", "selectByPrimaryKey")
	sf.CreateAttr("parameterType", ts.getPrimaryJdbcType())
	sf.CreateAttr("resultMap", DefaultResultMapName)
	sf.CreateText("\n\t\tselect ")
	si := sf.CreateElement("include")
	si.CreateAttr("refid", DefaultBCLName)
	sf.CreateText(fmt.Sprintf("\n\t\tfrom %s where %s=#{%s,jdbcType=%s} and deleted = false\n\t",
		ts.Table,
		ts.PrimaryColumn.Name,
		ts.PrimaryColumn.getPropertyName(),
		ts.PrimaryColumn.getJdbcType(),
	))
}
func (ts *TableStructure) writeSelectAllFunction(mapper *etree.Element) {
	sf := mapper.CreateElement("select")
	sf.CreateAttr("id", "selectAll")
	sf.CreateAttr("resultMap", DefaultResultMapName)
	sf.CreateText("\n\t\t select ")
	si := sf.CreateElement("include")
	si.CreateAttr("refid", DefaultBCLName)
	sf.CreateText(fmt.Sprintf("\n\t\t from %s where deleted = false\n\t", ts.Table))
}

func (ts *TableStructure) writeCountFunction(mapper *etree.Element) {
	sf := mapper.CreateElement("select")
	sf.CreateAttr("id", "countByPrimaryKey")
	sf.CreateAttr("parameterType", ts.getPrimaryJdbcType())
	sf.CreateAttr("resultType", "int")
	sf.CreateText(fmt.Sprintf("\n\t\tselect count(%s) \n\t\tfrom %s where deleted = false\n\t", ts.PrimaryColumn.Name, ts.Table))
}
func (ts *TableStructure) writeSetDeletedFunction(mapper *etree.Element, prefix string) {
	up := mapper.CreateElement("update")
	up.CreateAttr("id", "setDeleted")
	up.CreateAttr("parameterType", ts.getPrimaryJdbcType())
	up.CreateText(ts.generateSetDeletedSQL())
}
func (ts *TableStructure) writeUpdateTimeFunction(mapper *etree.Element, prefix string) {
	up := mapper.CreateElement("update")
	up.CreateAttr("id", "updateUpTime")
	up.CreateAttr("parameterType", ts.getPrimaryJdbcType())
	up.CreateText(ts.generateUpdateTimeSQL())
}

// writeMPFunctions 写入 MyBatis-Plus BaseMapper 内置 CRUD 语句
// （insert/deleteById/updateById/selectById/selectOne/selectList/selectPage/
// selectCount/selectBatchIds/deleteBatchIds）。
func (ts *TableStructure) writeMPFunctions(mapper *etree.Element, prefix string) {
	if ts.PrimaryColumn == nil {
		log.Warnf("skip mybatis-plus functions for table %s: no primary key", ts.Table)
		return
	}
	// insert：与 BaseMapper.insert 一致
	in := mapper.CreateElement("insert")
	in.CreateAttr("id", MPInsertID)
	in.CreateAttr("parameterType", ts.getModelName(prefix))
	in.CreateText(ts.generateInsertSQL())

	// deleteById：有逻辑删除列则 update 标记删除，否则物理删除
	de := mapper.CreateElement("delete")
	de.CreateAttr("id", MPDeleteByIDID)
	de.CreateAttr("parameterType", ts.getMPPrimaryJdbcType())
	de.CreateText(ts.generateMPDeleteByIDSQL())

	// updateById
	up := mapper.CreateElement("update")
	up.CreateAttr("id", MPUpdateByIDID)
	up.CreateAttr("parameterType", ts.getModelName(prefix))
	up.CreateText(ts.generateUpdateSQL())

	// selectById
	sf := ts.createMPSelectElement(mapper, MPSelectByIDID, ts.getMPPrimaryJdbcType())
	sf.CreateText(ts.whereByPrimarySQL())

	// selectOne / selectList / selectPage：均查全表（分页由 MP 端 IPage 追加 limit）
	for _, id := range []string{MPSelectOneID, MPSelectListID, MPSelectPageID} {
		sf := ts.createMPSelectElement(mapper, id, "")
		sf.CreateText(ts.listTailSQL())
	}

	// selectCount
	cf := mapper.CreateElement("select")
	cf.CreateAttr("id", MPSelectCountID)
	cf.CreateAttr("resultType", "long")
	cf.CreateText(ts.countTailSQL())

	// selectBatchIds：select ... where id in (foreach)；parameterType 为主键类型，
	// codegen 检测 foreach 后自动生成切片签名（[]int64 等）
	sb := ts.createMPSelectElement(mapper, MPSelectBatchIDsID, ts.getMPPrimaryJdbcType())
	sb.CreateText(fmt.Sprintf("\n\t\tfrom %s where %s in ", ts.Table, ts.PrimaryColumn.Name))
	ts.writeMPInForeach(sb, "id")
	if ts.hasColumn("deleted") {
		sb.CreateText(" and deleted = false")
	}
	sb.CreateText("\n\t")

	// deleteBatchIds：逻辑删除优先；parameterType 为主键类型，codegen 生成切片签名
	db := mapper.CreateElement("delete")
	db.CreateAttr("id", MPDeleteBatchIDsID)
	db.CreateAttr("parameterType", ts.getMPPrimaryJdbcType())
	if ts.hasColumn("deleted") {
		db.CreateText(fmt.Sprintf("\n\t\tupdate %s set deleted=true,delete_time=now() where %s in ", ts.Table, ts.PrimaryColumn.Name))
	} else {
		db.CreateText(fmt.Sprintf("\n\t\tdelete from %s where %s in ", ts.Table, ts.PrimaryColumn.Name))
	}
	ts.writeMPInForeach(db, "id")
	db.CreateText("\n\t")
}

// getMPPrimaryJdbcType 主键 JDBC 类型（MP 风格）：int64/uint64 主键 → java.lang.Long（codegen 生成 int64 签名），
// 其余走 ToJavaType（默认 java.lang.Integer → int32）。
func (ts *TableStructure) getMPPrimaryJdbcType() string {
	if ts.PrimaryColumn == nil {
		return ToJavaType(reflect.TypeOf(""))
	}
	switch ts.PrimaryColumn.Type.Kind() {
	case reflect.Int64, reflect.Uint64:
		return "java.lang.Long"
	}
	return ToJavaType(ts.PrimaryColumn.Type)
}

// createMPSelectElement 创建 select 元素：select <include refid="base_column_list"/>
func (ts *TableStructure) createMPSelectElement(mapper *etree.Element, id, parameterType string) *etree.Element {
	sf := mapper.CreateElement("select")
	sf.CreateAttr("id", id)
	if len(parameterType) > 0 {
		sf.CreateAttr("parameterType", parameterType)
	}
	sf.CreateAttr("resultMap", DefaultResultMapName)
	sf.CreateText("\n\t\tselect ")
	si := sf.CreateElement("include")
	si.CreateAttr("refid", DefaultBCLName)
	return sf
}

// writeMPInForeach 追加 in (foreach item) 片段
func (ts *TableStructure) writeMPInForeach(parent *etree.Element, item string) *etree.Element {
	fe := parent.CreateElement("foreach")
	fe.CreateAttr("item", item)
	fe.CreateAttr("collection", "collection")
	fe.CreateAttr("open", "(")
	fe.CreateAttr("separator", ",")
	fe.CreateAttr("close", ")")
	fe.CreateText("#{" + item + "}")
	return fe
}

// whereByPrimarySQL 按主键过滤的 from 片段（含逻辑删除过滤）
func (ts *TableStructure) whereByPrimarySQL() string {
	sql := fmt.Sprintf("\n\t\tfrom %s where %s=#{%s,jdbcType=%s}",
		ts.Table, ts.PrimaryColumn.Name, ts.PrimaryColumn.getPropertyName(), ts.PrimaryColumn.getJdbcType())
	if ts.hasColumn("deleted") {
		sql += " and deleted = false"
	}
	return sql + "\n\t"
}

// listTailSQL 全表查询的 from 片段（含逻辑删除过滤）
func (ts *TableStructure) listTailSQL() string {
	if ts.hasColumn("deleted") {
		return fmt.Sprintf("\n\t\tfrom %s where deleted = false\n\t", ts.Table)
	}
	return fmt.Sprintf("\n\t\tfrom %s\n\t", ts.Table)
}

// countTailSQL count(*) 语句（含逻辑删除过滤）
func (ts *TableStructure) countTailSQL() string {
	if ts.hasColumn("deleted") {
		return fmt.Sprintf("\n\t\tselect count(*) \n\t\tfrom %s where deleted = false\n\t", ts.Table)
	}
	return fmt.Sprintf("\n\t\tselect count(*) \n\t\tfrom %s\n\t", ts.Table)
}

// generateMPDeleteByIDSQL deleteById：有逻辑删除列则 update 标记删除，否则物理删除
func (ts *TableStructure) generateMPDeleteByIDSQL() string {
	if ts.hasColumn("deleted") {
		return fmt.Sprintf("\n\t\tupdate %s \n\t\tset deleted=true,delete_time=now() \n\t\t where %s=#{%s,jdbcType=%s}\n\t",
			ts.Table, ts.PrimaryColumn.Name, ts.PrimaryColumn.getPropertyName(), ts.PrimaryColumn.getJdbcType())
	}
	return fmt.Sprintf("\n\t\tdelete from %s where %s=#{%s,jdbcType=%s}\n\t",
		ts.Table, ts.PrimaryColumn.Name, ts.PrimaryColumn.getPropertyName(), ts.PrimaryColumn.getJdbcType())
}
