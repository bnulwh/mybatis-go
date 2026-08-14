package types

import (
	"sync"
	"testing"
)

// P2-5：无参 SQL 生成结果缓存回归测试。
// 无参 SQL 的拼接结果静态不变，应只生成一次并在后续调用中复用。

func Test_GenerateSqlWithoutParamCached(t *testing.T) {
	fn := &SqlFunction{
		Id:   "selectAll",
		Type: SelectFunction,
		Items: []*sqlFragment{
			{Type: simpleSqlFragment, Sql: parseSimpleSqlFromText("select * from t_user")},
			{Type: simpleSqlFragment, Sql: parseSimpleSqlFromText(" where deleted = 0")},
		},
	}
	sql1, args1, err := fn.GenerateSQL()
	if err != nil {
		t.Errorf("GenerateSQL failed: %v", err)
		return
	}
	if len(args1) != 0 {
		t.Errorf("no-param sql should have no args, got %v", args1)
	}
	// 多次调用结果必须一致
	sql2, _, err := fn.GenerateSQL()
	if err != nil || sql1 != sql2 {
		t.Errorf("no-param sql should be stable: %q vs %q, err=%v", sql1, sql2, err)
	}
	// 缓存已填充
	if fn.noParamSQL != sql1 {
		t.Errorf("cached sql mismatch: %q", fn.noParamSQL)
	}
	// 每次 GenerateSQL 仍计入统计（GenerateSQL 包装层仍执行）
	if fn.GenerateCount != 2 {
		t.Errorf("GenerateCount should be 2, got %d", fn.GenerateCount)
	}
}

// 并发调用无参 SQL 不应出现重复生成或数据竞争
func Test_GenerateSqlWithoutParamConcurrent(t *testing.T) {
	fn := &SqlFunction{
		Id:   "selectAll",
		Type: SelectFunction,
		Items: []*sqlFragment{
			{Type: simpleSqlFragment, Sql: parseSimpleSqlFromText("select * from t_user")},
		},
	}
	first, _, err := fn.GenerateSQL()
	if err != nil {
		t.Errorf("GenerateSQL failed: %v", err)
		return
	}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, _, e := fn.GenerateSQL()
			if e != nil || s != first {
				t.Errorf("concurrent GenerateSQL mismatch: %q err=%v", s, e)
			}
		}()
	}
	wg.Wait()
}
