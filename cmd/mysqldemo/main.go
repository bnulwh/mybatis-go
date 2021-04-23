package main

import (
	log "github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/logger"
	"github.com/bnulwh/mybatis-go/mapper"
	"github.com/bnulwh/mybatis-go/orm"
	"github.com/bnulwh/mybatis-go/types"
	_ "github.com/go-sql-driver/mysql"
)

func init() {
	logger.ConfigLocalFileSystemLogger("/var/log","mysqldemo")
	orm.Initialize("application-mysql.properties")

}
func main() {
	defer orm.Close()
	mp := mapper.GetUserInfoModelMapper()
	//var rb sql.RawBytes
	//var rt mysql.NullTime
	//
	rs, err := mp.SelectAll()
	if err != nil {
		log.Errorf("select failed: %v", err)
	} else {
		for _, row := range rs {
			log.Infof("row: %v", types.ToJson(row))
		}
	}
	item, err := mp.SelectByPrimaryKey(1)
	if err == nil {
		log.Infof("item: %v", types.ToJson(item))
	}
}
