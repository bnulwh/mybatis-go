package orm

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// D1：Mapper[T] / MustMapper[T] / NewMapperE —— 类型安全 + error，绑定 panic 转错误。
func Test_Mapper(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	// 值类型取用 + 正常调用
	mp, err := Mapper[G0TestMapper]("G0TestMapper")
	if err != nil {
		t.Errorf("Mapper[G0TestMapper] failed: %v", err)
		return
	}
	if _, err := mp.Insert(context.Background(), G0TestModel{Name: "mapper_generic"}); err != nil {
		t.Errorf("insert via Mapper[T] failed: %v", err)
	}

	// 指针类型取用
	mpp, err := Mapper[*G0TestMapper]("G0TestMapper")
	if err != nil {
		t.Errorf("Mapper[*G0TestMapper] failed: %v", err)
		return
	}
	if mpp == nil {
		t.Error("Mapper[*G0TestMapper] should not be nil")
	}

	// 类型不匹配 → error 而非 panic
	if _, err := Mapper[TxScopeTestMapper]("G0TestMapper"); err == nil {
		t.Error("Mapper with mismatched type should return error")
	}

	// 未注册 → error
	if _, err := Mapper[G0TestMapper]("NoSuchMapper"); err == nil {
		t.Error("Mapper for unregistered name should return error")
	}
}

func Test_MustMapper(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	mp := MustMapper[G0TestMapper]("G0TestMapper")
	if _, err := mp.Insert(context.Background(), G0TestModel{Name: "must_mapper"}); err != nil {
		t.Errorf("insert via MustMapper failed: %v", err)
	}
	// 未注册 → panic
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustMapper for unregistered name should panic")
			}
		}()
		MustMapper[G0TestMapper]("NoSuchMapper")
	}()
}

func Test_NewMapperE(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	v, err := NewMapperE("G0TestMapper")
	if err != nil {
		t.Errorf("NewMapperE failed: %v", err)
		return
	}
	if _, ok := v.(G0TestMapper); !ok {
		t.Errorf("NewMapperE should return mapper value, got %T", v)
	}
	if _, err := NewMapperE("NoSuchMapper"); err == nil {
		t.Error("NewMapperE for unregistered name should return error")
	}
}

// D2：RegisteredMapperNames / MapperFuncCount。
func Test_RegisteredMapperNames(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	names := RegisteredMapperNames()
	found := false
	for _, n := range names {
		if n == "G0TestMapper" {
			found = true
		}
	}
	if !found {
		t.Errorf("G0TestMapper should be in registered names %v", names)
	}
}

func Test_MapperFuncCount(t *testing.T) {
	if initG0Test(t) == "" {
		return
	}
	defer Close()

	cnt, err := MapperFuncCount("G0TestMapper")
	if err != nil {
		t.Errorf("MapperFuncCount failed: %v", err)
		return
	}
	if cnt != 2 {
		t.Errorf("G0TestMapper func count = %d, want 2 (Insert/SelectAll)", cnt)
	}
	// 大小写/别名均可查
	if _, err := MapperFuncCount("g0testmapper"); err != nil {
		t.Errorf("MapperFuncCount with lowercase name failed: %v", err)
	}
	if _, err := MapperFuncCount("NoSuchMapper"); err == nil {
		t.Error("MapperFuncCount for unregistered name should return error")
	}
}

// D3 端到端：列级转换失败 —— 非严格返回部分数据 + LastConvertReport 可查；
// 严格模式返回 ErrResultConvertIncomplete。
func Test_StrictResult(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="StrictConvMapper">
  <select id="selectBad" resultType="StrictConvModel">
    select id, name as ns from t_d3
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "StrictConvMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return
	}
	dbPath := filepath.Join(dir, "d3.db")
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return
	}
	defer Close()
	if _, err := Execute(`CREATE TABLE t_d3 (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	for i := 0; i < 2; i++ {
		if _, err := Execute(`INSERT INTO t_d3 (name) VALUES (?)`, "x"); err != nil {
			t.Errorf("insert failed: %v", err)
			return
		}
	}
	RegisterModel(new(StrictConvModel))
	if err := RegisterMapper(new(StrictConvMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("StrictConvMapper").(StrictConvMapper)

	// 默认（非严格）：返回部分数据（行保留、失败列保持零值），err == nil
	rows, err := mp.SelectBad()
	if err != nil {
		t.Errorf("non-strict select should return partial data, got err: %v", err)
		return
	}
	if len(rows) != 2 {
		t.Errorf("non-strict select should keep all rows, got %d", len(rows))
	}
	// D3：LastConvertReport 可查
	report := LastConvertReport("StrictConvMapper", "selectBad")
	if report == nil {
		t.Error("LastConvertReport should record conversion errors in non-strict mode")
		return
	}
	if len(report.Errors) == 0 {
		t.Error("LastConvertReport should contain column errors")
	}

	// 严格模式：返回 ErrResultConvertIncomplete
	SetStrictResult(true)
	defer SetStrictResult(false)
	if _, err := mp.SelectBad(); !errors.Is(err, ErrResultConvertIncomplete) {
		t.Errorf("strict select should return ErrResultConvertIncomplete, got %v", err)
	}
	// 报告仍然可查（记录最后一次）
	if LastConvertReport("StrictConvMapper", "selectBad") == nil {
		t.Error("LastConvertReport should be available after strict failure")
	}
}

// D3：配置解析 —— mybatis.configuration.strict-result（默认关）。
func Test_ParseStrictResult(t *testing.T) {
	cm := map[string]string{
		"spring.datasource.url":               "jdbc:sqlite:test.db",
		"mybatis.configuration.strict-result": "true",
	}
	cfg := NewConfigFromSettings(cm)
	if !cfg.Setting.StrictResult {
		t.Error("strict-result=true should parse to true")
	}
	cfg2 := NewConfigFromSettings(map[string]string{
		"spring.datasource.url": "jdbc:sqlite:test.db",
	})
	if cfg2.Setting.StrictResult {
		t.Error("default strict-result should be false")
	}
}

// StrictConvMapper D3 测试用 Mapper：ns 列（TEXT）无法转换到 chan int，
// 每行产生列级转换错误（行保留、字段零值）。
type StrictConvMapper struct {
	BaseMapper
	SelectBad func() ([]StrictConvModel, error)
}

type StrictConvModel struct {
	Id int64    `db:"id,pk"`
	Ns chan int `db:"ns"`
}
