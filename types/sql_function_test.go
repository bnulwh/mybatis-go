package types

import (
	"strings"
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

// 并发调用 UpdateUsage 不应互相覆盖 Max/Min（CAS 语义，P2-5）。
// 断言改为与观测到的样本极值比较：无论调度如何延迟，Max/Min 都必须是全体样本的极值
// （旧断言依赖绝对毫秒阈值，goroutine 抢占会抬高/压低测量值而偶发失败）。
func Test_UpdateUsageConcurrent(t *testing.T) {
	fn := &SqlFunction{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var observed []int64
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(ms int) {
			defer wg.Done()
			start := time.Now().Add(-time.Duration(ms) * time.Millisecond)
			fn.UpdateUsage(start, true)
			mu.Lock()
			observed = append(observed, time.Since(start).Milliseconds())
			mu.Unlock()
		}((i % 5) * 10) // 0/10/20/30/40ms
	}
	wg.Wait()
	if fn.TotalUsage != 20 {
		t.Errorf("total usage should be 20, got %d", fn.TotalUsage)
	}
	if fn.FailedUsage != 0 {
		t.Errorf("failed usage should be 0, got %d", fn.FailedUsage)
	}
	expMax, expMin := int64(0), int64(1<<62-1)
	for _, d := range observed {
		if d > expMax {
			expMax = d
		}
		if d < expMin {
			expMin = d
		}
	}
	// UpdateUsage 内部的 time.Since 早于外部观测，二者相差不超过毫秒截断 + 调度窗口，容差 5ms
	if diff := fn.MaxDuration - expMax; diff > 5 || diff < -5 {
		t.Errorf("max duration mismatch: got %d, observed max %d", fn.MaxDuration, expMax)
	}
	if diff := fn.MinDuration - expMin; diff > 5 || diff < -5 {
		t.Errorf("min duration mismatch: got %d, observed min %d", fn.MinDuration, expMin)
	}
	if fn.MaxDuration < fn.MinDuration {
		t.Errorf("max %d < min %d", fn.MaxDuration, fn.MinDuration)
	}
}

// P2-5 跟进：真实最小值 0ms 不应被「未初始化」哨兵误覆盖。
// 旧实现以 MinDuration==0 兼作未初始化标记，真实 0ms 最小值会被后续较大值覆盖。
func Test_updateMinDuration_ZeroCollision(t *testing.T) {
	var target int64
	var init int32
	// 首调为 0ms（真实测量值），随后 20ms/5ms 都不应覆盖
	updateMinDuration(&target, &init, 0)
	if target != 0 {
		t.Errorf("first call min should be 0, got %d", target)
	}
	updateMinDuration(&target, &init, 20)
	if target != 0 {
		t.Errorf("min 0 overwritten by 20, got %d", target)
	}
	updateMinDuration(&target, &init, 5)
	if target != 0 {
		t.Errorf("min 0 overwritten by 5, got %d", target)
	}
	// 首次为较大值时，后续更小值仍能更新
	var t2 int64
	var i2 int32
	updateMinDuration(&t2, &i2, 30)
	updateMinDuration(&t2, &i2, 10)
	if t2 != 10 {
		t.Errorf("min should drop to 10, got %d", t2)
	}
	updateMinDuration(&t2, &i2, 10)
	if t2 != 10 {
		t.Errorf("min should stay 10, got %d", t2)
	}
}

// Test_parseSqlFunctionFromXmlNode_GeneratedKeys useGeneratedKeys/keyProperty 被解析并暴露（S-11）。
func Test_parseSqlFunctionFromXmlNode_GeneratedKeys(t *testing.T) {
	node := xmlNode{
		Id:   "insertJob",
		Name: "insert",
		Attrs: map[string]string{
			"parameterType":    "SysJob",
			"useGeneratedKeys": "true",
			"keyProperty":      "jobId",
			"keyColumn":        "job_id",
		},
	}
	fn := parseSqlFunctionFromXmlNode(node, nil, nil, "SysJobMapper")
	if fn == nil {
		t.Error("parseSqlFunctionFromXmlNode returned nil")
		return
	}
	if !fn.UseGeneratedKeys {
		t.Error("UseGeneratedKeys should be true")
	}
	if fn.KeyProperty != "jobId" {
		t.Error("KeyProperty =", fn.KeyProperty, "want jobId")
	}
	if fn.KeyColumn != "job_id" {
		t.Error("KeyColumn =", fn.KeyColumn, "want job_id")
	}
	// 未设置时默认关闭
	node2 := xmlNode{Id: "insertX", Name: "insert", Attrs: map[string]string{"parameterType": "SysJob"}}
	fn2 := parseSqlFunctionFromXmlNode(node2, nil, nil, "SysJobMapper")
	if fn2.UseGeneratedKeys {
		t.Error("UseGeneratedKeys should default false")
	}
	if fn2.KeyProperty != "" || fn2.KeyColumn != "" {
		t.Error("KeyProperty/KeyColumn should default empty")
	}
}

// Test_GeneratedKeys_Samples 使用 samples/（RuoYi Mapper）真实文件回归：
// 5 个 useGeneratedKeys insert 的 keyProperty 均被解析（S-11）。
func Test_GeneratedKeys_Samples(t *testing.T) {
	mps := NewSqlMappers("../samples")
	if mps == nil || len(mps.Mappers) == 0 {
		t.Error("load samples failed")
		return
	}
	expect := map[string]string{
		"insertJob":            "jobId",
		"insertPost":           "postId",
		"insertRole":           "roleId",
		"insertGenTable":       "tableId",
		"insertGenTableColumn": "columnId",
	}
	found := 0
	for _, m := range mps.Mappers {
		for id, keyProp := range expect {
			fn := m.NamedFunctions[id]
			if fn == nil {
				continue
			}
			found++
			if !fn.UseGeneratedKeys {
				t.Error(id, "UseGeneratedKeys should be true")
			}
			if fn.KeyProperty != keyProp {
				t.Error(id, "KeyProperty =", fn.KeyProperty, "want", keyProp)
			}
		}
	}
	if found != len(expect) {
		t.Error("expected", len(expect), "generated-keys functions, found", found)
	}
}

// Test_ContainsForEach 递归检测片段树中的 <foreach>（含 if/include/where 嵌套）。
func Test_ContainsForEach(t *testing.T) {
	if containsForEach(nil) {
		t.Error("nil items should not contain foreach")
	}
	if containsForEach([]*sqlFragment{{Type: simpleSqlFragment}}) {
		t.Error("plain sql should not contain foreach")
	}
	// 直接 foreach
	if !containsForEach([]*sqlFragment{{Type: forLoopSqlFragment, ForLoop: &sqlForLoop{}}}) {
		t.Error("direct foreach not detected")
	}
	// if 嵌套 foreach
	if !containsForEach([]*sqlFragment{{
		Type:   ifTestSqlFragment,
		IfTest: &sqlIfTest{Sql: []*sqlFragment{{Type: forLoopSqlFragment, ForLoop: &sqlForLoop{}}}},
	}}) {
		t.Error("foreach inside if not detected")
	}
	// include 嵌套 foreach
	if !containsForEach([]*sqlFragment{{
		Type:    includeSqlFragment,
		Include: &sqlInclude{Fragments: []*sqlFragment{{Type: forLoopSqlFragment, ForLoop: &sqlForLoop{}}}},
	}}) {
		t.Error("foreach inside include not detected")
	}
	// where 嵌套 foreach
	if !containsForEach([]*sqlFragment{{
		Type:  whereSqlFragment,
		Where: &sqlWhere{Sql: []*sqlFragment{{Type: forLoopSqlFragment, ForLoop: &sqlForLoop{}}}},
	}}) {
		t.Error("foreach inside where not detected")
	}
	// 无 foreach
	if containsForEach([]*sqlFragment{{
		Type:   ifTestSqlFragment,
		IfTest: &sqlIfTest{Sql: []*sqlFragment{{Type: simpleSqlFragment, Sql: &simpleSql{Sql: "x"}}}},
	}}) {
		t.Error("nested plain sql should not contain foreach")
	}
}

// Test_GenerateDefine_ForEachSlice 标量参数 + <foreach> 的批量方法生成切片签名（[]int64）。
func Test_GenerateDefine_ForEachSlice(t *testing.T) {
	mk := func(id string, withForEach bool) *SqlFunction {
		items := []*sqlFragment{{Type: simpleSqlFragment, Sql: &simpleSql{Sql: "delete from t where id in "}}}
		if withForEach {
			items = append(items, &sqlFragment{Type: forLoopSqlFragment, ForLoop: &sqlForLoop{}})
		}
		return &SqlFunction{
			Id:    id,
			Type:  DeleteFunction,
			Param: SqlParam{Name: "Long", TypeName: "Long", Type: BaseSqlParam, Need: true},
			Items: items,
		}
	}
	// 批量方法：含 foreach → []int64
	def := mk("deleteByIds", true).generateDefine()
	if !strings.Contains(def, "DeleteByIds \tfunc ([]int64) (int64,error)") {
		t.Error("foreach batch function should generate []int64 signature, got:", def)
	}
	// 标量方法：无 foreach → int64
	def = mk("deleteById", false).generateDefine()
	if !strings.Contains(def, "DeleteById \tfunc (int64) (int64,error)") {
		t.Error("scalar function should keep int64 signature, got:", def)
	}
	// map 参数 + foreach：不包切片（保持 map 签名）
	items := []*sqlFragment{{
		Type:    forLoopSqlFragment,
		ForLoop: &sqlForLoop{},
	}}
	fn := &SqlFunction{
		Id:    "deleteByMap",
		Type:  DeleteFunction,
		Param: SqlParam{Name: "Map", TypeName: "java.util.Map", Type: MapSqlParam, Need: true},
		Items: items,
	}
	def = fn.generateDefine()
	if !strings.Contains(def, "DeleteByMap \tfunc (map[string]interface{}) (int64,error)") {
		t.Error("map param with foreach should keep map signature, got:", def)
	}
}

// Test_GenerateDefine_ForEachSlice_Samples samples 真实回归：deleteConfigByIds（parameterType=Long + foreach）
// 生成 []int64 签名（此前为标量 int64）。
func Test_GenerateDefine_ForEachSlice_Samples(t *testing.T) {
	mps := NewSqlMappers("../samples")
	if mps == nil || len(mps.Mappers) == 0 {
		t.Error("load samples failed")
		return
	}
	found := false
	for _, m := range mps.Mappers {
		fn := m.NamedFunctions["deleteConfigByIds"]
		if fn == nil {
			continue
		}
		found = true
		def := fn.generateDefine()
		if !strings.Contains(def, "DeleteConfigByIds \tfunc ([]int64) (int64,error)") {
			t.Error("samples deleteConfigByIds should generate []int64 signature, got:", def)
		}
	}
	if !found {
		t.Error("deleteConfigByIds not found in samples")
	}
}
