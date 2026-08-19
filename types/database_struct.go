package types

import (
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/utils"
	"path/filepath"
	"strings"
)

type DatabaseStructure struct {
	Tables    []*TableStructure
	TableList []string
	TableMap  map[string]*TableStructure
}

func (ds *DatabaseStructure) SaveToDir(dir, prefix, tables string) error {
	return ds.saveToDir(dir, prefix, tables, false)
}

// SaveToDirMP 生成 MyBatis-Plus 风格内置 CRUD XML（BaseMapper 标准方法名）。
func (ds *DatabaseStructure) SaveToDirMP(dir, prefix, tables string) error {
	return ds.saveToDir(dir, prefix, tables, true)
}

func (ds *DatabaseStructure) saveToDir(dir, prefix, tables string, mp bool) error {
	err := utils.MakeDirAll(dir)
	if err != nil {
		log.Errorf("check dir %s failed.%v", dir, err)
		return err
	}
	exts := make([]string, 0)
	if len(tables) == 0 {
		exts = ds.TableList
	} else {
		exts = strings.Split(tables, ",")
	}
	tbmp := utils.List2map(exts)
	for name, ts := range ds.TableMap {
		_, ok := tbmp[name]
		if !ok {
			continue
		}
		filename := filepath.Join(dir, fmt.Sprintf("%s.xml", ts.getMapperName(prefix)))
		if mp {
			err = ts.SaveMPToFile(filename, prefix)
		} else {
			err = ts.SaveToFile(filename, prefix)
		}
		if err != nil {
			log.Warnf("save table %s failed. %v", name, err)
		}
	}
	return nil
}
