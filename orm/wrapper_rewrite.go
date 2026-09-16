package orm

import (
	"regexp"
	"strings"

	"github.com/bnulwh/mybatis-go/types"
)

var (
	// reWrapperTail SQL 尾部子句边界（与 buildCountSQL 同款正则风格）：
	// 匹配首个 GROUP BY/HAVING/ORDER BY/LIMIT/OFFSET/FOR UPDATE。
	// 已知限制：子查询内含这些关键字时无法区分层级（同 buildCountSQL），文档明示。
	reWrapperTail = regexp.MustCompile(`(?is)\s+(GROUP\s+BY|HAVING|ORDER\s+BY|LIMIT|OFFSET|FOR\s+UPDATE)\s`)
	// reWrapperWhere core 部分是否已含 WHERE。
	reWrapperWhere = regexp.MustCompile(`(?is)\sWHERE\s`)
)

// splitWrapperTail 切分 SQL：core（select/from/where）与 tail（GROUP BY 起的尾部子句）。
func splitWrapperTail(sqlStr string) (core, tail string) {
	if loc := reWrapperTail.FindStringIndex(sqlStr); loc != nil {
		return sqlStr[:loc[0]], sqlStr[loc[0]:]
	}
	return sqlStr, ""
}

// applyQueryWrapper 把 Wrapper 条件注入 SQL（P0-1 自动注入算法）：
//  1. 切分 core / tail（尾部 GROUP BY/HAVING/ORDER BY/LIMIT/OFFSET/FOR UPDATE）；
//  2. core 已含 WHERE → 追加 " AND (seg)"，否则追加 " WHERE (seg)"；
//  3. Wrapper 自身的 GROUP BY/HAVING/ORDER BY 追加在原 tail 子句之后，last 恒置末尾
//
// 空 Wrapper（无条件且无附加子句）原样返回。
func applyQueryWrapper(sqlStr string, w *QueryWrapper) string {
	if w == nil {
		return sqlStr
	}
	cond := w.conditionSQL()
	clause := w.clauseSQL()
	if cond == "" && clause == "" && w.lastSQL == "" {
		return sqlStr
	}
	core, tail := splitWrapperTail(sqlStr)
	if cond != "" {
		if reWrapperWhere.MatchString(core) {
			core = core + " AND (" + cond + ")"
		} else {
			core = core + " WHERE (" + cond + ")"
		}
	}
	var parts []string
	if core = strings.TrimSpace(core); core != "" {
		parts = append(parts, core)
	}
	if tail = strings.TrimSpace(tail); tail != "" {
		parts = append(parts, tail)
	}
	if clause != "" {
		parts = append(parts, clause)
	}
	out := strings.Join(parts, " ")
	if w.lastSQL != "" {
		out = out + " " + w.lastSQL
	}
	return strings.TrimSpace(out)
}

// hasNamedSlot 语句必参占位符（Slots）中是否含指定名（#{} / ${} 收集结果）。
func hasNamedSlot(sqlFunc *types.SqlFunction, name string) bool {
	key := strings.ToLower(strings.TrimSpace(name))
	for _, s := range sqlFunc.Param.Slots {
		if strings.ToLower(strings.TrimSpace(s)) == key {
			return true
		}
	}
	return false
}

// injectWrapperArg 显式 ${ew} 模式：Wrapper 以 "ew" 键并入渲染参数。
// 无参 → 单 map 参数；末参为 map（TagArgs 路径）→ 并入；其余 → 位置追加
// （buildParamMap 按 Slots 位置绑定，适用于 wrapper 为最后一个占位符的场景）。
func injectWrapperArg(args []interface{}, w *QueryWrapper) []interface{} {
	if len(args) == 0 {
		return []interface{}{map[string]interface{}{"ew": w}}
	}
	if m, ok := args[len(args)-1].(map[string]interface{}); ok {
		m["ew"] = w
		return args
	}
	return append(args, w)
}
