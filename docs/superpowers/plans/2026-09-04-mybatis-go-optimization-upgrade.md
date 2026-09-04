# mybatis-go 优化改造升级方案 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 依据《2026-09-04_mybatis-go优化点整理.md》把 mybatis-go（当前 main @ v0.1.15）升级为"共享 Java Mapper XML 零改造可接入、多环境前缀切换不依赖人工改 XML"的版本，目标 v0.2.0。

**Architecture:** 分三个里程碑推进，全部围绕既有分层（types=XML 解析/渲染，orm=注册/代理/执行）做**局部增强、不重构**：
1. **参数绑定层对齐 Java 语义**（P0）：参数需求从"XML 显式 parameterType 声明"改为"语句占位符自动推导"，并打通多参数按名/按位绑定（消除 1.1/1.2/1.3 三个实测踩坑）；
2. **表前缀写入/移除双能力 + 健壮性**（P0/P1）：在现有 tokenizeSQL 状态机里加"前缀映射/替换"通道（2.1），注册期增加宽松模式（5.1）；
3. **体验与可观测性**（P2）：日志降噪、gofmt、表集合 TTL、改写日志、prepared 降级告警、文档矩阵。

**Tech Stack:** Go 1.24；beevik/etree（XML 解析）；lib/pq + go-sql-driver/mysql + modernc.org/sqlite（端到端用 SQLite 免外部 DB）；无新依赖。

---

## 0. 现状核对（实施前先复核，防止对过期代码动手）

文档基于 v0.1.15 撰写，当前 main 亦为 v0.1.15。逐项核对结果（实施时以仓库实际为准，下表为 2026-09-04 走查快照）：

| 文档项 | 涉及代码（现状） | 结论 |
|---|---|---|
| 1.1 parameterType 硬依赖 | `types/sql_param.go::parseSqlParamFromXmlAttrs`（无属性即 `Need=false`）；`orm/param_type.go::checkSql`（`ArgsLen>0 && !Need` 报错） | **未修复，P0** |
| 1.2 多占位符绑 args[0] | `types/sql_fragments.go::simpleSql.prepareSqlWithParam/generateSqlWithParam`（每个 param 都用同一 `m`）；`orm/proxy_arg.go::buildArgs`（带 tag 已打包 map） | 未修复但**复现面收窄**：仅无 `args:` 标签的多参函数触发；带标签已可用 |
| 1.3 变参 panic | **两处**：`orm/param_type.go::makeParamType`、`orm/proxy_value.go::buildRemoteMethod:73-78`；`orm/proxy_value_test.go::Test_makeParamType` 断言 panic 行为 | **未修复，P0**（比文档多一处） |
| 1.4 resultType="map" 噪音 | `types/common.go::parseResultTypeFrom` default 分支 `log.Warnf("unsupport type to parse")` | 未修复，P2 |
| 2.1 前缀移除/替换 | `orm/table_prefix.go`（仅正向添加 `applyPrefix`，无映射/剥离） | **未实现，P0** |
| 2.2 MERGE INTO 等关键字 | tokenizeSQL 状态机：`into` 已触发 expectTable → MERGE INTO 大概率已可用 | **先验证补测试**，基表失败再实现 |
| 2.3 嵌套 CTE | 主循环逐 token 扫 `with` → 括号内嵌套 with 也会被访问 → 大概率已收集 | **先验证补测试** |
| 2.4 集合缓存 TTL | `orm/table_prefix.go::tableNamesCache` 无时间戳/TTL | 未实现，P2 |
| 2.5 改写无可观测性 | `DB.applyTablePrefix` 无日志 | 未实现，P2 |
| 3.1 schema 逐源覆盖 | `orm/multi_datasource.go::parseMultiDatabaseConfig` 把 `spring.datasource.<name>.*` 剥前缀后走 `parseDatabaseConfig` → `parseSchema` 已能读到逐源 schema；仅缺"继承默认源 schema" | 解析层已对称，**补文档矩阵 + 决定不继承** |
| 4.1 codegen 不带 parameterType | `types/table_struct.go`（SaveToFile/SaveMPToFile 均已 `CreateAttr("parameterType", …)`；`resources/mapper/UserInfoModelMapper.xml` 可印证） | **已实现**，补验证测试收尾 |
| 5.1 注册期错误中止 | `orm/mapper_cache.go::bindSql` 已聚合全部错误（combineErrors），非首个即停；但仍整体失败 | 半完成：补宽松模式 |
| 5.2 debug 日志过载 | `NewSqlMappers` 无汇总行；`parseSqlParamFromXmlAttrs` 等有 `--begin/--finish` 成对 Debugf | 未实现，P2 |
| 5.3 prepared 降级无告警 | `orm/prepared_stmt.go::ExecContext/QueryContext` 命中 `cacheFull()` 时静默直连 | 未实现，P2 |

**验收基线（每阶段结束必须通过）**：

```bash
go build ./...
go vet ./...
go test -count=1 ./...
```

---

## Phase 0（P0 阻断项，合入即为 v0.2.0）

### Task 0.1: 语句占位符自动推导 Need + slots（1.1）

**Files:**
- Modify: `types/sql_param.go`（`SqlParam` 增加 `Slots`/`AutoDerive` 字段）
- Modify: `types/sql_fragments.go`（新增 `collectSqlSlots`）
- Modify: `types/sql_function.go`（`parseSqlFunctionFromXmlNode` 接线）
- Modify: `orm/param_type.go`（`checkSql` 放宽）
- Test: `types/sql_param_test.go`（新增）、`orm/param_type_test.go`（新增）

- [ ] **Step 1: 写失败测试 —— types/sql_param_test.go 新增**

```go
// Test_collectSqlSlots_NoParameterType：samples（RuoYi 共享 XML，无 parameterType）
// 语句应推导出 Need=true 与占位符 slots（1.1 核心场景）
func Test_collectSqlSlots_NoParameterType(t *testing.T) {
	mps := NewSqlMappers("../samples")
	mp, ok := mps.NamedMappers[buildKey("MdmOrgRawMapper")]
	if !ok {
		t.Error("MdmOrgRawMapper not found in samples")
		return
	}
	sf, ok := mp.NamedFunctions["selectbyorgcode"]
	if !ok {
		t.Error("selectByOrgCode not found")
		return
	}
	if !sf.Param.Need {
		t.Error("selectByOrgCode contains #{orgCode}, Need should be true")
	}
	if len(sf.Param.Slots) != 1 || buildKey(sf.Param.Slots[0]) != "orgcode" {
		t.Errorf("slots = %v, want [orgCode]", sf.Param.Slots)
	}
	if !sf.Param.AutoDerive {
		t.Error("AutoDerive should be true when parameterType is absent")
	}
}

// Test_collectSqlSlots_Static：纯静态 SQL（无占位符、无动态标签）→ 推导 Need=false
func Test_collectSqlSlots_Static(t *testing.T) {
	n := xmlNode{
		Id: "selectOne", Name: "select",
		Elements: []xmlElement{{ElementType: xmlTextElem, Val: "select 1 as one"}},
	}
	f := parseSqlFunctionFromXmlNode(n, nil, nil, "test")
	if f.Param.Need {
		t.Error("static sql should not need params")
	}
	if !f.Param.AutoDerive {
		t.Error("AutoDerive should be true when parameterType is absent")
	}
}

// Test_collectSqlSlots_IfNoPlaceholder：动态 <if> 但体无占位符 → 不视为必参（1.1 风险项）
func Test_collectSqlSlots_IfNoPlaceholder(t *testing.T) {
	n := xmlNode{
		Id: "list", Name: "select",
		Elements: []xmlElement{
			{ElementType: xmlTextElem, Val: "select * from t where 1=1 "},
			{ElementType: xmlNodeElem, Val: xmlNode{
				Name:  "if",
				Attrs: map[string]string{"test": "deptCheckStrictly"},
				Elements: []xmlElement{
					{ElementType: xmlTextElem, Val: " and t.dept_id = 1"},
				},
			}},
		},
	}
	f := parseSqlFunctionFromXmlNode(n, nil, nil, "test")
	if f.Param.Need {
		t.Error("if without placeholder should not force need params")
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./types/ -run Test_collectSqlSlots -v`
Expected: `Need should be true` / `slots = []` 失败（`Slots` 字段尚不存在编译不过，先加字段再加推导逻辑，或先 `go test ./types/` 看编译错误即可）。

- [ ] **Step 3: 实现 —— types/sql_param.go**

