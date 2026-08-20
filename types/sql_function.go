package types

import (
	"bytes"
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type SqlFunction struct {
	Id               string
	Owner            string
	Type             SqlFunctionType
	Param            SqlParam
	Result           SqlResult
	Items            []*sqlFragment
	UseGeneratedKeys bool   // <insert useGeneratedKeys="true">：自增主键回填开关（S-11）
	KeyProperty      string // 回填目标属性名（如 jobId）
	KeyColumn        string // 回填目标列名（如 job_id，可选）
	TotalUsage       int64
	FailedUsage      int64
	TotalDuration    int64
	MaxDuration      int64
	MinDuration      int64
	GenerateCount    int64
	GenerateDuration int64

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
	if !in.Param.Need && len(args) == 0 {
		return in.generateSqlWithoutParam(), []interface{}{}, nil
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
	if !in.Param.Need && len(args) == 0 {
		return in.generateSqlWithoutParam(), []string{}, nil
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
	if in.Param.Need {
		pt := toGolangType(in.Param.TypeName)
		// 标量参数 + 含 <foreach> 的批量方法（如 deleteConfigByIds/selectBatchIds）：
		// 参数自动生成为切片签名（[]int64 等），运行时 effectiveParamType 已支持切片分派（S-05）
		if in.Param.Type == BaseSqlParam && containsForEach(in.Items) {
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
		buf.WriteString("[]")
		if in.Result.ResultM != nil {
			buf.WriteString("models.")
			buf.WriteString(GetShortName(in.Result.ResultM.TypeName))
		} else {
			buf.WriteString(toGolangType(in.Result.ResultT.String()))
		}
		buf.WriteString(",error")
	}
	buf.WriteString(")\n")
	return buf.String()
}

// containsForEach 递归检查片段树（含 if/include/choose/where/set 嵌套）中是否含 <foreach>。
func containsForEach(items []*sqlFragment) bool {
	for _, item := range items {
		if item == nil {
			continue
		}
		switch item.Type {
		case forLoopSqlFragment:
			return true
		case ifTestSqlFragment:
			if item.IfTest != nil && containsForEach(item.IfTest.Sql) {
				return true
			}
		case includeSqlFragment:
			if item.Include != nil && containsForEach(item.Include.Fragments) {
				return true
			}
		case chooseSqlFragment:
			if item.Choose != nil {
				for _, w := range item.Choose.When {
					if w != nil && containsForEach(w.Sql) {
						return true
					}
				}
			}
		case whereSqlFragment:
			if item.Where != nil && containsForEach(item.Where.Sql) {
				return true
			}
		case setSqlFragment:
			if item.Set != nil && containsForEach(item.Set.Sql) {
				return true
			}
		}
	}
	return false
}
func (in *SqlFunction) prepareSqlWithMap(m map[string]interface{}) (string, []string) {
	log.Debugf("sql function %v generate sql with map: %v", in.Id, m)
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Items {
		buf.WriteString(" ")
		sqlstr, items := item.prepareSqlWithMap(m, 0)
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
		buf.WriteString(item.generateSqlWithMap(m, 0))
	}
	return buf.String()
}
func (in *SqlFunction) prepareSqlWithSlice(m []interface{}) (string, []string) {
	log.Debugf("sql function %v prepare sql with slice: %v", in.Id, m)
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Items {
		buf.WriteString(" ")
		sqlstr, items := item.prepareSqlWithSlice(m, 0)
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
		buf.WriteString(item.generateSqlWithSlice(m, 0))
	}
	return buf.String()
}
func (in *SqlFunction) prepareSqlWithParam(m interface{}) (string, []string) {
	log.Debugf("sql function %v generate sql with param: %v", in.Id, m)
	var buf bytes.Buffer
	var results []string
	for _, item := range in.Items {
		buf.WriteString(" ")
		sqlstr, items := item.prepareSqlWithParam(m)
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
		buf.WriteString(item.generateSqlWithParam(m))
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
			buf.WriteString(item.generateSqlWithoutParam())
		}
		in.noParamSQL = buf.String()
	})
	return in.noParamSQL
}

func parseSqlFunctionFromXmlNode(node xmlNode, rms map[string]*ResultMap, sns map[string]*SqlElement, owner string) *SqlFunction {
	log.Debugf("begin parse sql function from %v %v", node.Id, node.Name)
	defer log.Debugf("finish parse sql function from %v %v", node.Id, node.Name)
	tp := parseSqlFunctionType(node.Name)
	return &SqlFunction{
		Id:               node.Id,
		Owner:            owner,
		Type:             tp,
		Param:            parseSqlParamFromXmlAttrs(node.Attrs),
		Result:           parseSqlResultFromXmlAttrs(node.Attrs, rms),
		Items:            parsesqlFragmentsFromXmlElements(node.Elements, sns),
		UseGeneratedKeys: strings.EqualFold(node.Attrs["useGeneratedKeys"], "true"),
		KeyProperty:      node.Attrs["keyProperty"],
		KeyColumn:        node.Attrs["keyColumn"],
		TotalDuration:    0,
		TotalUsage:       0,
		MinDuration:      0,
		MaxDuration:      0,
		FailedUsage:      0,
		GenerateDuration: 0,
		GenerateCount:    0,
	}
}
func parsesqlFragmentsFromXmlElements(elems []xmlElement, sns map[string]*SqlElement) []*sqlFragment {
	var sts []*sqlFragment
	for _, elem := range elems {
		st, err := parsesqlFragmentFromXmlElement(elem, sns)
		if err != nil {
			log.Errorf("parse error:%v", err)
			continue
		}
		sts = append(sts, st)
	}
	return sts
}
