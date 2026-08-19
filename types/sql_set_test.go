package types

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testSetMapperXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="TestSetMapper">
	<sql id="sqlsetConfig">
		<set>
			<if test="configKey != null and configKey != ''">
				config_key = #{configKey},
			</if>
		</set>
	</sql>

	<update id="updateConfig" parameterType="map">
		update sys_config
		<set>
			<if test="configName != null and configName != ''">config_name = #{configName},</if>
			<if test="configKey != null and configKey != ''">config_key = #{configKey},</if>
			<if test="remark != null">remark = #{remark},</if>
			update_time = current_timestamp
		</set>
		where config_id = #{configId}
	</update>

	<update id="updateAllMatched" parameterType="map">
		update sys_config
		<set>
			<if test="configName != null">config_name = #{configName},</if>
			<if test="configKey != null">config_key = #{configKey},</if>
		</set>
		where config_id = #{configId}
	</update>

	<update id="updateByInclude" parameterType="map">
		update sys_config
		<include refid="sqlsetConfig"/>
		where config_id = #{configId}
	</update>
</mapper>
`

func writeSetTestMapper(t *testing.T) string {
	dir := t.TempDir()
	file := filepath.Join(dir, "TestSetMapper.xml")
	if err := os.WriteFile(file, []byte(testSetMapperXML), 0644); err != nil {
		t.Error("write test mapper failed:", err)
	}
	return dir
}

// collapseSpace 将连续空白折叠为单个空格，便于断言。
func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func Test_SetFragment_AllMatch(t *testing.T) {
	mps := NewSqlMappers(writeSetTestMapper(t))
	fn := getFunction(t, mps, "updateConfig")
	if fn == nil {
		return
	}
	sql, _, err := fn.GenerateSQL(map[string]interface{}{
		"configName": "system",
		"configKey":  "sys.name",
		"remark":     "remark",
		"configId":   1,
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if !strings.Contains(sql, "update sys_config set") {
		t.Error("set clause missing, sql:", sql)
	}
	if strings.HasSuffix(strings.TrimSpace(sql), ",") {
		t.Error("trailing comma not stripped, sql:", sql)
	}
	if !strings.Contains(sql, "config_name = 'system'") {
		t.Error("first assignment missing, sql:", sql)
	}
	if !strings.Contains(sql, "update_time = current_timestamp") {
		t.Error("plain text assignment missing, sql:", sql)
	}
	if !strings.Contains(sql, "where config_id = 1") {
		t.Error("where clause missing, sql:", sql)
	}
	t.Log("updateConfig sql:", sql)
}

func Test_SetFragment_NoMatch(t *testing.T) {
	mps := NewSqlMappers(writeSetTestMapper(t))
	fn := getFunction(t, mps, "updateAllMatched")
	if fn == nil {
		return
	}
	sql, _, err := fn.GenerateSQL(map[string]interface{}{
		"configName": "",
		"configKey":  "",
		"configId":   1,
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	if strings.Contains(sql, "set") {
		t.Error("empty set should be omitted, sql:", sql)
	}
	if strings.Contains(sql, ",") {
		t.Error("empty set should not leave comma, sql:", sql)
	}
	t.Log("updateAllMatched(empty) sql:", sql)
}

func Test_SetFragment_PartialMatch(t *testing.T) {
	mps := NewSqlMappers(writeSetTestMapper(t))
	fn := getFunction(t, mps, "updateConfig")
	if fn == nil {
		return
	}
	// 最后一个 if 不匹配：尾部逗号必须被剥离
	sql, _, err := fn.GenerateSQL(map[string]interface{}{
		"configName": "system",
		"configKey":  "",
		"remark":     "",
		"configId":   1,
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	if !strings.Contains(sql, "set config_name = 'system', update_time = current_timestamp") {
		t.Error("partial set wrong, sql:", sql)
	}
	if strings.Contains(sql, "current_timestamp,") || strings.HasSuffix(strings.TrimSpace(sql), ",") {
		t.Error("trailing comma not stripped, sql:", sql)
	}
	t.Log("updateConfig(partial) sql:", sql)
}

func Test_SetFragment_AllIfOnly(t *testing.T) {
	mps := NewSqlMappers(writeSetTestMapper(t))
	fn := getFunction(t, mps, "updateAllMatched")
	if fn == nil {
		return
	}
	// 仅 if 片段、无末尾纯文本：最后一个匹配的 if 尾部逗号必须被剥离
	sql, _, err := fn.GenerateSQL(map[string]interface{}{
		"configName": "system",
		"configKey":  "sys.name",
		"configId":   1,
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if !strings.Contains(sql, "set config_name = 'system', config_key = 'sys.name'") {
		t.Error("all-if set wrong, sql:", sql)
	}
	if strings.HasSuffix(strings.TrimSpace(sql), ",") {
		t.Error("trailing comma not stripped, sql:", sql)
	}
	t.Log("updateAllMatched sql:", sql)
}

func Test_SetFragment_InInclude(t *testing.T) {
	mps := NewSqlMappers(writeSetTestMapper(t))
	fn := getFunction(t, mps, "updateByInclude")
	if fn == nil {
		return
	}
	sql, _, err := fn.GenerateSQL(map[string]interface{}{
		"configKey": "sys.name",
		"configId":  1,
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	if !strings.Contains(sql, "set config_key = 'sys.name'") {
		t.Error("include + set not rendered, sql:", sql)
	}
	if strings.HasSuffix(strings.TrimSpace(sql), ",") {
		t.Error("trailing comma not stripped, sql:", sql)
	}
	t.Log("updateByInclude sql:", sql)
}

func Test_SetFragment_PrepareSQL(t *testing.T) {
	mps := NewSqlMappers(writeSetTestMapper(t))
	fn := getFunction(t, mps, "updateConfig")
	if fn == nil {
		return
	}
	sql, args, err := fn.PrepareSQL(map[string]interface{}{
		"configName": "system",
		"configKey":  "sys.name",
		"remark":     "remark",
		"configId":   1,
	})
	if err != nil {
		t.Error("PrepareSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if !strings.Contains(sql, "set config_name = ?") {
		t.Error("prepare set missing, sql:", sql)
	}
	if strings.Contains(sql, ", where") {
		t.Error("prepare trailing comma not stripped, sql:", sql)
	}
	if len(args) != 4 {
		t.Error("prepare args count wrong:", args)
	}
	t.Log("updateConfig prepare sql:", sql, "args:", args)
}

// Test_SetFragment_SamplesUpdateConfig 使用 samples/（RuoYi Mapper）真实文件回归：
// updateConfig 原先因 <set> 被丢弃而缺失 set 子句（S-02）。
func Test_SetFragment_SamplesUpdateConfig(t *testing.T) {
	mps := NewSqlMappers("../samples")
	if mps == nil {
		t.Error("load samples failed")
		return
	}
	if len(mps.Mappers) == 0 {
		t.Error("no samples mapper loaded")
		return
	}
	found := 0
	for _, m := range mps.Mappers {
		fn := m.NamedFunctions["updateConfig"]
		if fn == nil {
			continue
		}
		found++
		sql, _, err := fn.GenerateSQL(map[string]interface{}{
			"configName":  "system",
			"configKey":   "sys.name",
			"configValue": "1",
			"configType":  "Y",
			"updateBy":    "admin",
			"remark":      "r",
			"configId":    1,
		})
		if err != nil {
			t.Error("GenerateSQL failed:", err)
			return
		}
		sql = collapseSpace(sql)
		t.Log("samples updateConfig sql:", sql)
		if !strings.Contains(sql, " set config_name =") {
			t.Error("set clause missing in samples updateConfig, sql:", sql)
		}
		if strings.Contains(sql, ", where") {
			t.Error("trailing comma before where, sql:", sql)
		}
	}
	if found == 0 {
		t.Error("updateConfig not found in samples")
	}
}