```go
type SqlParam struct {
	Name       string
	TypeName   string
	Type       SqlParamType
	Need       bool
	Slots      []string // 语句中出现过的占位符名（#{} / ${}，去重、按首现序）
	AutoDerive bool     // 未声明 parameterType，Need 由语句占位符推导
}
```

`parseSqlParamFromXmlAttrs` 保持不变（仍只在显式 `parameterType` 时置 `Need=true`，`AutoDerive=false`）。

- [ ] **Step 4: 实现 —— types/sql_fragments.go 新增 collectSqlSlots**

```go
// collectSqlSlots 深度遍历片段树，收集全部 #{} / ${} 占位符名（去重、按首现序）。
// 推导规则（1.1）：<if>/<choose>/<where>/<set> 的 test 表达式本身不算必参；
// <foreach> 体由 slice/集合驱动，占位符不入 slots（不计入多参按位绑定）。
func collectSqlSlots(items []*sqlFragment) []string {
	seen := map[string]bool{}
	var out []string
	var walk func(fl []*sqlFragment)
	walk = func(fl []*sqlFragment) {
		for _, it := range fl {
			if it == nil {
				continue
			}
			switch it.Type {
			case simpleSqlFragment:
				if it.Sql == nil {
					continue
				}
				for _, p := range it.Sql.Params {
					k := buildKey(p.Name)
					if k != "" && !seen[k] {
						seen[k] = true
						out = append(out, p.Name)
					}
				}
			case includeSqlFragment:
				if it.Include != nil {
					walk(it.Include.Fragments)
				}
			case ifTestSqlFragment:
				if it.IfTest != nil {
					walk(it.IfTest.Sql)
				}
			case chooseSqlFragment:
				if it.Choose != nil {
					for _, w := range it.Choose.When {
						if w != nil {
							walk(w.Sql)
						}
					}
					if it.Choose.Otherwise != nil {
						walk([]*sqlFragment{{Type: simpleSqlFragment, Sql: it.Choose.Otherwise}})
					}
				}
			case whereSqlFragment:
				if it.Where != nil {
					walk(it.Where.Sql)
				}
			case setSqlFragment:
				if it.Set != nil {
					walk(it.Set.Sql)
				}
			}
			// forLoopSqlFragment：不进入（运行期以 item/collection 驱动）
		}
	}
	walk(items)
	return out
}
```

- [ ] **Step 5: 实现 —— types/sql_function.go parseSqlFunctionFromXmlNode 接线**

```go
items := parsesqlFragmentsFromXmlElements(node.Elements, sns)
param := parseSqlParamFromXmlAttrs(node.Attrs)
slots := collectSqlSlots(items)
if !param.Need { // 无显式 parameterType：由语句推导
	param.AutoDerive = true
	param.Need = len(slots) > 0 || containsForEach(items)
}
param.Slots = slots
return &SqlFunction{ /* 其余不变 */ Items: items, Param: param, ... }
```

- [ ] **Step 6: 实现 —— orm/param_type.go checkSql 放宽（M-06 语义）**

```go
func (in *ParamType) checkSql(f *types.SqlFunction, name string) error {
	if in.ArgsLen == 0 && f.Param.Need {
		return fmt.Errorf("%v check sql function %v failed, need func args: %v", name, f.Id, f.Param.Slots)
	}
	// 1.1：有参函数 + 语句无显式 parameterType 不再失败。
	// Need 已由语句占位符推导（SqlParam.Slots）；语句无占位符但有参时按 Java 语义
	// 容忍「未使用参数」，仅 debug 提示。
	if in.ArgsLen > 0 && !f.Param.Need {
		log.Debugf("%v check sql function %v: statement has no param slots, %d func args ignored", name, f.Id, in.ArgsLen)
		return nil
	}
	return nil
}
```

- [ ] **Step 7: 注册期回归测试 —— orm/param_type_test.go 新增**

```go
type NoParamTypeMapper struct {
	BaseMapper
	SelectByOrgCode func(params map[string]interface{}) ([]map[string]interface{}, error) `args:"params"`
	CountAll        func() ([]int64, error)
}

// Test_bindSql_NoParameterType：无 parameterType 的共享 XML 语句 + 有参函数 → 注册成功
func Test_bindSql_NoParameterType(t *testing.T) {
	info := newMapperInfo(reflect.TypeOf(NoParamTypeMapper{}))
	mps := types.NewSqlMappers("../samples")
	mp := mps.NamedMappers[types.BuildKeyOf("MdmOrgRawMapper")]
	if mp == nil {
		t.Error("MdmOrgRawMapper not found in samples")
		return
	}
	sf := mp.NamedFunctions["selectbyorgcode"]
	if sf == nil {
		t.Error("selectByOrgCode not found")
		return
	}
	// SelectByOrgCode 有参、语句无 parameterType：绑定必须成功
	if err := info.Functions[0].bindSql(sf); err != nil {
		t.Errorf("bind with-params func to no-parameterType statement failed: %v", err)
	}
	// CountAll 无参、绑定到有占位符语句：必须失败（Need 推导生效）
	sf2 := &types.SqlFunction{Id: "countall", Owner: "t", Param: types.SqlParam{Need: true, Slots: []string{"id"}}}
	if err := info.Functions[1].bindSql(sf2); err == nil {
		t.Error("no-arg func bound to param statement should fail")
	}
}
```

> 注：`types.BuildKeyOf` 若不存在则直接在测试里 `strings.ToLower`/自实现小写键（参考 `types.common.go::buildKey`）。`newMapperInfo` 当前返回 `*mapperInfo`（Task 0.2 会改签名，届时同步本测试）。

- [ ] **Step 8: 跑全量测试并提交**

Run: `go test -count=1 ./types/ ./orm/`
Expected: 新增用例通过；既有 `Test_*` 全绿（放宽只影响"有参函数 + 无 parameterType"这一失败面，不影响已声明 parameterType 的语句）。

```bash
git add types/sql_param.go types/sql_fragments.go types/sql_function.go orm/param_type.go types/sql_param_test.go orm/param_type_test.go
git commit -m "feat(types): derive param need from statement placeholders (1.1, M-06)"
```

---

### Task 0.2: 变参函数不再 panic；注册期参数错误 panic→error（1.3）

**Files:**
- Modify: `orm/param_type.go`（`makeParamType` 返回 error、变参跳过长度校验、`ParamType` 增 `Variadic`）
- Modify: `orm/mapper_cache.go`（`getFunctions`/`newMapperInfo`/`mapperInfo` 聚合错误）
- Modify: `orm/orm_cache.go`（`bindSqls` 合并 `FuncErrs`）
- Modify: `orm/proxy_value.go`（`buildRemoteMethod` 变参跳过长度校验）
- Test: `orm/proxy_value_test.go`（改 `Test_makeParamType`）、`orm/robustness_test.go`（若有 panic 断言同步）

- [ ] **Step 1: 写失败测试 —— orm/proxy_value_test.go 更新 Test_makeParamType**

```go
func Test_makeParamType(t *testing.T) {
	pt, err := makeParamType("f", reflect.TypeOf(func() {}), reflect.StructTag(""))
	if err != nil {
		t.Errorf("makeParamType no-arg failed: %v", err)
	}
	if len(pt.Args) != 0 {
		t.Error("no-arg func should have 0 args")
	}
	pt2, err := makeParamType("f", reflect.TypeOf(func(a int) {}), reflect.StructTag(`args:a`))
	if err != nil {
		t.Errorf("makeParamType single failed: %v", err)
	}
	if pt2.TagArgsLen != 1 {
		t.Errorf("TagArgsLen = %d, want 1", pt2.TagArgsLen)
	}
	// 1.3：非变参 tag 长度不匹配 → 返回 error 而非 panic
	if _, err := makeParamType("f", reflect.TypeOf(func(a, b int) {}), reflect.StructTag(`args:x`)); err == nil {
		t.Error("tag length != args should return error")
	}
	if _, err := makeParamType("f", reflect.TypeOf(func(a int) {}), reflect.StructTag(`args:x,y`)); err == nil {
		t.Error("tag length > args should return error")
	}
	// 1.3：变参（func(args ...interface{}) NumIn=1）tag 长度 > NumIn 不再 panic、不再报错
	pt3, err := makeParamType("f", reflect.TypeOf(func(args ...interface{}) {}), reflect.StructTag(`args:schema,tableName`))
	if err != nil {
		t.Errorf("variadic + tag length mismatch should be allowed: %v", err)
	}
	if pt3 == nil || !pt3.Variadic {
		t.Error("ParamType.Variadic should be true")
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./orm/ -run Test_makeParamType -v`
Expected: 编译失败/断言失败（`makeParamType` 尚未返回 error）。

