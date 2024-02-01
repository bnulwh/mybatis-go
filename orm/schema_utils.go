package orm

import (
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
	"path/filepath"
)

func SchemaToCode(dir, prefix, tables string) {
	ds, err := newDatabaseStructure(gDbConn.Setting.Name, tables)
	if err != nil {
		log.Errorf("get database structure failed. %v", err)
		return
	}
	if ds == nil {
		return
	}
	mapperDir := filepath.Join(dir, "resources", "mapper")
	err = ds.SaveToDir(mapperDir, prefix, tables)
	if err != nil {
		log.Errorf("save to dir failed.%v", err)
		return
	}
	codeDir := filepath.Join(dir, "src")
	mps := types.NewSqlMappers(mapperDir)
	mps.GenerateFiles(codeDir, "src")
}
