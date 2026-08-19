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

// SchemaToCodeMP 生成 MyBatis-Plus 风格内置 CRUD（BaseMapper 标准方法名）XML + Go 代码：
// insert/deleteById/updateById/selectById/selectList/selectOne/selectPage/
// selectCount/selectBatchIds/deleteBatchIds 直接可用，业务层无需手写 GoExtraMapper（TODO P16）。
func SchemaToCodeMP(dir, prefix, tables string) {
	ds, err := newDatabaseStructure(gDbConn.Setting.Name, tables)
	if err != nil {
		log.Errorf("get database structure failed. %v", err)
		return
	}
	if ds == nil {
		return
	}
	mapperDir := filepath.Join(dir, "resources", "mapper")
	err = ds.SaveToDirMP(mapperDir, prefix, tables)
	if err != nil {
		log.Errorf("save to dir failed.%v", err)
		return
	}
	codeDir := filepath.Join(dir, "src")
	mps := types.NewSqlMappers(mapperDir)
	mps.GenerateFiles(codeDir, "src")
}