- [ ] **Step 3: 实现 —— orm/param_type.go**

```go
type ParamType struct {
	TagArgs    []TagArg
	TagArgsLen int
	Args       []reflect.Type
	ArgsLen    int
	Variadic   bool // 变参函数（...T）：不校验 tag 与 NumIn 严格相等
}

func makeParamType(funcName string, funcType reflect.Type, funcTag reflect.StructTag) (*ParamType, error) {
	if funcType.Kind() != reflect.Func {
		return nil, fmt.Errorf("[mybatis-go] %v is not a func kind, got %v", funcName, funcType.Kind())
	}
	if funcType.NumIn() == 0 {
		return &ParamType{
			TagArgs:    []TagArg{},
			TagArgsLen: 0,
			Args:       []reflect.Type{},
			ArgsLen:    0,
		}, nil
	}
	tagArgs := parseTagArgs(getTagArgNames(funcTag))
	variadic := funcType.IsVariadic()
	if !variadic { // 变参不校验：NumIn 只反映 []T 一个槽位（1.3）
		if len(tagArgs) > funcType.NumIn() {
			return nil, fmt.Errorf(`[mybatis-go] method fail! the tag "args" length can not > arg length ! filed=` + funcName)
		}
		if len(tagArgs) > 0 && funcType.NumIn() != len(tagArgs) {
			return nil, fmt.Errorf(`[mybatis-go] method fail! the tag "args" length  != args length ! filed = ` + funcName)
		}
	}
	var args []reflect.Type
	for i := 0; i < funcType.NumIn(); i++ {
		args = append(args, funcType.In(i))
	}
	return &ParamType{
		TagArgs:    tagArgs,
		TagArgsLen: len(tagArgs),
		Args:       args,
		ArgsLen:    len(args),
		Variadic:   variadic,
	}, nil
}
```

- [ ] **Step 4: 实现 —— orm/mapper_cache.go（错误聚合）**

```go
type mapperInfo struct {
	Name           string
	Type           reflect.Type
	Functions      []*funcInfo
	NamedFunctions map[string]*funcInfo
	SqlMapper      *types.SqlMapper
	FuncErrs       []error // 参数解析失败（如 tag 长度不匹配）的函数，注册期聚合上报
}

func getFunctions(typ reflect.Type) ([]*funcInfo, []error) {
	var infos []*funcInfo
	var errs []error
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldName := typ.Field(i).Name
		fieldType := typ.Field(i).Type
		fieldTag := typ.Field(i).Tag
		if fieldType.Kind() != reflect.Func {
			continue
		}
		methodFieldCheck(&typ, &field, true)
		pt, err := makeParamType(fieldName, fieldType, fieldTag)
		if err != nil {
			errs = append(errs, fmt.Errorf("%v.%v: %v", typ.Name(), fieldName, err))
			continue
		}
		infos = append(infos, &funcInfo{
			Name:       fieldName,
			Type:       fieldType,
			Tag:        fieldTag,
			ParamType:  pt,
			ReturnType: makeReturnType(fieldName, fieldType),
			SqlFunc:    nil,
		})
	}
	return infos, errs
}

func newMapperInfo(typ reflect.Type) *mapperInfo {
	fs, errs := getFunctions(typ)
	mfs := makeNamedFunctions(fs)
	return &mapperInfo{
		Name:           typ.Name(),
		Type:           typ,
		Functions:      fs,
		NamedFunctions: mfs,
		FuncErrs:       errs,
	}
}
```

- [ ] **Step 5: 实现 —— orm/orm_cache.go bindSqls 合并 FuncErrs**

```go
for name := range in.mappers.Mappers {
	...
	mi := in.mappers.Mappers[name]
	errs = append(errs, mi.FuncErrs...)
	err := mi.bindSql(smp)
	if err != nil {
		errs = append(errs, err)
	}
}
```

- [ ] **Step 6: 实现 —— orm/proxy_value.go buildRemoteMethod 变参放行**

```go
var tagArgs = parseTagArgs(getTagArgNames(structField.Tag))
if !fieldTyp.IsVariadic() { // 1.3：变参不在代理安装期校验 tag 长度（注册期已放行）
	if len(tagArgs) > fieldTyp.NumIn() {
		panic(`[mybatis-go] method fail! the tag "args" length can not > arg length ! filed=` + structField.Name)
	}
	var tagArgsLen = len(tagArgs)
	if tagArgsLen > 0 && fieldTyp.NumIn() != tagArgsLen {
		panic(`[mybatis-go] method fail! the tag "args" length  != args length ! filed = ` + structField.Name)
	}
}
```

- [ ] **Step 7: 全量测试**

Run: `go build ./... && go vet ./... && go test -count=1 ./...`
Expected: 全绿；`Test_makeParamType` 新断言通过；无其它用例依赖旧 panic 行为（若有，同步改为断言 error）。

```bash
git add orm/param_type.go orm/mapper_cache.go orm/orm_cache.go orm/proxy_value.go orm/proxy_value_test.go
git commit -m "fix(orm): variadic functions no longer panic; arg-tag errors become errors (1.3)"
```

---

### Task 0.3: 多参数按占位符名/按位绑定（1.2）

**Files:**
- Modify: `types/sql_function.go`（`GenerateSQL`/`PrepareSQL` 多参路由 + `buildParamMap`）
- Modify: `types/sql_param.go`（`validParam` 增加多参校验分支）
- Test: `types/sql_function_test.go`（新增）

> 背景：`orm/proxy_arg.go::buildArgs` 对带 `args:` 标签的函数已把参数打包为 map（args[0]），不影响；本任务修复**无标签多参**（`func(schema, tableName string)`）场景：所有占位符被同一个 `args[0]` 替换。

- [ ] **Step 1: 写失败测试 —— types/sql_function_test.go 新增**

```go
// Test_GenerateSQL_MultiParam：无 args 标签的多参函数，占位符按位绑定（1.2）
func Test_GenerateSQL_MultiParam(t *testing.T) {
	n := xmlNode{
		Id: "selectColumns", Name: "select",
		Elements: []xmlElement{{ElementType: xmlTextElem, Val: "select * from information_schema.COLUMNS where TABLE_SCHEMA = #{schema} and TABLE_NAME = #{tableName}"}},
	}
	f := parseSqlFunctionFromXmlNode(n, nil, nil, "test")
	sql, _, err := f.GenerateSQL("gzwsk", "sys_user")
	if err != nil {
		t.Fatalf("GenerateSQL failed: %v", err)
	}
	if !strings.Contains(sql, "'gzwsk'") || !strings.Contains(sql, "'sys_user'") {
		t.Errorf("multi-param not bound per-slot: %v", sql)
	}
	if strings.Count(sql, "'gzwsk'") != 1 {
		t.Errorf("schema should appear exactly once: %v", sql)
	}
}

// Test_PrepareSQL_MultiParam：预编译路径同样按位绑定（占位符参数数量=占位符数）
func Test_PrepareSQL_MultiParam(t *testing.T) {
	n := xmlNode{
		Id: "selectColumns2", Name: "select",
		Elements: []xmlElement{{ElementType: xmlTextElem, Val: "select * from t where a = #{a} and b = #{b}"}},
	}
	f := parseSqlFunctionFromXmlNode(n, nil, nil, "test")
	sql, params, err := f.PrepareSQL("x", "y")
	if err != nil {
		t.Fatalf("PrepareSQL failed: %v", err)
	}
	if !strings.Contains(sql, "?") || len(params) != 2 {
		t.Errorf("prepare multi-param: sql=%v params=%v want 2 placeholders", sql, params)
	}
	if params[0] != "x" || params[1] != "y" {
		t.Errorf("prepare params = %v, want [x y]", params)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./types/ -run 'Test_GenerateSQL_MultiParam|Test_PrepareSQL_MultiParam' -v`
Expected: `multi-param not bound per-slot`（现在两个占位符都渲染成 `'gzwsk'`）。

- [ ] **Step 3: 实现 —— types/sql_function.go**

`GenerateSQL`/`PrepareSQL` 在 `validParam` 之后、`effectiveParamType` 分派之前插入多参分支，并新增 buildParamMap：

```go
// buildParamMap 将多参数按语句占位符名（Slots）位置绑定为渲染 map（1.2）。
// 与 Java @Param 按名取数对齐：第 i 个参数绑定第 i 个去重占位符名；
// 无 slots（纯静态/仅 foreach）时按 arg0/arg1... 兜底。
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
```

