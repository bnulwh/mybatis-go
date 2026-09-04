package types

import (
	"os/exec"
	"testing"
)

// Test_GeneratedFiles_gofmtClean：codegen 产物必须 gofmt 干净且语法合法（4.2）。
// 覆盖 samples（RuoYi 共享 XML：无 parameterType、resultType=map、变参等边界）。
// 回归：历史上 resultType=map → "models.map[string]interface {}"、无 parameterType →
// "models." 两类非法签名（修复见 generateDefine）。
func Test_GeneratedFiles_gofmtClean(t *testing.T) {
	dir := t.TempDir()
	mps := NewSqlMappers("../samples")
	mps.GenerateFiles(dir, "src")
	out, err := exec.Command("gofmt", "-l", dir).CombinedOutput()
	if err != nil {
		t.Errorf("gofmt -l failed (generated code has syntax errors): %v\n%s", err, out)
		return
	}
	if len(out) > 0 {
		t.Errorf("generated files not gofmt-clean:\n%s", out)
	}
}
