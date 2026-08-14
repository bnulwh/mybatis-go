package types

import (
	"sync"
	"testing"
	"time"
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

// P2-5：UpdateUsage 统计（Min/Max/TotalDuration）与并发安全回归测试
func Test_UpdateUsageStats(t *testing.T) {
	fn := &SqlFunction{}
	// 首调 50ms：Min/Max 均为首值
	fn.UpdateUsage(time.Now().Add(-50*time.Millisecond), true)
	if fn.TotalUsage != 1 || fn.FailedUsage != 0 {
		t.Errorf("usage counters wrong: %d/%d", fn.TotalUsage, fn.FailedUsage)
	}
	if fn.MinDuration < 40 || fn.MaxDuration < 40 {
		t.Errorf("first call min/max should be ~50ms, got %d/%d", fn.MinDuration, fn.MaxDuration)
	}
	// 第二次 20ms：Min 下降，Max 保持
	fn.UpdateUsage(time.Now().Add(-20*time.Millisecond), true)
	if fn.MinDuration > 30 {
		t.Errorf("min should drop to ~20ms, got %d", fn.MinDuration)
	}
	if fn.MaxDuration < 40 {
		t.Errorf("max should stay ~50ms, got %d", fn.MaxDuration)
	}
	// 第三次 100ms：Max 上升，Min 保持
	fn.UpdateUsage(time.Now().Add(-100*time.Millisecond), false)
	if fn.MaxDuration < 90 {
		t.Errorf("max should rise to ~100ms, got %d", fn.MaxDuration)
	}
	if fn.FailedUsage != 1 {
		t.Errorf("failed counter wrong: %d", fn.FailedUsage)
	}
	if fn.TotalDuration < 160 {
		t.Errorf("total duration should be ~170ms, got %d", fn.TotalDuration)
	}
}

// 并发调用 UpdateUsage 不应互相覆盖 Max/Min（CAS 语义）
func Test_UpdateUsageConcurrent(t *testing.T) {
	fn := &SqlFunction{}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(ms int) {
			defer wg.Done()
			fn.UpdateUsage(time.Now().Add(-time.Duration(ms)*time.Millisecond), true)
		}((i % 5) * 10) // 0/10/20/30/40ms
	}
	wg.Wait()
	if fn.TotalUsage != 20 {
		t.Errorf("total usage should be 20, got %d", fn.TotalUsage)
	}
	// 最大应接近 40ms，最小应接近 0ms
	if fn.MaxDuration < 35 {
		t.Errorf("max should be ~40ms, got %d", fn.MaxDuration)
	}
	if fn.MinDuration > 15 {
		t.Errorf("min should be <= ~10ms, got %d", fn.MinDuration)
	}
}
