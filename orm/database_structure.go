package orm

import (
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
	"github.com/bnulwh/mybatis-go/utils"
	"strings"
)

func newDatabaseStructure(dbName, tables string) (*types.DatabaseStructure, error) {
	tns, err := fetchTables(dbName)
	if err != nil {
		return nil, err
	}
	pds := &types.DatabaseStructure{
		TableList: tns,
		TableMap:  map[string]*types.TableStructure{},
		Tables:    []*types.TableStructure{},
	}
	exts := make([]string, 0)
	if len(tables) == 0 {
		exts = tns
	} else {
		exts = strings.Split(tables, ",")
	}
	tbmp := utils.List2map(exts)
	for _, table := range tns {
		_, ok := tbmp[table]
		if !ok {
			continue
		}
		pts, err := newTableStruct(dbName, table)
		if err != nil {
			continue
		}
		pds.Tables = append(pds.Tables, pts)
		pds.TableMap[table] = pts
	}
	return pds, nil
}

func fetchTables(dbName string) ([]string, error) {
	sqlStr := tableListSQL(gDbConn.Setting, dbName)
	if sqlStr == "" {
		log.Errorf("unsupport database type %v to get table list", gDbConn.Setting.Type)
		return nil, fmt.Errorf("unsupport database type %v to get table list", gDbConn.Setting.Type)
	}
	res, err := Query(sqlStr)
	if err != nil {
		log.Errorf("get tables from %s structure failed.%v", dbName, err)
		return nil, err
	}
	tables := []string{}
	for _, row := range res {
		tables = append(tables, row["table_name"].(string))
	}
	return tables, nil
}
