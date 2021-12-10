package orm

import (
	"fmt"
	"github.com/beevik/etree"
	log "github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/types"
	"io/ioutil"
	"reflect"
	"strings"
)

const (
	DefaultResultMapName = "BaseResultMap"
	DefaultBCLName       = "base_column_list"
)

type TableStructure struct {
	Columns       []*ColumnStucture
	ColumnMap     map[string]*ColumnStucture
	Table         string
	PrimaryColumn *ColumnStucture
}

func newTableStructFromMysql(dbName, table string) (*TableStructure, error) {
	sql := fmt.Sprintf("select TABLE_NAME,COLUMN_NAME,COLUMN_TYPE,COLUMN_COMMENT,COLUMN_KEY from information_schema.`COLUMNS` WHERE TABLE_SCHEMA='%s' AND TABLE_NAME='%s'", dbName, table)
	log.Debugf("sql: %v", sql)
	res, err := Query(sql)
	if err != nil {
		log.Errorf("get table %s structure failed.%v", table, err)
		return nil, err
	}
	ret := &TableStructure{
		Columns:       []*ColumnStucture{},
		ColumnMap:     map[string]*ColumnStucture{},
		Table:         table,
		PrimaryColumn: nil,
	}
	for _, row := range res {
		pcs := newColumnStructureFromMysl(row)
		ret.Columns = append(ret.Columns, pcs)
		ret.ColumnMap[pcs.Name] = pcs
		if pcs.Primary {
			ret.PrimaryColumn = pcs
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
func newTableStructFromPostgres(dbName, table string) (*TableStructure, error) {
	sql := fmt.Sprintf(`SELECT
    A.ordinal_position,A.column_name,CASE A.is_nullable WHEN 'NO' THEN 0 ELSE 1 END AS is_nullable,
    col_description(B.attrelid,B.attnum) as column_comment,
    A.data_type as column_type,coalesce(A.character_maximum_length, A.numeric_precision, -1) as length,
    A.numeric_scale,CASE WHEN length(B.attname) > 0 THEN 'PRI' ELSE '' END AS column_key
    FROM information_schema.columns A,pg_attribute B
    WHERE A.column_name = B.attname AND B.attrelid = '%s' :: regclass   
          AND  A.table_schema = 'public'  AND A.table_name = '%s'
    ORDER BY A.ordinal_position ASC`, table, table)
	log.Debugf("sql: %v", sql)
	res, err := Query(sql)
	if err != nil {
		log.Errorf("get table %s structure failed.%v", table, err)
		return nil, err
	}
	ret := &TableStructure{
		Columns:       []*ColumnStucture{},
		ColumnMap:     map[string]*ColumnStucture{},
		Table:         table,
		PrimaryColumn: nil,
	}
	find := false
	for _, row := range res {
		pcs := newColumnStructureFromPostgres(row)
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

func (ts *TableStructure) saveToFile(filename string) error {
	doc := etree.NewDocument()
	ts.writeHeader(doc)
	mapper := ts.createMapper(doc)
	ts.writeResultMap(mapper)
	ts.writeBaseColumnList(mapper)
	ts.writeDeleteFunction(mapper)
	ts.writeInsertFunction(mapper)
	ts.writeUpdateFunction(mapper)
	ts.writeSelectFunction(mapper)
	ts.writeSelectAllFunction(mapper)
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

func (ts *TableStructure) getMapperName() string {
	arr := strings.Split(ts.Table, "_")
	var res []string
	for _, item := range arr {
		res = append(res, types.UpperFirst(strings.TrimSpace(item)))
	}
	res = append(res, "Mapper")
	return strings.Join(res, "")
}
func (ts *TableStructure) getModelName() string {
	arr := strings.Split(ts.Table, "_")
	var res []string
	for _, item := range arr {
		res = append(res, types.UpperFirst(strings.TrimSpace(item)))
	}
	res = append(res, "Model")
	return strings.Join(res, "")
}

func (ts *TableStructure) createMapper(doc *etree.Document) *etree.Element {
	mapper := doc.CreateElement("mapper")
	mapper.CreateAttr("namespace", ts.getMapperName())
	return mapper
}

func (ts *TableStructure) writeResultMap(mapper *etree.Element) {
	resultMap := mapper.CreateElement("resultMap")
	resultMap.CreateAttr("id", DefaultResultMapName)
	resultMap.CreateAttr("type", ts.getModelName())
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
	sql.CreateText(strings.Join(cnames, ","))
}
func (ts *TableStructure) getPrimaryJdbcType() string {
	if ts.PrimaryColumn != nil {
		return types.ToJavaType(ts.PrimaryColumn.Type)
	}
	return types.ToJavaType(reflect.TypeOf(""))
}
func (ts *TableStructure) generateDeleteSQL() string {
	return fmt.Sprintf("delete from %s where %s=#{%s,jdbcType=%s}",
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
	cns := strings.Join(cnames, ",")
	cvs := strings.Join(cvalues, ",")
	sql := fmt.Sprintf("insert into %s (%s) values (%s)", ts.Table, cns, cvs)
	return sql
}
func (ts *TableStructure) writeInsertFunction(mapper *etree.Element) {
	in := mapper.CreateElement("insert")
	in.CreateAttr("id", "insert")
	in.CreateAttr("parameterType", ts.getModelName())
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
	cvs := strings.Join(cvalues, ",")
	sql := fmt.Sprintf("update %s set %s where %s=#{%s,jdbcType=%s}",
		ts.Table,
		cvs,
		ts.PrimaryColumn.Name,
		ts.PrimaryColumn.getPropertyName(),
		ts.PrimaryColumn.getJdbcType(),
	)
	return sql
}
func (ts *TableStructure) writeUpdateFunction(mapper *etree.Element) {
	up := mapper.CreateElement("update")
	up.CreateAttr("id", "updateByPrimaryKey")
	up.CreateAttr("parameterType", ts.getModelName())
	up.CreateText(ts.generateUpdateSQL())
}
func (ts *TableStructure) writeSelectFunction(mapper *etree.Element) {
	sf := mapper.CreateElement("select")
	sf.CreateAttr("id", "selectByPrimaryKey")
	sf.CreateAttr("parameterType", ts.getPrimaryJdbcType())
	sf.CreateAttr("resultMap", DefaultResultMapName)
	sf.CreateText(" select ")
	si := sf.CreateElement("include")
	si.CreateAttr("refid", DefaultBCLName)
	sf.CreateText(fmt.Sprintf(" from %s where %s=#{%s,jdbcType=%s}",
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
	sf.CreateText(" select ")
	si := sf.CreateElement("include")
	si.CreateAttr("refid", DefaultBCLName)
	sf.CreateText(fmt.Sprintf(" from %s ", ts.Table))
}
