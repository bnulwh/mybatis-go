# mybatis-go 框架自身需提升点（v0.3.18 实读）

> 生成日期：2026-10-21
> 版本基线：`github.com/bnulwh/mybatis-go v0.3.18`（源码实读，模块缓存路径 `%GOMODCACHE%/github.com/bnulwh/mybatis-go@v0.3.18`）
> 用途：可直接整理为 upstream issue / PR 清单
> 配套文档：同目录 `2026-10-21_backend-go使用mybatis-go需提升点.md`（本项目侧改造，与本文件互补不重复）
> 取证方式：文中每条均给框架源码 `文件:行`，并附本项目（backend-go）的**真实踩坑证据**，便于 upstream 复现

---

## 0. 结论速览

| # | 问题 | 类型 | 影响面 | 建议优先级 |
|---|---|---|---|---|
| A1 | `<foreach>` 循环体只支持纯文本，嵌套标签被丢弃（甚至 panic） | 动态 SQL 缺陷 | 共享 Java XML 里 `<foreach>` 内套 `<if>` 的语句全部不可用 | 高（✅ v0.3.19 已修复） |
| A2 | `<include>` 的 `<sql>` 片段「静态文本 + 动态标签混排」时丢静态文本 | 动态 SQL 缺陷 | 拼出的 SQL 缺片段，直接语法错误 | **最高**（静默生成错误 SQL）（✅ v0.3.19 已修复） |
| A3 | 未识别的 `<if test>` 表达式被静默丢弃 → 条件恒真 → 条件体恒渲染 | 动态 SQL 缺陷 | 与 v0.3.11「字符串等值恒不渲染」同源，仍未根治 | **最高**（静默错数据）（✅ v0.3.31 已修复：UnsupportedIfTests 记录/汇总 + strict-if-test 严格模式，默认行为不变） |
| B1/B2 | `buildCountSQL` / `ApplyPagination` 用正则剥 `ORDER BY`/`LIMIT`，会误剥子查询内层 | 分页缺陷 | 启用 `PageParam` 后 total 错、内层 LIMIT 被删 | 高（✅ v0.3.20 已修复：深度感知扫描器替代正则） |
| B3 | 全局行数上限 10000 静默截断，无按语句覆盖，`MaxRows` 是死字段 | 正确性 | 大结果集静默丢行 | 高（✅ v0.3.32 已修复：严格模式 + max-rows 配置（v0.3.20）+ 语句级 `<select maxRows>` 覆盖 + `LastQueryTruncated()` 可观测） |
| B4 | `PageParam` 与 XML 手写 `LIMIT` 共存无诊断 | 分页缺陷 | 双重分页静默错数据 | 低（✅ v0.3.20 已修复：运行时告警） |
| C1 | `orm.Begin()` 全局单事务槽仍在且无 Deprecated 引导（`WithTx` 已可用） | 并发/事务 | 误用即串行或事务互相劫持 | 高（✅ v0.3.21 已修复：Deprecated + 运行时迁移告警，行为不变） |
| C2 | `WithTx` 与「Mapper 方法未声明 ctx」组合静默走非事务连接 | 事务 | 事务语义失效且无提示 | 中（✅ v0.3.21 已修复：WithTx 作用域检测 + 运行时告警） |
| D1 | `NewMapper(name) interface{}` 找不到即 panic，无 error 版本、无泛型取用 | API 设计 | 调用方只能裸类型断言 | 中高（✅ v0.3.22 已修复：Mapper[T]/MustMapper[T]/NewMapperE） |
| D2 | 无 Mapper 注册表自省 API | 可观测 | 使用方只能硬编码计数 | 中（✅ v0.3.22 已修复：RegisteredMapperNames/MapperFuncCount） |
| D3 | 行级 scan/convert 失败只打 ERROR 日志仍返回部分数据，无 strict 选项 | 正确性 | "少了行"无法被调用方感知 | 中高（✅ v0.3.22 已修复：SetStrictResult + ErrResultConvertIncomplete + LastConvertReport） |
| E1 | `CollectSqlSlots` 不采集 `<if>/<foreach>` 体内占位符 → xml2go 生成签名丢动态参数 | 代码生成 | 动态 SQL Mapper 无法用生成层，只能手写 | 高（✅ v0.3.23 已修复：CollectAllSqlSlots/AllSlots → 动态体占位符退化为 map 参数） |
| E2 | 跨 Mapper resultMap 限定名引用退化为 `interface{}` | 代码生成 | 共享 Java XML 多 Mapper 复用时关联字段类型丢失 | 中（✅ v0.3.24 已修复：resultMap 全局索引静态解析） |
| E3 | namespace 冲突无诊断/别名机制 | 代码生成 | 共享 XML 被迫改名抬进仓库 | 中（✅ v0.3.25 已修复：重复 namespace 检测 + NamespacePrefix + RegisterNamespaceAlias） |
| F1 | 缺 MySQL→PG 族函数/语法翻译层（`ifnull`/`find_in_set`/`date_format`/反引号/`col = 0`） | 方言能力 | 共享 Java XML 必须手改副本才能跑 PG/金仓 PG 模式 | 中高（✅ v0.3.26 已修复：RewriteMySqlCompat 翻译引擎 + function-rewrite 开关；`col = 0` 类型推断保守跳过） |
| F2 | `||` 跨模式语义差异无检测 | 方言能力 | 共享 XML 换 database_mode 语义全变 | 中（✅ v0.3.27 已修复：portability-check 开关 + 执行收口点告警） |
| G1 | `docs/mybatis-go使用问题与解决方案.md` 停留在 v0.1.7，多条结论已过时且无废弃标注 | 文档 | 使用方按旧文档做过时变通（本项目已中招） | 中高（低成本）（✅ v0.3.28 已修复：文档状态横幅 + 7 条目 Fixed in 标注 + README 能力矩阵） |
| G2 | 生产级开关（safe-update/explain-slow/pool-stats/cache）默认全关，无推荐预设 | 运维 | 上线默认无防护、无观测 | 中（✅ v0.3.29 已修复：RecommendedProductionSettings + 启动期未开启项提示） |
| H1 | lib/pq → pgx/v5 暂缓；RETURNING 唯一推荐路径未文档化 | 技术选型 | currval 变通残留、不稳定 | 跟踪（✅ v0.3.30 已文档化：RETURNING 唯一推荐路径 + currval 废弃指引；迁移保持暂缓） |

