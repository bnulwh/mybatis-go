# mybatis-go 优化点整理

> 整理时间：2026-09-04 ｜ 基于代码版本：mybatis-go **v0.1.15**（bnulwh/mybatis-go）
> 素材来源：本仓库（三库平台 Go 后端）实际接入 mybatis-go 0.1.15 过程中实测发现的问题 + 源码阅读 + 官方 TODO.md 脉络
> 场景背景：Go 后端共享 Java 侧 Mapper XML，连接人大金仓 KingbaseES（MySQL 兼容模式），多环境 schema/表前缀差异（dev=public+threedb_，prod=subsp+无前缀）

---

## 一、参数绑定与渲染层（与 Java MyBatis 语义对齐）

### 1.1 【高】`parameterType` 硬依赖导致"有参函数注册失败"（实测踩坑）
- **现状**：`types/sql_param.go::parseSqlParamFromXmlAttrs` 只有 XML 显式声明 `parameterType` 才置 `Need=true`；否则 `orm/param_type.go::checkSql` 判定"函数不需要参数"，对**有参数**的 Go 函数直接报
  `xxx check sql function xxx failed, not need func args` 并中止 Mapper 注册。
- **实测影响**：三库项目 32 个 XML 中约 **199 处** `<select|insert|update|delete>` 因缺 `parameterType` 触发注册失败（EvalDataMapper 29 处、PlatformApiMapper 等共 170 处），全部需手工补齐才能启动服务。
- **根因**：Java MyBatis 里 `#{xxx}`/动态标签天然从方法参数（@Param/POJO）取数，不需要 `parameterType` 也能工作；Go 端 codegen 生成的函数一律带参数（`func(params map[string]interface{})`），与"共享 Java XML"冲突。
- **建议**：注册期改为**从 SQL 文本/节点自动推导 Need**：语句含 `#{}`、`${}` 占位符或 `<if>/<foreach>/<choose>/<where>/<set>` 动态节点 → 视为需要参数（按 map 处理）；仅当语句纯静态且无占位符时才允许无参绑定。彻底去掉对 `parameterType` 属性的依赖，与 Java 语义对齐。
- **收益**：共享 Java XML 零改造；codegen 函数签名与 XML 自动匹配；注册失败面消失。
- **风险**：`<if test>` 含常量表达式（如 `<if test="1==1">`）误判需参数——可在推导时仅统计 `#{}/${}` 占位符与 `<foreach>`，`<if>/<choose>` 条件不视为必参。

### 1.2 【高】BaseSqlParam 多占位符全部绑定 `args[0]`（实测踩坑）
- **现状**：`types/sql_fragments.go` 的 `simpleSql.prepareSqlWithParam/generateSqlWithParam` 中，**语句内每个 `#{a}`、`#{b}` 占位符都用同一个 `m`（即 `args[0]`）替换**，后续参数被丢弃。
- **实例**：`SELECT * FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = #{schema} AND TABLE_NAME = #{tableName}`，调用 `SelectTableColumns("gzwsk", "sys_user")`，两个占位符都渲染为 `'gzwsk'`。
- **根因**：`effectiveParamType` 只按 `args[0]` 分派（String→BaseSqlParam、map→MapSqlParam），且 Base/Map 渲染路径只认单个参数对象；多参数只能走 map 包装（struct tag `args:"a,b"` 已解析但**渲染层未使用 TagArgs 按名绑定**）。
- **建议**：`GenerateSQL/PrepareSQL` 入口增加"多参数 + TagArgs"路由：当 `len(args)>1 && TagArgsLen==len(args)` 时，将 args 按 TagArgs 名字打包为 map 再走 `prepareSqlWithMap/generateSqlWithMap`；同时 `validParam` 校验 TagArgs 与占位符一致性。
- **收益**：多参函数（JDBC 式 `#{a}#{b}`）正确绑定，去掉"同名替换"隐性 bug。
- **风险**：低；打包路径与既有 MapSqlParam 一致，仅新增入口分支。

### 1.3 【中】变参函数 + tag 长度不符直接 panic（实测踩坑）
- **现状**：`orm/param_type.go::makeParamType` 中 `tagArgsLen > funcType.NumIn()` 时 **panic**（`the tag "args" length can not > arg length !`）。
- **实例**：`SelectTableColumns func(args ...interface{}) \`args:"schema,tableName"\``，变参反射 `NumIn=1`（`[]interface{}`），tag 长度 2 > 1 → panic 崩掉整个服务启动。
- **建议**：① `IsVariadic` 时不适用该长度校验（按变参元素个数或跳过后绑定）；② 该类错误从 `panic` 改为返回 `error`（与框架其它注册错误一致，便于调用方处理而非进程崩溃）。
- **收益**：健壮性；错误可观测。

