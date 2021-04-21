package logger

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
)

type SimpleFormatter struct {
	Colored bool
}

func (f *SimpleFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil{
		b = entry.Buffer
	}else{
		b = &bytes.Buffer{}
	}
	if f.Colored{
		switch entry.Level{
		case logrus.TraceLevel,logrus.DebugLevel:
			b.WriteString("\x1b[34;1m")
		case logrus.InfoLevel:
			b.WriteString("\x1b[32;1m")
		case logrus.WarnLevel:
			b.WriteString("\x1b[35;1m")
		case logrus.ErrorLevel,logrus.FatalLevel, logrus.PanicLevel:
			b.WriteString("\x1b[31;1m")
		}
	}
	b.WriteString(fmt.Sprintf("[%s] [%8s] [%20s] : %v",
		entry.Time.Format("2006-01-02 15:04:05.000"),
		entry.Level.String(),
		entry.Data["source"],
		entry.Message,
	))
	if f.Colored{
		b.WriteString("\x1b[0m")
	}
	b.WriteString("\n")
	return b.Bytes(),nil
}