# 代码生成

## generator — 从 XML Mapper 生成 Go 代码

```bash
go build -o generator cmd/generator/main.go
./generator -p mypackage -d temp -m resources/mapper
```

参数说明：
- `-p` 包名（默认 `temp`）
- `-d` 输出目录（默认 `temp`）
- `-m` XML Mapper 文件目录（默认 `resources/mapper`）

## schema2code — 从数据库表结构生成代码

```bash
go build -o schema2code cmd/schema2code/main.go
./schema2code -type mysql -host localhost -port 3306 -username root -password 123456 -db mydb -output temp
# MyBatis-Plus 内置 CRUD：加 -mp 生成 BaseMapper 标准方法名（insert/deleteById/updateById/selectById/selectList/selectOne/selectPage/selectCount/selectBatchIds/deleteBatchIds）
./schema2code -type postgres -host localhost -port 5432 -username root -password 123456 -db mydb -prefix sys_ -tables sys_user -output temp -mp
```

参数说明：
- `-type` 数据库类型：`mysql` / `postgres` / `kingbase` / `sqlite` / `tidb` / `tdsql` / `polardb` / `opengauss` / `gaussdb` / `highgo` / `vastbase` / `oceanbase` / `oceanbase-oracle` / `dameng` / `gbase8s` / `mssql` / `oracle` / `db2`
- `-host` 数据库地址
- `-port` 端口
- `-username` / `-password` 认证信息（SQLite 无需填写）
- `-db` 数据库名（SQLite 为 `.db` 文件路径）
- `-output` 输出目录
- `-prefix` 可选，表名前缀
- `-tables` 可选，指定表名（逗号分隔），为空则生成全部表
- `-mp` 可选，生成 MyBatis-Plus 内置 CRUD XML（BaseMapper 标准方法名；批量方法自动生成切片签名，如 `DeleteBatchIds func([]int64)`；`SelectCount` 返回 `[]int64`）

## xml2go — 从 XML Mapper 逆向生成 Go 代码

```bash
go build -o xml2go cmd/xml2go/main.go
# 入口一：已知 Mapper XML 目录
./xml2go -m resources/mapper -d gen -p github.com/xxx/app
# 入口二：直接给 MyBatis/MyBatis-Plus 配置文件（先解析 mapper 位置再生成）
./xml2go -c src/main/resources/application.yml -d gen -p github.com/xxx/app
```

参数说明：
- `-c` mybatis/mybatis-plus 配置文件（.properties/.yml/.yaml/.xml），先从配置解析 Mapper XML 位置（`mybatis[-plus].mapper-locations`、`<mappers><mapper resource/>`、`<property name="mapperLocations">`；支持 `classpath*:` 前缀、`*`/`**` 通配、逗号多值、`config-location` 链式解析）再生成；设置时 `-m` 被忽略
- `-m` XML Mapper 目录（默认 `resources/mapper`）
- `-d` 输出目录（默认 `gen`，内含 `models/`、`mapper/` 两个子目录）
- `-p` 输出目录自身的模块导入路径前缀（生成 import 为 `<prefix>/models` 等；留空退化为裸导入，GOPATH 兼容）
- `-skip-mp` 可选，跳过 MyBatis-Plus 内置 CRUD 字段
- `-v` 可选，打印跳过语句及原因

生成代码用法与手写 Mapper 一致（`mapper.GetSysUserMapper().SelectUserByUserName("admin")`）；完整签名推导规则与已知边界见 **docs/agents/xml2go.md**。

## sqlc — 从静态 select 生成类型安全 Querier

```bash
go run ./cmd/sqlc -m resources/mapper -d querier -p querier
```

参数说明：
- `-p` 输出包名（默认 `querier`）
- `-d` 输出目录（默认 `querier`）
- `-m` XML Mapper 文件目录（默认 `resources/mapper`）
- `-v` 可选，打印跳过语句及原因

扫描 XML Mapper，把静态 `<select>`（无动态标签、无 `${}`）抽取为类型安全的 Querier 接口 + 实现（model / `XxxQuerier` 接口 / `NewXxxQuerier(db)` 构造，一个 mapper 一个文件）；免连库——resultMap 按 jdbcType、resultType 按声明类型定型，标量参数按占位符名具名定型；动态语句自动跳过（`-v` 查看原因）；生成代码走 `QueryToContext`（方言占位符转换 / 表名前缀 / ctx 事务全部由 orm 层承担）。与 xml2go 互补——动态语句/MP CRUD 选 xml2go，纯静态查询选 sqlc。
