package orm

import "github.com/bnulwh/mybatis-go/types"

type sqlCache struct {
	Mappers *types.SqlMappers
}

func newSqlCache(dir string) *sqlCache {
	return &sqlCache{
		Mappers: types.NewSqlMappers(dir),
	}
}
