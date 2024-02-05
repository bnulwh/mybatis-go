package orm

import (
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
)

func newTableStruct(dbName, table string) (*types.TableStructure, error) {
	var sql string
	switch gDbConn.Setting.Type {
	case PostgresDb:
		sql = fmt.Sprintf(`SELECT
    A.ordinal_position,A.table_name,A.column_name,CASE A.is_nullable WHEN 'NO' THEN 0 ELSE 1 END AS is_nullable,
    col_description(B.attrelid,B.attnum) as column_comment,
    A.data_type as column_type,coalesce(A.character_maximum_length, A.numeric_precision, -1) as length,
    A.numeric_scale,CASE WHEN length(B.attname) > 0 THEN 'PRI' ELSE '' END AS column_key
    FROM information_schema.columns A,pg_attribute B
    WHERE A.column_name = B.attname AND B.attrelid = '%s' :: regclass   
          AND  A.table_schema = 'public'  AND A.table_name = '%s'
    ORDER BY A.ordinal_position ASC`, table, table)
	case MySqlDb:
		sql = fmt.Sprintf(`select TABLE_NAME as table_name,COLUMN_NAME as column_name,
    COLUMN_TYPE as column_type,COLUMN_COMMENT as column_comment,COLUMN_KEY as column_key 
    from information_schema.COLUMNS WHERE TABLE_SCHEMA='%s' AND TABLE_NAME='%s'
    ORDER BY ORDINAL_POSITION ASC`, dbName, table)
	default:
		log.Errorf("unsupport database type %v to get table structure", gDbConn.Setting.Type)
		return nil, fmt.Errorf("unsupport database type %v to get table structure", gDbConn.Setting.Type)
	}

	log.Debugf("sql: %v", sql)
	res, err := Query(sql)
	if err != nil {
		log.Errorf("get table %s structure failed.%v", table, err)
		return nil, err
	}
	return types.NewTableStruct(table, res)
}
