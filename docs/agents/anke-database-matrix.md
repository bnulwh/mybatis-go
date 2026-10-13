# 安全可靠测评数据库支持情况矩阵

> 调研日期：2026-10-12；更新：2026-10-13 — 7 款零成本别名落地（v0.3.17 / v0.3.18）。覆盖安全可靠测评公告中的集中式 + 分布式数据库产品（23 个送测项 / 19 款去重：达梦 2 项合 1 款、GoldenDB 3 项合 1 款、SeaboxSQL 2 项合 1 款）。
>
> 相关文档：信创适配计划 `docs/agents/plans/2026-10-12-xinchuang-databases.md`、安可零成本别名方案 `docs/agents/plans/2026-10-12-anke-zero-cost-aliases.md`、Oscar 调研 `docs/agents/oscar-driver-research.md`、新增方言手册 `docs/agents/add-dialector.md`。

## 总览

| 支持状态 | 数量 | 产品 |
|----------|------|------|
| ✅ 已支持 | 16 项（12 款） | 达梦（集中式/分布式）、PolarDB MySQL、GoldenDB Lite/V7/EBASE、OceanBase 集中式、GaussDB 分布式；**TaurusDB、UPDRDB、Kingwow 金乌、TeleDB、DWS、AnalyticDB PG、SeaboxSQL（集中式+分布式）**（后 7 款为 v0.3.17/v0.3.18 新增） |
| 🟡 需换驱动/待实测 | 2 款 | 磐维 PanweiDB、GBase 8c |
| ❌ 无路径/不适用 | 5 款 | YashanDB、RASESQL、ArgoDB、DolphinDB、TimechoDB |

零成本别名方案（v0.3.17 MySQL 族 + v0.3.18 PG 族）落地后，按款计覆盖率 **12/19（63%）**；若后续引入 openGauss 官方纯 Go 驱动解决磐维/GBase 8c 的 sha256 认证问题，可达 **14/19（74%）**。
（原文「若完成 7 个零成本别名 + openGauss 驱动引入可达 17/19（89%）」统计有误，已修正——款去重口径下原已支持为 5 款而非 7 款。）

---

## ✅ 一、已支持（8 项）

| # | 产品 | 送测单位 | 现有 dialector | 方言族 | 备注 |
|---|------|----------|---------------|--------|------|
| 集-1 | 达梦数据库管理系统 V9 | 武汉达梦 | `dameng` | Oracle 方言族 | 完整支持 |
| 分-1 | 达梦数据库管理系统（分布式版）V9 | 武汉达梦 | `dameng` | Oracle 方言族 | 与集中式同驱动 |
| 集-3 | 阿里云 PolarDB 数据库（MySQL 版）V2.0 | 阿里云 | `polardb` | MySQL 兼容族 | 完整支持 |
| 集-6 | GoldenDB Lite 数据库 V7 | 中兴通讯 | `goldendb` | MySQL 兼容族 | MySQL 协议，内嵌 MySqlDialector |
| 分-4 | GoldenDB 数据库 V7 | 中兴通讯 | `goldendb` | MySQL 兼容族 | 同上 |
| 分-14 | GoldenDB EBASE V3 | 中兴通讯 | `goldendb` | MySQL 兼容族 | GoldenDB 家族，MySQL 协议 |
| 集-7 | OceanBase 数据库软件（集中式版）V4 | 北京奥星贝斯 | `oceanbase` / `oceanbase-oracle` | MySQL 模式 + Oracle 模式 | 双 dialector 覆盖两种兼容模式 |
| 分-3 | GaussDB V3.0（分布式版） | 华为云 | `gaussdb` | PG 兼容族 | ⚠️ 走 lib/pq；openGauss 默认 sha256 认证下连不上，需部署侧启用 md5 认证（同下文「openGauss 认证边界」） |
| 集-8 | 华为云 TaurusDB V3.0 | 华为云 | `taurusdb` | MySQL 兼容族 | ✅ v0.3.17（2026-10-13）；100% MySQL 兼容，内嵌 MySqlDialector |
| 分-7 | 中国银联 UPDRDB V2 | 中国银联 | `updrdb` | MySQL 兼容族 | ✅ v0.3.17；MySQL 协议，内嵌 MySqlDialector |
| 分-15 | Kingwow（金乌）数据库 V6 | 北京自然原数 | `kingwow` | MySQL 兼容族 | ✅ v0.3.17；⚠️ 低置信度（社区来源），待真机验证 |
| 分-6 | 天翼云 TeleDB V5.1.9 | 天翼云 | `teledb` | PG 兼容族 | ✅ v0.3.18；端口 15432 为示例值（官方未公布默认端口） |
| 分-8 | 华为云 DWS 数据仓库 V8 | 华为云 | `dws` | PG 兼容族 | ✅ v0.3.18；⚠️ 须 MD5 认证账户（lib/pq 不支持 DWS SHA256） |
| 分-9 | 阿里云 AnalyticDB PostgreSQL 版 V2.0 | 阿里云 | `adbpg` | PG 兼容族 | ✅ v0.3.18；OLAP 场景 |
| 集-4/分-12 | 海盒 SeaboxSQL V21（集中式+分布式） | 东方金信 | `seaboxsql` | PG 兼容族 | ✅ v0.3.18；⚠️ 待真机验证 |

