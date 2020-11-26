package logger

import (
	"encoding/json"
	"fmt"
	log "github.com/astaxie/beego/logs"
	"os"
	"path/filepath"
)

const logPath = "/var/log"

type LogConfig struct {
	Filename string   `json:"filename"`
	Separate []string `json:"separate"`
	MaxLines int      `json:"maxlines"`
	MaxSize  int      `json:"maxsize"`
	MaxDays  int      `json:"maxdays"`
	Level    int      `json:"level"`
	Daily    bool     `json:"daily"`
}

func Initialize(logName string) {
	log.EnableFuncCallDepth(true)
	log.SetLogFuncCallDepth(3)
	if !pathExists(logPath) {
		fmt.Printf("dir: %s not found.", logPath)
		err := os.MkdirAll(logPath, 0711)
		if err != nil {
			fmt.Printf("mkdir %s failed: %v", logPath, err)
		}
	}
	c := &LogConfig{
		Filename: filepath.Join(logPath, logName),
		Separate: []string{"emergency", "alert", "critical", "error", "warning", "notice", "info", "debug"},
	}
	value, _ := json.Marshal(c)
	err := log.SetLogger(log.AdapterMultiFile, string(value))
	if err != nil {
		fmt.Println(err)
	}
	err = log.SetLogger(log.AdapterConsole, `{"level":7}`)
	if err != nil {
		fmt.Println(err)
	}
}
func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}
