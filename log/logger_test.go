package log

import "testing"

// P0-2 / P1-3：日志级别行为回归测试。
// 默认 ConsoleLogger 为 Warn 级别：Debug 关闭（跳过昂贵的调试参数求值）、Warn/Error 可见；
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
	// 手动设 Enable=true 且未设 Level 时（零值 DebugLevel）应返回 true
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

// P1-3：默认日志级别为 Warn，Warn/Error 可见，Debug/Info 不可见
func Test_DefaultLevel(t *testing.T) {
	SetLogger(newConsoleLogger())
	if !WarnEnabled() || !ErrorEnabled() {
		t.Error("default logger should show warn/error")
	}
	if DebugEnabled() || InfoEnabled() {
		t.Error("default logger should hide debug/info")
	}
	// 显式设置为 Debug 级别后全部可见
	SetLogger(&ConsoleLogger{Enable: true, Level: DebugLevel})
	if !DebugEnabled() || !InfoEnabled() || !WarnEnabled() || !ErrorEnabled() {
		t.Error("debug-level logger should show all levels")
	}
	SetLogger(newConsoleLogger())
}
