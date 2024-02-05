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
		err = ts.saveToFile(filename, prefix)
		if err != nil {
			log.Warnf("save table %s failed. %v", name, err)
		}
	}
	return nil
}
