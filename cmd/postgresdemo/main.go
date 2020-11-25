package main

import (
	log "github.com/astaxie/beego/logs"
	"github.com/bnulwh/mybatis-go/logger"
	"github.com/bnulwh/mybatis-go/orm"
	"github.com/bnulwh/mybatis-go/utils"
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
	logger.Initialize("postgresdemo.log")
	orm.Initialize("application-pg.properties")
	orm.RegisterModel(new(UserInfoModel))
	orm.RegisterMapper(new(UserInfoModelMapper))
}
func main() {
	defer orm.Close()
	mp := orm.NewMapper("UserInfoModelMapper").(UserInfoModelMapper)
	rs, err := mp.SelectAll()
	if err != nil {
		log.Error("select failed: %v", err)
	} else {
		for _, row := range rs {
			log.Info("row: %v", utils.ToJson(row))
		}
	}
}