---

## A. 动态 SQL 引擎

### A1【高】`<foreach>` 循环体只接受纯文本，嵌套标签被静默丢弃

> ✅ **已于 v0.3.19 修复**：`sqlForLoop.Sql` 改为 `[]Node`（`types/sqlfragment/foreach.go`），循环体经 `ParseFragments` 解析，支持嵌套 `<if>`/`<choose>`/`<foreach>`；非检查类型断言 panic 一并消除。回归测试见 `types/sqlfragment/dynamic_fix_test.go`。

**框架证据**：`types/sqlfragment/foreach.go:178-195`
```go
func parseSqlForLoopFromXmlNode(attrs map[string]string, elems []XmlElement) (*sqlForLoop, error) {
	...
	return &sqlForLoop{
		...
		Sql: parseSimpleSqlFromText(elems[0].Val.(string)),  // ← 只取第 0 个元素，且断言为 string
	}, nil
}
```
- 循环体结构体字段是 `Sql *simpleSql`（`:17-25`），**不是 `[]Node`**，所以循环体内**不可能**出现 `<if>`/`<choose>`/嵌套 `<foreach>`；
- `elems[0].Val.(string)` 是**非检查类型断言**：若循环体首个元素是子标签（`Val` 非 string），直接 panic；
- 若循环体「文本 + 子标签」混排，子标签部分被丢弃（只保留 `elems[0]`）。

**本项目踩坑**：`backend-go/mybatis/mappers/EtlTaskMapper.xml:14-17` 头注释明确记录"`<foreach>` 内不支持嵌套标签，故 aggregateTable 的「字符串列非空数 e_」改为对所有列统一用 `CAST(col AS text)` 计算"，即**为了绕开该缺陷而改变了 SQL 语义**（数值列 `e_` 变成恒等于 `n_`）。

**建议**：`sqlForLoop.Sql` 改为 `[]Node`，与 `sqlIfTest.Sql` / `sqlChoose` 一致；解析走统一 `parseFragmentsFromXmlElements`；`elems[0].Val.(string)` 改安全断言 + 明确错误。

**最小复现**：
```xml
<select id="t" resultType="map">
  SELECT 1 FROM t WHERE
  <foreach collection="items" item="it" open="(" close=")" separator=",">
    <if test="it != null">#{it}</if>
  </foreach>
</select>
```

---

### A2【最高】`<include>` 渲染丢弃 `<sql>` 片段的静态文本

> ✅ **已于 v0.3.19 修复**（双根因）：
> ① 注释切割：`ParseXmlNode`（`types/sqlfragment/xml.go`）解析前经 `stripXmlComments` 剥离注释（CDATA 感知、注释替换为单空格），消除「注释把文本切成多段、逐段 TrimSpace 拼接丢中间空格」；
> ② 混排丢失：`ParseSqlElementFromXmlNode`（`types/sqlfragment/element.go`）在「文本 + 子标签」混排时按文档序把静态文本以 `simpleSql` 节点放入 `Fragments`，`renderFragments` 统一渲染，静态前后缀不再丢失。回归测试见 `types/sqlfragment/dynamic_fix_test.go`。

**框架证据**：`types/sqlfragment/include.go:39-44`
```go
func (in *sqlInclude) renderFragments(render func(f Node, depth int) (string, []string), depth int) (string, []string) {
	if len(in.Fragments) == 0 {
		return in.Sql, []string{}   // 只有「纯文本片段」才返回静态文本
	}
	var buf bytes.Buffer
	for _, item := range in.Fragments {   // 有 Fragments 时完全忽略 in.Sql
		...
	}
}
```
即：`<sql id="x"> SELECT a, b FROM t <where>...</where> </sql>` 这类「静态前后缀 + 动态子节点」的片段，被 `<include>` 引用后**只剩动态部分**，前面的 `SELECT a, b FROM t` 直接消失。

**本项目踩坑**：`backend-go/mybatis/mappers/EtlTaskMapper.xml:20-22` 注释：`"<sql>/<include> 片段在「静态文本 + 嵌套动态标签混排」时会丢失静态文本，故 strtonumValueExpr 直接内联进 applyStrtonumPair（出现两次）"` —— 为避免该 bug，同一段 SQL 被迫复制两份，后续维护双改。

**建议**：把静态文本作为片段树的第一个文本节点（`simpleSql`）放入 `Fragments`，`renderFragments` 统一渲染；或 `renderFragments` 先写 `in.Sql` 再渲染 `Fragments`。**这是"静默生成非法 SQL"级别的缺陷，建议优先修**。

---

### A3【最高】未识别的 `<if test>` 表达式 → 条件恒真 → 条件体恒渲染

> ✅ **已于 v0.3.31 修复**：① 解析期未命中模式的子表达式记录到 `sqlIfTest.Unsupported`，语句级聚合为 `SqlFunction.UnsupportedIfTests`（`types/sqlfragment.CollectUnsupportedIfTests`，含 `<choose>/<when>` 嵌套），注册期按语句汇总告警；② 新增 `orm.SetStrictIfTest(true)`（配置 `mybatis.configuration.strict-if-test`，默认关）：注册期存在未识别表达式即返回 error（与 strictReg 一致）；③ 严格模式下求值期「整个 test 均未识别」按 false 处理（不渲染）+ Warn，**默认（非严格）恒真行为保持不变**（存量兼容）。测试见 `types/sqlfragment/a3_unsupported_test.go` + `types/a3_unsupported_test.go` + `orm/a3_strict_iftest_test.go`。

