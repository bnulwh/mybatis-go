package utils

import (
	"fmt"
	"testing"
)

func TestGetAllEnv(t *testing.T) {
	mp := GetAllEnv()
	fmt.Println(mp)
	if len(mp) == 0 {
		t.Error("test GetAllEnv failed.")
	}
}