`GenerateSQL` 中（替换现有 `if !in.Param.Need && len(args) == 0 {...}` 之后的 `switch in.effectiveParamType(args)` 前的分派）：

```go
if len(args) > 1 {
	// 1.2：多参数按语句占位符名绑定（消除"全部绑定 args[0]"隐性 bug）
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
```

`PrepareSQL` 中对应分支：

```go
if len(args) > 1 {
	sqlstr, results := in.prepareSqlWithMap(in.buildParamMap(args))
	return sqlstr, results, nil
}
switch in.effectiveParamType(args) { /* 原有 Base/Slice 分支不变 */ }
nmp := convert2Map(reflect.Indirect(reflect.ValueOf(args[0])))
sqlstr, results := in.prepareSqlWithMap(nmp)
return sqlstr, results, nil
```

> `generateSqlWithMap`/`prepareSqlWithMap` 已存在（`MapSqlParam` 路径），`lookupParam` 支持按 buildKey 键取值，直接复用。

- [ ] **Step 4: 实现 —— types/sql_param.go validParam 多参放宽**

在 `validParam` 的 `switch in.Type` 之前插入：

```go
// 1.2：多参数（无 parameterType 或 Base/Map 声明均可）→ 按 Slots 逐参数绑定，
// 允许多个标量/map 参数；只校验非 nil。
if len(args) > 1 {
	for _, a := range args {
		val := reflect.ValueOf(a)
		if (a == nil) || (val.Kind() == reflect.Ptr && val.IsNil()) {
			return fmt.Errorf("need param, got nil in multi-param: %v", a)
		}
	}
	return nil
}
```

- [ ] **Step 5: 端到端复核（SQLite 真实执行）**

在 `orm/table_prefix_test.go` 同款 harness（`Test_Sqlite*` 模式）新增 `Test_Sqlite_MultiParamNoTag`：
建表 `columns_t(a text, b text)`，XML 语句 `select a from columns_t where a = #{a} and b = #{b}`（无 parameterType、无 args 标签），Go 结构体函数 `SelectByA func(a, b string) ([]map[string]interface{}, error)`，插入 2 行后按 `("x","y")` 查询断言只匹配 `a='x' and b='y'` 的行。

- [ ] **Step 6: 全量测试并提交**

Run: `go build ./... && go test -count=1 ./...`
Expected: 全绿；既有单参（Base/Slice/Map/Struct）路径行为不变（`len(args)==1` 不受影响）。

```bash
git add types/sql_function.go types/sql_param.go types/sql_function_test.go orm/table_prefix_test.go
git commit -m "fix(types): bind multi-args per statement slot instead of args[0] (1.2)"
```

---

### Task 0.4: 表前缀映射/替换能力（2.1，逆场景）

**Files:**
- Modify: `orm/database_config.go`（`MyBatisSetting.TablePrefixMap` + `parseTablePrefixMap` 接线）
- Modify: `orm/table_prefix.go`（`SetTablePrefixMap`/`tablePrefixMap`/`mappedPrefix`/`stripPrefix`/`rewriteSQLTablesWithMap` 与 `applyTablePrefix` 走新函数）
- Modify: `orm/multi_datasource.go`（可选：多源继承映射配置）
- Test: `orm/table_prefix_test.go`（新增单测 + SQLite 端到端）

- [ ] **Step 1: 写失败测试 —— orm/table_prefix_test.go 新增纯函数断言**

```go
// Test_rewriteSQLTablesWithMap：前缀移除/替换（2.1）
func Test_rewriteSQLTablesWithMap(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		prefix  string
		pMap    map[string]string
		want    string
	}{
		{"remove", "select * from threedb_sys_user", "", map[string]string{"threedb_": ""}, "select * from sys_user"},
		{"remove-noop", "select * from sys_user", "", map[string]string{"threedb_": ""}, "select * from sys_user"},
		{"replace", "select * from threedb_sys_user", "", map[string]string{"threedb_": "app_"}, "select * from app_sys_user"},
		{"map-plus-prefix", "select * from sys_user", "test_", map[string]string{"threedb_": ""}, "select * from test_sys_user"},
		{"map-into-join", "select * from threedb_a join threedb_b on a.id = b.id", "", map[string]string{"threedb_": ""}, "select * from a join b on a.id = b.id"},
		{"remove-schema-qualified", "select * from subsp.threedb_ods_x", "", map[string]string{"threedb_": ""}, "select * from subsp.ods_x"},
		{"unmatched-keeps", "select * from app_sys_user", "", map[string]string{"threedb_": ""}, "select * from app_sys_user"},
	}
	for _, c := range cases {
		got := rewriteSQLTablesWithMap(c.in, c.prefix, c.pMap, nil)
		if got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

// Test_rewriteSQLTablesWithMap_tableSet：映射值为空时走表集合判定（2.1 优先级约定）
func Test_rewriteSQLTablesWithMap_tableSet(t *testing.T) {
	set := tableSetOf("sys_user") // 库中只有无前缀表
	// SQL 引用带前缀表名 + 库中无该带前缀表 → 剥前缀
	got := rewriteSQLTablesWithMap("select * from threedb_sys_user", "", map[string]string{"threedb_": ""}, set)
	if got != "select * from sys_user" {
		t.Errorf("remove when unprefixed exists: got %q", got)
	}
	set2 := tableSetOf("threedb_sys_user") // 库中只有带前缀表
	got2 := rewriteSQLTablesWithMap("select * from threedb_sys_user", "", map[string]string{"threedb_": ""}, set2)
	if got2 != "select * from threedb_sys_user" {
		t.Errorf("keep when prefixed table exists: got %q", got2)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./orm/ -run Test_rewriteSQLTablesWithMap -v`
Expected: 编译失败（函数不存在）——正是预期。

- [ ] **Step 3: 实现 —— orm/database_config.go**

```go
type MyBatisSetting struct {
	...
	TablePrefix      string            // 数据表名前缀（现有，正向添加）
	TablePrefixMap   map[string]string // 前缀映射：oldprefix → newprefix（空值=移除），配置键 mybatis.table-prefix-map
}

// parseTablePrefixMap 解析前缀映射：mybatis.table-prefix-map = threedb_:,app_:subsp_
// 逗号分隔多个 old:new；new 为空表示移除；形如 `:x`（新前缀缺省）不合法被忽略。
func parseTablePrefixMap(m map[string]string) map[string]string {
	raw := strings.TrimSpace(m["mybatis.table-prefix-map"])
	if raw == "" {
		return nil
	}
	out := map[string]string{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		kv := strings.SplitN(item, ":", 2)
		oldp := strings.TrimSpace(kv[0])
		newp := ""
		if len(kv) == 2 {
			newp = strings.TrimSpace(kv[1])
		}
		if oldp == "" {
			continue
		}
		out[oldp] = newp
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
```

`parseDatabaseConfig` 里与 `TablePrefix: parseTablePrefix(m)` 并列加一行：`TablePrefixMap: parseTablePrefixMap(m)`。

- [ ] **Step 4: 实现 —— orm/table_prefix.go**

新增：

```go
// defaultTablePrefixMap 全局默认前缀映射（SetTablePrefixMap 设置，DB 未单独配置时生效）。
var defaultTablePrefixMap map[string]string

func SetTablePrefixMap(pm map[string]string) {
	defaultTablePrefixMap = pm
	if len(pm) > 0 {
		log.Infof("set default table prefix map: %v", pm)
	}
}

func (db *DB) tablePrefixMap() map[string]string {
	if db != nil && db.Config != nil && len(db.Setting.TablePrefixMap) > 0 {
		return db.Setting.TablePrefixMap
	}
	return defaultTablePrefixMap
}

// mappedPrefix 返回标识符命中的前缀映射（newprefix, 命中的旧前缀, 是否命中）。
// 命中规则：标识符以 oldp 开头（大小写不敏感）；映射优先级高于 tableSet 判定。
func mappedPrefix(name string, prefixMap map[string]string) (string, string, bool) {
	lower := strings.ToLower(name)
	for oldp, newp := range prefixMap {
		ol := strings.ToLower(oldp)
		if ol != "" && strings.HasPrefix(lower, ol) {
			return newp, oldp, true
		}
	}
	return "", "", false
}

// stripPrefix 在标识符 token 上移除开头前缀（引号标识符在开引号之后切）。
func stripPrefix(tok *sqlToken, oldp string) {
	if oldp == "" {
		return
	}
	if tok.typ == tkQuoted && len(tok.text) >= 2 {
		inner := tok.text[1 : len(tok.text)-1]
		if strings.HasPrefix(strings.ToLower(inner), strings.ToLower(oldp)) {
			tok.text = tok.text[:1] + inner[len(oldp):] + tok.text[len(tok.text)-1:]
		}
		return
	}
	if strings.HasPrefix(strings.ToLower(tok.text), strings.ToLower(oldp)) {
		tok.text = tok.text[len(oldp):]
	}
}
```