**框架证据**：`types/sqlfragment/if.go:353-420`（解析）+ `:181-190`（求值）
```go
// 解析：只认 5 种模式，其余 continue（静默丢弃）
reNC     // x != null
reEC     // x != ''
reStrCmp // x == 'literal'
reCmp    // x != 0 / x > 18 / x.length > 0
reBool   // 裸标识符
// 未命中任何模式 → continue（不告警、不报错）

// 求值：Conditions 为空时「全条件满足」恒真
func (in *sqlIfTest) checkConditions(m map[string]interface{}) bool {
	for _, cond := range in.Conditions { if !cond.checkValue(m) { return false } }
	return true   // ← 空条件 = true
}
```
后果：`<if test="a == b">`、`<if test="x != 'y'.toString()">`、`<if test="flag ? true : false">`、`<if test="@java.util.Objects@nonNull(x)">`、`<if test="x.size() > 0">` 等一律**恒渲染**（把 `AND ...` 无条件拼进 SQL），既不报错也不告警。

**本项目踩坑（同源事故）**：v0.3.11 之前 `<if test="status == '白名单'">` 就是这条路被静默丢弃 → 条件恒渲染/恒不渲染，导致 WorkerMapper 名单筛选与 QualityMapper 分支全失效（现由 `mybatis/string_equality_render_test.go` 在 v0.3.12 后锁定回归）。**v0.3.12 只补了「字符串等值」一种形态，其余形态仍恒真**。

**建议**：
1. 解析期：任一 `reSplit` 后的子条件未命中任何模式 → `log.Warnf` + 记录到 `SqlFunction.UnsupportedIfTests`，并在 `RegisterMapper` 阶段汇总打印；
2. 提供 `orm.SetStrictIfTest(true)`：严格模式下注册期直接返回 error（与 `strictReg` 一致）；
3. 求值期：`Conditions` 为空时按 **false**（不渲染）处理，并 Warn —— 与 MyBatis「表达式非法即失败」更接近，避免静默错数据。

---

### A4【低】无参路径与 A3 语义不一致

`if.go:147-149`：`GenerateSqlWithoutParam() string { return "" }` —— 无参调用时 `<if>` 体**恒不渲染**；而有参路径空条件是**恒渲染**。同一节点两种默认，容易在「无参 Mapper 方法」上产生隐蔽差异。

**建议**：统一为一种默认（推荐"恒不渲染 + Warn"），并在文档写明。

---

### A5【中】`${}` 内联无内建校验/白名单 API

框架对 `${}` 只做「原样字符串替换」，无任何标识符校验能力。本项目 61 个 XML 里 `**238 处** `${}``，全部靠代码自律 + 自写守卫脚本（`scripts/2026-10-09_check_mysql_dialect_in_mappers.py`）兜底。

**建议**：新增 `orm.RegisterIdentWhitelist(table string, cols ...string)` 或语法扩展 `${col:whitelist}`；渲染时非白名单值直接 error。参考 MyBatis 生态（MyBatis-Plus 的 `SqlInjectionUtils`）也可提供 `orm.SafeIdentifier(s)` 工具。

---

## B. 分页与行数上限

### B1【高】`buildCountSQL` 正则剥子句，会误剥子查询内层

> ✅ **已于 v0.3.20 修复**：`buildCountSQL`（`orm/sql_rewrite.go`）改为先经 `dialector.StripTopLevelTailClauses` 只剥离**括号深度 0** 的尾部子句（ORDER BY/LIMIT/OFFSET/RETURNING/FOR UPDATE 族），SELECT...FROM 替换边界同样按顶层深度定位（`TopLevelKeywordEnd`/`TopLevelClausePos`）；子查询内层 `ORDER BY`/`LIMIT`（含 `ROW_NUMBER() OVER (ORDER BY ...)`、标量子查询 `(SELECT ... LIMIT 1)`）原样保留，并新支持 WITH/CTE 前缀。扫描器见 `orm/dialector/sql_scan.go`（引号/注释/词边界感知），测试见 `orm/dialector/sql_scan_test.go` + `orm/sql_rewrite_test.go`。

**框架证据**：`orm/sql_rewrite.go:23-38`
```go
reLimit     = regexp.MustCompile(`(?is)\s+LIMIT\s+\S+`)
reOffset    = regexp.MustCompile(`(?is)\s+OFFSET\s+\S+`)
reOrderBy   = regexp.MustCompile(`(?is)\s+ORDER\s+BY\s+.+$`)

func buildCountSQL(query string) string {
	q = reLimit.ReplaceAllString(q, "")     // 全局替换：子查询里的 LIMIT 也删
	q = reOrderBy.ReplaceAllString(q, "")   // `.+$` 贪婪到句尾：子查询 ORDER BY 会被吞到外层
	...
}
```
派生 COUNT 时对 SQL 做**无 AST 的全局正则清洗**，对「子查询里含 `ORDER BY` / `LIMIT`」的语句（本项目大量存在，例如 `ROW_NUMBER() OVER (...)` 外层包一层排名后取 `_rn = 1`）会产出错误的 COUNT。

### B2【高】`ApplyPagination` 同样剥全部 `LIMIT`

> ✅ **已于 v0.3.20 修复**：`dialector.ApplyPagination`（`orm/dialector/base.go`）及 Oracle/MSSQL/DB2/ClickHouse 方言的 ApplyPagination 均改用 `StripTopLevelTailClauses(q, "limit", "offset")` —— 只替换**顶层** LIMIT/OFFSET 再拼接新分页，标量子查询 `(SELECT ... LIMIT 1)` 原样保留。测试见 `orm/sql_rewrite_test.go` `Test_applyPagination` 子查询用例。

