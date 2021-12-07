package utils

import (
	"os"
	"strings"
)

func GetAllEnv() map[string]string {
	envMap := map[string]string{}
	for _, envLine := range os.Environ() {
		kv := strings.Split(envLine, "=")
		envMap[kv[0]] = kv[1]
	}
	return envMap
}