新增 `rewriteSQLTablesWithMap`（状态机逻辑与 `rewriteSQLTablesWithSet` 相同，仅表位置分支增加映射处理），并让旧函数委托：

```go
func rewriteSQLTablesWithSet(query, prefix string, tableSet map[string]struct{}) string {
	return rewriteSQLTablesWithMap(query, prefix, nil, tableSet)
}

// rewriteSQLTablesWithMap 同 rewriteSQLTablesWithSet，但叠加前缀映射（2.1）：
// 命中映射（表名以旧前缀开头）→ 映射优先于表集合：新前缀非空则替换，为空则按集合判定是否剥离
// （库中该无前缀表不存在且带前缀表不存在时按配置剥；带前缀表存在则保留原样）。
func rewriteSQLTablesWithMap(query, prefix string, prefixMap map[string]string, tableSet map[string]struct{}) string {
	if (prefix == "" || query == "") && len(prefixMap) == 0 {
		return query
	}
	tokens := tokenizeSQL(query)
	expectTable := false
	commaList := false
	cteNames := map[string]bool{}

	// applyToken 处理处于表位置的单个标识符（含 schema.table 限定名的表名部分），
	// 先在映射通道处理，未命中再走原有 prefixRequired+applyPrefix。
	applyToken := func(t *sqlToken) {
		name := identContent(*t)
		if name == "" {
			return
		}
		if newp, oldp, hit := mappedPrefix(name, prefixMap); hit {
			if newp == "" {
				// 移除：带前缀表物理存在 → 保留；其余（含集合缺失）→ 剥
				if len(tableSet) > 0 {
					lower := strings.ToLower(name)
					if _, ok := tableSet[lower]; ok {
						return // 无前缀表已存在，保持
					}
					if _, ok := tableSet[strings.ToLower(prefix)+lower]; ok {
						return // 带前缀表存在，不剥（避免指向错误表）
					}
				}
				stripPrefix(t, oldp)
				return
			}
			// 替换：剥旧前缀再套新前缀
			stripPrefix(t, oldp)
			applyPrefix(t, newp)
			return
		}
		if !cteNames[strings.ToLower(name)] && prefixRequired(*t, prefix, tableSet) {
			applyPrefix(t, prefix)
		}
	}
	_ = applyToken
	// ... 以下循环体与 rewriteSQLTablesWithSet 相同，仅把两处「表位置标识符处理」替换为 applyToken 调用 ...
	return strings.Join(tokenTexts(tokens), "")
}
```

实施要点（Step 4 内完成）：
1. 把 `rewriteSQLTablesWithSet` 的循环体抽出为 `rewriteTableTokens(tokens []sqlToken, prefix string, prefixMap map[string]string, tableSet map[string]struct{}) string`，两处表位置分支（普通标识符、`schema.table` 限定名的表名 token）统一调用 `applyToken`；原 `rewriteSQLTablesWithSet`/`rewriteSQLTables` 变为薄委托，**保证既有 20+24+8 组单测零回归**（prefixMap=nil 时行为与现状逐字节一致）。
2. 注意 `schema.table` 限定分支：映射对表名部分生效**不区分 schema 是否为 public/main**（映射为显式配置意图；`mappedPrefix` 命中即处理），未命中映射时维持原有 `isPrefixableSchema` 跳过逻辑。

- [ ] **Step 5: 实现 —— DB.applyTablePrefix 走映射 + 多源继承**

```go
func (db *DB) applyTablePrefix(query string) string {
	prefix := db.tablePrefix()
	pMap := db.tablePrefixMap()
	if prefix == "" && len(pMap) == 0 {
		return query
	}
	return rewriteSQLTablesWithMap(query, prefix, pMap, db.tableNameSet())
}
```

`orm/multi_datasource.go`：附加数据源未单独配置 `spring.datasource.<name>.table-prefix-map` 时继承默认源映射（参照现有 `TablePrefix` 继承写法，新增一行）。

> 说明：`spring.datasource.<name>.table-prefix-map` 逐源覆盖键经 `parseMultiDatabaseConfig` 剥前缀后自动生效（复用 parseDatabaseConfig），无需额外解析。

- [ ] **Step 6: 端到端（SQLite，模拟三库 prod 场景）**

新增 `Test_TablePrefixMap_SqliteRemovePrefix`：库中建**无前缀**表 `sys_user`；Setting 配 `TablePrefixMap: {"threedb_": ""}`（通过 `mybatis.table-prefix-map` 配置键加载）；Mapper XML 语句硬编码 `select * from threedb_sys_user where id = #{id}`；执行断言命中 `sys_user` 真实表。

- [ ] **Step 7: 全量测试并提交**

Run: `go build ./... && go test -count=1 ./orm/ -run 'Test_rewriteSQLTables|Test_TablePrefix' -v && go test -count=1 ./...`
Expected: 新增用例通过；既有前缀单测（basic/noFalsePositive/tableSet）全部保持原断言。

```bash
git add orm/database_config.go orm/table_prefix.go orm/multi_datasource.go orm/table_prefix_test.go
git commit -m "feat(orm): table prefix map to remove/replace prefixes (2.1)"
```

---

### Task 0.5: P0 端到端验收（共享 Java XML 零改造注册 + 前缀切换）

**Files:**
- Test: `orm/sqlite_p0_test.go`（新增，不含生产代码）

- [ ] **Step 1: 写验收测试**

```go
// Test_P0_SharedJavaXml_NoParameterType：samples（Java 侧共享 XML）无 parameterType
// 语句 + 有参函数注册成功，SQLite 真实执行命中数据（1.1+1.2+1.3 组合验证）
func Test_P0_SharedJavaXml_NoParameterType(t *testing.T) {
	dir := t.TempDir()
	// 1) 建库建表 + 写入无 parameterType 的 XML（拷贝 samples/mapper/mdm 语义）
	...
	// 2) orm.Initialize / NewConfig 指向 dir；RegisterMapper(new(MdmOrgRawMapper副本struct))
	// 3) SelectByOrgCode(map[string]interface{}{"orgCode": "O1"}) → 1 行
}
```

实施要点：
- 复用 `orm/table_prefix_test.go` 的 SQLite harness（`InitializeDatabase("sqlite", ...)` + 临时 XML 目录 + `RegisterMapper`）。
- Go struct：`SelectByOrgCode func(params map[string]interface{}) ([]*MdmOrgRaw, error)`（模型字段 `OrgCode`/`DeletedAt`，resultType 映射 `MdmOrgRaw`）。
- 同一测试内再验证：`func(args ...interface{})` + `args:"orgCode"` 变参注册不 panic（1.3 验收）。

- [ ] **Step 2: 运行**

Run: `go test -count=1 ./orm/ -run Test_P0_SharedJavaXml_NoParameterType -v`
Expected: 全绿。

```bash
git add orm/sqlite_p0_test.go
git commit -m "test(orm): end-to-end shared java xml registration without parameterType"
```

> **Phase 0 完成 = v0.2.0 候选**。README changelog 增补 v0.2.0 条目（1.1/1.2/1.3/2.1 + M-06 关闭 TODO）。

---

## Phase 1（P1 稳定性/可用性，合入为 v0.2.1）

### Task 1.1: 宽松注册模式（5.1）

**Files:**
- Modify: `orm/orm_cache.go`（`SetStrictRegister` + `bindSqls` 模式分支 + `bindMapper` 容错）
- Modify: `orm/mapper_cache.go`（`mapperInfo.bindSql` 宽松分支）
- Test: `orm/robustness_test.go`（新增）

- [ ] **Step 1: 写失败测试**

```go
// Test_SetStrictRegister_Lax：strict=false 时失败函数跳过、其余正常绑定
func Test_SetStrictRegister_Lax(t *testing.T) {
	dir := t.TempDir()
	// XML 仅含 selectGood；struct 内 Good + Bad 两个函数（Bad 无对应 XML 语句）
	os.WriteFile(filepath.Join(dir, "LaxMapper.xml"), []byte(`<?xml version="1.0" encoding="UTF-8" ?><mapper namespace="com.t.LaxMapper">
	<select id="selectGood" resultType="map">select 1 as g</select>