**框架证据**：`orm/dialector/base.go:159-170`
```go
q = limitRe.ReplaceAllString(q, "")   // 删掉 SQL 里所有 LIMIT
q = offsetRe.ReplaceAllString(q, "")
return fmt.Sprintf("%s LIMIT %d OFFSET %d", q, limit, offset)
```
本项目 `mybatis/mappers/RecordsMapper.xml:150` 有标量子查询 `... (SELECT ... LIMIT 1) AS orgName`。若对该语句启用 `*orm.PageParam`，**子查询的 `LIMIT 1` 会被删掉**，语义从"取一行"变成"取全部（多行即报错）"。

### B3【高】行数上限：全局、静默、无按语句覆盖

> ✅ **已于 v0.3.20 部分修复（第 1/3 条 + 第 3 条）**：
> ① **严格模式**：`orm.SetRowLimitStrict(true)`（或配置 `mybatis.configuration.row-limit-strict=true`）后，Query/QueryTo/RowStream 达到上限返回 `ErrRowLimitExceeded`（`%w` wrap，`errors.Is` 可判），不再静默截断；RowStream 的上限与严格开关在打开时快照；
> ② **`MaxRows` 死字段接入**：`mybatis.max-rows` 配置经 `InitializeFromSettings` 套用到 `SetDefaultRowLimit`（0=未设置保持默认 10000，负数不限）；
> ③ 语句级覆盖（`<select maxRows>`）与 `Truncated` 可观测返回**未做**（涉及签名变更，留待后续）。
> 测试见 `orm/row_limit_strict_test.go`。
>
> ✅ **v0.3.32 完成 ③（语句级覆盖 + 截断可观测）**：
> - **语句级覆盖**：`<select maxRows="N">` 属性解析到 `SqlFunction.MaxRows`（0=未设置沿用全局，-1 不限，>0 语句级上限，非法值告警回退全局）；经 context 下发（不改公共 API 签名），Mapper 代理全路径（executeMethod/executeStream/executePage）与原生查询（Query/QueryTo/QueryStream）统一经 `effectiveRowLimit` 解析生效上限；
> - **截断可观测**：新增 `orm.LastQueryTruncated() *QueryTruncation`（Statement=namespace.id / Limit / Time），静默截断与严格拦截均记录（`orm/row_limit_ctx.go`）；另有 `orm.ResetQueryTruncation()`（测试用）。
> 测试见 `orm/b3_stmt_maxrows_test.go`（语句级覆盖端到端 / -1 不限 / 非法值回退 / 流式快照 / 严格模式叠加 / 截断记录）。

**框架证据**：
- 上限常量与开关：`orm/sql_execute.go:20`（`defaultRowLimit = 10000`）、`:47-52`（`SetDefaultRowLimit`）；
- 截断处只 Warn 不报错：`orm/sql_execute.go:169-176`、`orm/scan_rows.go:237/275/309`、`orm/row_stream.go:92`；
- 配置字段 `MyBatisSetting.MaxRows int64`（`orm/database_config.go:36`）**无任何读取点**（全仓 grep 仅此一处）→ **死字段**。

后果：
1. 只能全局调，不能按语句/按 Mapper 调；
2. 超限时**静默返回前 10000 行**，调用方无法区分"确实只有 1 万行"和"被截断了"；
3. 本项目因此出现真实风险：`WorkerMapper.SelectLatestPersonnelScores`（全量人员最新得分，人员库数万级）、`SysOperLogMapper.selectOperLogList`（无 LIMIT 全表）都会静默少数据。

**建议**：
1. 补齐 `MaxRows` 配置（`mybatis.max-rows`）与语句级覆盖（XML `<select maxRows="-1">` 或 `PageParam` 旁路）；
2. 截断时在返回值上可观测：Mapper 方法支持 `(*Result, error)` 形态返回 `Truncated bool`，或提供 `orm.LastQueryTruncated()`；
3. 至少把 Warn 提升为可配置的 error（`mybatis.configuration.row-limit-strict=true` 时返回 `ErrRowLimitExceeded`）。

### B4【低】`PageParam` 与 XML 手写 `LIMIT` 共存无诊断

> ✅ **已于 v0.3.20 修复（运行时告警）**：`executePage`（`orm/base_mapper.go`）在派生 COUNT 前调用 `warnPageParamLimitConflict` —— 检测语句含**顶层** LIMIT/OFFSET 且方法声明 `*PageParam` 时 `log.Warnf`（每 namespace.id 仅告警一次）。选运行时而非注册期：SQL 是动态的（LIMIT 可能条件渲染），运行时检测准确。

XML 已写 `LIMIT #{limit} OFFSET #{offset}`（本项目 130 处）又声明 `*PageParam` 时，框架会再注入一次分页（B2 的正则先删再拼），过程中**不告警**。建议注册期检测「方法含 PageParam 且语句自带 LIMIT」→ Warn。

---

## C. 事务

### C1【高】`orm.Begin()` 全局单事务槽仍在，且没有 Deprecated 引导

> ✅ **已于 v0.3.21 修复（Deprecated + 运行时迁移告警）**：`orm.Begin`/`orm.BeginTx`（含 `DB.Begin`/`DB.BeginTx` 多数据源变体，`orm/transaction.go`）全部标记 `Deprecated` 指向 `WithTx(ctx, fn)`；首次使用时 `log.Warnf` 迁移提示（每进程一次）；docs/features.md 事务章节重写为 WithTx 优先 + 迁移指引。行为不变（存量代码零破坏）。

**框架证据**：`orm/transaction.go:15-40`
```go
// Begin 在全局连接上开启事务。
// 注意：事务绑定全局连接（单事务槽），请勿在多个 goroutine 中交错开启事务。
func Begin() (*Transaction, error) { ... db.setCurTx(t) ... }
```
v0.3.6 已提供正确解 `orm.WithTx(ctx, fn)`（`:88-120`，`beginDetachedTx` 不占全局槽、嵌套复用、多 goroutine 安全），但 `Begin/Commit/Rollback` 三件套既无 `Deprecated` 标记，也无迁移提示。

