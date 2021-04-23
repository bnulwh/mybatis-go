package logger

import (
	"github.com/bnulwh/logrus"
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/pkg/errors"
	"io"
	"path"
	"time"
)

func createFileLogger(level, logPath string) (*rotatelogs.RotateLogs, error) {
	prefix := ""
	if len(level) > 0 {
		prefix = "." + level
	}
	return rotatelogs.New(
		logPath+prefix+".%Y%m%d%H%M.log",
		rotatelogs.WithLinkName(logPath+".log"),
		rotatelogs.WithMaxAge(time.Hour*24*7),
		rotatelogs.WithRotationTime(time.Hour),
	)
}

func ConfigLocalFileSystemLogger(logPath, logFileName string) {
	baseLogPath := path.Join(logPath, logFileName)
	debugWriter, err := createFileLogger("debug", baseLogPath)
	infoWriter, err := createFileLogger("info", baseLogPath)
	warnWriter, err := createFileLogger("warn", baseLogPath)
	errorWriter, err := createFileLogger("error", baseLogPath)
	commonWriter, err := createFileLogger("", baseLogPath)
	multiErrorWriter := io.MultiWriter(errorWriter, commonWriter)
	if err != nil {
		logrus.Errorf("config local file system logger error: %+v", errors.WithStack(err))
	}
	lfHook := NewLocalFileSystemHook(WriterMap{
		logrus.DebugLevel: io.MultiWriter(debugWriter, commonWriter),
		logrus.InfoLevel:  io.MultiWriter(infoWriter, commonWriter),
		logrus.WarnLevel:  io.MultiWriter(warnWriter, commonWriter),
		logrus.ErrorLevel: multiErrorWriter,
		logrus.FatalLevel: multiErrorWriter,
		logrus.PanicLevel: multiErrorWriter,
	}, &logrus.SimpleFormatter{})
	//logrus.AddHook(NewContextHook())
	logrus.AddHook(lfHook)
	//logrus.SetFormatter(&logrus.SimpleFormatter{Colored: true})
}
