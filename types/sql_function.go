package types

import (
	"bytes"
	"fmt"
	"strconv"
	"github.com/bnulwh/mybatis-go/log"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bnulwh/mybatis-go/types/sqlfragment"
)

type SqlFunction struct {
	Id           string
	Owner        string
	Type         SqlFunctionType
	Param        SqlParam
	Result       SqlResult
	Items        []sqlfragment.Node
	IfTestFields []string
	// UnsupportedIfTests 语句内所有 <if>/<when test> 中未识别的子表达式（A3，
	// 去重后按首现序）：注册期汇总告警/严格模式报错的依据。空表示全部识别。
	UnsupportedIfTests []string
	// MaxRows 语句级行数上限覆盖（<select maxRows="N">，B3）：
	// 0=未设置（沿用全局 DefaultRowLimit），-1=不限制，>0=语句级上限。
	MaxRows int64
	// HasBind 语句内含 <bind>（P0-3）：渲染必须走 map 路径（绑定变量写回参数 map）。
	HasBind bool
	// BoundNames 语句内 <bind> 产生的变量名（不从调用方取值，已在 Slots/AllSlots 中剔除）。
	BoundNames []string
	// BindRefs <bind> 表达式引用的参数路径（调用方需提供，已并入 Slots/AllSlots）。
	BindRefs []string
	// UnsupportedTags 语句内未注册的 XML 标签（P0-3，去重按首现序）：
	// 注册期默认汇总 Warn，strict-unknown-tag 严格模式下报错。
	UnsupportedTags []string
	UseGeneratedKeys   bool   // <insert useGeneratedKeys="true">：自增主键回填开关（S-11）
	KeyProperty        string // 回填目标属性名（如 jobId）
	KeyColumn          string // 回填目标列名（如 job_id，可选）
	HasVersion         bool   // 乐观锁：updateById 涉及 version 列，CAS 检测
	FlushCache         bool   // 执行后刷新当前 namespace 缓存
	UseCache           bool   // 是否启用二级缓存（默认 true，false 时跳过）
	TotalUsage         int64
	FailedUsage        int64
	TotalDuration      int64
	MaxDuration        int64
	MinDuration        int64
	GenerateCount      int64
	GenerateDuration   int64

	// 无参 SQL 结果静态不变，首次生成后缓存（P2-5）
	noParamOnce sync.Once
	noParamSQL  string

	// minDurationInit 标记 MinDuration 是否已初始化（原子）。
	// 注意：MinDuration==0 兼作「无数据」语义，但真实 0ms 测量值也是 0，
	// 若以 0 作未初始化哨兵，真实最小值 0 会被后续较大值误覆盖（CAS 分支 bug），
	// 故单独用标志区分「未初始化」与「已初始化为 0」。
	minDurationInit int32
}

func (in *SqlFunction) UpdateUsage(start time.Time, success bool) {
	atomic.AddInt64(&in.TotalUsage, 1)
	if !success {
		atomic.AddInt64(&in.FailedUsage, 1)
	}
	d := time.Since(start).Milliseconds()
	atomic.AddInt64(&in.TotalDuration, d)
	updateMaxDuration(&in.MaxDuration, d)
	updateMinDuration(&in.MinDuration, &in.minDurationInit, d)
}

// updateMaxDuration 以 CAS 循环原子更新最大值（避免并发 Swap 互相覆盖）。
func updateMaxDuration(target *int64, v int64) {
	for {
		cur := atomic.LoadInt64(target)
		if v <= cur || atomic.CompareAndSwapInt64(target, cur, v) {
			return
		}
	}
}

// updateMinDuration 以 CAS 循环原子更新最小值。
// init 单独标记是否已初始化：真实测量值可以是 0ms，与「无数据」的 0 冲突，
// 因此不能用 MinDuration==0 兼作未初始化哨兵（否则真实最小值 0 会被后续较大值覆盖）。
func updateMinDuration(target *int64, init *int32, v int64) {
	for {
		if atomic.LoadInt32(init) == 0 {
			if atomic.CompareAndSwapInt32(init, 0, 1) {
				atomic.StoreInt64(target, v)
				return
			}
			continue
		}
		cur := atomic.LoadInt64(target)
		if v >= cur || atomic.CompareAndSwapInt64(target, cur, v) {
			return
		}
	}
}

