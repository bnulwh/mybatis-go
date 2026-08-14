package log

import "testing"

// P0-2：IsDebugEnabled 行为回归测试。
// 默认 ConsoleLogger 关闭时返回 false（跳过昂贵的调试参数求值）；
// 自定义 Logger 未实现 debugEnabler 时保守返回 true，不丢失日志。

type plainLogger struct{}

func (p *plainLogger) Debugf(format string, args ...interface{}) {}
func (p *plainLogger) Printf(format string, args ...interface{}) {}
func (p *plainLogger) Infof(format string, args ...interface{})  {}
func (p *plainLogger) Warnf(format string, args ...interface{})  {}
func (p *plainLogger) Errorf(format string, args ...interface{}) {}

func Test_IsDebugEnabled(t *testing.T) {
	// 默认 ConsoleLogger：Enable=false
	SetLogger(newConsoleLogger())
	if IsDebugEnabled() {
		t.Error("default console logger (Enable=false) should report debug disabled")
	}
	// 开启后应返回 true
	SetLogger(&ConsoleLogger{Enable: true})
	if !IsDebugEnabled() {
		t.Error("console logger with Enable=true should report debug enabled")
	}
	// 未实现 debugEnabler 的自定义 Logger 保守返回 true
	SetLogger(&plainLogger{})
	if !IsDebugEnabled() {
		t.Error("logger without IsDebugEnabled should conservatively return true")
	}
	// 恢复默认，避免影响同进程其他测试
	SetLogger(newConsoleLogger())
}
