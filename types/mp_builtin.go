package types

import (
	"bytes"
	"github.com/beevik/etree"
	"github.com/bnulwh/mybatis-go/log"
	"reflect"
	"strings"
	"unicode"
)

// MP 内置 CRUD 方法 ID（与 BaseMapper 标准方法名对齐，见 table_struct.go 常量）
var mpBuiltinIDs = []string{
	MPInsertID, MPDeleteByIDID, MPUpdateByIDID, MPSelectByIDID,
	MPSelectOneID, MPSelectListID, MPSelectPageID, MPSelectCountID,
	MPSelectBatchIDsID, MPDeleteBatchIDsID,
}

// jdkTypeNames resultMap type 为 JDK 基础类型时不可推导表结构
var jdkTypeNames = map[string]bool{
	"map": true, "hashmap": true, "treemap": true,
	"list": true, "arraylist": true, "array": true, "slice": true,
	"string": true, "varchar": true, "char": true,
	"int": true, "integer": true, "int32": true, "int64": true,
	"long": true, "bigint": true, "short": true, "byte": true,
	"bool": true, "boolean": true,
	"float": true, "float32": true, "float64": true, "double": true,
	"time": true, "timestamp": true, "datetime": true, "date": true,
}

// ensureMPBuiltinCRUD：Mapper 缺少 MP 内置 CRUD 时，若存在可推导表结构的 resultMap
// （含基本类型列 id/result，type 为业务模型），则在内存中补生成缺失的内置方法
// （insert/deleteById/updateById/selectById/selectOne/selectList/selectPage/
// selectCount/selectBatchIds/deleteBatchIds），按方法 ID 逐个补缺、已有（手写）不覆盖。
func (in *SqlMapper) ensureMPBuiltinCRUD() {
	ts, rm := buildTableStructureFromResultMap(in)
	if ts == nil {
		return
	}
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)
	mapper := doc.CreateElement("mapper")
	mapper.CreateAttr("namespace", "__mp_builtin__")
	ts.writeBaseColumnList(mapper)
	ts.writeMPFunctions(mapper, "")
	bts, err := doc.WriteToBytes()
	if err != nil {
		log.Warnf("generate mp builtin crud for %v failed: %v", in.Namespace, err)
		return
	}
	node, err := parseXmlNode(bytes.NewReader(bts))
	if err != nil || node == nil {
		log.Warnf("parse generated mp builtin crud for %v failed: %v", in.Namespace, err)
		return
	}
	sns := makeNamedSql(filterSqlElement(node.Elements))
	rms := map[string]*ResultMap{buildKey(DefaultResultMapName): rm}
	added := 0
	for _, fn := range filterSqlFunction(node.Elements, rms, sns, "__mp_builtin__") {
		if in.NamedFunctions[fn.Id] != nil {
			continue // 已有（手写或原 XML 自带），不覆盖
		}
		fn.Owner = in.Namespace
		in.Functions = append(in.Functions, fn)
		in.NamedFunctions[fn.Id] = fn
		in.NamedFunctions[buildKey(fn.Id)] = fn
		in.NamedFunctions[strings.ToLower(fn.Id)] = fn
		added++
	}
	if added > 0 {
		log.Infof("mapper %v: generated %d mybatis-plus builtin crud functions in memory (table %s)", in.Namespace, added, ts.Table)
	}
}

// buildTableStructureFromResultMap 从 Mapper 中选取第一个可推导表结构的 resultMap：
//   - type 为业务模型（非 JDK 类型），且含至少一个基本类型列（<id>/<result>）
//   - 表名：type 短名去 Model 后缀 + camelCase → snake_case（SysUser → sys_user）
//   - 列：<id>/<result> 项；jdbcType 缺失时主键默认 int64（BIGINT）、普通列默认 string（VARCHAR）
//
// 返回 nil 表示无法推导（不生成 CRUD）。
func buildTableStructureFromResultMap(in *SqlMapper) (*TableStructure, *ResultMap) {
	var target *ResultMap
	for _, rm := range in.Maps {
		if rm == nil || rm.TypeName == "" {
			continue
		}
		sname := GetShortName(rm.TypeName)
		if jdkTypeNames[strings.ToLower(sname)] {
			continue
		}
		hasBasic := false
		for _, item := range rm.Results {
			if item.Kind == ResultItemKindId || item.Kind == ResultItemKindResult {
				hasBasic = true
				break
			}
		}
		if hasBasic {
			target = rm
			break
		}
	}
	if target == nil {
		return nil, nil
	}
	table := camelToSnake(strings.TrimSuffix(GetShortName(target.TypeName), "Model"))
	if table == "" {
		return nil, nil
	}
	ts := &TableStructure{
		Table:         table,
		ModelName:     target.TypeName,
		Columns:       []*ColumnStructure{},
		ColumnMap:     map[string]*ColumnStructure{},
		PrimaryColumn: nil,
	}
	for _, item := range target.Results {
		if item.Kind != ResultItemKindId && item.Kind != ResultItemKindResult {
			continue
		}
		if item.Column == "" {
			continue
		}
		cs := &ColumnStructure{
			Name:    item.Column,
			DbType:  item.JdbcType,
			Primary: item.PrimaryKey,
			Comment: "",
		}
		if item.JdbcType != "" {
			cs.Type = ParseJdbcTypeFrom(item.JdbcType)
			cs.DbType = item.JdbcType
		} else if item.PrimaryKey {
			cs.Type = reflect.TypeOf(int64(0))
			cs.DbType = "BIGINT" // 主键无 jdbcType 默认 BIGINT（占位符 #{x,jdbcType=BIGINT} 可被解析）
		} else {
			cs.Type = reflect.TypeOf("")
			cs.DbType = "VARCHAR" // 普通列无 jdbcType 默认 VARCHAR
		}
		ts.Columns = append(ts.Columns, cs)
		ts.ColumnMap[cs.Name] = cs
		if cs.Primary && ts.PrimaryColumn == nil {
			ts.PrimaryColumn = cs
		}
	}
	if ts.PrimaryColumn == nil {
		log.Debugf("resultMap %v has no primary key, skip mp builtin crud", target.Id)
		return nil, nil
	}
	return ts, target
}

// camelToSnake CamelCase → snake_case（SysUser → sys_user）
func camelToSnake(s string) string {
	var buf strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				buf.WriteByte('_')
			}
			buf.WriteRune(unicode.ToLower(r))
		} else {
			buf.WriteRune(r)
		}
	}
	return buf.String()
}