**本项目代价**：`backend-go/domain/evalscore/tx.go:10-34` 为此加了 `dbMu` 全局互斥，把"评分批次事务"整体串行；`domain/evalscore/runcontext.go:18-25` 记录"Java 侧并行 worker 因事务槽冲突被迫改串行"。

**建议**：`Begin/Commit/Rollback` 标 `Deprecated: 请使用 WithTx`；`Begin()` 内部改为"检测到已有 ctx 事务则复用"，或用 `gDbConn.BeginTx(ctx)` 强制要求 ctx；文档置顶迁移指引。

### C2【中】`WithTx` 与「Mapper 方法未声明 ctx」的组合无告警

> ✅ **已于 v0.3.21 修复（运行时告警）**：新增 `orm/tx_scope.go` —— `WithTx` 进入/退出时在当前 goroutine 打作用域标记（嵌套计数、panic 路径平衡）；Mapper 代理（executeMethod/executeStream/executePage）检测「语句 ctx 未携带事务且处于 WithTx 作用域内」→ `log.Warnf`（每 namespace.id 一次），提示该语句走独立连接、不随 WithTx 提交/回滚。开关：`orm.SetTxScopeWarn(false)` / `mybatis.configuration.tx-scope-warn=false`（默认开）。测试见 `orm/tx_scope_test.go`（含端到端：无 ctx 方法在 WithTx 回滚后仍留存的行为记录）。

`WithTx` 只对 `ExecContext/QueryContext/QueryToContext` 与「Mapper 方法声明 `context.Context` 参数」的执行路径生效。若方法签名没带 ctx（本项目 **0/1203 条语句**带 ctx），`WithTx` 内调用会**静默走非事务连接**——语义上等于事务失效，但无任何提示。

**建议**：`WithTx` 内首次执行未带 ctx 的 Mapper 方法时 `log.Warnf`（可用开关关闭），并在文档/注册期提示。

---

## D. API 类型安全与可观测性

### D1【中高】`NewMapper` 返回 `interface{}`，找不到即 panic，无 error 版本/泛型版本

> ✅ **已于 v0.3.22 修复**：新增 `orm/mapper_generic.go` —— `orm.Mapper[T](name) (T, error)`（类型安全 + error，T 支持值/指针/接口形态）、`orm.MustMapper[T](name) T`（显式 panic 命名）、`orm.NewMapperE(name) (interface{}, error)`；bindMapper 绑定 panic 一并转为 error（不再击穿调用栈）。`NewMapper` 原样保留（存量兼容）。

**框架证据**：`orm/orm_cache.go:104-111`
```go
func NewMapper(name string) interface{} {
	mp, err := gCache.createMapper(name)
	if err != nil { log.Warnf(...); panic(err) }   // ← panic
	...
}
```
- 无 `NewMapperE(name) (interface{}, error)`；
- 无泛型取用 `orm.Mapper[T](name)` / `orm.MustMapper[T](name)`。

**本项目代价**：`backend-go` 有 **120 处** `mybatis.NewMapper("X").(mybatis.XMapper)` —— 字符串名与类型名靠人肉同步，断言无 `ok` 检查（类型漂移即 panic）；项目还额外维护了 `SetMapperForTest` 全局覆盖表来支持单测。

**建议**：
```go
func Mapper[T any](name string) (T, error)      // 类型安全 + error
func MustMapper[T any](name string) T           // 失败 panic（显式命名）
```

### D2【中】无 Mapper / Model 注册表自省 API

> ✅ **已于 v0.3.22 修复**：新增 `orm.RegisteredMapperNames() []string`（去重排序的规范结构体名，跨包 init() 注册数量可自证）与 `orm.MapperFuncCount(name) (int, error)`（函数字段数，别名/大小写均可查）；见 `orm/orm_cache.go`。

找不到 `RegisteredMappers()` / `MapperNames()` / `MapperStats()` 之类接口。本项目只能把生成层 Mapper 数**硬编码**成常量（`backend-go/mybatis/init.go:34` `const genMapperCount = 11`，注释里已因增删手改两次：13 → 11），跨包 `init()` 注册的数量无法自证。

**建议**：`orm.RegisteredMapperNames() []string` / `orm.MapperFuncCount(name) int`；xml2go 的 `init()` 里维护包级计数并提供 `genmapper.Count()`（生成器层面也可解决）。

### D3【中高】行级转换失败只记日志，仍返回部分数据

> ✅ **已于 v0.3.22 修复**：新增 `orm.SetStrictResult(on)` / `orm.StrictResult()` + 配置 `mybatis.configuration.strict-result`（默认关）—— 开启后结果集转换不完整（Skipped>0 或 Errors 非空）的查询返回 `ErrResultConvertIncomplete`（`%w` wrap，`errors.Is` 可判）而非部分数据，且不进二级缓存；默认关闭保持旧行为，但新增 `orm.LastConvertReport(namespace, id)` 查询最近一次转换异常报告（仅异常语句记录）。见 `orm/result_convert.go` + `orm/base_mapper.go` executeMethod。

**框架证据**：`orm/base_mapper.go:395-400`
```go
results, report := convert2Results(rows, sqlFunc.Result)
if report.Skipped > 0 || len(report.Errors) > 0 {
	log.Errorf("result convert report [...]: total=%d converted=%d skipped=%d errors=%v", ...)
}
// 之后照常 return results, nil
```
比 v0.1.8 的"静默丢行"进步很多，但调用方仍拿不到"数据不完整"的信号（`err == nil`）。

**建议**：`orm.SetStrictResult(true)`（或语句级 `strictResult="true"`）时，把 `report.Errors` 非空转成返回 error；否则保持现状但提供 `orm.LastConvertReport()` 查询。

---

## E. 代码生成（xml2go / sqlc）

### E1【高】`CollectSqlSlots` 不采集 `<if>/<choose>/<foreach>` 体内占位符 → 生成签名丢动态参数