</mapper>`), 0644)
	cfg := NewConfigFromSettings(map[string]string{
		"spring.datasource.driver-class-name": "org.sqlite.JDBC",
		"spring.datasource.type":              "sqlite",
		"spring.datasource.db":                filepath.Join(dir, "t.db"),
		"mybatis.mapper-locations":            dir,
	})
	_ = cfg
	// 复用 table_prefix_test 的 sqlite 初始化（InitializeDatabase + RegisterMapper）
	type LaxMapper struct {
		BaseMapper
		SelectGood func() ([]map[string]interface{}, error)
		SelectBad  func() ([]map[string]interface{}, error)
	}
	orm.SetStrictRegister(false)          // 宽松
	defer orm.SetStrictRegister(true)     // 还原默认
	// ... 完成 sqlite 初始化后 RegisterMapper(new(LaxMapper))，err 必须为 nil ...
	// NewMapper("LaxMapper") 后调用 SelectBad → err 非 nil（明确错误）而非 panic
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./orm/ -run Test_SetStrictRegister_Lax -v`
Expected: `RegisterMapper` 返回错误（当前宽松未实现）。

- [ ] **Step 3: 实现 —— orm/orm_cache.go**

```go
var strictRegister atomic.Bool

func init() {
	strictRegister.Store(true) // 默认严格：任一函数注册失败整体失败（现状保持）
}

// SetStrictRegister 设置注册模式：true=严格（默认），任一函数绑定失败即整体失败；
// false=宽松：失败函数记 error 日志后跳过（该函数运行时返回明确错误），其余正常注册。
func SetStrictRegister(strict bool) { strictRegister.Store(strict) }

// IsStrictRegister 返回当前注册模式。
func IsStrictRegister() bool { return strictRegister.Load() }
```

`bindSqls` 中 `bindSql` 返回错误时的处理改为：

```go
err := mi.bindSql(smp)
if err != nil {
	if !IsStrictRegister() {
		log.Errorf("lax register: skip mapper %s: %v", name, err)
		continue
	}
	errs = append(errs, err)
}
```

`mapperInfo.bindSql`（mapper_cache.go）内部同样加宽松分支（跳过失败函数、继续绑定其余）：

```go
err := in.Functions[i].bindSql(sf)
if err != nil {
	if !IsStrictRegister() {
		log.Errorf("lax register: skip %v.%v: %v", in.Name, fi.Name, err)
		continue
	}
	errs = append(errs, err)
}
```

- [ ] **Step 4: 实现 —— bindMapper 容错（宽松模式下未绑定函数运行时返回明确错误）**

`orm/orm_cache.go::bindMapper` 中 `bm.fetchSqlFunction(funcName)` 返回 err 时，不再 panic，改为产出错误代理函数：

```go
sqlFunc, err := bm.fetchSqlFunction(funcName)
if err != nil {
	// 宽松注册：该函数注册期被跳过，运行时返回明确错误而非 panic
	rerr := err
	var proxyFunc = func(arg ProxyArg) []reflect.Value {
		return buildReturnValues(returnType, reflect.Value{}, rerr)
	}
	methodFieldCheck(&outTyp, &funcField, true)
	... 与正常路径相同的 MakeFunc 绑定 ...
	return
}
```

（具体接线参照 `bindMapper` 现有 proxyFunc 注册结构；`buildReturnValues(returnType, reflect.Value{}, error)` 已在库内使用，直接复用。）

- [ ] **Step 5: 全量测试并提交**

Run: `go build ./... && go vet ./... && go test -count=1 ./...`
Expected: 全绿；默认严格模式行为不变（既有测试未受影响）。

```bash
git add orm/orm_cache.go orm/mapper_cache.go orm/robustness_test.go
git commit -m "feat(orm): lax registration mode SetStrictRegister (5.1)"
```

---

### Task 1.2: codegen parameterType 验证收尾（4.1，已实现→补测试）

**Files:**
- Test: `types/table_struct_test.go`（新增断言）
- Docs: `docs/agents/mybatis-plus.md`（注明产物已带 parameterType；MyBatis-Plus 章节补一句）

- [ ] **Step 1: 写验证测试**

```go
// Test_GeneratedXML_AllStatementsHaveParameterType：schema2code 产物每条语句都带 parameterType（4.1）
func Test_GeneratedXML_AllStatementsHaveParameterType(t *testing.T) {
	ts := buildTestTableStructure(t) // 复用测试内已有构造（refcode/desc 字段）
	dir := t.TempDir()
	if err := ts.SaveToFile(filepath.Join(dir, "Xxx.xml"), ""); err != nil {
		t.Fatalf("save: %v", err)
	}
	bts, _ := os.ReadFile(filepath.Join(dir, "Xxx.xml"))
	xmlStr := string(bts)
	stmts := []string{"<select", "<insert", "<update", "<delete"}
	for _, tag := range stmts {
		idx := strings.Index(xmlStr, tag)
		for idx >= 0 {
			end := strings.Index(xmlStr[idx:], ">")
			head := xmlStr[idx : idx+end]
			if strings.Contains(head, " resultMap") || strings.Contains(head, " useGeneratedKeys") {
				idx2 := strings.Index(xmlStr[idx+len(tag):], tag)
				idx = idx2 + len(tag)
				continue
			}
			if !strings.Contains(head, "parameterType") {
				t.Errorf("%v statement missing parameterType: %v", tag, head[:min(120, len(head))])
			}
			idx2 := strings.Index(xmlStr[idx+len(tag):], tag)
			idx = idx2 + len(tag)
		}
	}
}
```

（`SaveMPToFile` 同理跑一遍。若发现有语句确实缺失 parameterType（如 count/selectAll），按 `types/table_struct.go` 现有 `CreateAttr("parameterType", …)` 模式补齐并断言通过。）

- [ ] **Step 2: 运行并修复缺口**

Run: `go test ./types/ -run Test_GeneratedXML_AllStatementsHaveParameterType -v`
Expected: 通过（现状已全带）；若有失败按 Step 1 补 `CreateAttr` 后重跑。

- [ ] **Step 3: 文档更新并提交**

`docs/agents/mybatis-plus.md` 增补：schema2code/saveMP 生成 XML 均内嵌 `parameterType`（基础类型/模型名/主键类型），配合 1.1 自动推导注册期无需再手工补参。`git add && git commit -m "test(types): assert generated xml carries parameterType (4.1)"`。

---

### Task 1.3: schema 逐源能力验证 + 文档矩阵（3.1/3.2）

**Files:**
- Test: `orm/multi_datasource_test.go`（新增验证）
- Docs: `README.md`（多数据源 × schema × 前缀矩阵）、`docs/agents/table-prefix.md`（schema 限定名约定说明）

- [ ] **Step 1: 写验证测试**

```go
// Test_parseMultiDatabaseConfig_schemaOverride：逐源 schema 键被解析（3.1）
func Test_parseMultiDatabaseConfig_schemaOverride(t *testing.T) {
	cm := map[string]string{
		"spring.datasource.url":  "jdbc:postgresql://h/pg?currentSchema=public",
		"spring.datasource.type": "postgres",
		"mybatis.datasources":    "report",
		"spring.datasource.report.url":      "jdbc:postgresql://h/pg",
		"spring.datasource.report.schema":   "subsp",
		"spring.datasource.report.type":     "postgres",
	}
	cfgs := parseMultiDatabaseConfig(cm)
	if cfgs["report"].Setting.Schema != "subsp" {
		t.Errorf("report schema = %q, want subsp", cfgs["report"].Setting.Schema)
	}
}
```

- [ ] **Step 2: 运行**

Run: `go test ./orm/ -run Test_parseMultiDatabaseConfig_schemaOverride -v`
Expected: 通过（parseDatabaseConfig 已对称支持）。若失败则说明逐源解析有缺口，按 `parseMultiDatabaseConfig` 剥前缀逻辑排查补修。

- [ ] **Step 3: 文档**

README「多数据源」/「schema」章节补矩阵示例：

```md
| 数据源 | schema | 前缀 | 配置键 |
|---|---|---|---|
| default | public | threedb_ | mybatis.table-prefix=threedb_ / mybatis.table-prefix-map=... |
| report | subsp | （移除） | spring.datasource.report.schema=subsp / spring.datasource.report.table-prefix-map=threedb_: |
```

`docs/agents/table-prefix.md`「samples 兼容性缺陷/已知边界」追加 3.2 约定：**SQL 一律写无 schema 名 + search_path 路由**（schema 限定名仅 public/main 参与前缀改写，其余整段跳过；前缀映射为显式配置意图、对限定名表名部分也生效）。

