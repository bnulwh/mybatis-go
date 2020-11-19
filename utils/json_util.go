package utils

import (
	"encoding/json"
)

func ToJson(v interface{}) string {
	dt, err := json.Marshal(v)
	if err != nil {
		fmt.Printf("to json failed: %v", err)
		return ""
	}
	return string(dt)
}