> ✅ **已于 v0.3.23 修复**：新增 `sqlfragment.CollectAllSqlSlots`（递归收集片段树全部占位符，含动态体）与 `SqlParam.AllSlots`；xml2go 签名推导不再仅依赖无条件位置 Slots——AutoDerive 语句存在「动态体专属占位符」（AllSlots ⊃ Slots，含占位符全在 `<if>/<choose>` 体内的语句）时退化为 `param map[string]interface{}` 按名取数（与 foreach 惯例一致），动态参数不再丢失；纯无条件占位符语句保持具名参数、纯无占位符语句保持无参不变。运行期 `Need` 推导与按位绑定（Slots）行为不变。

**框架证据**：`types/sqlfragment/node.go:250-269`（设计注释明说"条件体由运行时求值，不视为必参"）
```go
// 推导规则（1.1）：<if>/<choose>/<foreach> 体内的占位符不统计——条件体由运行时求值，
// 不视为必参（selectAll 类「占位符全在 if 体内」语句保持无参调用契约）；
```
该规则对**运行期 `Need` 推导**是合理的（避免注册失败），但被 xml2go **复用来推导函数签名** → 生成的方法只含"无条件位置"的参数。

**本项目后果（4 个 Mapper 被迫去生成化）**：`backend-go/scripts/gen_mappers_xml2go.py:58-64` 注释：
> `xml2go 的 CollectSqlSlots 不采集 <if>/<foreach> 体内占位符，导致生成的签名丢 lastPersonId/period/status/message 等动态参数（断点续跑、状态流转语义残缺），改由 mybatis/evalscore_extra.go 手写 struct 绑定同一份 XML`

**建议**：xml2go **不要复用** `CollectSqlSlots` 定签名，改为：
- 含 `<if>/<choose>/<foreach>` 的语句 → 生成 `func(params map[string]interface{})`（与项目惯例一致）；或
- 把 if 体内 slot 作为**可选具名参数**生成（`func(lastPersonId interface{}, ...)`），保持可读性。

### E2【中】跨 Mapper resultMap 引用退化为 `interface{}`

> ✅ **已于 v0.3.24 修复**：`GenerateXml2GoFiles` 先扫全量 XML 建 resultMap 全局索引（全限定 `namespace.id` + 短前缀 `Mapper短名.id` 两种键，同键先注册者生效），嵌套 association/collection 的限定名引用（`resultMap="OtherMapper.Xxx"`）静态解析为真实关联类型（`*SysDept` / `[]SysRole`），不再退化 `interface{}`；裸 id 跨 Mapper 引用保持退化（与 MyBatis 裸 id 仅本 Mapper 解析语义一致，避免同名 id 歧义）。见 `types/result_item.go` lookupNestedResultMap + `types/xml2go_gen.go`。

`docs/agents/xml2go.md` 已列为已知边界（`resultMap="OtherMapper.Xxx"` 静态不可解析 → 字段 `interface{}`）。对共享 Java XML 的多 Mapper 复用场景影响面不小。建议：支持 `-resultmap-index` 全局索引（先扫全量 XML 建 resultMap 表，再生成）。

### E3【中】namespace 冲突无诊断/别名机制

> ✅ **已于 v0.3.25 修复**：
> ① **重复 namespace 检测**：`NewSqlMappersFromSources` 加载期检测「同 namespace 多文件」，覆盖行为不变（后者覆盖前者）但 `log.Warnf` 列出全部文件，并记入 `SqlMappers.NamespaceConflicts`（orm 侧 `orm.NamespaceConflicts()` 查询）；
> ② **`MapperSource.NamespacePrefix`**：该来源所有 namespace 加 `prefix.` 前缀（函数 Owner 等 namespace 消费点一致），绑定键改「前缀+短名」且不注册裸短名键 —— 共享 Java XML 与本地同名 Mapper 并存互不覆盖，Go 结构体命名 前缀+短名（如 QualityPlatformDisplayMapper）即可绑定；
> ③ **`orm.RegisterNamespaceAlias(alias, name)`**（新增 `orm/namespace_alias.go`）：别名各查找键解析到与 name 相同的 `*SqlMapper` 并自动重绑全部结构体 —— namespace 改名/加前缀后旧引用不断；alias 覆盖既有不同 Mapper 仅告警。
> 见 `types/sql_mappers.go` + `types/sql_mapper.go` + `orm/namespace_alias.go`；测试 `types/namespace_e3_test.go` + `orm/e3_namespace_test.go`（别名/前缀 sqlite 端到端）。

`RegisterMapperFS` 是"替换全部 SQL 定义"语义（`orm/embed_fs.go:38-45`），同一 namespace 多来源仅"后者覆盖前者"（`:49`）。**没有**「同名 Mapper 结构体已存在」或「两文件同 namespace」的诊断。

**本项目代价**：`scripts/sync_mappers.py` 不得不把 Java 侧 `PlatformDisplayMapper.xml` 改名为 `QualityPlatformDisplayMapper.xml` 以避开 Go 端已有同名 Mapper；这类"改名抬进仓库"的做法对共享单一事实源是长期的维护负担。

**建议**：加载期检测重复 namespace → Warn/error；提供 `MapperSource{FS, Patterns, NamespacePrefix}` 或 `orm.RegisterNamespaceAlias(old, new)`。

---

## F. 方言层

### F1【中高】缺 MySQL→PG 族函数/语法翻译 ✅（v0.3.26 已修复）

框架在 `orm/`、`types/` 内**没有任何** MySQL 函数翻译（grep `ifnull|find_in_set|date_format|str_to_date` 零命中）。而共享 Java XML 里这些函数是常态，PG/金仓 PG 模式不认。

**本项目代价**：靠 `backend-go/scripts/gen_mybatis_go.py`（已失效）+ 手改 XML 副本 + 守卫脚本 `scripts/2026-10-09_check_mysql_dialect_in_mappers.py --strict` 三重兜底，且守卫只"拦截"不"修复"。

