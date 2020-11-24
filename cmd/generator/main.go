package main

import (
	"flag"
	"github.com/bnulwh/mybatis-go/logger"
	"github.com/bnulwh/mybatis-go/types"
)

func init(){
	logger.Initialize("generator.log")
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