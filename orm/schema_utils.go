package orm

import (
	log "github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/types"
	"path/filepath"
)

func SchemaToCode(dir string) {
	ds, err := newDatabaseStructure(gDbConn.dbName)
	if err != nil {
		log.Errorf("get database structure failed. %v", err)
		return
	}
	if ds == nil {
		return
	}
	mapperDir := filepath.Join(dir, "resources", "mapper")
	err = ds.SaveToDir(mapperDir)
	if err != nil {
		log.Errorf("save to dir failed.%v", err)
		return
	}
	codeDir := filepath.Join(dir, "src")
	mps := types.NewSqlMappers(mapperDir)
	mps.GenerateFiles(codeDir, "src")
}
