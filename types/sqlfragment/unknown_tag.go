package sqlfragment

import "sync/atomic"

// strictUnknownTags 未注册标签严格模式开关（P0-3，mybatis.configuration.strict-unknown-tag，
// 默认关）。开启后注册期存在未注册标签即报错；默认只在注册期按语句汇总 Warn，
// 避免「XML 里写了 <selectKey>/<bind> 之类未支持标签 → 标签内容被静默丢弃」。
var strictUnknownTags atomic.Bool

// SetStrictUnknownTags 设置未注册标签严格模式。
func SetStrictUnknownTags(on bool) { strictUnknownTags.Store(on) }

// StrictUnknownTags 返回未注册标签严格模式开关状态。
func StrictUnknownTags() bool { return strictUnknownTags.Load() }