## 🔵 二、零成本别名 —— 已全部落地（v0.3.17 / v0.3.18）

原「零成本可支持」7 款已按 `docs/agents/plans/2026-10-12-anke-zero-cost-aliases.md` 全部完成，移入上文「已支持」。落地明细：

- **v0.3.17（MySQL 族，commit `7b1c632` + `7d9cf0b`）**：TaurusDB（`taurusdb`）、UPDRDB（`updrdb`）、Kingwow 金乌（`kingwow`，低置信度标注）
- **v0.3.18（PG 族，commit `1675f11`）**：TeleDB（`teledb`，端口示例值）、DWS（`dws`，MD5 认证注记）、AnalyticDB PG（`adbpg`，OLAP）、SeaboxSQL（`seaboxsql`，待真机验证）
- 实现模式与前序 goldendb/greatdb/antdb 一致：内嵌 MySql/PostgresDialector + init 驱动别名注册（`go-sql-driver/mysql` / `lib/pq`），零新增依赖；专属测试入 `orm/dialector/mysql_family_test.go` / `pg_family_test.go`，表驱动条目在 kingbase_test.go；每款含 demo 三件套（`cmd/<db>demo`）

## 🔵 二、零成本别名调研依据（7 款，已全部落地）

以下为立项时的协议/端口/置信度调研依据，供溯源（落地状态见「已支持」表）。

