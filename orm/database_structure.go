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
	var sql string
	switch gDbConn.Setting.Type {
	case MySqlDb:
		// MySQL 下 schema 即数据库名；显式配置 spring.datasource.schema 时以它为准
		sql = fmt.Sprintf("select DISTINCT TABLE_NAME as table_name from information_schema.COLUMNS WHERE TABLE_SCHEMA='%s'", gDbConn.Setting.effectiveSchema(dbName))
	case PostgresDb, KingbaseDb:
		// 配置了 schema 时仅列该 schema 的表（join pg_namespace）；未配置保持历史行为（列出所有 schema）
		if schema := gDbConn.Setting.Schema; schema != "" {
			sql = fmt.Sprintf("select relname as TABLE_NAME from pg_class c join pg_namespace n on n.oid = c.relnamespace where c.relkind = 'r' and n.nspname = '%s' and c.relname not like 'pg_%%' and c.relname not like 'sql_%%'", schema)
		} else {
			sql = "select relname as TABLE_NAME from pg_class where  relkind = 'r' and relname not like 'pg_%' and relname not like 'sql_%'"
		}
	case SqliteDb:
		sql = "select name as table_name from sqlite_master where type='table' and name not like 'sqlite_%'"
	default:
		log.Errorf("unsupport database type %v to get table list", gDbConn.Setting.Type)
		return nil, fmt.Errorf("unsupport database type %v to get table list", gDbConn.Setting.Type)
	}
	res, err := Query(sql)
	if err != nil {
		log.Errorf("get tables from %s structure failed.%v", dbName, err)
		return nil, err
	}
	//fmt.Println(res)
	tables := []string{}
	for _, row := range res {
		tables = append(tables, row["table_name"].(string))
	}
	return tables, nil
}
