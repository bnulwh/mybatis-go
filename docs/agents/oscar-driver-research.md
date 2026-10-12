# 神通 Oscar Go 驱动可行性调研

> 调研日期：2026-10-12（信创适配 Task 0 spike，对应计划 `docs/agents/plans/2026-10-12-xinchuang-databases.md`）

## 结论（A / B / C 三选一）

**B：无纯 Go 驱动，仅有 cgo + ACI（私有客户端库）路径 → 不进主库，README 文档化为「规划中」，关闭本轮适配。**

- 社区存在 `database/sql` 兼容的 Go 驱动（见下表），但全部为 **cgo 包装 ACI（Oscar Call Interface）**，编译期依赖神舟通用的私有客户端库 `libaci`（来自数据库安装目录 `$SZ_OSCAR_HOME/aci/`）+ `pkg-config` + C 工具链。
- Oscar 使用**自有通信协议**（ACI 类似 Oracle 的 OCI，但非 Oracle TNS 线协议），`sijms/go-ora` 无法直连，PG/MySQL 族驱动均不可复用。
- mybatis-go 现有依赖全部为纯 Go、支持 `CGO_ENABLED=0` 交叉编译，且本轮适配硬约束为「零新增依赖、Go 1.24」；cgo + 私有库路径与该约束冲突，故**不建议进主库**。
- 若未来确有 Oscar 需求，可行路径为**独立可选模块**（如 `orm/dialector/oscar_cgo/` + 构建标签 `//go:build cgo`，或独立 module），由使用方自带 ACI 环境自行编译。本轮不做。

## 调研证据

| 渠道 | 来源 | 结果 |
|------|------|------|
| pkg.go.dev 搜索 `shentong` | https://pkg.go.dev/search?q=shentong | 15 个模块：`Mystery00/go-shentong`（官方流出、cgo+ACI）、`team-ide/go-driver/db_shentong`（cgo+ACI 封装）、`yuluo-yx/go-shentong`（前者的维护 fork，cgo+ACI，内置多平台 libaci）、若干 gorm 方言封装 |
| pkg.go.dev 搜索 `oscar database` | https://pkg.go.dev/search?q=oscar+database | 0 结果（"oscar" 关键词无直接命中，实际驱动均以 shentong 命名） |
| GitHub `Mystery00/go-shentong` | https://github.com/Mystery00/go-shentong | 自述「神通数据库官方驱动——从官方版本拿过来的，没有持续更新」；README 明确要求 aci.pc + libaci.so/aci.dll（cgo）；v0.0.2（2022-11）后停更，0 star |
| GitHub `yuluo-yx/go-shentong` | https://github.com/yuluo-yx/go-shentong | 维护中的 fork（v0.0.5，2026-06）：同样 cgo + ACI，内置 amd64/arm64/loong64/mips64/sw64/win64 客户端库；**要求 Go 1.25.8**（> 项目固定 1.24，不满足约束）；提供 Docker E2E（`teamide/shentong:7.0.8`，端口 2003） |
| 官网 shentongdata.com.cn | http://shentongdata.com/ | 「下载与服务」仅提供产品文档下载；驱动（JDBC/ODBC/ACI C SDK）随数据库安装介质分发，**无公开纯 Go 驱动** |
| Oracle TNS 协议兼容性 | 各驱动 README / pkg.go.dev 文档 | Oscar 走自有协议（默认端口 2003，ACI 接口风格类似 OCI）；无任何 TNS 兼容证据，`go-ora` 不可复用 |

## 关键技术参数（供未来可选模块参考）

- 驱动 import path：`github.com/Mystery00/go-shentong`（MIT，官方流出停更版）或 `github.com/yuluo-yx/go-shentong`（维护 fork，Go 1.25.8）
- 驱动注册名：`aci`（须先调用 `shentong.Init()`，再 `sql.Open("aci", dsn)`）
- DSN 格式：`[用户名/密码@]主机[:端口][/数据库名][?参数]`，示例 `sysdba/szoscar55@127.0.0.1:2003/OSRDB?questionph=true`
- 占位符风格：默认 Oracle 风格 `:n`；DSN 参数 `questionph=true` 切换为 `?`（适配 `database/sql` 惯用写法）
- 默认端口 / 默认库：`2003` / `OSRDB`；Docker 验证镜像 `teamide/shentong:7.0.8`
- 编译依赖：CGO + C 工具链 + `pkg-config` + 私有 `libaci`（`$SZ_OSCAR_HOME/aci/` 或 fork 内置 `third_party/shentong/aci/`），Windows 需 `aci.dll`/`aci.lib`/`pthreadVC2_x64.dll`
- `database/sql` 接口完整性：已由 pkg.go.dev 文档核实（Driver/Connector/Conn/Ping/BeginTx/Stmt/Rows/Tx/CheckNamedValue 齐全）；**连通性标注「待真机验证」**（本机无 Oscar 实例，且无 CGO+libaci 工具链，编译 spike 未执行——结论不依赖该项）

## 后续行动

1. 本轮（信创四库适配）**关闭 Oscar 主库适配**；README 多数据库清单维持现状（不新增 Oscar 条目）。
2. 若用户提出真实 Oscar 需求：另立计划做「可选 cgo 模块」——`orm/dialector` 增加 `OscarDb` 常量 + `NewForType` 路由照常进主库（纯类型/路由，无驱动依赖），dialector 实现与驱动注册放入 `//go:build cgo` 保护的独立包/模块，由使用方自行启用。
3. 长期跟踪：若神舟通用发布官方纯 Go 驱动或社区出现纯协议实现（无 cgo/私有库），按 add-dialector.md 的 Oracle 方言族路径（参照 `oceanbase_oracle.go` 模式）重新评估立项。
