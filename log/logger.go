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

type ConsoleLogger struct {
	Enable bool
}

func newConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{Enable: false}
}

func (log ConsoleLogger) IsDebugEnabled() bool {
	return log.Enable
}

func (log ConsoleLogger) Debugf(format string, args ...interface{}) {
	if log.Enable {

		fmt.Println(current(), "[DEBUG]", fmt.Sprintf(format, args...))
	}
}
func (log ConsoleLogger) Printf(format string, args ...interface{}) {
	if log.Enable {
		fmt.Println(current(), "[INFO]", fmt.Sprintf(format, args...))
	}
}
func (log ConsoleLogger) Infof(format string, args ...interface{}) {
	if log.Enable {
		fmt.Println(current(), "[INFO]", fmt.Sprintf(format, args...))
	}
}
func (log ConsoleLogger) Warnf(format string, args ...interface{}) {
	if log.Enable {
		fmt.Println(current(), "[WARN]", fmt.Sprintf(format, args...))
	}
}
func (log ConsoleLogger) Errorf(format string, args ...interface{}) {
	if log.Enable {
		fmt.Println(current(), "[ERROR]", fmt.Sprintf(format, args...))
	}
}

func current() string {
	return time.Now().Format("2006-01-02 15:04:05.000000000")
}

func SetLogger(logger Logger) {
	root = logger
}

// IsDebugEnabled 报告当前 Logger 是否启用调试日志。
// 未实现 debugEnabler 接口的自定义 Logger 保守返回 true，保证不丢失现有日志输出。
func IsDebugEnabled() bool {
	if d, ok := root.(debugEnabler); ok {
		return d.IsDebugEnabled()
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
