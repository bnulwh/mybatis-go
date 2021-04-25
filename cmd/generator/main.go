package main

import (
	"flag"
	log "github.com/bnulwh/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/types"
)

func init(){
	log.ConfigLocalFileSystemLogger("logs","generator")
}

func main(){
	var pkg,dir,mp string
	flag.StringVar(&pkg,"p","temp","package name,default: temp")
	flag.StringVar(&dir,"d","temp","saving directory,default: temp")
	flag.StringVar(&mp,"m","resources/mapper","sql mapper file directory,default: resources/mapper")
	flag.Parse()
	mps := types.NewSqlMappers(mp)
	mps.GenerateFiles(dir,pkg)
}