package types

import (
	"testing"

	"github.com/bnulwh/mybatis-go/types/sqlfragment"
)

// P3-3：静态 SQL 生成缓存基准
func BenchmarkGenerateSQL_NoParam(b *testing.B) {
	items, _ := sqlfragment.ParseFragments([]xmlElement{
		{ElementType: xmlTextElem, Val: "select * from t_user"},
		{ElementType: xmlTextElem, Val: " where deleted = 0"},
	}, nil)
	fn := &SqlFunction{
		Id:    "selectAll",
		Type:  SelectFunction,
		Items: items,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := fn.GenerateSQL(); err != nil {
			b.Fatal(err)
		}
	}
}