func (in *SqlFunction) String() string {
	return fmt.Sprintf("%v,%v,%v,%v,%v,%v,%v,%v,%v", in.Owner, in.Id,
		atomic.LoadInt64(&in.TotalUsage)-atomic.LoadInt64(&in.FailedUsage),
		atomic.LoadInt64(&in.TotalUsage),
		atomic.LoadInt64(&in.MinDuration), atomic.LoadInt64(&in.MaxDuration),
		atomic.LoadInt64(&in.TotalDuration),
		atomic.LoadInt64(&in.GenerateCount), atomic.LoadInt64(&in.GenerateDuration))
}

func (in *SqlFunction) updateGenerate(start time.Time) {
	atomic.AddInt64(&in.GenerateCount, 1)
	atomic.AddInt64(&in.GenerateDuration, time.Since(start).Milliseconds())
}

// RequiresCallerArgs 语句是否需要调用方提供参数（占位符或 <foreach> 集合）。
// 与 Param.Need 的区别：<bind> 产生的绑定变量与纯字面量 <bind> 不计入——
// 它们由语句自身求值，不要求函数声明对应参数（P0-3）。
func (in *SqlFunction) RequiresCallerArgs() bool {
	return len(in.Param.Slots) > 0 || sqlfragment.ContainsForEach(in.Items)
}

// bindParamMap 把调用参数规整为可写入绑定变量的渲染 map（P0-3）：
// 无参 → 空 map；多参 → 按 Slots 按位绑定（1.2）；单参 map/struct → 拷贝/转换；
// 标量参数无法命名注入，返回明确错误（不再把 #{bindVar} 原样发给数据库）。
func (in *SqlFunction) bindParamMap(args []interface{}) (map[string]interface{}, error) {
	switch {
	case len(args) == 0:
		return map[string]interface{}{}, nil
	case len(args) > 1:
		return in.buildParamMap(args), nil
	}
	v := reflect.Indirect(reflect.ValueOf(args[0]))
	if !v.IsValid() {
		return map[string]interface{}{}, nil
	}
	switch v.Kind() {
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			break
		}
		out := make(map[string]interface{}, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			key := iter.Key().String()
			val := iter.Value().Interface()
			out[key] = val
			// 键名大小写不敏感：同步 buildKey 别名，与 #{} / <if test> 取值规则一致
			if k := buildKey(key); k != key {
				if _, exists := out[k]; !exists {
					out[k] = val
				}
			}
		}
		return out, nil
	case reflect.Struct:
		return convert2Map(v), nil
	}
	// 标量参数：按语句首个（唯一）参数名包装，保持与 Base 渲染路径等价。
	if len(in.Param.Slots) == 1 {
		return map[string]interface{}{in.Param.Slots[0]: args[0]}, nil
	}
	return nil, fmt.Errorf("sql function %v: <bind> requires a map/struct parameter or a single named slot, got %v (slots=%v)", in.Id, v.Kind(), in.Param.Slots)
}

// effectiveParamType 结合静态 parameterType 与实际参数类型决定渲染路径：
// 实际传入切片/数组时走 Slice 路径（如 parameterType="Long" + collection="array" 批量删除），
// 标量走 Base，map/struct（time.Time 除外）走 Map；无法判定时回退静态类型。
func (in *SqlFunction) effectiveParamType(args []interface{}) SqlParamType {
	if len(args) == 0 {
		return in.Param.Type
	}
	v := reflect.ValueOf(args[0])
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		return SliceSqlParam
	}
	val := reflect.Indirect(v)
	switch val.Kind() {
	case reflect.Map, reflect.Struct:
		if val.Type().String() == "time.Time" {
			return BaseSqlParam
		}
		return MapSqlParam
	case reflect.String,
		reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return BaseSqlParam
	}
	return in.Param.Type
}

