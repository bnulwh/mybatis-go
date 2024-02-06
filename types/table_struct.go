package types

import (
	"fmt"
	"github.com/beevik/etree"
	"github.com/bnulwh/mybatis-go/log"
	"io/ioutil"
	"reflect"
	"strings"
)

const (
	DefaultResultMapName = "BaseResultMap"
	DefaultBCLName       = "base_column_list"
)

func NewTableStruct(table string, res []map[string]interface{}) (*TableStructure, error) {
	ret := &TableStructure{
		Columns:       []*ColumnStucture{},
		ColumnMap:     map[string]*ColumnStucture{},
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
	Columns       []*ColumnStucture
	ColumnMap     map[string]*ColumnStucture
	Table         string
	PrimaryColumn *ColumnStucture
}

func (ts *TableStructure) saveToFile(filename, prefix string) error {
	doc := etree.NewDocument()
	ts.writeHeader(doc)
	mapper := ts.createMapper(doc, prefix)
	ts.writeResultMap(mapper, prefix)
	ts.writeBaseColumnList(mapper)
	ts.writeDeleteFunction(mapper)
	ts.writeInsertFunction(mapper, prefix)
	ts.writeUpdateFunction(mapper, prefix)
	ts.writeUpdateTimeFunction(mapper, prefix)
	ts.writeSetDeletedFunction(mapper, prefix)
	ts.writeSelectFunction(mapper)
	ts.writeSelectAllFunction(mapper)
	ts.writeCountFunction(mapper)
	doc.IndentTabs()
	bts, err := doc.WriteToBytes()
	if err != nil {
		return err
	}
	//fmt.Println(string(bts))
	return ioutil.WriteFile(filename, bts, 0640)
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

func (ts *TableStructure) createMapper(doc *etree.Document, prefix string) *etree.Element {
	mapper := doc.CreateElement("mapper")
	mapper.CreateAttr("namespace", ts.getMapperName(prefix))
	return mapper
}

func (ts *TableStructure) writeResultMap(mapper *etree.Element, prefix string) {
	resultMap := mapper.CreateElement("resultMap")
	resultMap.CreateAttr("id", DefaultResultMapName)
	resultMap.CreateAttr("type", ts.getModelName(prefix))
	for _, column := range ts.Columns {
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
	sf.CreateText(fmt.Sprintf("\n\t\tfrom %s where %s=#{%s,jdbcType=%s}\n\t",
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
	sf.CreateText(fmt.Sprintf("\n\t\t from %s \n\t", ts.Table))
}

func (ts *TableStructure) writeCountFunction(mapper *etree.Element) {
	sf := mapper.CreateElement("select")
	sf.CreateAttr("id", "countByPrimaryKey")
	sf.CreateAttr("parameterType", ts.getPrimaryJdbcType())
	sf.CreateAttr("resultType", "int")
	sf.CreateText(fmt.Sprintf("\n\t\tselect count(%s) \n\t\tfrom %s\n\t", ts.PrimaryColumn.Name, ts.Table))
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
