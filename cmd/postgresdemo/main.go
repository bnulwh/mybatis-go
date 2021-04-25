package main

import (
	log "github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/orm"
	"github.com/bnulwh/mybatis-go/types"
	_ "github.com/lib/pq"
	"time"
)

type UserInfoModel struct {
	Id          int
	CreatedBy   string
	UpdatedBy   string
	CreateTime  time.Time
	UpdateTime  time.Time
	GroupId     int
	Username    string
	PassMd5     string
	Roles       string
	Description string
	Avatar      string
}

type UserInfoModelMapper struct {
	orm.BaseMapper
	DeleteByPrimaryKey func(id int) (int64, error)
	Insert             func(model UserInfoModel) (int64, error)
	UpdateByPrimaryKey func(model UserInfoModel) (int64, error)
	SelectByPrimaryKey func(id int) ([]UserInfoModel, error)
	SelectAll          func() ([]UserInfoModel, error)
}

func init() {
	log.ConfigLocalFileSystemLogger("/var/log","postgresdemo")
	orm.Initialize("application-pg.properties")
	orm.RegisterModel(new(UserInfoModel))
	orm.RegisterMapper(new(UserInfoModelMapper))
}
func main() {
	defer orm.Close()
	mp := orm.NewMapper("UserInfoModelMapper").(UserInfoModelMapper)
	rs, err := mp.SelectAll()
	if err != nil {
		log.Errorf("select failed: %v", err)
	} else {
		for _, row := range rs {
			log.Infof("row: %v", types.ToJson(row))
		}
	}
}
