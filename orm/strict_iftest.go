package orm

import (
	"github.com/bnulwh/mybatis-go/types/sqlfragment"
)

// SetStrictIfTest 设置 <if test> 严格模式（A3，mybatis.configuration.strict-if-test，
// 默认关）。开启后：
// ① 注册期：任一语句存在未识别 test 表达式时 RegisterMapper 返回 error
//
//	（配合默认严格注册模式；宽松注册模式下仅记 error 日志跳过该检查）；
//
// ② 求值期：整个 test 均未识别的 <if>/<when> 按 false 处理（不渲染），不再恒真。
// 默认（非严格）保持存量行为：未识别表达式静默丢弃、空条件恒真（解析期已 Warn）。
func SetStrictIfTest(on bool) { sqlfragment.SetStrictIfTest(on) }

// StrictIfTest 返回 <if test> 严格模式开关状态（A3）。
func StrictIfTest() bool { return sqlfragment.StrictIfTest() }
