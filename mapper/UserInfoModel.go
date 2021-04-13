package mapper

import(
	"github.com/bnulwh/mybatis-go/orm"
	"time"
)

type UserInfoModel struct{
	Id 	int
	CreatedBy 	string
	UpdatedBy 	string
	CreateTime 	time.Time
	UpdateTime 	time.Time
	GroupId 	int
	Username 	string
	PassMd5 	string
	Roles 	string
	Description 	string
	Avatar 	string
}

func init(){
	orm.RegisterModel(new(UserInfoModel))
}

