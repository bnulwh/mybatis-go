package orm

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reSelectFrom = regexp.MustCompile(`(?is)^(\s*SELECT\s+)(.+?)(\s+FROM\s)`)
	reOrderBy    = regexp.MustCompile(`(?is)\s+ORDER\s+BY\s+.+$`)
	reLimit      = regexp.MustCompile(`(?is)\s+LIMIT\s+\S+`)
	reOffset     = regexp.MustCompile(`(?is)\s+OFFSET\s+\S+`)
	reReturning  = regexp.MustCompile(`(?is)\s+RETURNING\s+.+$`)
	reForUpdate  = regexp.MustCompile(`(?is)\s+FOR\s+UPDATE\s*$`)
)

func applyPagination(query string, limit, offset int) string {
	if limit <= 0 {
		return query
	}
	q := strings.TrimRight(query, " \t\n\r;")
	q = reLimit.ReplaceAllString(q, "")
	q = reOffset.ReplaceAllString(q, "")
	if offset > 0 {
		return fmt.Sprintf("%s LIMIT %d OFFSET %d", q, limit, offset)
	}
	return fmt.Sprintf("%s LIMIT %d", q, limit)
}

func buildCountSQL(query string) string {
	q := strings.TrimRight(query, " \t\n\r;")
	q = reForUpdate.ReplaceAllString(q, "")
	q = reReturning.ReplaceAllString(q, "")
	q = reLimit.ReplaceAllString(q, "")
	q = reOffset.ReplaceAllString(q, "")
	q = reOrderBy.ReplaceAllString(q, "")
	m := reSelectFrom.FindStringSubmatch(q)
	if m != nil {
		return m[1] + "COUNT(*)" + m[3] + q[len(m[0]):]
	}
	return q
}
