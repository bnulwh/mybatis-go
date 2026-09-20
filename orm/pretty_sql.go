package orm

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/bnulwh/mybatis-go/orm/dialector"
)

var prettySQLEnabled bool

func SetPrettySQL(on bool) { prettySQLEnabled = on }
func PrettySQLEnabled() bool { return prettySQLEnabled }

func formatArg(v interface{}) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case string:
		return "'" + strings.ReplaceAll(val, "'", "''") + "'"
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return fmt.Sprintf("%v", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case time.Time:
		return "'" + val.Format("2006-01-02 15:04:05.999") + "'"
	case *time.Time:
		if val == nil {
			return "NULL"
		}
		return "'" + val.Format("2006-01-02 15:04:05.999") + "'"
	case sql.NullString:
		if !val.Valid {
			return "NULL"
		}
		return "'" + strings.ReplaceAll(val.String, "'", "''") + "'"
	case sql.NullInt64:
		if !val.Valid {
			return "NULL"
		}
		return fmt.Sprintf("%v", val.Int64)
	case sql.NullFloat64:
		if !val.Valid {
			return "NULL"
		}
		return fmt.Sprintf("%v", val.Float64)
	case sql.NullBool:
		if !val.Valid {
			return "NULL"
		}
		if val.Bool {
			return "true"
		}
		return "false"
	case sql.NullTime:
		if !val.Valid {
			return "NULL"
		}
		return "'" + val.Time.Format("2006-01-02 15:04:05.999") + "'"
	case []byte:
		return "x'" + fmt.Sprintf("%x", val) + "'"
	default:
		return fmt.Sprintf("%v", val)
	}
}

func FormatSQLWithArgs(sqlStr string, args []interface{}, style dialector.PlaceholderStyle) string {
	if len(args) == 0 {
		return sqlStr
	}
	switch style {
	case dialector.PlaceholderDollar:
		return replaceNumericPlaceholders(sqlStr, args, `\$(\d+)`)
	case dialector.PlaceholderColon:
		return replaceNumericPlaceholders(sqlStr, args, `:(\d+)`)
	case dialector.PlaceholderAtP:
		return replaceNumericPlaceholders(sqlStr, args, `@p(\d+)`)
	default:
		count := 0
		var b strings.Builder
		b.Grow(len(sqlStr) + len(args)*16)
		for i := 0; i < len(sqlStr); i++ {
			if sqlStr[i] == '?' && count < len(args) {
				b.WriteString(formatArg(args[count]))
				count++
			} else {
				b.WriteByte(sqlStr[i])
			}
		}
		return b.String()
	}
}

func replaceNumericPlaceholders(sqlStr string, args []interface{}, pattern string) string {
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllStringFunc(sqlStr, func(match string) string {
		sub := re.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		idx := 0
		for _, c := range sub[1] {
			if c >= '0' && c <= '9' {
				idx = idx*10 + int(c-'0')
			}
		}
		if idx >= 1 && idx <= len(args) {
			return formatArg(args[idx-1])
		}
		return match
	})
}

var lineBreakKeywords = []string{
	"SELECT ", "FROM ", "WHERE ", "AND ", "OR ", "ORDER BY ", "GROUP BY ",
	"HAVING ", "LIMIT ", "OFFSET ", "UNION ", "UNION ALL ",
	"JOIN ", "LEFT JOIN ", "RIGHT JOIN ", "INNER JOIN ", "CROSS JOIN ", "FULL JOIN ",
	"ON ", "SET ", "VALUES ", "RETURNING ",
	"INSERT INTO ", "UPDATE ", "DELETE FROM ",
}

var subqueryKeywords = []string{"SELECT ", "UNION ", "UNION ALL "}

func PrettySQL(sqlStr string) string {
	sqlStr = strings.TrimSpace(sqlStr)
	if sqlStr == "" {
		return ""
	}
	upper := strings.ToUpper(sqlStr)
	var b strings.Builder
	b.Grow(len(sqlStr) * 2)
	indent := 0
	pos := 0
	for pos < len(upper) {
		matched := false
		for _, kw := range lineBreakKeywords {
			kwUpper := kw
			if pos+len(kwUpper) > len(upper) || upper[pos:pos+len(kwUpper)] != kwUpper {
				continue
			}
			chBefore := byte(0)
			if pos > 0 {
				chBefore = sqlStr[pos-1]
			}
			if chBefore != 0 && !isSQLBreak(chBefore) {
				continue
			}
			if !strings.HasSuffix(kw, " ") {
				chAfter := byte(0)
				if pos+len(kw) < len(sqlStr) {
					chAfter = sqlStr[pos+len(kw)]
				}
				if chAfter != 0 && (unicode.IsLetter(rune(chAfter)) || unicode.IsDigit(rune(chAfter)) || chAfter == '_') {
					continue
				}
			}
			if kw == "AND " || kw == "OR " {
				indent = max(indent, 1)
			}
			for _, sq := range subqueryKeywords {
				if kw == sq && pos > 0 {
					indent++
					break
				}
			}
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			for i := 0; i < indent; i++ {
				b.WriteString("  ")
			}
			kwOrig := sqlStr[pos : pos+len(kw)]
			if kw == "AND " || kw == "OR " {
				kwOrig = strings.TrimRight(kwOrig, " ") + " "
			}
			b.WriteString(kwOrig)
			pos += len(kw)
			matched = true
			break
		}
		if !matched {
			b.WriteByte(sqlStr[pos])
			pos++
		}
	}
	return b.String()
}

func isSQLBreak(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '(' || ch == ','
}

func getPlaceholderStyle() dialector.PlaceholderStyle {
	if gDbConn != nil && gDbConn.Dialector != nil {
		return gDbConn.Dialector.PlaceholderStyle()
	}
	return dialector.PlaceholderQuestion
}

func formatSQLForLog(sqlStr string, args []interface{}) string {
	if !PrettySQLEnabled() {
		return sqlStr
	}
	withArgs := FormatSQLWithArgs(sqlStr, args, getPlaceholderStyle())
	return PrettySQL(withArgs)
}
