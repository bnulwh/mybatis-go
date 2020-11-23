package orm

import (
	"errors"
	"fmt"
	log "github.com/astaxie/beego/logs"
	"github.com/bnulwh/mybatis-go/types"
	"io/ioutil"
	"reflect"
	"os"
	"strings"
	"regexp"
	"strconv"
)

type DatabaseConfig struct{
	Host string
	Port int64
	Username string 
	Password string 
	DbName string
}

type MyBatisConfig struct{
	DbConfig DatabaseConfig
	MapperLocations string
	TypeAliasPackage string
	MaxRows int64
}

func NewConfig(filename string) *MyBatisConfig{
	cm := LoadSettings(filename)
	dbc := parseDatabaseConfig(cm)
	ml :=cm["mybatis.mapper-locations"]
	return &MyBatisConfig{
		DbConfig: *dbc,
		MapperLocations: ml,
	}
}

func (in *DatabaseConfig) GenerateConnString() string{
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			in.Host,in.Port,in.Username,in.Password,in.DbName)
}

func parseDatabaseConfig(m map[string]string) *DatabaseConfig{
	h,P,d,err := parseAddr(m)
	if err != nil{
		log.Error("parse postgres addr failed: %v",err)
		panic(err)
	}
	u,ok := m["spring.datasource.username"]
	if !ok{
		log.Error("get database username failed.")
		panic("get database username failed.")
	}
	p,ok := m["spring.datasource.password"]
	if !ok{
		log.Error("get database password failed.")
		panic("get database password failed.")
	}
	return &DatabaseConfig{
		Host: h,
		Port: P,
		Username: u,
		Password: p,
		DbName: d,
	}
}

func parseAddr(m map[string]string) (string,int64,string,error) {
	val,ok := m["spring.datasource.url"]
	if !ok{
		return "",0,"",errors.New("not found key spring.datasource.url")
	}
	re := regexp.MustCompile(`jdbc:postgresql://([\w\\.]+):([\d]+)/([\w_-]+)`)
	matched := re.FindStringSubmatch(val)
	if len(matched) <4{
		return "",0,"",errors.New("unsupport formate of spring.datasource.url")
	}
	i,_ := strconv.Atoi(matched[2])
	return matched[1],int64(i),matched[3],nil
}