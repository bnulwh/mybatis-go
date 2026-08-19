package types

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testRawParamMapperXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="TestRawParamMapper">
	<select id="selectDeptList" parameterType="map" resultType="map">
		select * from sys_dept
		<where>
			<if test="deptName != null and deptName != ''">
				AND dept_name like concat('%', #{deptName}, '%')
			</if>
		</where>
		${params.dataScope}
		order by d.parent_id, d.order_num
	</select>

	<update id="createTable">
		${sql}
	</update>
</mapper>
`

func writeRawParamTestMapper(t *testing.T) string {
	dir := t.TempDir()
	file := filepath.Join(dir, "TestRawParamMapper.xml")
	if err := os.WriteFile(file, []byte(testRawParamMapperXML), 0644); err != nil {
		t.Error("write test mapper failed:", err)
	}
	return dir
}

func Test_RawParam_DataScopeInjected(t *testing.T) {
	mps := NewSqlMappers(writeRawParamTestMapper(t))
	fn := getFunction(t, mps, "selectDeptList")
	if fn == nil {
		return
	}
	sql, _, err := fn.GenerateSQL(map[string]interface{}{
		"deptName": "研发",
		"params": map[string]interface{}{
			"dataScope": " and d.dept_id in (1,2,3)",
		},
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if strings.Contains(sql, "${") || strings.Contains(sql, "#{") {
		t.Error("raw param literal remains, sql:", sql)
	}
	// dataScope 原样注入：不加引号
	if !strings.Contains(sql, "and d.dept_id in (1,2,3)") {
		t.Error("dataScope not injected raw, sql:", sql)
	}
	if strings.Contains(sql, "' and d.dept_id") {
		t.Error("dataScope wrongly quoted, sql:", sql)
	}
	// #{deptName} 仍然正常
	if !strings.Contains(sql, "dept_name like concat('%', '研发', '%')") {
		t.Error("normal param broken, sql:", sql)
	}
	t.Log("selectDeptList sql:", sql)
}

func Test_RawParam_DataScopeEmpty(t *testing.T) {
	mps := NewSqlMappers(writeRawParamTestMapper(t))
	fn := getFunction(t, mps, "selectDeptList")
	if fn == nil {
		return
	}
	// dataScope 为空字符串：应替换为空，不残留字面量
	sql, _, err := fn.GenerateSQL(map[string]interface{}{
		"deptName": "研发",
		"params": map[string]interface{}{
			"dataScope": "",
		},
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if strings.Contains(sql, "${params.dataScope}") {
		t.Error("empty dataScope literal remains, sql:", sql)
	}
	t.Log("selectDeptList(empty) sql:", sql)
}

func Test_RawParam_CreateTableMap(t *testing.T) {
	mps := NewSqlMappers(writeRawParamTestMapper(t))
	fn := getFunction(t, mps, "createTable")
	if fn == nil {
		return
	}
	sql, _, err := fn.GenerateSQL(map[string]interface{}{
		"sql": "create table t (id int, name varchar(32))",
	})
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if sql != "create table t (id int, name varchar(32))" {
		t.Error("createTable map raw substitution wrong, sql:", sql)
	}
	t.Log("createTable(map) sql:", sql)
}

func Test_RawParam_CreateTableScalar(t *testing.T) {
	mps := NewSqlMappers(writeRawParamTestMapper(t))
	fn := getFunction(t, mps, "createTable")
	if fn == nil {
		return
	}
	// 无 parameterType，直接传字符串（对应手动 Mapper: CreateTable func(sql string)）
	sql, _, err := fn.GenerateSQL("create table t (id int)")
	if err != nil {
		t.Error("GenerateSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if sql != "create table t (id int)" {
		t.Error("createTable scalar raw substitution wrong, sql:", sql)
	}
	t.Log("createTable(scalar) sql:", sql)
}

func Test_RawParam_PrepareSQL(t *testing.T) {
	mps := NewSqlMappers(writeRawParamTestMapper(t))
	fn := getFunction(t, mps, "selectDeptList")
	if fn == nil {
		return
	}
	sql, args, err := fn.PrepareSQL(map[string]interface{}{
		"deptName": "研发",
		"params": map[string]interface{}{
			"dataScope": " and d.dept_id in (1,2,3)",
		},
	})
	if err != nil {
		t.Error("PrepareSQL failed:", err)
		return
	}
	sql = collapseSpace(sql)
	if strings.Contains(sql, "${") || strings.Contains(sql, "#{") {
		t.Error("prepare raw literal remains, sql:", sql)
	}
	// dataScope 内联注入，不入占位符参数
	if !strings.Contains(sql, "and d.dept_id in (1,2,3)") {
		t.Error("prepare dataScope not injected, sql:", sql)
	}
	if len(args) != 1 {
		t.Error("prepare args should only contain #{deptName}, got:", args)
	}
	t.Log("selectDeptList prepare sql:", sql, "args:", args)
}

// Test_RawParam_Samples 使用 samples/（RuoYi Mapper）真实文件回归：
// selectDeptList 的 ${params.dataScope} 与 createTable 的 ${sql}。
func Test_RawParam_Samples(t *testing.T) {
	mps := NewSqlMappers("../samples")
	if mps == nil {
		t.Error("load samples failed")
		return
	}
	if len(mps.Mappers) == 0 {
		t.Error("no samples mapper loaded")
		return
	}
	// 1) selectDeptList: ${params.dataScope}
	for _, m := range mps.Mappers {
		fn := m.NamedFunctions["selectDeptList"]
		if fn == nil {
			continue
		}
		sql, _, err := fn.GenerateSQL(map[string]interface{}{
			"deptName": "研发",
			"params": map[string]interface{}{
				"dataScope": " and d.dept_id in (1,2,3)",
			},
		})
		if err != nil {
			t.Error("selectDeptList GenerateSQL failed:", err)
			return
		}
		sql = collapseSpace(sql)
		t.Log("selectDeptList sql:", sql)
		if strings.Contains(sql, "${") {
			t.Error("selectDeptList raw literal remains, sql:", sql)
		}
		if !strings.Contains(sql, "and d.dept_id in (1,2,3)") {
			t.Error("selectDeptList dataScope not injected, sql:", sql)
		}
		if strings.Contains(sql, "' and d.dept_id") {
			t.Error("selectDeptList dataScope wrongly quoted, sql:", sql)
		}
	}
	// 2) createTable: ${sql}
	for _, m := range mps.Mappers {
		fn := m.NamedFunctions["createTable"]
		if fn == nil {
			continue
		}
		sql, _, err := fn.GenerateSQL(map[string]interface{}{
			"sql": "create table gen_table (table_id bigint)",
		})
		if err != nil {
			t.Error("createTable GenerateSQL failed:", err)
			return
		}
		sql = collapseSpace(sql)
		t.Log("createTable sql:", sql)
		if strings.Contains(sql, "${") {
			t.Error("createTable raw literal remains, sql:", sql)
		}
		if !strings.Contains(sql, "create table gen_table (table_id bigint)") {
			t.Error("createTable sql not injected, sql:", sql)
		}
	}
}
