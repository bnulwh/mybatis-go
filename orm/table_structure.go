package orm

import (
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
)

func newTableStruct(dbName, table string) (*types.TableStructure, error) {
	if gDbConn == nil || gDbConn.Dialector == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	sqlStr := gDbConn.Dialector.TableStructureSQL(table)
	if sqlStr == "" {
		return nil, fmt.Errorf("unsupport database type %v to get table structure", gDbConn.Setting.Type)
	}

	log.Debugf("sql: %v", sqlStr)
	res, err := Query(sqlStr)
	if err != nil {
		log.Errorf("get table %s structure failed.%v", table, err)
		return nil, err
	}
	return types.NewTableStruct(table, res)
}
