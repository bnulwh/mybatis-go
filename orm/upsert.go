package orm

import (
	"fmt"
	"strings"

	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/orm/dialector"
	"github.com/bnulwh/mybatis-go/types"
)

// upsertSQLProviderCallback types 包回调：根据当前数据库方言生成 upsert SQL。
// 未初始化数据库或方言不支持 upsert 时返回空串（不生成 insertOrUpdate 方法）。
func upsertSQLProviderCallback(args types.UpsertSQLArgs) string {
	if gDbConn == nil || gDbConn.Dialector == nil {
		return ""
	}
	family := gDbConn.Dialector.Family()
	return generateUpsertSQL(family, args)
}

// ensureUpsertFunctions 在数据库初始化后补生成各 Mapper 缺失的 insertOrUpdate 方法。
// XML 加载期（ensureMPBuiltinCRUD）因 gDbConn 未就绪而跳过 upsert，
// 此函数在 gDbConn 可用后遍历所有 Mapper，为有主键但缺 insertOrUpdate 的 Mapper 补生成。
func ensureUpsertFunctions() {
	if gDbConn == nil || gDbConn.Dialector == nil {
		return
	}
	family := string(gDbConn.Dialector.Family())
	if generateUpsertSQL(gDbConn.Dialector.Family(), types.UpsertSQLArgs{}) == "" {
		return
	}
	for i := range gCache.sqls.Mappers {
		gCache.sqls.Mappers[i].EnsureUpsertFunction(family)
	}
}

// upsertSQLByFamilyCallback 按 DatabaseFamily 字符串生成 upsert SQL 的回调（供 types 包调用）。
func upsertSQLByFamilyCallback(family string, args types.UpsertSQLArgs) string {
	return generateUpsertSQL(dialector.DatabaseFamily(family), args)
}

// generateUpsertSQL 根据数据库方言生成 upsert（INSERT ... ON CONFLICT / ON DUPLICATE KEY / MERGE）语句。
// 返回空串表示方言不支持 upsert。
func generateUpsertSQL(family dialector.DatabaseFamily, args types.UpsertSQLArgs) string {
	switch family {
	case dialector.FamilyPostgres, dialector.FamilySQLite:
		return generateUpsertSQLPostgres(args)
	case dialector.FamilyMySQL:
		return generateUpsertSQLMySQL(args)
	case dialector.FamilyMSSQL:
		return generateUpsertSQLMSSQL(args)
	case dialector.FamilyOracle:
		return generateUpsertSQLOracle(args)
	default:
		log.Debugf("dialect family %q does not support upsert, skip insertOrUpdate", family)
		return ""
	}
}

// generateUpsertSQLPostgres PostgreSQL / SQLite: INSERT INTO ... VALUES (...) ON CONFLICT (pk) DO UPDATE SET ...
func generateUpsertSQLPostgres(args types.UpsertSQLArgs) string {
	var cnames, cvalues, updates []string
	for i, col := range args.Columns {
		prop := args.Properties[i]
		jt := args.JdbcTypes[i]
		cnames = append(cnames, col)
		cvalues = append(cvalues, fmt.Sprintf("#{%s,jdbcType=%s}", prop, jt))
		if col != args.PkColumn {
			updates = append(updates, fmt.Sprintf("%s=EXCLUDED.%s", col, col))
		}
	}
	return fmt.Sprintf("insert into %s \n\t\t(%s) \n\t\tvalues \n\t\t(%s) \n\t\ton conflict (%s) do update set %s",
		args.Table,
		strings.Join(cnames, ",\n\t\t"),
		strings.Join(cvalues, ",\n\t\t"),
		args.PkColumn,
		strings.Join(updates, ",\n\t\t"))
}

// generateUpsertSQLMySQL MySQL: INSERT INTO ... VALUES (...) ON DUPLICATE KEY UPDATE ...
func generateUpsertSQLMySQL(args types.UpsertSQLArgs) string {
	var cnames, cvalues, updates []string
	for i, col := range args.Columns {
		prop := args.Properties[i]
		jt := args.JdbcTypes[i]
		cnames = append(cnames, col)
		cvalues = append(cvalues, fmt.Sprintf("#{%s,jdbcType=%s}", prop, jt))
		if col != args.PkColumn {
			updates = append(updates, fmt.Sprintf("%s=VALUES(%s)", col, col))
		}
	}
	return fmt.Sprintf("insert into %s \n\t\t(%s) \n\t\tvalues \n\t\t(%s) \n\t\ton duplicate key update %s",
		args.Table,
		strings.Join(cnames, ",\n\t\t"),
		strings.Join(cvalues, ",\n\t\t"),
		strings.Join(updates, ",\n\t\t"))
}

// generateUpsertSQLMSSQL MSSQL: MERGE INTO ... USING (VALUES (...)) AS source ON ... WHEN MATCHED THEN UPDATE ... WHEN NOT MATCHED THEN INSERT ...
func generateUpsertSQLMSSQL(args types.UpsertSQLArgs) string {
	var cnames, cvalues, updates, insertCols, insertVals []string
	for i, col := range args.Columns {
		prop := args.Properties[i]
		jt := args.JdbcTypes[i]
		cnames = append(cnames, col)
		cvalues = append(cvalues, fmt.Sprintf("#{%s,jdbcType=%s}", prop, jt))
		insertCols = append(insertCols, col)
		insertVals = append(insertVals, fmt.Sprintf("source.%s", col))
		if col != args.PkColumn {
			updates = append(updates, fmt.Sprintf("%s=source.%s", col, col))
		}
	}
	return fmt.Sprintf("merge into %s as target \n\t\tusing (values (%s)) as source (%s) \n\t\ton target.%s = source.%s \n\t\twhen matched then update set %s \n\t\twhen not matched then insert (%s) values (%s)",
		args.Table,
		strings.Join(cvalues, ",\n\t\t"),
		strings.Join(cnames, ","),
		args.PkColumn, args.PkColumn,
		strings.Join(updates, ",\n\t\t"),
		strings.Join(insertCols, ","),
		strings.Join(insertVals, ","))
}

// generateUpsertSQLOracle Oracle / DM: MERGE INTO ... USING (SELECT ... FROM DUAL) AS source ON ...
func generateUpsertSQLOracle(args types.UpsertSQLArgs) string {
	var cnames, cvalues, updates, insertCols, insertVals []string
	for i, col := range args.Columns {
		prop := args.Properties[i]
		jt := args.JdbcTypes[i]
		cnames = append(cnames, col)
		cvalues = append(cvalues, fmt.Sprintf("#{%s,jdbcType=%s} as %s", prop, jt, col))
		insertCols = append(insertCols, col)
		insertVals = append(insertVals, fmt.Sprintf("source.%s", col))
		if col != args.PkColumn {
			updates = append(updates, fmt.Sprintf("%s=source.%s", col, col))
		}
	}
	return fmt.Sprintf("merge into %s target \n\t\tusing (select %s from dual) source \n\t\ton (target.%s = source.%s) \n\t\twhen matched then update set %s \n\t\twhen not matched then insert (%s) values (%s)",
		args.Table,
		strings.Join(cvalues, ",\n\t\t"),
		args.PkColumn, args.PkColumn,
		strings.Join(updates, ",\n\t\t"),
		strings.Join(insertCols, ","),
		strings.Join(insertVals, ","))
}