**建议**：`Dialector.FunctionRewrites() map[string]string` + `mybatis.configuration.function-rewrite=true`，内置 PG 族映射：
| MySQL | PG / 金仓 PG 模式 |
|---|---|
| `ifnull(a,b)` | `coalesce(a,b)` |
| `find_in_set(x, csv) > 0` | `position(','||x||',' in ','||csv||',') > 0` |
| `date_format(d, fmt)` | `to_char(d, fmt)`（fmt 需转换表） |
| `` `col` `` | `"col"` 或裸名 |
| `col = 0`（varchar 列） | `col = '0'`（需类型推断，可保守跳过） |

> **✅ 修复说明（v0.3.26）**：新增 `orm/dialector/mysql_compat.go` 翻译引擎（`RewriteMySqlCompat`，未按建议的简单 `map[string]string` 实现——`find_in_set`/`date_format` 需参数级重写，简单名称映射不够）+ `orm/function_rewrite.go` 开关（`orm.SetFunctionRewrite` / 配置 `mybatis.configuration.function-rewrite`，默认关）+ `DB.ExecContext/QueryContext/QueryRowContext` 三执行收口点挂载（全路径覆盖）。上表前 4 项已实现；`col = 0` 类型推断按建议保守跳过（文档明示边界，同 `find_in_set` 返回值序号差异）。仅 PG 族数据源生效，MySQL 族原生语法不动。详见 docs/features.md「MySQL 方言函数翻译」与 CHANGELOG v0.3.26。

### F2【中】`||` 跨模式语义差异无检测 ✅（v0.3.27 已修复）

金仓 MySQL 兼容模式下 `||` 是**逻辑或**，Oracle/PG 模式才是**字符串拼接**（`AGENTS.sql-rules.md` 已记录）。框架 `sql_rewrite` 不做任何检测，共享 XML 一旦带 `||` 就是"换个 database_mode 语义全变"。

**建议**：`mybatis.configuration.portability-check=true` 时扫描 `||` 并在非 Oracle/PG 模式下 Warn（或提供 `mybatis.sql-guards` 插件点，让使用方注册自定义守卫——本项目现有三个守卫脚本可迁进来）。

> **✅ 修复说明（v0.3.27）**：新增 `orm/portability_check.go`（`orm.SetPortabilityCheck` / 配置 `mybatis.configuration.portability-check`，默认关）+ 引号/注释感知扫描器 `dialector.ContainsConcatOp`；三个执行收口点（`DB.ExecContext/QueryContext/QueryRowContext`）检测：MySQL 族执行 SQL 含 `||` → 告警「默认逻辑或（除非 PIPES_AS_CONCAT）」；KingbaseES（框架无法感知实际 database_mode）含 `||` → 告警「语义随兼容模式变化」；纯 PG/Oracle 不告警。每条 SQL 仅告警一次（sync.Map 去重）。仅诊断不改写。`mybatis.sql-guards` 插件点未实现（现有守卫脚本迁入属使用方工程化范畴，非框架缺陷）。详见 CHANGELOG v0.3.27。

---

## G. 文档与生产默认值

### G1【中高，低成本】使用问题文档停留在 v0.1.7，多结论已过时且无废弃标注 ✅（v0.3.28 已修复）

`docs/mybatis-go使用问题与解决方案.md`（397 行，标注"版本：v0.1.7"）中至少 3 条已失效：

| 文档结论 | 现状 |
|---|---|
| P22「无分页支持（PageHelper 等价物缺失）→ 用内存分页」 | v0.3.0 M-07 已实现 `orm.PageParam` / `orm.Page`（`orm/page.go`、`orm/m07_page_test.go`） |
| P10「不支持 `!= 0` 数值比较，只能靠零值剔除」 | M-02 已支持数值比较（`if.go:355-421`）；v0.3.12 又补了字符串等值 |
| P4「PG/金仓不支持 `LastInsertId`，需 `currval` 回读」 | v0.1.12 已实现 RETURNING 回填（`orm/base_mapper.go`） |

**本项目已中招**：`backend-go/domain/monitor/common.go:52` 注释仍写"mybatis-go 无 PageHelper → 内存分页"；`mybatis/querymap.go:8-11` 注释仍写"mybatis-go 不支持 `!= 0` 求值"；`domain/system/common.go:103` 仍走 `currval` 回读。

**建议**：文档按版本小节重构（"≤v0.3.5 适用 / v0.3.6+ 已解决"），每条问题加 `Fixed in` 标记；README 顶部加"能力矩阵（按版本）"。

> **✅ 修复说明（v0.3.28）**：① 使用问题文档顶部加「文档状态」横幅（历史记录定位 + 指引新版能力）；7 个过时条目加 `✅ Fixed in` 标注（P4→v0.1.12 RETURNING、P8 坑1→v0.1.8 M-01、P10→v0.1.8 M-02 + v0.3.12、P14→v0.3.22 D3 可检测、P20/P21→v0.3.26 F1 自动翻译、P22→v0.3.0 M-07 分页），原文保留作历史记录；② README 顶部新增「能力矩阵（按版本）」表（9 项能力 × 最早版本 × 入口）；③ `backend-go` 使用方代码注释不属本仓库，待使用方自行按标注更新（currval 回读应在升级后删除，RETURNING 已是推荐路径）。未按建议做「按版本小节全量重构」（成本高且历史记录本身有价值，横幅 + 逐条标注已消除误导）。

### G2【中】生产级开关默认全关，无推荐预设 ✅（v0.3.29 已修复）

`safe-update`、`explain-slow-sql`、`pool-stats-interval`、`cache-enabled`、`pretty-sql` 默认全部关闭或 0（`docs/configuration.md`），而它们是"上线就该开"的防护/观测项。使用方往往不知道存在，直到出事。

**建议**：`orm.RecommendedProductionSettings() map[string]string` 一键生成推荐配置（safe-update=true、pool-stats-interval=60、explain-slow-sql=true + threshold），并在启动日志打印"当前未开启的推荐项"清单。

