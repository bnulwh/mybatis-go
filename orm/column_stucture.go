package orm

import (
	log "github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/types"
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

func newColumnStructureFromMysl(row map[string]interface{}) *ColumnStucture {
	log.Debugf("row %v", row)
	return &ColumnStucture{
		Name:    row["COLUMN_NAME"].(string),
		Type:    types.ParseJdbcTypeFrom(row["COLUMN_TYPE"].(string)),
		DbType:  row["COLUMN_TYPE"].(string),
		Comment: row["COLUMN_COMMENT"].(string),
		Primary: row["COLUMN_KEY"].(string) == "PRI",
	}
}
func newColumnStructureFromPostgres(row map[string]interface{}) *ColumnStucture {
	log.Debugf("row %v", row)
	return &ColumnStucture{
		Name:    row["column_name"].(string),
		Type:    types.ParseJdbcTypeFrom(row["column_type"].(string)),
		DbType:  row["column_type"].(string),
		Comment: row["column_comment"].(string),
		Primary: row["column_key"].(string) == "PRI",
	}
}

func (cs ColumnStucture) getJdbcType() string {
	return strings.ToUpper(types.GetJdbcTypePart(cs.DbType))
}

func (cs ColumnStucture) getPropertyName() string {
	arr := strings.Split(cs.Name, "_")
	var res []string
	res = append(res, arr[0])
	for i := 1; i < len(arr); i++ {
		res = append(res, types.UpperFirst(strings.TrimSpace(arr[i])))
	}
	return strings.Join(res, "")
}
