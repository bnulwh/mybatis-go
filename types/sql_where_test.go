package types

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testWhereMapperXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="TestWhereMapper">
	<sql id="sqlwhereSearch">
		<where>
			<if test="configKey !=null and configKey != ''">
				and config_key = #{configKey}
			</if>
		</where>
	</sql>

	<select id="selectConfigList" parameterType="map" resultType="map">
		select * from sys_config
		<where>
			<if test="configName != null and configName != ''">
				AND config_name like concat('%', #{configName}, '%')
			</if>
			<if test="configType != null and configType != ''">
				AND config_type = #{configType}
			</if>
		</where>
	</select>

	<select id="selectConfigByKey" parameterType="map" resultType="map">
		select * from sys_config
		<include refid="sqlwhereSearch"/>
	</select>

	<select id="selectAll" resultType="map">
		select * from sys_config
		<where>
			<if test="configName != null">
				AND config_name = #{configName}
			</if>
		</where>
	</select>

	<select id="selectConfigs" parameterType="map" resultType="map">
		select * from sys_config
		<where>
			<if test="ids != null">
				AND config_id in
				<foreach collection="ids" item="id" open="(" separator="," close=")">
					#{id}
				</foreach>
			</if>
		</where>
	</select>
</mapper>
`

func writeTestMapper(t *testing.T) string {
	dir := t.TempDir()
	file := filepath.Join(dir, "TestWhereMapper.xml")
	if err := os.WriteFile(file, []byte(testWhereMapperXML), 0644); err != nil {
		t.Error("write test mapper failed:", err)
	}
	return dir
}

func getFunction(t *testing.T, mps *SqlMappers, id string) *SqlFunction {
	if mps == nil {
		t.Error("load mapper failed")
		return nil
	}
	if len(mps.Mappers) == 0 {
		t.Error("no mapper loaded")
		return nil
	}
	fn := mps.Mappers[0].NamedFunctions[id]
	if fn == nil {
		t.Error("function not found:", id)
		return nil
	}
	return fn
}

func Test_WhereFragment_AllMatch(t *testing.T) {
	mps := NewSqlMappers(writeTestMapper(t))
	fn := getFunction(t, mps, "selectConfigList")
	if fn == nil {
		return
	}
	sql, _, err := fn.GenerateSQL(map[string]interface{}{
		"configName": "system",
		"configType": "Y",
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	if !strings.Contains(sql, " where") {
		t.Error("where clause missing, sql:", sql)
	}
	if strings.Contains(sql, "where AND") || strings.Contains(sql, "whereconfig") {
		t.Error("leading AND not stripped, sql:", sql)
	}
	if !strings.Contains(sql, "config_name like concat('%', 'system', '%')") {
		t.Error("first condition missing or malformed, sql:", sql)
	}
	if !strings.Contains(sql, "config_type = 'Y'") {
		t.Error("second condition missing, sql:", sql)
	}
	t.Log("selectConfigList sql:", sql)
}

func Test_WhereFragment_NoMatch(t *testing.T) {
	mps := NewSqlMappers(writeTestMapper(t))
	fn := getFunction(t, mps, "selectConfigList")
	if fn == nil {
		return
	}
	sql, _, err := fn.GenerateSQL(map[string]interface{}{
		"configName": "",
		"configType": "",
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	if strings.Contains(sql, "where") {
		t.Error("empty where should be omitted, sql:", sql)
	}
	t.Log("selectConfigList(empty) sql:", sql)
}

func Test_WhereFragment_PartialMatch(t *testing.T) {
	mps := NewSqlMappers(writeTestMapper(t))
	fn := getFunction(t, mps, "selectConfigList")
	if fn == nil {
		return
	}
	// 第一个条件不匹配，第二个匹配：不应输出 where 前缀（无 AND 可剥离，且仅剩 OR/AND 之外的第二个条件）
	sql, _, err := fn.GenerateSQL(map[string]interface{}{
		"configName": "",
		"configType": "Y",
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	if !strings.Contains(sql, "where config_type = 'Y'") {
		t.Error("partial match where wrong, sql:", sql)
	}
	t.Log("selectConfigList(partial) sql:", sql)
}

func Test_WhereFragment_InInclude(t *testing.T) {
	mps := NewSqlMappers(writeTestMapper(t))
	fn := getFunction(t, mps, "selectConfigByKey")
	if fn == nil {
		return
	}
	sql, _, err := fn.GenerateSQL(map[string]interface{}{
		"configKey": "sys.name",
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	if !strings.Contains(sql, " where config_key = 'sys.name'") {
		t.Error("include + where not rendered, sql:", sql)
	}
	t.Log("selectConfigByKey sql:", sql)
}

func Test_WhereFragment_NoParam(t *testing.T) {
	mps := NewSqlMappers(writeTestMapper(t))
	fn := getFunction(t, mps, "selectAll")
	if fn == nil {
		return
	}
	sql, _, err := fn.GenerateSQL()
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	if strings.Contains(sql, "where") {
		t.Error("no-param where should be omitted, sql:", sql)
	}
	t.Log("selectAll sql:", sql)
}

func Test_WhereFragment_WithForeach(t *testing.T) {
	mps := NewSqlMappers(writeTestMapper(t))
	fn := getFunction(t, mps, "selectConfigs")
	if fn == nil {
		return
	}
	sql, _, err := fn.GenerateSQL(map[string]interface{}{
		"ids": []int64{1, 2, 3},
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	if !strings.Contains(sql, "where config_id in") {
		t.Error("where+foreach not rendered, sql:", sql)
	}
	if !strings.Contains(sql, "1, 2, 3") {
		t.Error("foreach items missing, sql:", sql)
	}
	t.Log("selectConfigs sql:", sql)
}

func Test_WhereFragment_PrepareSQL(t *testing.T) {
	mps := NewSqlMappers(writeTestMapper(t))
	fn := getFunction(t, mps, "selectConfigList")
	if fn == nil {
		return
	}
	sql, args, err := fn.PrepareSQL(map[string]interface{}{
		"configName": "system",
		"configType": "Y",
	})
	if err != nil {
		t.Error("PrepareSQL failed:", err)
		return
	}
	if !strings.Contains(sql, " where") {
		t.Error("prepare where missing, sql:", sql)
	}
	if strings.Contains(sql, "where AND") || strings.Contains(sql, "whereconfig") {
		t.Error("prepare leading AND not stripped, sql:", sql)
	}
	if len(args) != 2 {
		t.Error("prepare args count wrong:", args)
	}
	t.Log("selectConfigList prepare sql:", sql, "args:", args)
}