// GenerateSQL
func (in *SqlFunction) GenerateSQL(args ...interface{}) (string, []interface{}, error) {
	log.Debugf("========================================")
	log.Debugf("sql function %v begin generate sql args: %v", in.Id, args)
	start := time.Now()
	defer log.Debugf("sql function %v finish  generate sql", in.Id)
	defer in.updateGenerate(start)
	err := in.Param.validParam(args)
	if err != nil {
		log.Warnf("valid param failed: %v", err)
		return "", []interface{}{}, err
	}
	if len(args) > 0 && args[0] == nil {
		// S-09：nil 参数反射零值 panic 防御（无 parameterType 的函数走 validParam 前已放行）
		log.Warnf("sql function %v got nil param", in.Id)
		return "", []interface{}{}, fmt.Errorf("sql function %v param is nil", in.Id)
	}
	if in.HasBind {
		// P0-3：语句含 <bind>，绑定变量需写回参数 map —— 必须走 map 渲染路径
		// （含无参语句：bind 字面量也在此求值，避免 #{bindVar} 原样下发）。
		if len(args) > 0 && in.effectiveParamType(args) == SliceSqlParam {
			// 顶层切片参数：bind 只在 <foreach> 体内的逐项 map 上下文生效
			smp := sliceArgsFrom(args)
			return in.generateSqlWithSlice(smp), []interface{}{}, nil
		}
		mp, err := in.bindParamMap(args)
		if err != nil {
			return "", []interface{}{}, err
		}
		return in.generateSqlWithMap(mp), []interface{}{}, nil
	}
	if !in.Param.Need && len(args) == 0 {
		return in.generateSqlWithoutParam(), []interface{}{}, nil
	}
	if len(args) > 1 {
		// 1.2：多参数按语句占位符名（Slots）位置绑定为 map，走 Map 渲染路径。
		// 占位符名来自 XML（#{schema} / #{tableName}），与 Java @Param 按名取数对齐；
		// 无 slots（纯静态/仅 foreach 或参数多于占位符）时按 arg0/arg1... 兜底。
		return in.generateSqlWithMap(in.buildParamMap(args)), []interface{}{}, nil
	}
	switch in.effectiveParamType(args) {
	case BaseSqlParam:
		return in.generateSqlWithParam(args[0]), []interface{}{}, nil
	case SliceSqlParam:
		smp := sliceArgsFrom(args)
		return in.generateSqlWithSlice(smp), []interface{}{}, nil
	}
	nmp := convert2Map(reflect.Indirect(reflect.ValueOf(args[0])))
	return in.generateSqlWithMap(nmp), []interface{}{}, nil
}
func (in *SqlFunction) PrepareSQL(args ...interface{}) (string, []string, error) {
	log.Debugf("========================================")
	log.Debugf("sql function %v begin prepare sql args: %v", in.Id, args)
	defer log.Debugf("sql function %v finish  prepare sql", in.Id)
	err := in.Param.validParam(args)
	if err != nil {
		log.Warnf("valid param failed: %v", err)
		return "", nil, err
	}
	if len(args) > 0 && args[0] == nil {
		// S-09：nil 参数反射零值 panic 防御（无 parameterType 的函数走 validParam 前已放行）
		log.Warnf("sql function %v got nil param", in.Id)
		return "", nil, fmt.Errorf("sql function %v param is nil", in.Id)
	}
	if in.HasBind {
		// P0-3：<bind> 语句统一走 map 渲染路径（同 GenerateSQL）。
		if len(args) > 0 && in.effectiveParamType(args) == SliceSqlParam {
			smp := sliceArgsFrom(args)
			sqlstr, results := in.prepareSqlWithSlice(smp)
			return sqlstr, results, nil
		}
		mp, err := in.bindParamMap(args)
		if err != nil {
			return "", nil, err
		}
		sqlstr, results := in.prepareSqlWithMap(mp)
		return sqlstr, results, nil
	}
	if !in.Param.Need && len(args) == 0 {
		return in.generateSqlWithoutParam(), []string{}, nil
	}
	if len(args) > 1 {
		// 1.2：多参数按语句占位符名位置绑定（同 GenerateSQL）
		sqlstr, results := in.prepareSqlWithMap(in.buildParamMap(args))
		return sqlstr, results, nil
	}
	switch in.effectiveParamType(args) {
	case BaseSqlParam:
		sqlstr, results := in.prepareSqlWithParam(args[0])
		return sqlstr, results, nil
	case SliceSqlParam:
		smp := sliceArgsFrom(args)
		sqlstr, results := in.prepareSqlWithSlice(smp)
		return sqlstr, results, nil
	}
	nmp := convert2Map(reflect.Indirect(reflect.ValueOf(args[0])))
	sqlstr, results := in.prepareSqlWithMap(nmp)
	return sqlstr, results, nil

}
func (in *SqlFunction) generateDefine() string {
	var buf bytes.Buffer
	buf.WriteString("\t")
	buf.WriteString(UpperFirst(in.Id))
	buf.WriteString(" \tfunc (")
	if in.Id == MPSelectPageID {
		buf.WriteString("*orm.PageParam")
	} else if in.Param.Need {
		pt := toGolangType(in.Param.TypeName)
		if in.Param.TypeName == "" {
			pt = "map[string]interface{}"
		}
		if in.Param.Type == BaseSqlParam && sqlfragment.ContainsForEach(in.Items) {
			buf.WriteString("[]")
			buf.WriteString(pt)
		} else {
			buf.WriteString(pt)
		}
	}
	buf.WriteString(") (")
	switch in.Type {
	case UpdateFunction, InsertFunction, DeleteFunction:
		buf.WriteString("int64,error")
	case SelectFunction:
		if in.Id == MPSelectPageID {
			buf.WriteString("*orm.Page,error")
		} else {
			buf.WriteString("[]")
			if in.Result.ResultM != nil {
				buf.WriteString("models.")
				buf.WriteString(GetShortName(in.Result.ResultM.TypeName))
			} else if in.Result.ResultT.Kind() == reflect.Map {
				buf.WriteString("map[string]interface{}")
			} else {
				buf.WriteString(toGolangType(in.Result.ResultT.String()))
			}
			buf.WriteString(",error")
		}
	}
	buf.WriteString(")\n")
	return buf.String()
}

