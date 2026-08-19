package types

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testDotParamMapperXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="TestDotParamMapper">
	<select id="selectList" parameterType="map" resultType="map">
		select * from sys_config
		<where>
			<if test="configName != null and configName != ''">
				AND config_name like concat('%', #{configName}, '%')
			</if>
			<if test="params.beginTime != null and params.beginTime != ''">
				AND date_format(create_time,'%Y%m%d') &gt;= date_format(#{params.beginTime},'%Y%m%d')
			</if>
			<if test="params.endTime != null and params.endTime != ''">
				AND date_format(create_time,'%Y%m%d') &lt;= date_format(#{params.endTime},'%Y%m%d')
			</if>
		</where>
	</select>

	<update id="updateChildren" parameterType="java.util.List">
		update sys_dept set ancestors =
		<foreach collection="depts" item="item" index="index"
			separator=" " open="case dept_id" close="end">
			when #{item.deptId} then #{item.ancestors}
		</foreach>
		where dept_id in
		<foreach collection="depts" item="item" index="index"
			separator="," open="(" close=")">
			#{item.deptId}
		</foreach>
	</update>
</mapper>
`

func writeDotParamTestMapper(t *testing.T) string {
	dir := t.TempDir()
	file := filepath.Join(dir, "TestDotParamMapper.xml")
	if err := os.WriteFile(file, []byte(testDotParamMapperXML), 0644); err != nil {
		t.Error("write test mapper failed:", err)
	}
	return dir
}

// deptItem 用于 foreach 结构体切片测试
type deptItem struct {
	DeptId    int64
	Ancestors string
}

func Test_DotParam_MapNested(t *testing.T) {
	mps := NewSqlMappers(writeDotParamTestMapper(t))
	fn := getFunction(t, mps, "selectList")
	if fn == nil {
		return
	}
	sql, _, err := fn.GenerateSQL(map[string]interface{}{
		"configName": "system",
		"params": map[string]interface{}{
			"beginTime": "2024-01-01",
			"endTime":   "2024-12-31",
		},
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if strings.Contains(sql, "#{") {
		t.Error("dotted param literal not replaced, sql:", sql)
	}
	if !strings.Contains(sql, "date_format(create_time,'%Y%m%d') >= date_format('2024-01-01','%Y%m%d')") {
		t.Error("params.beginTime not replaced, sql:", sql)
	}
	if !strings.Contains(sql, "date_format(create_time,'%Y%m%d') <= date_format('2024-12-31','%Y%m%d')") {
		t.Error("params.endTime not replaced, sql:", sql)
	}
	t.Log("selectList sql:", sql)
}

func Test_DotParam_MapFlatKey(t *testing.T) {
	mps := NewSqlMappers(writeDotParamTestMapper(t))
	fn := getFunction(t, mps, "selectList")
	if fn == nil {
		return
	}
	sql, _, err := fn.GenerateSQL(map[string]interface{}{
		"params.beginTime": "2024-01-01",
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if strings.Contains(sql, "#{") {
		t.Error("flat dotted param literal not replaced, sql:", sql)
	}
	if !strings.Contains(sql, "'2024-01-01'") {
		t.Error("flat params.beginTime not replaced, sql:", sql)
	}
	t.Log("selectList(flat) sql:", sql)
}

func Test_DotParam_IfEmptyOmitted(t *testing.T) {
	mps := NewSqlMappers(writeDotParamTestMapper(t))
	fn := getFunction(t, mps, "selectList")
	if fn == nil {
		return
	}
	// beginTime/endTime 为空：dotted 条件应识别为不匹配，整段 if 被剔除
	sql, _, err := fn.GenerateSQL(map[string]interface{}{
		"configName": "system",
		"params": map[string]interface{}{
			"beginTime": "",
			"endTime":   "",
		},
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if strings.Contains(sql, "date_format") {
		t.Error("empty dotted condition should be omitted, sql:", sql)
	}
	if strings.Contains(sql, "#{") {
		t.Error("literal remains, sql:", sql)
	}
	t.Log("selectList(empty) sql:", sql)
}

func Test_DotParam_ForeachStruct(t *testing.T) {
	mps := NewSqlMappers(writeDotParamTestMapper(t))
	fn := getFunction(t, mps, "updateChildren")
	if fn == nil {
		return
	}
	sql, _, err := fn.GenerateSQL([]deptItem{
		{DeptId: 100, Ancestors: "0,100"},
		{DeptId: 101, Ancestors: "0,100,101"},
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if strings.Contains(sql, "#{") {
		t.Error("foreach dotted param literal not replaced, sql:", sql)
	}
	if !strings.Contains(sql, "case dept_id when 100 then '0,100' when 101 then '0,100,101'") {
		t.Error("foreach struct params wrong, sql:", sql)
	}
	if !strings.Contains(sql, "in ( 100, 101)") {
		t.Error("foreach in clause wrong, sql:", sql)
	}
	t.Log("updateChildren sql:", sql)
}

func Test_DotParam_ForeachMap(t *testing.T) {
	mps := NewSqlMappers(writeDotParamTestMapper(t))
	fn := getFunction(t, mps, "updateChildren")
	if fn == nil {
		return
	}
	sql, _, err := fn.GenerateSQL([]map[string]interface{}{
		{"deptId": int64(100), "ancestors": "0,100"},
		{"deptId": int64(101), "ancestors": "0,100,101"},
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if strings.Contains(sql, "#{") {
		t.Error("foreach map dotted param literal not replaced, sql:", sql)
	}
	if !strings.Contains(sql, "case dept_id when 100 then '0,100' when 101 then '0,100,101'") {
		t.Error("foreach map params wrong, sql:", sql)
	}
	if !strings.Contains(sql, "in ( 100, 101)") {
		t.Error("foreach map in clause wrong, sql:", sql)
	}
	t.Log("updateChildren(map) sql:", sql)
}

func Test_DotParam_PrepareSQL(t *testing.T) {
	mps := NewSqlMappers(writeDotParamTestMapper(t))
	fn := getFunction(t, mps, "selectList")
	if fn == nil {
		return
	}
	sql, args, err := fn.PrepareSQL(map[string]interface{}{
		"configName": "system",
		"params": map[string]interface{}{
			"beginTime": "2024-01-01",
		},
	})
	if err != nil {
		t.Error("PrepareSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if strings.Contains(sql, "#{") {
		t.Error("prepare dotted param literal not replaced, sql:", sql)
	}
	if !strings.Contains(sql, "date_format(?,'%Y%m%d')") {
		t.Error("prepare dotted param placeholder missing, sql:", sql)
	}
	if len(args) != 2 {
		t.Error("prepare args count wrong:", args)
	}
	t.Log("selectList prepare sql:", sql, "args:", args)
}

// Test_DotParam_Samples 使用 samples/（RuoYi Mapper）真实文件回归：
// selectConfigList 的 #{params.beginTime} 与 updateDeptChildren 的 #{item.deptId}。
func Test_DotParam_Samples(t *testing.T) {
	mps := NewSqlMappers("../samples")
	if mps == nil {
		t.Error("load samples failed")
		return
	}
	if len(mps.Mappers) == 0 {
		t.Error("no samples mapper loaded")
		return
	}
	// 1) selectConfigList: params.beginTime / params.endTime 点号参数 + dotted if 条件
	for _, m := range mps.Mappers {
		fn := m.NamedFunctions["selectConfigList"]
		if fn == nil {
			continue
		}
		sql, _, err := fn.GenerateSQL(map[string]interface{}{
			"configName": "system",
			"configType": "Y",
			"params": map[string]interface{}{
				"beginTime": "2024-01-01",
				"endTime":   "2024-12-31",
			},
		})
		if err != nil {
			t.Error("selectConfigList GenerateSQL failed:", err)
			return
		}
		sql = collapseSpace(sql)
		t.Log("selectConfigList sql:", sql)
		if strings.Contains(sql, "#{") {
			t.Error("selectConfigList dotted literal remains, sql:", sql)
		}
		if !strings.Contains(sql, "date_format('2024-01-01','%Y%m%d')") {
			t.Error("selectConfigList params.beginTime not replaced, sql:", sql)
		}
	}
	// 2) updateDeptChildren: foreach 内 #{item.deptId} / #{item.ancestors}
	for _, m := range mps.Mappers {
		fn := m.NamedFunctions["updateDeptChildren"]
		if fn == nil {
			continue
		}
		sql, _, err := fn.GenerateSQL([]map[string]interface{}{
			{"deptId": int64(100), "ancestors": "0,100"},
			{"deptId": int64(101), "ancestors": "0,100,101"},
		})
		if err != nil {
			t.Error("updateDeptChildren GenerateSQL failed:", err)
			return
		}
		sql = collapseSpace(sql)
		t.Log("updateDeptChildren sql:", sql)
		if strings.Contains(sql, "#{") {
			t.Error("updateDeptChildren dotted literal remains, sql:", sql)
		}
		if !strings.Contains(sql, "when 100 then '0,100'") {
			t.Error("updateDeptChildren item.deptId not replaced, sql:", sql)
		}
	}
}
