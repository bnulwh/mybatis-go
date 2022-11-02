package main

import (
	log "github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/mapper"
	"github.com/bnulwh/mybatis-go/orm"
	"github.com/bnulwh/mybatis-go/types"
	_ "github.com/go-sql-driver/mysql"
	"sync"
)

func init() {
	log.ConfigLocalFileSystemLogger("./logs", "mysqldemo")
	err := orm.Initialize("application-mysql.properties")
	if err != nil {
		panic(err)
	}
}
func main() {
	wg := sync.WaitGroup{}
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
	for j := 0; j < 10; j++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				item, err := mp.SelectByPrimaryKey(1)
				if err == nil {
					log.Infof("item: %v", types.ToJson(item))
				}
			}
		}(j)

	}
	wg.Wait()
}
