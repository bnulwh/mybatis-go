package logger

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"runtime"
	"strings"
)

const(
	DefaultCallerSkip int = 8
)

type contextHook struct {
	Field string
	Skip int
	levels []logrus.Level
}

func NewContextHook(levels ...logrus.Level) logrus.Hook {
	hook := contextHook {
		Field: "source",
		Skip: DefaultCallerSkip,
		levels: levels,
	}
	if len(hook.levels) == 0{
		hook.levels = logrus.AllLevels
	}
	return &hook
}

func (hook contextHook) Levels() []logrus.Level {
	return hook.levels
}

func (hook contextHook) Fire(entry *logrus.Entry) error {
	var pc uintptr
	var file string
	var line int 
	var ok bool
	for i:=0;i<10;i++{
		pc, file, line, ok = runtime.Caller(hook.Skip+i)
		if !ok{
			continue
		}
		if !strings.Contains(file,"logrus/"){
			break
		}
	}
	// ret := ""
	// if ok{
	fullFn := runtime.FuncForPC(pc)
	fullFnName := fullFn.Name()
	pos := strings.LastIndex(fullFnName,".")
	// fnNames := strings.Split(fullFnName,".")
	ret := fmt.Sprintf("%v : %v : %v()",filepath.Base(file),line, fullFnName[pos+1:])
	// }
	entry.Data[hook.Field] = ret
	return nil
}
