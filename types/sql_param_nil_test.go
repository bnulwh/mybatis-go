package types

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const testNilParamMapperXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="TestNilParamMapper">
	<select id="selectById" parameterType="Long" resultType="map">
		select * from t where id = #{id}
	</select>
	<select id="selectByUser" parameterType="SysUser" resultType="map">
		select * from t where user_id = #{userId}
	</select>
	<delete id="deleteByIds" parameterType="Long">
		delete from t where id in
		<foreach item="id" collection="array" open="(" separator="," close=")">
			#{id}
		</foreach>
	</delete>
	<select id="selectAll" resultType="map">
		select * from t where deleted = 0
	</select>
</mapper>
`

type nilParamUser struct {
	UserId int64
}

func writeNilParamTestMapper(t *testing.T) string {
	dir := t.TempDir()
	file := filepath.Join(dir, "TestNilParamMapper.xml")
	if err := os.WriteFile(file, []byte(testNilParamMapperXML), 0644); err != nil {
		t.Error("write test mapper failed:", err)
	}
	return dir
}

// Test_ValidParam_Nil S-09：validParam 对 nil 参数返回错误而非反射零值 panic。
func Test_ValidParam_Nil(t *testing.T) {
	p := SqlParam{Name: "Long", TypeName: "Long", Type: BaseSqlParam, Need: true}
	if err := p.validParam([]interface{}{nil}); err == nil {
		t.Error("BaseSqlParam validParam(nil) should error")
	}
	p = SqlParam{Name: "Long", TypeName: "Long", Type: SliceSqlParam, Need: true}
	if err := p.validParam([]interface{}{nil}); err == nil {
		t.Error("SliceSqlParam validParam(nil) should error")
	}
	p = SqlParam{Name: "SysUser", TypeName: "SysUser", Type: StructSqlParam, Need: true}
	if err := p.validParam([]interface{}{nil}); err == nil {
		t.Error("StructSqlParam validParam(nil) should error")
	}
	p = SqlParam{TypeName: "Long", Type: BaseSqlParam, Need: true}
	if err := p.validParam([]interface{}{}); err == nil {
		t.Error("BaseSqlParam validParam() missing arg should error")
	}
}

// Test_GenerateSQL_NilParam S-09：GenerateSQL/PrepareSQL 传 nil 不再反射零值 panic。
func Test_GenerateSQL_NilParam(t *testing.T) {
	mps := NewSqlMappers(writeNilParamTestMapper(t))
	if mps == nil || len(mps.Mappers) == 0 {
		t.Error("load mapper failed")
		return
	}
	// BaseSqlParam nil
	fn := mps.Mappers[0].NamedFunctions["selectById"]
	if fn == nil {
		t.Error("selectById not found")
		return
	}
	if sql, _, err := fn.GenerateSQL(nil); err == nil {
		t.Error("GenerateSQL(nil) should error, sql:", sql)
	}
	if sql, _, err := fn.PrepareSQL(nil); err == nil {
		t.Error("PrepareSQL(nil) should error, sql:", sql)
	}
	// BaseSqlParam 类型化 nil 指针
	var nilLong *int64
	if sql, _, err := fn.GenerateSQL(nilLong); err == nil {
		t.Error("GenerateSQL((*int64)(nil)) should error, sql:", sql)
	}
	// StructSqlParam nil 与类型化 nil 指针
	fn = mps.Mappers[0].NamedFunctions["selectByUser"]
	if fn != nil {
		if sql, _, err := fn.GenerateSQL(nil); err == nil {
			t.Error("GenerateSQL(nil) struct should error, sql:", sql)
		}
		if sql, _, err := fn.GenerateSQL((*nilParamUser)(nil)); err == nil {
			t.Error("GenerateSQL((*nilParamUser)(nil)) should error, sql:", sql)
		}
	}
	// SliceSqlParam nil 与切片含 nil 元素
	fn = mps.Mappers[0].NamedFunctions["deleteByIds"]
	if fn != nil {
		if sql, _, err := fn.GenerateSQL(nil); err == nil {
			t.Error("GenerateSQL(nil) slice should error, sql:", sql)
		}
		if _, _, err := fn.GenerateSQL([]interface{}{nil, int64(1)}); err != nil {
			t.Error("GenerateSQL slice with nil element should not panic, err:", err)
		}
	}
	// 无 parameterType 的函数传 nil：返回错误而非 panic
	fn = mps.Mappers[0].NamedFunctions["selectAll"]
	if fn != nil {
		if sql, _, err := fn.GenerateSQL(nil); err == nil {
			t.Error("GenerateSQL(nil) no-param fn should error, sql:", sql)
		}
	}
}

func Test_ValidValue_Nil(t *testing.T) {
	if validValue(nil) {
		t.Error("validValue(nil) should be false")
	}
}

func Test_Convert2Map_InvalidValue(t *testing.T) {
	// reflect.Indirect(reflect.ValueOf(nil)) 是零 Value，convert2Map 应返回空 map 而非 panic
	mp := convert2Map(reflect.Value{})
	if mp == nil || len(mp) != 0 {
		t.Error("convert2Map(zero) should return empty map, got:", mp)
	}
}
