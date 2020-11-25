package main

import (
	log "github.com/astaxie/beego/logs"
	"github.com/bnulwh/mybatis-go/logger"
	"github.com/bnulwh/mybatis-go/orm"
	"github.com/bnulwh/mybatis-go/utils"
	_ "github.com/go-sql-driver/mysql"
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
	DeleteByPrimaryKey orm.ExecuteFunc
	Insert             orm.ExecuteFunc
	UpdateByPrimaryKey orm.ExecuteFunc
	SelectByPrimaryKey orm.QueryRowsFunc
	SelectAll          orm.QueryRowsFunc
}

func init() {
	logger.Initialize("mysqldemo.log")
	orm.Initialize("application-mysql.properties")
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
