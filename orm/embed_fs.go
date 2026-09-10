package orm

import (
	"io/fs"

	"github.com/bnulwh/mybatis-go/types"
)

// MapperSource 是 types.MapperSource 的别名，便于调用方无需额外 import types 包。
// FS 为 nil 时表示磁盘文件系统；Patterns 为目录（递归）或单个 .xml 文件路径。
type MapperSource = types.MapperSource

// 本文件提供 go:embed 内嵌 Mapper XML 的加载入口。
// 传统方式要求 XML 以真实文件落在磁盘上（mybatis.mapper-locations 指向目录），
// 单文件二进制部署时需在运行时把内嵌 XML 解出到临时目录；以下 API 让 embed.FS
// 的内容直接参与解析，无需落盘。
//
// 典型用法：
//
//	//go:embed resources/mapper/*.xml
//	var mapperXML embed.FS
//
//	func init() {
//	    if err := orm.RegisterMapperFS(mapperXML, "resources/mapper"); err != nil {
//	        log.Fatalf("load mapper failed: %v", err)
//	    }
//	    orm.RegisterMapper(new(mapper.SysUserMapper))
//	}
//
// 加载顺序无关：RegisterMapperFS 只负责装载 XML 定义，RegisterMapper 负责注册
// Go 结构体并触发绑定；先注册结构体、后加载 XML 同样可用（InitializeFromSettings
// 之后的 RegisterMapper 会自动重新绑定）。

// RegisterMapperFS 从 embed.FS（或其他只读文件系统）加载全部 Mapper XML。
// patterns 每一项既可以是目录（递归收集其下全部 .xml，与 mybatis.mapper-locations
// 语义一致），也可以是单个 .xml 文件路径。可为多个目录/文件混合传入。
//
// 注意：这会替换当前已加载的全部 SQL 定义（与 Initialize 的行为一致）；
// 若需在磁盘目录之外**追加**内嵌 Mapper，请使用 RegisterMapperSources。
func RegisterMapperFS(fsys fs.FS, patterns ...string) error {
	if fsys == nil {
		return nil
	}
	return gCache.initSqlsFromFS(fsys, patterns...)
}

// RegisterMapperSources 合并多个 Mapper 来源（磁盘目录 / embed.FS / 内存）一次性加载。
// FS 为 nil 的条目按磁盘路径处理，与 mybatis.mapper-locations 行为一致；
// 同一 namespace 在多个来源中重复出现时，后列出的来源覆盖先列出的。
//
// 典型用法（内嵌 Mapper 为正式来源，磁盘目录提供本地热修覆盖）：
//
//	err := orm.RegisterMapperSources(
//	    orm.MapperSource{FS: mapperXML, Patterns: []string{"resources/mapper"}},
//	    orm.MapperSource{Patterns: []string{"extra/mapper"}}, // 磁盘覆盖目录
//	)
func RegisterMapperSources(sources ...MapperSource) error {
	return gCache.initSqlsFromSources(sources)
}

// ReloadMappers 按当前已加载的 XML 定义重新执行一次 Mapper 绑定。
// 在 RegisterMapperFS 之前已调用过 RegisterMapper 的场景下，用于手动补绑（通常无需调用：
// RegisterMapper 自身在 XML 已就绪时会自动重新绑定）。
func ReloadMappers() error {
	if gCache.sqls == nil {
		return nil
	}
	return gCache.bindSqls()
}
