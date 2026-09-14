package orm

import (
	"reflect"
)

type PageParam struct {
	PageNum  int
	PageSize int
}

func (p *PageParam) Offset() int {
	if p.PageNum <= 1 {
		return 0
	}
	return (p.PageNum - 1) * p.PageSize
}

func (p *PageParam) Limit() int {
	return p.PageSize
}

func (p *PageParam) Valid() bool {
	return p.PageNum > 0 && p.PageSize > 0
}

type Page struct {
	Total    int64
	Records  []map[string]interface{}
	PageNum  int
	PageSize int
}

var pageType = reflect.TypeOf((*Page)(nil))

var pageParamType = reflect.TypeOf((*PageParam)(nil))
