package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
)

func init() {
	logrus.ConfigLocalFileSystemLogger("logs", "xml2go")
	log.SetLogger(logrus.StandardLogger())
}

// xml2go：XML Mapper 逆向代码生成 —— 从既有 MyBatis / MyBatis-Plus XML 生成
// Go 模型（<output>/models/，db tag 元数据 + TableName）与 Mapper 代理结构体
// （<output>/mapper/，orm.BaseMapper + 函数字段 + init 注册 + 单例 getter）。
//
// 与 sqlc 的分工：sqlc 面向纯静态 select 生成类型安全 Querier 函数；
// xml2go 面向全量语句（含动态 SQL / MP 内置 CRUD）生成反射代理 Mapper，
// 二者可共存于同一工程。
func main() {
	var mpDir, outDir, prefix string
	var skipMP, verbose bool
	flag.StringVar(&mpDir, "m", "resources/mapper", "sql mapper file directory, default: resources/mapper")
	flag.StringVar(&outDir, "d", "gen", "output directory (contains models/ and mapper/ subdirs), default: gen")
	flag.StringVar(&prefix, "p", "", "module import path prefix of generated packages, e.g. github.com/xxx/app")
	flag.BoolVar(&skipMP, "skip-mp", false, "skip mybatis-plus builtin crud function fields")
	flag.BoolVar(&verbose, "v", false, "print skipped statements with reasons")
	flag.Parse()
	mps := types.NewSqlMappers(mpDir)
	if len(mps.Mappers) == 0 {
		fmt.Printf("no mapper xml found in %v\n", mpDir)
		os.Exit(1)
	}
	result, err := mps.GenerateXml2GoFiles(&types.Xml2GoOptions{
		OutputDir:    outDir,
		ModulePrefix: prefix,
		SkipMP:       skipMP,
	})
	if err != nil {
		fmt.Println("generate xml2go files failed:", err)
		os.Exit(1)
	}
	totalFn, skipped := 0, 0
	for _, st := range result.Stats {
		totalFn += st.Functions
		skipped += len(st.Skipped)
		file := st.MapperFile
		if file == "" {
			file = "-"
		}
		fmt.Printf("%-40v %3v functions %3v models -> %v (skipped %v)\n",
			types.GetShortName(st.Mapper), st.Functions, st.Models, file, len(st.Skipped))
		if verbose {
			for _, sk := range st.Skipped {
				fmt.Printf("    skip %-30v %v\n", sk.Id, sk.Reason)
			}
		}
	}
	fmt.Printf("total: %v model files, %v mappers, %v functions generated, %v statements skipped\n",
		len(result.ModelFiles), len(result.Stats), totalFn, skipped)
}
