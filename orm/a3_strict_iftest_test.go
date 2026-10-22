package orm

import (
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/bnulwh/mybatis-go/types"
)

// A3：<if test> 严格模式——注册期报错与配置解析。

type a3StrictMapper struct {
	BaseMapper
	SelectUsers func(params map[string]interface{}) ([]map[string]interface{}, error) `args:"params"`
}

func a3StrictFS() fstest.MapFS {
	return fstest.MapFS{
		"m/A3StrictMapper.xml": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="A3StrictMapper">
  <select id="selectUsers" resultType="map">
    select * from t_user
    <if test="name != null and a == b">and name = #{name}</if>
  </select>
</mapper>`)},
	}
}

// Test_bindSql_StrictIfTestError 严格 if-test 开启时，注册期（mapperInfo.bindSql）
// 对存在未识别表达式的语句返回 error。
func Test_bindSql_StrictIfTestError(t *testing.T) {
	defer SetStrictIfTest(StrictIfTest())
	defer SetStrictRegister(IsStrictRegister())

	mps := types.NewSqlMappersFromSources(types.MapperSource{FS: a3StrictFS(), Patterns: []string{"m"}})
	smp := mps.NamedMappers["A3StrictMapper"]
	if smp == nil {
		t.Error("A3StrictMapper missing")
		return
	}

	// 默认（非严格 if-test）：绑定成功，仅告警，行为不变
	SetStrictIfTest(false)
	SetStrictRegister(true)
	info := newMapperInfo(reflect.TypeOf(a3StrictMapper{}))
	if err := info.bindSql(smp); err != nil {
		t.Errorf("default mode bindSql should succeed (warn only), got %v", err)
	}

	// 严格 if-test + 严格注册：报错并列出未识别表达式
	SetStrictIfTest(true)
	info2 := newMapperInfo(reflect.TypeOf(a3StrictMapper{}))
	err := info2.bindSql(smp)
	if err == nil {
		t.Error("strict if-test should fail bindSql on unsupported expressions")
		return
	}
	if !strings.Contains(err.Error(), "a == b") {
		t.Errorf("error should list unsupported expression, got %v", err)
	}
}

// Test_ConfigParseStrictIfTest 配置 mybatis.configuration.strict-if-test 解析与套用。
func Test_ConfigParseStrictIfTest(t *testing.T) {
	defer SetStrictIfTest(StrictIfTest())

	cm := map[string]string{
		"spring.datasource.url":                "jdbc:sqlite:test.db",
		"mybatis.configuration.strict-if-test": "true",
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("InitializeFromSettings failed: %v", err)
		return
	}
	defer Close()
	if !StrictIfTest() {
		t.Error("strict-if-test=true should enable StrictIfTest")
	}

	cm2 := map[string]string{
		"spring.datasource.url": "jdbc:sqlite:test.db",
	}
	if err := InitializeFromSettings(cm2); err != nil {
		t.Errorf("InitializeFromSettings failed: %v", err)
		return
	}
	defer Close()
	if StrictIfTest() {
		t.Error("strict-if-test default should be off")
	}
}