```bash
git add orm/multi_datasource_test.go README.md docs/agents/table-prefix.md
git commit -m "docs(orm): datasource schema override matrix + qualified-name convention (3.1/3.2)"
```

---

## Phase 2（P2 体验/可观测性，合入为 v0.2.2）

### Task 2.1: resultType="map" 不再刷 warning（1.4）

**Files:**
- Modify: `types/common.go`（`parseResultTypeFrom` 增加 map 分支）
- Test: `types/sql_result_test.go`（新增）

- [ ] **Step 1: 失败测试**

```go
// Test_ParseResultTypeFrom_Map：map 识别为合法通用类型，不再走 default warn 路径
func Test_ParseResultTypeFrom_Map(t *testing.T) {
	typ := parseResultTypeFrom("map")
	if typ.Kind() != reflect.Map {
		t.Errorf("parseResultTypeFrom(map) = %v, want map kind", typ)
	}
	if parseResultTypeFrom("MAP").Kind() != reflect.Map {
		t.Error("MAP uppercase should also be map")
	}
}
```

- [ ] **Step 2: 运行**（`go test ./types/ -run Test_ParseResultTypeFrom_Map -v` → FAIL）

- [ ] **Step 3: 实现**

```go
case "MAP", "HASHMAP", "TREEMAP":
	return reflect.TypeOf(map[string]interface{}{})
```

（不降级 default 的 Warn：保留对真实拼写错误的告警。）重跑通过后 `git commit -m "fix(types): map resultType is a valid generic type (1.4)"`。

---

### Task 2.2: 表关键字边界验证/补全（2.2）

**Files:**
- Test: `orm/table_prefix_test.go`（新增 MERGE/CREATE INDEX/RENAME 断言）
- Modify: `orm/table_prefix.go`（若断言失败才实现）

- [ ] **Step 1: 写验证测试（现状可能已支持，先测再改）**

```go
func Test_rewriteSQLTables_edgeKeywords(t *testing.T) {
	cases := []struct{ in, want string }{
		{"merge into target using src on a.id=b.id", "merge into test_target using test_src on a.id=b.id"},
		{"create index idx_x on sys_user(id)", "create index idx_x on test_sys_user(id)"},
		{"rename table a to b", "rename table test_a to test_b"},
	}
	for _, c := range cases {
		got := rewriteSQLTables(c.in, "test_")
		if got != c.want {
			t.Errorf("edge keyword: in=%q got=%q want=%q", c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 2: 运行确认**（`go test ./orm/ -run Test_rewriteSQLTables_edgeKeywords -v`）
  - `merge into`：现有 `into` 已触发 expectTable，应已通过（RUN 验证）。
  - `create index ... on`：`on` 是断句词不清零 expectTable，需状态机补 `index→on` 上下文。
  - `rename table a to b`：`to` 未触发，需补 `rename` 上下文。

- [ ] **Step 3: 按失败项实现（增量，保持既有行为）**

状态机补充（`rewriteSQLTablesWithMap` 词法循环内）：
- 新增两个上下文标记 `pendingIndex bool`、`renameMode bool`：
  - 遇到 word `index` → `pendingIndex = true`；
  - word `on` 且 `pendingIndex` → `expectTable = true; pendingIndex = false`；
  - word `rename` → `renameMode = true`（纯标记，`table` 分支自然处理 a）；
  - 表位置处理完成后若 `renameMode` 且当前词是首个表名 → 设置 `expectToTable = true`；
  - word `to` 且 `expectToTable` → `expectTable = true; expectToTable = false`。
- 仅新增分支，不改动 `clauseBreakWords` 与既有路径；每个新分支配单测断言（Step 1 用例补全到全绿）。

- [ ] **Step 4: 全量测试并提交**（`go test -count=1 ./orm/ && git commit -m "feat(orm): table rewrite edge keywords MERGE/CREATE INDEX/RENAME (2.2)"`）

---

### Task 2.3: 嵌套 CTE 名称收集验证（2.3）

**Files:**
- Test: `orm/table_prefix_test.go`（新增嵌套 WITH 断言）

- [ ] **Step 1: 写验证测试**

```go
func Test_rewriteSQLTables_nestedCTE(t *testing.T) {
	q := "select * from outer_cte where id in (with inner_cte as (select id from sys_user) select id from inner_cte)"
	got := rewriteSQLTables(q, "test_")
	if strings.Contains(got, "test_inner_cte") {
		t.Errorf("nested CTE name should not be prefixed: %v", got)
	}
	if strings.Contains(got, "test_outer_cte") {
		t.Errorf("outer CTE name should not be prefixed: %v", got)
	}
	if !strings.Contains(got, "test_sys_user") {
		t.Errorf("real table should be prefixed: %v", got)
	}
}
```

- [ ] **Step 2: 运行确认**（`go test ./orm/ -run Test_rewriteSQLTables_nestedCTE -v`）
  若通过（主循环逐 token 访问括号内 `with` → 已收集）→ 直接在 `docs/agents/table-prefix.md` 移除「嵌套 CTE 未收集」的已知边界描述并提交；若失败再实现递归收集（在 `matchParenIndex` 返回的括号区间内对 `with` token 再调 `cteModeCollect`）。

---

### Task 2.4: 表集合缓存 TTL（2.4）

**Files:**
- Modify: `orm/database_config.go`（`MyBatisSetting.TablePrefixSetTTL time.Duration` + `parseTablePrefixSetTTL`）
- Modify: `orm/table_prefix.go`（`tableNamesCache` 加 `fetchedAt`，`tableNameSet` 过期重取）
- Test: `orm/table_prefix_test.go`（新增）

- [ ] **Step 1: 实现（TTL 默认关闭，行为零变化）**

`database_config.go`：

```go
// parseTablePrefixSetTTL 解析表集合缓存 TTL：mybatis.table-prefix-set-ttl=1h
func parseTablePrefixSetTTL(m map[string]string) time.Duration {
	raw := strings.TrimSpace(m["mybatis.table-prefix-set-ttl"])
	if raw == "" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Warnf("bad table-prefix-set-ttl %q: %v", raw, err)
		return 0
	}
	return d
}
```

`table_prefix.go`：

```go
type tableNamesCache struct {
	mu        sync.RWMutex
	names     map[string]struct{}
	done      bool
	fetchedAt time.Time
	ttl       time.Duration // 0=永不过期（历史行为）
}

func (c *tableNamesCache) get() (map[string]struct{}, bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	stale := c.ttl > 0 && !c.fetchedAt.IsZero() && time.Since(c.fetchedAt) > c.ttl
	return c.names, c.done, stale
}
```

`tableNameSet`：`get()` 返回 stale 时重新 `fetchTableNames` 并更新 `fetchedAt`（失败时保留旧集合并记 Warn，不降级为空集）；`db.tablePrefixSetTTL()` 取 DB 配置或 0。

- [ ] **Step 2: 测试**

```go
// Test_tableNamesCache_TTL：TTL 过期后 get() 标记 stale（弱校验路径触发重取）
func Test_tableNamesCache_TTL(t *testing.T) {
	c := &tableNamesCache{names: map[string]struct{}{"a": {}}, done: true, fetchedAt: time.Now(), ttl: time.Millisecond}
	_, done, stale := c.get()
	if !done {
		t.Error("done should be true")
	}
	if stale {
		t.Error("freshly fetched should not be stale")
	}
	time.Sleep(2 * time.Millisecond)
	_, _, stale2 := c.get()
	if !stale2 {
		t.Error("ttl elapsed should mark stale")
	}
}
```

- [ ] **Step 3: 全量测试并提交**（`go test -count=1 ./... && git commit -m "feat(orm): table name set cache TTL refresh (2.4)"`）

---

### Task 2.5: 前缀改写日志 + prepared 降级告警（2.5/5.3）

**Files:**
- Modify: `orm/table_prefix.go`（`applyTablePrefix` debug 日志）
- Modify: `orm/prepared_stmt.go`（降级计数器 + 一次性 Warn）
- Test: `orm/prepared_stmt_test.go`（新增）

- [ ] **Step 1: 实现 —— table_prefix.go**

```go
func (db *DB) applyTablePrefix(query string) string {
	prefix := db.tablePrefix()
	pMap := db.tablePrefixMap()
	if prefix == "" && len(pMap) == 0 {
		return query
	}
	out := rewriteSQLTablesWithMap(query, prefix, pMap, db.tableNameSet())
	if log.IsDebugEnabled() && out != query {
		log.Debugf("[table-prefix] %s: rewrite %q -> %q", db.Setting.Name, query, out)
	}
	return out
}
```

- [ ] **Step 2: 实现 —— prepared_stmt.go**

```go
type PreparedStmtDB struct {
	...
	directExec       atomic.Int64 // 缓存满/无参降级直连的次数
	directWarnedOnce atomic.Bool  // 降级告警只打一次
}

