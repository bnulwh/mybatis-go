package orm

import "strings"

type TagArg struct {
	Name  string
	Index int
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
