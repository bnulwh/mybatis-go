package types

import (
	"os/exec"

	"github.com/bnulwh/mybatis-go/log"
)

// gofmtFile 对生成文件执行 gofmt -w（4.2）：生成器固定宽度对齐不符合 gofmt，
// 每次 CI/IDE 格式检查大量报出；生成写盘后统一格式化，失败仅 Warn 不阻塞生成。
func gofmtFile(path string) {
	if path == "" {
		return
	}
	out, err := exec.Command("gofmt", "-w", path).CombinedOutput()
	if err != nil {
		log.Warnf("gofmt %v failed: %v: %s", path, err, out)
	}
}