// noteDirectExec 记录降级直连执行：首次数 Stats 可观测，首次发生 Warn 一次。
func (db *PreparedStmtDB) noteDirectExec() {
	db.directExec.Add(1)
	if db.directWarnedOnce.CompareAndSwap(false, true) {
		log.Warnf("prepared stmt cache full (>=%d), degraded to direct execution", maxPreparedStmts)
	}
}
```

将 `ExecContext`/`QueryContext`/`QueryRowContext` 中 `db.cacheFull()`（或 `len(args)==0` 直连）分支改为调用 `db.noteDirectExec()` 后直连。

- [ ] **Step 3: 测试**

```go
// Test_PreparedStmtDirectExecCounter：filled 缓存之后直连计数增加且不 panic
func Test_PreparedStmtDirectExecCounter(t *testing.T) {
	pdb := &PreparedStmtDB{Stmts: map[string]*Stmt{}, PreparedSQL: []string{}, Mux: &sync.RWMutex{}}
	// 填满 maxPreparedStmts 个占位，使 cacheFull()=true
	for i := 0; i < maxPreparedStmts; i++ {
		pdb.Stmts[fmt.Sprintf("q%d", i)] = &Stmt{prepared: make(chan struct{})}
	}
	pdb.noteDirectExec()
	if pdb.directExec.Load() != 1 {
		t.Errorf("directExec = %d, want 1", pdb.directExec.Load())
	}
}
```

- [ ] **Step 4: 全量测试并提交**（`go test -count=1 ./... && git commit -m "feat(orm): observable table-prefix rewrite log + prepared degrade warn (2.5/5.3)"`）

---

### Task 2.6: 解析日志汇总 + 生成文件 gofmt（5.2/4.2）

**Files:**
- Modify: `types/sql_mappers.go`（`NewSqlMappers` 每文件汇总行）
- Modify: `types/sql_mapper.go` + `types/result_map.go`/`types/table_struct.go` 生成路径（gofmt -w）
- Modify: `orm/schema_utils.go`（`SchemaToCode`/`SchemaToCodeMP` 末尾统一 gofmt）
- Test: `types/sql_mappers_test.go`（汇总行数量断言，可选）

- [ ] **Step 1: 实现 —— NewSqlMappers 汇总日志**

```go
func NewSqlMappers(dir string) *SqlMappers {
	filenames := filterMapperFiles(dir)
	var mps []SqlMapper
	nmp := map[string]*SqlMapper{}
	for _, filename := range filenames {
		start := time.Now()
		mp := loadMapper(filename)
		if mp != nil {
			mps = append(mps, *mp)
			...
		}
		if log.IsDebugEnabled() {
			log.Debugf("parsed mapper %s: %d statements in %s", filename, len(mp.Functions(nil)), time.Since(start))
		}
	}
	...
}
```

> 若 `SqlMapper` 无现成的"语句数"访问器，改用文件内 `<select|insert|update|delete` 计数（`strings.Count` 于原始 XML）或对 `NamedFunctions` 长度/2（每个函数按 id+lcase 双键）。选定一种在注释中说明即可，目标是一文件一行汇总。

- [ ] **Step 2: 实现 —— 生成文件 gofmt（4.2）**

`types/sql_mapper.go` 新建 helper 并接入 `generateMapperFile` 与模型 GenerateFile 写盘处；`orm/schema_utils.go` 两个入口末尾对各目录 `exec.Command("gofmt", "-w", f)`（失败仅 Warn，不阻塞生成）：

```go
// gofmtFile 对生成文件执行 gofmt -w（生成器固定宽度对齐不符合 gofmt，CI 噪音源）。
func gofmtFile(path string) {
	out, err := exec.Command("gofmt", "-w", path).CombinedOutput()
	if err != nil {
		log.Warnf("gofmt %v failed: %v: %s", path, err, out)
	}
}
```

- [ ] **Step 3: 验证**

Run: `go run ./cmd/sqlitedemo` 或 `go test ./types/ -run Test_GenerateFiles` 后，对生成的 `*.go` 执行 `gofmt -l` 断言为空（CI 步骤中新增 `gofmt -l` 检查或人工复核一次）。

- [ ] **Step 4: 提交**（`git add ... && git commit -m "chore(codegen): per-file parse summary + gofmt generated go (5.2/4.2)"`）

---

### Task 2.7: 文档与 TODO 收尾

**Files:**
- Modify: `README.md`（v0.2.0/v0.2.1/v0.2.2 changelog + 3.1 矩阵 + 1.2 多参按位绑定说明 + 2.1 前缀映射示例）
- Modify: `TODO.md`（关闭 M-06；新增 2.1/1.3/5.1 完成记录；2.2/2.3 验证结论：已支持则标注"已覆盖"）
- Modify: `docs/agents/table-prefix.md`（更新 2.1 映射/2.4 TTL/已知边界）
- Modify: `docs/agents/conventions.md`（注册期参数绑定新约定：无 parameterType 自动推导；多参无 tag 时按占位符顺序绑定）
- Modify: `docs/agents/mybatis-plus.md`（4.1 parameterType 已内嵌说明）

- [ ] **Step 1~N:** 随各 Task 同步更新对应文档；最终 `git commit -m "docs: changelog v0.2.x + optimization plan follow-ups"`。

---

## 自检（Self-Review）

**1. Spec 覆盖核对：**

| 文档项 | 任务 |
|---|---|
| 1.1 parameterType 自动推导 | Task 0.1（+0.5 验收） |
| 1.2 多占位符按位绑定 | Task 0.3 |
| 1.3 变参 panic→error | Task 0.2 |
| 1.4 map warning 噪音 | Task 2.1 |
| 2.1 前缀移除/替换 | Task 0.4 |
| 2.2 表关键字 | Task 2.2（先验证后实现） |
| 2.3 嵌套 CTE | Task 2.3（先验证后实现） |
| 2.4 集合缓存 TTL | Task 2.4 |
| 2.5 改写日志 | Task 2.5 |
| 3.1 schema 逐源/文档矩阵 | Task 1.3 |
| 3.2 schema.table 约定 | Task 1.3 |
| 4.1 codegen parameterType | Task 1.2（已实现→验证） |
| 4.2 gofmt | Task 2.6 |
| 4.3 if 静态检查 | **暂缓**：随注册期 run 校验成本高，收益低；留作后续（文档标注）。 |
| 5.1 宽松注册 | Task 1.1 |
| 5.2 日志过载 | Task 2.6 |
| 5.3 prepared 降级告警 | Task 2.5 |
| 官方 TODO 暂缓项（pgx 迁移等） | 不实施，仅文档说明 |

**2. 占位符扫描：** 无 TBD/TODO；所有代码步骤均给出完整实现或明确"测试先行、按失败项增量实现"的路径。

**3. 类型一致性：**
- `SqlParam` 新增字段 `Slots`/`AutoDerive` 全局同名；
- `ParamType.Variadic`、`makeParamType(...) (*ParamType, error)`、`getFunctions(...) ([]*funcInfo, []error)`、`mapperInfo.FuncErrs` 在各 Task 引用一致；
- 前缀新函数命名统一：`rewriteSQLTablesWithMap(query, prefix, prefixMap, tableSet)`、`mappedPrefix(name, prefixMap) (newp, oldp string, hit bool)`、`stripPrefix(tok, oldp)`、`SetTablePrefixMap`/`tablePrefixMap`；
- `SetStrictRegister(bool)` / `IsStrictRegister()` 命名一致（T1.1 与其测试）。

**4. 变更面控制：** 全部为增量、向后兼容：默认严格注册、默认无 TTL、默认无映射、单参路径零改动；既有 `Test_rewriteSQLTables_*`（20+24+8 组）与 `Test_Sqlite*` 端到端全部保持。

---

## 执行交接（Execution Handoff）

计划已保存至 `docs/superpowers/plans/2026-09-04-mybatis-go-optimization-upgrade.md`。两种执行方式：

1. **Subagent-Driven（推荐）**：每个 Task 派发独立 subagent，任务间我来审查；
2. **Inline Execution**：本会话内按 executing-plans 批量推进，检查点处暂停复审。

选哪种？确认后按 Phase 0 依次实施（每 Task 结束后跑验收基线 `go build ./... && go vet ./... && go test -count=1 ./...`）。