| # | 产品 | 协议依据 | 接入族 | 默认端口 | 置信度 |
|---|------|----------|--------|----------|--------|
| 集-8 | 华为云 TaurusDB V3.0 | 100% 兼容 MySQL（[官方产品页](https://www.huaweicloud.com/product/taurusdb.html)） | MySQL 族 | 3306 | 高（官方） |
| 分-7 | 中国银联 UPDRDB V2 | MySQL 协议，通用 MySQL 客户端可连（[银联云文档](https://doc.yun.unionpay.com/tcloud/Database/UPDRDB/quickstart/connectinstance)） | MySQL 族 | 3306（官方） | 高（官方） |
| 分-6 | 天翼云 TeleDB V5.1.9 | 完全兼容 PostgreSQL 协议（[天翼云产品页](https://www.ctyun.cn/products/topic/xcteledb)；开源版 [OpenTeleDB](https://gitee.com/teledb/openteledb) 基于 PG 17） | PG 族 | 未公开（示例 15432） | 高（官方） |
| 分-8 | 华为云 DWS 数据仓库 V8 | PG libpq 协议（[官方 FAQ](https://support.huaweicloud.com/dws_faq/dws_03_0017.html)） | PG 族 | 8000（官方） | 高（官方），**须在文档注明：集群须启用 MD5 认证，开源 PG 驱动不支持 DWS 的 SHA256** |
| 分-9 | 阿里云 AnalyticDB PostgreSQL 版 V2.0 | PG 协议，Greenplum 系 MPP（阿里云帮助文档） | PG 族 | 5432 | 高（官方），OLAP 场景注记 |
| 集-4/分-12 | 海盒 SeaboxSQL V21（集中式+分布式） | PostgreSQL 血统（第三方实测 + 姊妹产品 SeaboxMPP 使用 postgresql JDBC 驱动） | PG 族 | 5432（推断） | 中（社区/间接），待真机验证 |
| 分-15 | Kingwow（金乌）数据库 V6 | MySQL 线协议，可直接用 MySQL 客户端/JDBC 驱动访问（MODB 百科） | MySQL 族 | 3306（推断） | 中（社区来源），接入时标注低置信度 |

## 🟡 三、需引入 openGauss 官方驱动 / 待实测（2 款）

### openGauss 认证边界（两款共性问题）

mybatis-go 现有 `opengauss`/`gaussdb` dialector 走 `github.com/lib/pq`。openGauss 系数据库默认启用 **SHA256/SM3 认证**，原生 lib/pq 不支持；仅当部署侧配置 `password_encryption_type` 为 md5 时可连。彻底解法是引入 openGauss 官方纯 Go 驱动 **`gitee.com/opengauss/openGauss-connector-go-pq`**（lib/pq 的 fork，MIT，适配 SHA256/SM3，驱动注册名 `opengauss`，无 CGO）。

| # | 产品 | 情况 | 动作 |
|---|------|------|------|
| 集-5 | 磐维数据库 V3.0（中移动） | 内核基于华为 openGauss（多来源一致） | 现有 `opengauss` dialector 在 md5 认证下可直接连；默认 sha256 认证需引入 openGauss-connector-go-pq。端口 5432，占位符 `$n`，PG 方言全复用 |
| 分-10 | 南大通用 GBase 8c V6 | openGauss 系（高置信：DBCOMPATIBILITY 参数、gs_preinstall/gs_install 工具链、WDR Snapshot 等均为 openGauss 特征；厂商未公开声明） | 官方仅提供 JDBC/ODBC，无 Go 驱动；候选 openGauss-connector-go-pq 或 lib/pq，**需真机实测**（认证方式是关键变量）。社区示例端口 15400，占位符 `$n` |

## ❌ 四、无路径 / 不适用（5 款）

| # | 产品 | 结论 | 依据 |
|---|------|------|------|
| 分-2 | 崖山分布式数据库 YashanDB V23（深圳计算科学研究院） | **同 Oscar 结论 B**：Oracle 兼容（1:1 替换 Oracle 定位），官方驱动 `github.com/yashan-technologies/yashandb-go`（Apache-2.0）为 **CGO + 崖山 C 客户端库**，非纯 Go，与本项目「零新增依赖、纯 Go」约束冲突，不进主库 | 官方 GitHub + pkg.go.dev；默认端口 1688，DSN `sys/password@127.0.0.1:1688`（若未来做可选 cgo 模块可参考） |
| 集-2 | 平安 RASESQL V3（平安科技） | PG 生态兼容（安全可靠测评公告 + 公开词条），但**无公开的驱动/端口/协议文档**（官网不可达，GitHub 零结果） | 需商务渠道获取 JDBC/ODBC 驱动文档后验证 lib/pq 直连，再评估立项 |
| 分-11 | 星环 Transwarp ArgoDB V6 | **不适用**：OLAP 分析型引擎，HiveServer2/Inceptor SQL 系（Thrift 私有协议），无任何 database/sql Go 驱动 | 官方文档；与 MyBatis 风格 OLTP ORM 场景不符 |
| 分-13 | DolphinDB V2.0（浙江智臾） | **无路径**：自有 RPC 协议，官方 Go 客户端 `github.com/dolphindb/api-go` 为自定义 API（RunScript 模式），非 database/sql；默认端口 8848 | 官方文档 |
| 分-5 | 工业时序数据库 TimechoDB V2.0（天谋科技） | **无路径**：基于 **Apache IoTDB**（天谋为 IoTDB 核心团队创立，非 TDengine），私有 RPC 协议；官方 Go 客户端 `github.com/apache/iotdb-client-go` 为 Session API，非 database/sql；默认端口 6667 | 官方产品页 + Apache 仓库 |

---

## 后续行动建议

1. ~~批量零成本别名~~ **已完成**（v0.3.17 / v0.3.18，commit `7b1c632` + `7d9cf0b` + `1675f11`，见 `docs/agents/plans/2026-10-12-anke-zero-cost-aliases.md`）。
2. **openGauss 驱动替换评估**：引入 `openGauss-connector-go-pq` 替换 opengauss/gaussdb dialector 的 lib/pq，一次性解决 PanweiDB/GBase 8c/GaussDB 三款的 sha256 认证问题（需评估：新增一个纯 Go 依赖 vs 保持零依赖让用户配置 md5 认证）。
3. **真机连通性验证清单**：Kingwow、SeaboxSQL（提升置信度）；GBase 8c（认证方式为关键变量）；TeleDB（确认默认端口）。
4. **YashanDB / RASESQL**：按 Oscar 先例关闭主库适配；RASESQL 待商务资料到位后重开评估。
5. 本矩阵随支持状态变化同步更新（新增别名后在对应表格行标注 ✅ 并移入「已支持」段）。
