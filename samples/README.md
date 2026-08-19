# threedb.jar 提取的 MyBatis Mapper XML

从 `threedb.jar`（`BOOT-INF/classes/`）原样提取的 MyBatis Mapper XML（含全部 SQL），
供接口对齐改造（backend-go）与 SQL 分析参考。提取脚本见仓库 `extract_api.py` 同目录说明。

> 提取日期：2026-08-18　|　共 23 个 XML / **166 条 SQL**
> 数据库方言：人大金仓 KingbaseES（`com.kingbase8.Driver`），部分 SQL 含金仓特有语法
> 表命名：RuoYi 标准结构（`sys_user`、`sys_role`、`sys_menu` 等）

## 目录

### mybatis/system（RuoYi 系统管理，16 个 Mapper / 154 条 SQL）

| Mapper | SQL 数 | 说明 |
|--------|--------|------|
| SysUserMapper | 12 | 用户 CRUD、按用户名查询、用户列表（含部门/角色联查）、改密 |
| SysRoleMapper | 12 | 角色 CRUD、按用户查角色、角色列表联查 |
| SysMenuMapper | 16 | 菜单树、按用户查菜单/权限、路由 |
| SysDeptMapper | 14 | 部门树、子部门、联查 |
| SysDictDataMapper | 10 | 字典数据 CRUD、按类型查 |
| SysDictTypeMapper | 9 | 字典类型 CRUD |
| SysConfigMapper | 8 | 参数配置 CRUD、按 key 查 |
| SysNoticeMapper | 6 | 通知公告 CRUD、标记已读 |
| SysNoticeReadMapper | 7 | 已读记录 |
| SysPostMapper | 11 | 岗位 CRUD |
| SysJobMapper | 7 | 定时任务 CRUD |
| SysJobLogMapper | 7 | 任务日志 |
| SysUserRoleMapper / SysRoleMenuMapper / SysRoleDeptMapper / SysUserPostMapper | 4+4+4+4 | 多对多关联 |

### mybatis/monitor（监控，2 个 Mapper / 9 条 SQL）

| Mapper | SQL 数 | 说明 |
|--------|--------|------|
| SysLogininforMapper | 4 | 登录日志（记录/查询/清理） |
| SysOperLogMapper | 5 | 操作日志（记录/查询/清理） |

### mybatis/tool（代码生成，2 个 Mapper / 17 条 SQL）

| Mapper | SQL 数 | 说明 |
|--------|--------|------|
| GenTableMapper | 11 | 代码生成表信息 |
| GenTableColumnMapper | 6 | 代码生成表字段 |

### mapper/mdm（MDM 数据源，2 个 Mapper / 3 条 SQL）

| Mapper | SQL 数 | 说明 |
|--------|--------|------|
| MdmOrgRawMapper | 1 | 组织原始数据（ESB） |
| MdmStaffRawMapper | 2 | 人员原始数据（ESB） |

### 其他

- `mybatis-config.xml` — MyBatis 全局配置（驼峰映射、分页插件等）
- `logback.xml`、`pom.xml` — 非 SQL，未提取

## 与 backend-go 改造的关联

实施改造方案（`docs/generate/2026-08-18_backend-go接口对齐jar接口实施改造方案.md`）中
P0/P1 模块的 SQL 参考均来自本目录：
- 建表结构 → 参照各 Mapper 的 resultMap/column
- 查询逻辑 → 参照 select/join 片段
- 注意金仓 ↔ MySQL 方言差异（`SELECT 1 FROM DUAL`、`LIMIT` 语法等）
