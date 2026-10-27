package orm

import "github.com/bnulwh/mybatis-go/types/sqlfragment"

// SetStrictUnknownTags 设置未注册标签严格模式（P0-3，mybatis.configuration.strict-unknown-tag，
// 默认关）。开启后注册期存在未注册 XML 标签（如 <selectKey>、写错的标签名）即报错；
// 默认仅在注册期按 mapper.语句 汇总 Warn，避免「启动成功、运行期才发现 SQL 缺了一段」。
func SetStrictUnknownTags(on bool) { sqlfragment.SetStrictUnknownTags(on) }

// StrictUnknownTags 返回未注册标签严格模式开关状态。
func StrictUnknownTags() bool { return sqlfragment.StrictUnknownTags() }
