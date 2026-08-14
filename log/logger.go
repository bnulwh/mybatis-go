package log

import (
	"fmt"
	"time"
)

type Logger interface {
	Debugf(format string, args ...interface{})
	Printf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

var root Logger = newConsoleLogger()

// debugEnabler 可选接口：实现者可通过 IsDebugEnabled 精确控制调试日志开销
// （避免日志级别关闭时仍执行昂贵参数求值，如整结果集 JSON 序列化）。
type debugEnabler interface {
	IsDebugEnabled() bool
}

// LogLevel 日志级别：数值越小越详细。
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

type ConsoleLogger struct {
	Enable bool
	Level  LogLevel
}

// newConsoleLogger 默认开启 Warn/Error 级别：
// 保证不调用 SetLogger 时错误和警告也可见（不再静默吞日志）。
func newConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{Enable: true, Level: WarnLevel}
}

func (log ConsoleLogger) isEnabled(level LogLevel) bool {
	return log.Enable && log.Level <= level
}

func (log ConsoleLogger) IsDebugEnabled() bool {
	return log.isEnabled(DebugLevel)
}

func (log ConsoleLogger) Debugf(format string, args ...interface{}) {
	if log.isEnabled(DebugLevel) {
		fmt.Println(current(), "[DEBUG]", fmt.Sprintf(format, args...))
	}
}
func (log ConsoleLogger) Printf(format string, args ...interface{}) {
	if log.isEnabled(InfoLevel) {
		fmt.Println(current(), "[INFO]", fmt.Sprintf(format, args...))
	}
}
func (log ConsoleLogger) Infof(format string, args ...interface{}) {
	if log.isEnabled(InfoLevel) {
		fmt.Println(current(), "[INFO]", fmt.Sprintf(format, args...))
	}
}
func (log ConsoleLogger) Warnf(format string, args ...interface{}) {
	if log.isEnabled(WarnLevel) {
		fmt.Println(current(), "[WARN]", fmt.Sprintf(format, args...))
	}
}
func (log ConsoleLogger) Errorf(format string, args ...interface{}) {
	if log.isEnabled(ErrorLevel) {
		fmt.Println(current(), "[ERROR]", fmt.Sprintf(format, args...))
	}
}

func current() string {
	return time.Now().Format("2006-01-02 15:04:05.000000000")
}

func SetLogger(logger Logger) {
	root = logger
}

// levelEnabler 可选接口：ConsoleLogger 通过它按级别报告开关状态。
type levelEnabler interface {
	isEnabled(level LogLevel) bool
}

// IsDebugEnabled 报告当前 Logger 是否启用调试日志。
// 未实现 levelEnabler 的自定义 Logger 保守返回 true，保证不丢失现有日志输出。
func IsDebugEnabled() bool {
	if d, ok := root.(debugEnabler); ok {
		return d.IsDebugEnabled()
	}
	return isLevelEnabled(DebugLevel)
}

// DebugEnabled / InfoEnabled / WarnEnabled / ErrorEnabled 按级别查询当前 Logger 开关状态。
func DebugEnabled() bool { return isLevelEnabled(DebugLevel) }
func InfoEnabled() bool  { return isLevelEnabled(InfoLevel) }
func WarnEnabled() bool  { return isLevelEnabled(WarnLevel) }
func ErrorEnabled() bool { return isLevelEnabled(ErrorLevel) }

func isLevelEnabled(level LogLevel) bool {
	if e, ok := root.(levelEnabler); ok {
		return e.isEnabled(level)
	}
	return true
}

func Debugf(format string, args ...interface{}) {
	root.Debugf(format, args...)
}
func Printf(format string, args ...interface{}) {
	root.Printf(format, args...)
}
func Infof(format string, args ...interface{}) {
	root.Infof(format, args...)
}
func Warnf(format string, args ...interface{}) {
	root.Warnf(format, args...)
}
func Errorf(format string, args ...interface{}) {
	root.Errorf(format, args...)
}