> **✅ 修复说明（v0.3.29）**：新增 `orm/production_settings.go`——① `orm.RecommendedProductionSettings()` 一键生成推荐配置（safe-update=true / pool-stats-interval=60 / explain-slow-sql=true / slow-sql-threshold=3000），使用方合并进配置 map 即可；② 启动期（`InitializeFromSettings`）自动打印"未开启的推荐配置项"清单（Warn 级，每进程一次，`UnrecommendedSettingItems` 纯函数可测）；③ 使用文档见 docs/features.md「生产推荐配置」。未把 `cache-enabled` 纳入推荐（缓存正确性依赖 DML flush 语义理解，保守由使用方显式开启）；默认值保持不变（存量行为零变化）。

---

## H. 长期技术选型

### H1【跟踪】lib/pq → pgx/v5 官方暂缓 ✅（v0.3.30 已按建议文档化）

`TODO.md` 记录「lib/pq 迁移 pgx/v5 已评估，当前 v1.12.3 仍可维护，收益有限暂缓」。影响本项目的能力缺口：
- 无 per-connection 会话保证 → `currval` 回读主键（当前用 `PlatformApiMapper.SelectCurrval` 兜底）不稳；
- 无 `COPY` 批量写入 → 大批量导入只能多行 `VALUES`；
- 无 `search_path` 之外的原生 session 参数能力。

**建议**：保持暂缓结论，但把"RETURNING 回填"作为 PG 族主键回填的**唯一推荐路径**写进文档（这样 currval 变通自然消亡）；未来若做 COPY，再评估 pgx。

> **✅ 处理说明（v0.3.30）**：暂缓结论保持（TODO.md 不变）；docs/features.md 新增「自增主键回填（RETURNING / LastInsertId）」章节——明确 **RETURNING 是 PG 族主键回填唯一推荐路径**、`currval` 回读属 v0.1.7 时代变通且连接池下不可靠（应删除）；使用问题文档 P4 标注同步指路（见 G1/v0.3.28）。使用方 `SelectCurrval` 兜底代码待其升级后自行移除。未来做 COPY 时再评估 pgx。

---

## 附：建议提交 upstream 的 issue 清单（按优先级，可直接复制标题）

1. **`<include>` 引用含动态子节点的 `<sql>` 片段时丢失静态文本，生成非法 SQL**（A2，附最小 XML）
2. **未识别的 `<if test>` 表达式被静默丢弃 → 条件恒真渲染，且无任何告警**（A3，附 5 种表达式形态）
3. **`<foreach>` 循环体仅支持纯文本，嵌套 `<if>`/`<choose>` 被丢弃或 panic**（A1，附 `elems[0].Val.(string)` 位置）
4. **`buildCountSQL` / `ApplyPagination` 使用全局正则清洗 SQL，误剥子查询内 `ORDER BY`/`LIMIT`**（B1/B2，附子查询 `LIMIT 1` 用例）
5. **全局行数上限无按语句覆盖，超限静默截断；`MyBatisSetting.MaxRows` 为未使用死字段**（B3）
6. **`CollectSqlSlots` 被 xml2go 复用于签名推导，导致动态 SQL Mapper 生成签名丢参**（E1，附本项目 4 个 Mapper 被迫手写的证据）
7. **`NewMapper` 仅返回 `interface{}` 且内部 panic，缺 `NewMapperE` / 泛型 `Mapper[T]`**（D1）
8. **`Begin/Commit/Rollback` 全局单事务槽应标 Deprecated 并引导 `WithTx`**（C1）
9. **行级结果转换失败仅 ERROR 日志，调用方无法得知数据不完整（缺 strict 选项）**（D3）
10. **方言层缺 MySQL→PG 族函数/语法翻译与 `||` 可移植性检测**（F1/F2）
11. **缺 Mapper 注册表自省 API（`RegisteredMapperNames`）**（D2）
12. **文档 `docs/mybatis-go使用问题与解决方案.md` 版本滞后，P4/P10/P22 结论已失效，需加 Fixed-in 标注**（G1）
13. **`WithTx` 内调用未声明 `context.Context` 的 Mapper 方法会静默逃逸事务，建议告警**（C2）
14. **namespace 重复加载无诊断、无别名/前缀机制**（E3）

---

## 证据索引（框架源码，v0.3.18）

| 条目 | 框架位置 |
|---|---|
| `<foreach>` 循环体仅纯文本 | `types/sqlfragment/foreach.go:17-25, 178-195` |
| `<include>` 丢静态文本 | `types/sqlfragment/include.go:39-56` |
| `<if test>` 表达式白名单 + 空条件恒真 | `types/sqlfragment/if.go:353-421, 181-190` |
| 无参 `<if>` 恒不渲染 | `types/sqlfragment/if.go:147-149` |
| COUNT 派生正则清洗 | `orm/sql_rewrite.go:23-38` |
| 分页 LIMIT/OFFSET 注入（先删后拼） | `orm/dialector/base.go:159-170` |
| 行数上限与静默截断 | `orm/sql_execute.go:20, 47-52, 169-176`；`orm/scan_rows.go:237,275,309`；`orm/row_stream.go:92` |
| `MaxRows` 死字段 | `orm/database_config.go:36`（全仓无读取点） |
| 全局事务槽 | `orm/transaction.go:15-40` |
| `WithTx`（不占槽） | `orm/transaction.go:62-120` |
| `NewMapper` panic + interface{} | `orm/orm_cache.go:104-122` |
| 注册表自省缺失 | 全仓无 `RegisteredMapper*` 导出函数 |
| 转换报告只记日志 | `orm/base_mapper.go:395-400` |
| `CollectSqlSlots` 规则 | `types/sqlfragment/node.go:250-269` |
| `RegisterMapperFS` 替换语义 | `orm/embed_fs.go:38-56` |
| 连接池 / 超时默认值 | `orm/dialector/base.go:17-18, 48-60`；`orm/database_config.go:158-159` |
