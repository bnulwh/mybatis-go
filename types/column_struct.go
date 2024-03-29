package types

import (
	"github.com/bnulwh/mybatis-go/log"
	"reflect"
	"strings"
)

type ColumnStucture struct {
	Name    string
	Type    reflect.Type
	DbType  string
	Comment string
	Primary bool
}

func newColumnStructure(row map[string]interface{}) *ColumnStucture {
	log.Debugf("row %v", row)
	return &ColumnStucture{
		Name:    row["column_name"].(string),
		Type:    ParseJdbcTypeFrom(row["column_type"].(string)),
		DbType:  row["column_type"].(string),
		Comment: row["column_comment"].(string),
		Primary: row["column_key"].(string) == "PRI",
	}
}

func (cs ColumnStucture) getJdbcType() string {
	jt := strings.ToUpper(GetJdbcTypePart(cs.DbType))
	if jt == "TEXT" || jt == "LONGTEXT" || jt == "TINYTEXT" {
		return "VARCHAR"
	}
	if jt == "CHARACTER" {
		return "VARCHAR"
	}
	return jt
}

func (cs ColumnStucture) getPropertyName() string {
	arr := strings.Split(cs.Name, "_")
	var res []string
	res = append(res, arr[0])
	for i := 1; i < len(arr); i++ {
		res = append(res, UpperFirst(strings.TrimSpace(arr[i])))
	}
	return strings.Join(res, "")
}
