package sqlfragment

import (
	"regexp"
	"strings"
)

// sqlTrim 对应 MyBatis <trim> 标签：通用前后缀容器。
// 子片段全部为空时不输出任何内容；否则先按 PrefixOverrides/SuffixOverrides
// 剥离首尾（"|" 分隔、忽略大小写、仅剥离一次），再加 Prefix/Suffix。
type sqlTrim struct {
	containerBase // 嵌入 containerBase 即得 Node 全部接口方法，本类型只提供 wrapBody
	Prefix        string
	Suffix        string
	rePrefix      *regexp.Regexp
	reSuffix      *regexp.Regexp
}

// wrapBody 容器包装函数：① 剥头 ② 剥尾 ③ TrimSpace ④ 加 Prefix/Suffix（内容非空时）。
func (t *sqlTrim) wrapBody(body string) string {
	if t.rePrefix != nil {
		body = t.rePrefix.ReplaceAllString(body, "")
	}
	if t.reSuffix != nil {
		body = t.reSuffix.ReplaceAllString(body, "")
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	var sb strings.Builder
	if t.Prefix != "" {
		sb.WriteString(t.Prefix)
		sb.WriteString(" ")
	}
	sb.WriteString(body)
	if t.Suffix != "" {
		sb.WriteString(" ")
		sb.WriteString(t.Suffix)
	}
	return sb.String()
}

// buildOverrideRegex 构造首/尾剥离正则：属性值按 "|" 分隔（逐 token 转义、忽略大小写、
// token 去空白），前缀形 `(?i)^\s*(?:t1|t2)(?:\s+|$)`、后缀形 `(?i)\s*(?:t1|t2)\s*$`。
// 锚定正则至多替换一次（与 MyBatis 仅剥离一次的语义一致）；无有效 token 返回 nil。
// 后缀形不要求覆盖符前有空白，`suffixOverrides=","` 可剥离 "?," 形式的尾部逗号。
func buildOverrideRegex(overrides string, leading bool) *regexp.Regexp {
	var tokens []string
	for _, part := range strings.Split(overrides, "|") {
		if p := strings.TrimSpace(part); p != "" {
			tokens = append(tokens, regexp.QuoteMeta(p))
		}
	}
	if len(tokens) == 0 {
		return nil
	}
	pattern := `(?i)(?:` + strings.Join(tokens, "|") + `)`
	if leading {
		return regexp.MustCompile(`^\s*` + pattern + `(?:\s+|$)`)
	}
	return regexp.MustCompile(`\s*` + pattern + `\s*$`)
}

// newTrimNode 解析 <trim> 标签：子元素经 ParseFragments 递归收集，
// prefix/suffix/prefixOverrides/suffixOverrides 属性在解析期一次编译。
func newTrimNode(node XmlNode, sns map[string]*SqlElement) (Node, error) {
	sts, err := ParseFragments(node.Elements, sns)
	if err != nil {
		return nil, err
	}
	t := &sqlTrim{
		Prefix:   strings.TrimSpace(node.Attrs["prefix"]),
		Suffix:   strings.TrimSpace(node.Attrs["suffix"]),
		rePrefix: buildOverrideRegex(node.Attrs["prefixOverrides"], true),
		reSuffix: buildOverrideRegex(node.Attrs["suffixOverrides"], false),
	}
	t.Sql = sts
	t.Wrap = t.wrapBody
	return t, nil
}

func init() {
	RegisterNodeParser("trim", newTrimNode)
}
