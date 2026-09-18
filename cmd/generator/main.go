package main

import (
	"flag"
	"fmt"
	"github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
)

// Deprecated: generator 已被 xml2go（cmd/xml2go）全面取代，仅为兼容保留。
// xml2go 生成带 db tag 元数据的模型 + 与 orm 注册期校验对齐的 Mapper 代理，
// 支持模块路径前缀（-p）、MyBatis-Plus 内置 CRUD、跨 Mapper 模型去重，
// 且产物保证可编译；新项目请使用 xml2go。
func init() {
	logrus.ConfigLocalFileSystemLogger("logs", "generator")
	log.SetLogger(logrus.StandardLogger())
	fmt.Println("warning: generator is deprecated and superseded by xml2go (github.com/bnulwh/mybatis-go/cmd/xml2go)")
}

func main() {
	var pkg, dir, mp string
	flag.StringVar(&pkg, "p", "temp", "package name,default: temp")
	flag.StringVar(&dir, "d", "temp", "saving directory,default: temp")
	flag.StringVar(&mp, "m", "resources/mapper", "sql mapper file directory,default: resources/mapper")
	flag.Parse()
	mps := types.NewSqlMappers(mp)
	mps.GenerateFiles(dir, pkg)
}