### 1.4 【低】`resultType="map"` 解析刷 warning 噪音
- **现状**：每次加载 XML 语句都打 `unsupport type to parse: map`（`parse sql result` 阶段，对 map 类型仅 warning 后走通用路径）。一次 Init 数万行日志被这类噪音淹没。
- **建议**：map 识别为合法通用类型（或降级 debug）；`checkSql`/`return_type` 相关"失败"信息补充**namespace + statement id + XML 路径**，便于定位（目前只有函数名）。

---

## 二、数据表名前缀改写（table-prefix，0.1.13/0.1.15）

### 2.1 【中】缺少"前缀移除/替换"能力（逆场景）
- **现状**：0.1.15 语义=按配置前缀**正向添加**（辅以真实表集合判断"带前缀存在→改写 / 无前缀存在→保持原样"），**没有"把硬编码前缀剥成无前缀"的能力**。
- **实例**：三库 prod 是 `subsp` schema + **无前缀**物理表；若 XML 仍硬编码 `threedb_x`，配置空前缀时保持原样 → 必然 42P01。本项目最终靠**批量改写 XML 剥前缀**绕开（679 处），而非框架能力。
- **建议**：支持**前缀映射/替换**配置，如 `mybatis.table-prefix-map = threedb_:`（旧前缀→新前缀，空表示移除），改写时先按映射翻译再叠加表集合校验；与现有 `table-prefix` 并存（后者为纯添加）。
- **收益**：存量硬编码前缀 XML 无需人工批量修改即可切生产；两套命名规范并行时一处配置切换。
- **风险**：映射规则与表集合判定的组合需要仔细定义优先级（建议：映射结果优先于集合，映射值空 时走集合判定）。

### 2.2 【低】表关键字集合有限
- **现状**：仅识别 `FROM / JOIN / INTO / UPDATE / TABLE`；`MERGE INTO`、`CREATE INDEX ... ON`、MySQL `RENAME TABLE a TO b` 目标名等不加前缀（已知边界）。
- **建议**：补充 `MERGE INTO`、`CREATE INDEX ... ON`、`RENAME TABLE ... TO`、`INSERT OVERWRITE` 等常见位置。

### 2.3 【低】嵌套 CTE 名称未收集
- **现状**：顶层 CTE（含 RECURSIVE/NOT MATERIALIZED）已收集；**子查询内再 `WITH`** 的名称未收集，命中会误加前缀（显式 SQL 错误，非静默错数据）。
- **建议**：递归收集嵌套 CTE；或在文档中明确"不建议在 Mapper 子查询内嵌套 WITH"。

### 2.4 【低】表集合缓存刷新依赖本进程 DDL
- **现状**：表名集合按数据源惰性缓存，**仅本进程执行 DDL 后失效**；外部会话新建/改名表不会主动刷新（需重启或执行任意 DDL 触发）。
- **建议**：增加可配置 TTL（如 `mybatis.table-prefix-set-ttl=1h`）或按"schema+库"周期性弱校验（低成本 `count` 探测），降低多会话/发布场景下的陈旧集合窗口。

### 2.5 【低】改写动作无可观测性
- **现状**：`applyTablePrefix` 改写前后 SQL 无日志/埋点，排障只能靠 DB 报错反推。
- **建议**：debug 级打印 `[table-prefix] rewrite: <before> -> <after>`（或仅在命中表集合改判时打印），附带数据源名；便于多环境排查"为何没改写/改写成什么"。

---

## 三、schema 配置（0.1.14，已实用）

### 3.1 【已满足】schema 三来源配置 + search_path 注入
- `spring.datasource.schema` 键 > JDBC URL（`currentSchema`/`search_path`/`schema`）；PG/Kingbase DSN 自动追加 `search_path=<schema>`；表结构查询按 schema 过滤。
- **实测**：三库项目 `spring.datasource.schema=public` 验证通过（`ods_hive_table_metadata` 等查询正确命中 public；表集合匹配按 schema 独立）。
- **建议**：README 补充**多数据源 × schema × 前缀**的矩阵示例（现网既有 spring.datasource.<name>.table-prefix 继承，schema 是否也逐源覆盖需明确说明——建议支持 `spring.datasource.<name>.schema` 对称能力）。

### 3.2 【低】SQL 内显式 `schema.table` 限定名与改写的关系需文档化
- 现状：非 `public`/`main` 的 schema 限定名（如 `subsp.sys_user`）整体跳过前缀改写（防跨库误改）。与 `search_path` 方案共存时，应明确推荐"SQL 一律写无 schema 名 + search_path 路由"的单一约定，避免混用。

---

## 四、codegen / schema2code 工具链（本项目痛点）

