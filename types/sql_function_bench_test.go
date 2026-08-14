package types

import "testing"

// P3-3：静态 SQL 生成缓存基准
func BenchmarkGenerateSQL_NoParam(b *testing.B) {
	fn := &SqlFunction{
		Id:   "selectAll",
		Type: SelectFunction,
		Items: []*sqlFragment{
			{Type: simpleSqlFragment, Sql: parseSimpleSqlFromText("select * from t_user")},
			{Type: simpleSqlFragment, Sql: parseSimpleSqlFromText(" where deleted = 0")},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := fn.GenerateSQL(); err != nil {
			b.Fatal(err)
		}
	}
}
