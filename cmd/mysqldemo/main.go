package main

import (
	log "github.com/astaxie/beego/logs"
	"github.com/bnulwh/mybatis-go/logger"
	"github.com/bnulwh/mybatis-go/mapper"
	"github.com/bnulwh/mybatis-go/orm"
	"github.com/bnulwh/mybatis-go/types"
	_ "github.com/go-sql-driver/mysql"
)


func init() {
	logger.Initialize("mysqldemo.log")
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
		log.Error("select failed: %v", err)
	} else {
		for _, row := range rs {
			log.Info("row: %v", types.ToJson(row))
		}
	}
}