// buildParamMap 将多参数按语句占位符名（Slots）位置绑定为渲染 map（1.2）。
// 与 Java @Param 按名取数对齐：第 i 个参数绑定第 i 个去重占位符名；
// 无 slots（纯静态/仅 foreach）或参数多于占位符时按 arg0/arg1... 兜底。
func (in *SqlFunction) buildParamMap(args []interface{}) map[string]interface{} {
	nmp := map[string]interface{}{}
	for i, a := range args {
		key := fmt.Sprintf("arg%d", i)
		if i < len(in.Param.Slots) && in.Param.Slots[i] != "" {
			key = in.Param.Slots[i]
		}
		nmp[buildKey(key)] = a
	}
	return nmp
}

// containsForEach 已迁移至 sqlfragment.ContainsForEach（按 Node 接口递归）。

func (in *SqlFunction) prepareSqlWithMap(m map[string]interface{}) (string, []string) {
	log.Debugf("sql function %v generate sql with map: %v", in.Id, m)
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Items {
		buf.WriteString(" ")
		sqlstr, items := item.PrepareSqlWithMap(m, 0)
		buf.WriteString(sqlstr)
		results = append(results, items...)
	}
	return buf.String(), results
}
func (in *SqlFunction) generateSqlWithMap(m map[string]interface{}) string {
	log.Debugf("sql function %v prepare sql with map: %v", in.Id, m)
	var buf bytes.Buffer
	for _, item := range in.Items {
		buf.WriteString(" ")
		buf.WriteString(item.GenerateSqlWithMap(m, 0))
	}
	return buf.String()
}
func (in *SqlFunction) prepareSqlWithSlice(m []interface{}) (string, []string) {
	log.Debugf("sql function %v prepare sql with slice: %v", in.Id, m)
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Items {
		buf.WriteString(" ")
		sqlstr, items := item.PrepareSqlWithSlice(m, 0)
		buf.WriteString(sqlstr)
		results = append(results, items...)
	}
	return buf.String(), results
}
func (in *SqlFunction) generateSqlWithSlice(m []interface{}) string {
	log.Debugf("sql function %v generate sql with slice: %v", in.Id, m)
	var buf bytes.Buffer
	for _, item := range in.Items {
		buf.WriteString(" ")
		buf.WriteString(item.GenerateSqlWithSlice(m, 0))
	}
	return buf.String()
}
func (in *SqlFunction) prepareSqlWithParam(m interface{}) (string, []string) {
	log.Debugf("sql function %v generate sql with param: %v", in.Id, m)
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Items {
		buf.WriteString(" ")
		sqlstr, items := item.PrepareSqlWithParam(m)
		buf.WriteString(sqlstr)
		results = append(results, items...)
	}
	return buf.String(), results
}
func (in *SqlFunction) generateSqlWithParam(m interface{}) string {
	log.Debugf("sql function %v generate sql with param: %v", in.Id, m)
	var buf bytes.Buffer
	for _, item := range in.Items {
		buf.WriteString(" ")
		buf.WriteString(item.GenerateSqlWithParam(m))
	}
	return buf.String()
}

