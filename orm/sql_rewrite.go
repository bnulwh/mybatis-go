package orm

import (
	"strings"

	"github.com/bnulwh/mybatis-go/orm/dialector"
)

// countTailKeywords 派生 COUNT SQL 时从尾部剥离的顶层子句（B1）：
// 括号深度 0 之外（子查询/窗口函数内）的同名关键字不受影响。
var countTailKeywords = []string{
	"order by", "limit", "offset", "returning",
	"for update", "for share", "for no key update", "for key share",
}

func applyPagination(query string, limit, offset int) string {
	return dialector.ApplyPagination(query, limit, offset)
}

// buildCountSQL 由分页查询派生 COUNT 查询（B1 修复）：
//   - 只剥离「顶层」（括号深度 0）的尾部子句（ORDER BY/LIMIT/OFFSET/RETURNING/FOR UPDATE 等），
//     子查询内层的 ORDER BY/LIMIT 原样保留（如 ROW_NUMBER() OVER (ORDER BY ...)、
//     `... (SELECT ... LIMIT 1) AS x`）；
//   - SELECT 列表替换为 COUNT(*) 时同样按顶层 FROM 定位，SELECT 列表里的
//     标量子查询（内含 FROM/LIMIT）不再干扰替换边界。
func buildCountSQL(query string) string {
	q := strings.TrimRight(query, " \t\n\r;")
	q = dialector.StripTopLevelTailClauses(q, countTailKeywords...)
	selEnd := dialector.TopLevelKeywordEnd(q, "select")
	fromPos := dialector.TopLevelClausePos(q, "from")
	if selEnd < 0 || fromPos < selEnd {
		return q
	}
	wsStart := fromPos
	for wsStart > selEnd && dialector.IsSQLSpace(q[wsStart-1]) {
		wsStart--
	}
	return q[:selEnd] + "COUNT(*)" + q[wsStart:]
}
