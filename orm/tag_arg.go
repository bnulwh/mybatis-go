package orm

import (
	"reflect"
	"strings"
)

type TagArg struct {
	Name  string
	Index int
}

// getTagArgNames 兼容两种 tag 写法：
//   - args:name（文档格式，StructTag.Get 解析不到未加引号的值）
//   - args:"name,other"（Go 标准 tag 格式）
func getTagArgNames(tag reflect.StructTag) string {
	if v := tag.Get("args"); v != "" {
		return v
	}
	raw := string(tag)
	const prefix = "args:"
	if strings.HasPrefix(raw, prefix) {
		return strings.Trim(raw[len(prefix):], `"`)
	}
	return ""
}

func parseTagArgs(tagstr string) []TagArg {
	var tagArgs = make([]TagArg, 0)
	if len(tagstr) == 0 {
		return tagArgs
	}
	tagParams := strings.Split(tagstr, `,`)
	if len(tagParams) != 0 {
		for index, v := range tagParams {
			var tagArg = TagArg{
				Index: index,
				Name:  v,
			}
			tagArgs = append(tagArgs, tagArg)
		}
	}
	return tagArgs
}
