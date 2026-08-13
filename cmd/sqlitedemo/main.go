package main

import (
	log "github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/orm"
	"github.com/bnulwh/mybatis-go/types"
	_ "modernc.org/sqlite"
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
	log.ConfigLocalFileSystemLogger("logs", "sqlitedemo")
	orm.SetLogger(log.StandardLogger())
	orm.Initialize("application-sqlite.properties")
	orm.RegisterModel(new(UserInfoModel))
	orm.RegisterMapper(new(UserInfoModelMapper))
}

func main() {
	defer orm.Close()
	// 初始化表结构（SQLite 无 DDL 迁移，演示时自动建表）
	_, err := orm.Execute(`CREATE TABLE IF NOT EXISTS user_info (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_by TEXT, updated_by TEXT,
		create_time DATETIME, update_time DATETIME,
		group_id INTEGER, username TEXT, pass_md5 TEXT,
		roles TEXT, description TEXT, avatar TEXT)`)
	if err != nil {
		log.Errorf("create table failed: %v", err)
		return
	}
	_, err = orm.Execute(`DELETE FROM user_info`)
	if err != nil {
		log.Errorf("clean table failed: %v", err)
		return
	}
	mp := orm.NewMapper("UserInfoModelMapper").(UserInfoModelMapper)
	_, err = mp.Insert(UserInfoModel{
		CreatedBy:   "sqlite",
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
		Username:    "demo_user",
		Roles:       "admin",
		Description: "sqlite demo",
	})
	if err != nil {
		log.Errorf("insert failed: %v", err)
		return
	}
	rs, err := mp.SelectAll()
	if err != nil {
		log.Errorf("select failed: %v", err)
		return
	}
	for _, row := range rs {
		log.Infof("row: %v", types.ToJson(row))
	}
}