func (in *SqlFunction) generateSqlWithoutParam() string {
	// 无参 SQL 的拼接结果静态不变，只生成一次（sync.Once 保证并发安全）
	in.noParamOnce.Do(func() {
		log.Debugf("sql function %v generate sql without param", in.Id)
		var buf bytes.Buffer
		for _, item := range in.Items {
			buf.WriteString(" ")
			buf.WriteString(item.GenerateSqlWithoutParam())
		}
		in.noParamSQL = buf.String()
	})
	return in.noParamSQL
}

func parseSqlFunctionFromXmlNode(node xmlNode, rms map[string]*ResultMap, sns map[string]*SqlElement, owner string) *SqlFunction {
	log.Debugf("begin parse sql function from %v %v", node.Id, node.Name)
	defer log.Debugf("finish parse sql function from %v %v", node.Id, node.Name)
	tp := parseSqlFunctionType(node.Name)
	items, _ := sqlfragment.ParseFragments(node.Elements, sns)
	param := parseSqlParamFromXmlAttrs(node.Attrs)
	// P0-3：<bind> 产生的变量由语句自身求值，不再要求调用方传参；
	// 表达式引用的参数路径则并入 Slots/AllSlots（参与按位绑定与签名推导）。
	boundNames := sqlfragment.CollectBoundNames(items)
	bindRefs := sqlfragment.CollectBindRefs(items)
	hasBind := sqlfragment.ContainsBind(items)
	slots := mergeSqlSlots(sqlfragment.CollectSqlSlots(items), boundNames, bindRefs)
	if !param.Need { // 无显式 parameterType：由语句占位符推导（1.1)
		param.AutoDerive = true
		param.Need = len(slots) > 0 || sqlfragment.ContainsForEach(items)
	}
	param.Slots = slots
	param.AllSlots = mergeSqlSlots(sqlfragment.CollectAllSqlSlots(items), boundNames, bindRefs)
	return &SqlFunction{
		Id:                 node.Id,
		Owner:              owner,
		Type:               tp,
		Param:              param,
		Result:             parseSqlResultFromXmlAttrs(node.Attrs, rms),
		Items:              items,
		HasBind:            hasBind,
		BoundNames:         boundNames,
		BindRefs:           bindRefs,
		UnsupportedTags:    sqlfragment.CollectUnsupportedTags(items),
		IfTestFields:       sqlfragment.CollectIfTestFieldNames(items),
		UnsupportedIfTests: sqlfragment.CollectUnsupportedIfTests(items),
		MaxRows:            parseMaxRowsAttr(node.Attrs["maxRows"], owner, node.Id),
		UseGeneratedKeys:   strings.EqualFold(node.Attrs["useGeneratedKeys"], "true"),
		KeyProperty:        node.Attrs["keyProperty"],
		KeyColumn:          node.Attrs["keyColumn"],
		TotalDuration:      0,
		TotalUsage:         0,
		MinDuration:        0,
		MaxDuration:        0,
		FailedUsage:        0,
		GenerateDuration:   0,
		GenerateCount:      0,
	}
}

// mergeSqlSlots 合并占位符名与绑定变量（P0-3）：
//   - 剔除 <bind> 产生的名字（不是调用方参数）；
//   - 并入 <bind> 表达式引用的参数路径（调用方需提供）；
//
// 去重按首现序，保持与 1.2 按位绑定、xml2go 签名推导的顺序语义一致。
func mergeSqlSlots(slots, boundNames, bindRefs []string) []string {
	bound := make(map[string]bool, len(boundNames))
	for _, n := range boundNames {
		bound[strings.ToLower(n)] = true
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(slots)+len(bindRefs))
	appendSlot := func(name string) {
		k := strings.ToLower(strings.TrimSpace(name))
		if k == "" || bound[k] || seen[k] {
			return
		}
		seen[k] = true
		out = append(out, name)
	}
	for _, s := range slots {
		appendSlot(s)
	}
	for _, s := range bindRefs {
		appendSlot(s)
	}
	return out
}

// parseMaxRowsAttr 解析 <select maxRows="N"> 语句级行数上限属性（B3）：
// 0/缺省=未设置（沿用全局），-1=不限制，>0=语句级上限；非法值告警后按未设置处理。
func parseMaxRowsAttr(v, owner, id string) int64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		log.Warnf("invalid maxRows %q on %v.%v, ignored: %v", v, owner, id, err)
		return 0
	}
	return n
}

// parsesqlFragmentsFromXmlElements 已迁移至 sqlfragment.ParseFragments（log-and-continue 容错保持不变）。
