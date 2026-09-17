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
	logrus.ConfigLocalFileSystemLogger("logs", "sqlc")
	log.SetLogger(logrus.StandardLogger())
}

// sqlc：XML 存量静态 select 抽取 → 类型安全 Querier 代码生成（可行性报告 §3.4 方案 A）。
// 扫描 Mapper 目录，对每个纯静态 select（无动态标签、无 ${} 注入）生成
// 「接口 + 实现」普通 Go 函数，经 (db *orm.DB) QueryToContext 执行；
// 动态 / 无法静态推断的语句跳过并以 -v 输出原因。
func main() {
	var pkg, dir, mp string
	var verbose bool
	flag.StringVar(&pkg, "p", "querier", "package name of generated files, default: querier")
	flag.StringVar(&dir, "d", "querier", "saving directory, default: querier")
	flag.StringVar(&mp, "m", "resources/mapper", "sql mapper file directory, default: resources/mapper")
	flag.BoolVar(&verbose, "v", false, "print skipped statements with reasons")
	flag.Parse()
	mps := types.NewSqlMappers(mp)
	if len(mps.Mappers) == 0 {
		fmt.Printf("no mapper xml found in %v\n", mp)
		os.Exit(1)
	}
	stats, err := mps.GenerateQuerierFiles(dir, pkg)
	if err != nil {
		fmt.Println("generate querier files failed:", err)
		os.Exit(1)
	}
	total, skipped, files := 0, 0, 0
	for _, st := range stats {
		total += st.Functions
		skipped += len(st.Skipped)
		if st.File != "" {
			files++
		}
		fmt.Printf("%-40v %3v functions %3v models -> %v (skipped %v)\n",
			types.GetShortName(st.Mapper), st.Functions, st.Models, st.File, len(st.Skipped))
		if verbose {
			for _, sk := range st.Skipped {
				fmt.Printf("    skip %-30v %v\n", sk.Id, sk.Reason)
			}
		}
	}
	fmt.Printf("total: %v files, %v functions generated, %v statements skipped\n", files, total, skipped)
}
