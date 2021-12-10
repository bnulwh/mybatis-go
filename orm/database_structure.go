package orm

import (
	"fmt"
	log "github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/utils"
	"path/filepath"
)

type DatabaseStructure struct {
	Tables    []*TableStructure
	TableList []string
	TableMap  map[string]*TableStructure
}

func newDatabaseStructure(dbName string) (*DatabaseStructure, error) {
	tables, err := fetchTables(dbName)
	if err != nil {
		return nil, err
	}
	pds := &DatabaseStructure{
		TableList: tables,
		TableMap:  map[string]*TableStructure{},
		Tables:    []*TableStructure{},
	}
	for _, table := range tables {
		pts, err := newTableStruct(dbName, table)
		if err != nil {
			continue
		}
		pds.Tables = append(pds.Tables, pts)
		pds.TableMap[table] = pts
	}
	return pds, nil
}

func (ds *DatabaseStructure) SaveToDir(dir string) error {
	err := utils.MakeDirAll(dir)
	if err != nil {
		log.Errorf("check dir %s failed.%v", dir, err)
		return err
	}
	for name, ts := range ds.TableMap {
		filename := filepath.Join(dir, fmt.Sprintf("%s.xml", ts.getMapperName()))
		err = ts.saveToFile(filename)
		if err != nil {
			log.Warnf("save table %s failed. %v", name, err)
		}
	}
	return nil
}

func fetchTables(dbName string) ([]string, error) {
	var sql string
	switch gDbConn.dbType {
	case MySqlDb:
		sql = fmt.Sprintf("select DISTINCT TABLE_NAME as table_name from information_schema.COLUMNS WHERE TABLE_SCHEMA='%s'", dbName)
	case PostgresDb:
		sql = "select relname as TABLE_NAME from pg_class where  relkind = 'r' and relname not like 'pg_%' and relname not like 'sql_%'"
	default:
		log.Errorf("unsupport database type %v to get table list", gDbConn.dbType)
		return nil, fmt.Errorf("unsupport database type %v to get table list", gDbConn.dbType)
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
