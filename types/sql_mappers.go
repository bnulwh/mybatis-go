package types

import (
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/utils"
	"os"
	"path/filepath"
	"strings"
)

type SqlMappers struct {
	Mappers      []SqlMapper
	NamedMappers map[string]*SqlMapper
}

//func NewSqlMappersEx(ds *DatabaseStructure) *SqlMappers {
//	var mps []SqlMapper
//	nmp := map[string]*SqlMapper{}
//	for _,tableName := range ds.TableList{
//
//	}
//}

func NewSqlMappers(dir string) *SqlMappers {
	filenames := filterMapperFiles(dir)
	var mps []SqlMapper
	nmp := map[string]*SqlMapper{}
	for _, filename := range filenames {
		log.Debugf("begin parse mapper file: %v", filename)
		mp := loadMapper(filename)
		if mp != nil {
			mps = append(mps, *mp)
			nmp[mp.Namespace] = mp
			sname := GetShortName(mp.Namespace)
			nmp[sname] = mp
			nmp[buildKey(sname)] = mp
			nmp[strings.ToLower(sname)] = mp
		}
	}
	return &SqlMappers{
		Mappers:      mps,
		NamedMappers: nmp,
	}
}

func (in *SqlMappers) GenerateFiles(dir, pkg string) {
	err := utils.MakeDirAll(dir)
	if err != nil {
		return
	}
	err = utils.MakeDirAll(filepath.Join(dir, "mapper"))
	if err != nil {
		return
	}
	err = utils.MakeDirAll(filepath.Join(dir, "models"))
	if err != nil {
		return
	}
	for _, mapper := range in.Mappers {
		mapper.GenerateFiles(dir, pkg)
	}
}

func filterMapperFiles(dir string) []string {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if strings.Compare(strings.ToLower(path[len(path)-4:]), ".xml") == 0 {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		log.Warnf("walk dir %v failed: %v", dir, err)
	}
	return files
}
