package mapper

import (
	"github.com/bnulwh/mybatis-go/orm"
	"sync"
) 

type UserInfoModelMapper struct {
	orm.BaseMapper
	DeleteByPrimaryKey 	func (int32) (int64,error)
	Insert 	func (UserInfoModel) (int64,error)
	UpdateByPrimaryKey 	func (UserInfoModel) (int64,error)
	SelectByPrimaryKey 	func (int32) ([]UserInfoModel,error)
	SelectAll 	func () ([]UserInfoModel,error)
}

var (
	gUserInfoModelMapper  *UserInfoModelMapper
	gUserInfoModelMapperOnce  sync.Once
)

func init() {
	orm.RegisterMapper(new(UserInfoModelMapper))
}

func GetUserInfoModelMapper() *UserInfoModelMapper{
	gUserInfoModelMapperOnce.Do(func() {
		gUserInfoModelMapper = orm.NewMapperPtr("UserInfoModelMapper").(*UserInfoModelMapper)
	})
	return gUserInfoModelMapper
}