### 4.1 【高】生成的 XML 不带 `parameterType`
- **现状**：`schema2code`/generator 生成 Mapper XML（含 MP CRUD）**不写 `parameterType`**，与 1.1 的校验组合后：任何"有参函数 + 无 parameterType 语句"都注册失败 → 本项目被迫手工补 ~199 处。
- **建议**：① 生成 XML 时按参数推断写入 `parameterType`（map→`map`、标量→`Long/String`、切片→`array`）；② 若采纳 1.1 的自动推导，则 codegen 只需保证"有参则生成有参函数"与"占位符存在"一致性即可。
- **收益**：生成物开箱即用，消灭人工修补。

### 4.2 【低】生成 Go 文件不满足 gofmt
- **现状**：`mappers_gen.go`/`models_gen.go` 等生成文件字段使用固定宽度对齐（非 gofmt tab 对齐），`gofmt -l` 大片报出；每次 CI/IDE 格式检查噪音大。
- **建议**：生成流程末尾 `gofmt -w` 统一格式化。

### 4.3 【低】codegen 对动态 SQL 的签名推断边界
- 现状：`<foreach>` 自动生成切片签名（已实现）；`<if>` 引用不存在字段时静默不渲染（与 Java 一致）。
- **建议**：codegen 对 `<if test>` 引用的字段做静态检查并告警，减少"拼出空 where"类问题。

---

## 五、健壮性与运维

### 5.1 【中】注册期错误中止策略
- 现状：`RegisterMapper` 遇第一个失败函数即返回 error，`Init` 整体失败（本项目一次性被 29+ 个"签名不匹配"阻塞，只能逐个修）。
- **建议**：增加"宽松注册"模式（`orm.SetStrictRegister(bool)`）：strict=false 时对失败函数记录 error 日志并跳过（该函数运行时返回明确错误），其余函数正常注册——便于大 XML 接入时先跑起来、后逐个修。

### 5.2 【低】debug 日志过载
- 现状：XML 解析/绑定逐条刷屏（`begin parse mapper file`、`--begin/--finish parse sql param`、`++begin/++finish parse sql fragment` 阶层式），单次 Init 数万行 debug。
- **建议**：热路径采用汇总日志（如每文件 1 行 `parsed N statements in Xms`）；`unsupport type` 类 warning 降级；保留 `IsDebugEnabled()` 守卫（已实现，但调用点仍多）。

### 5.3 【低】prepared-stmt 缓存上限与退化路径
- 现状：`maxPreparedStmts=100`，满后降级直接执行（README 已声明）。
- **建议**：降级事件加**统计计数/告警日志**（当前静默），便于发现缓存命中率劣化（如大量动态 SQL 文本导致缓存抖动）。

---

## 六、官方 TODO 中"暂缓/待选"项（供决策）

- **lib/pq → pgx/v5 迁移**：官方已评估，结论"v1.12.3 仍可维护，收益有限暂缓"。注意：pgx/v5 原生支持 `search_path`/`currentSchema` 启动参数、`?` 占位符（可省去 formatSQL `?→$n` 转换）。若未来需要"更好连接池/多语句 COPY/简化方言层"，可重新评估。
- **`?` 占位符方言转换**：当前 PG/Kingbase 无条件 `?→$n`，但金仓 MySQL 兼容模式部分场景可接受 `?`——保持现状即可（统一转换更稳）。

---

## 七、本次实测的 Demo/复现（三库项目内）

| 问题 | 复现场景 | 现网处理 |
|---|---|---|
| 1.1 注册失败 | `EvalDataMapper.selectValidPersonnel` 等 199 处缺 parameterType | 批量补 `parameterType="map"/"String"/"Long"` |
| 1.2 多占位符绑定 | `PlatformDisplayMapper.SelectTableColumns(#{schema},#{tableName})` | 签名改定参（仍受 1.2 限制，SQL 为 MySQL 方言原本金仓不可用，留待框架修复） |
| 1.3 变参 panic | `SelectTableColumns func(args ...interface{})` + tag | 改为 `func(schema string, tableName string)` |
| 前缀差异 | prod=subsp+无前缀 vs dev=public+threedb_ | XML 剥 679 处前缀 + `KB_SCHEMA/KB_TABLE_PREFIX` 环境化（见仓库 2026-09-04 提交与 docs/problems） |

---

## 优先级建议

- **P0（阻断类，建议下版本修）**：1.1 parameterType 自动推导、1.3 panic→error、2.1 前缀移除/替换能力
- **P1（稳定性/可用性）**：1.2 多参数按 TagArgs 绑定、5.1 宽松注册、4.1 codegen 带 parameterType
- **P2（体验/可观测）**：1.4/4.2/5.2 日志与格式、2.2~2.5 前缀细节、3.1/3.2 文档矩阵、5.3 降级告警