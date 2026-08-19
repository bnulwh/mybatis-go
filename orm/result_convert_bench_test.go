package orm

import (
	"testing"
	"time"

	"github.com/bnulwh/mybatis-go/types"
)

// P3-3：结果集转换基准（100 行 -> SqliteTestModel 切片）
func BenchmarkConvert2Results(b *testing.B) {
	RegisterModel(new(SqliteTestModel))
	rmp := &types.ResultMap{
		TypeName: "SqliteTestModel",
		ColumnMap: map[string]*types.ResultItem{
			"id":          {Column: "id", Property: "Id"},
			"name":        {Column: "name", Property: "Name"},
			"create_time": {Column: "create_time", Property: "CreateTime"},
		},
	}
	tm := time.Now()
	rows := make([]map[string]interface{}, 100)
	for i := range rows {
		rows[i] = map[string]interface{}{
			"id":          int64(i),
			"name":        "benchmark",
			"create_time": tm,
		}
	}
	resInfo := types.SqlResult{ResultM: rmp}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = convert2Results(rows, resInfo)
	}
}
