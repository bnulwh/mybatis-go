package orm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	_ "modernc.org/sqlite"
)

// TypeHandlerTag 自定义类型：DB 中 TEXT 列存 JSON 字符串，Go 侧为 struct。
type TypeHandlerTag struct {
	Label string `json:"label"`
	Level int    `json:"level"`
}

// TypeHandlerModel 测试模型：Tags 字段走自定义 TypeHandler 反序列化。
type TypeHandlerModel struct {
	Id   int64
	Name string
	Tags TypeHandlerTag
}

type TypeHandlerMapper struct {
	BaseMapper
	SelectAll  func() ([]TypeHandlerModel, error)
	SelectAuto func() ([]TypeHandlerModel, error)
	StreamAll  func() (*RowStream, error)
}

// registerTypeHandlerTagJSON 注册 JSON → TypeHandlerTag 处理器（幂等，可覆盖）。
func registerTypeHandlerTagJSON(t *testing.T) {
	t.Helper()
	RegisterTypeHandlerFor(func(source interface{}) (TypeHandlerTag, error) {
		var tag TypeHandlerTag
		switch s := source.(type) {
		case string:
			if err := json.Unmarshal([]byte(s), &tag); err != nil {
				return TypeHandlerTag{}, fmt.Errorf("unmarshal tags %q: %v", s, err)
			}
		case []byte:
			if err := json.Unmarshal(s, &tag); err != nil {
				return TypeHandlerTag{}, fmt.Errorf("unmarshal tags: %v", err)
			}
		case nil:
			// DB NULL → 零值
		default:
			return TypeHandlerTag{}, fmt.Errorf("unsupported source type %T", source)
		}
		return tag, nil
	})
}

// Test_RegisterTypeHandler 注册/查询/覆盖与未注册回退（零行为变化）。
func Test_RegisterTypeHandler(t *testing.T) {
	typ := reflect.TypeOf(TypeHandlerTag{})
	if h := getTypeHandler(typ); h != nil {
		t.Error("unregistered type should have nil handler")
	}
	// 未注册类型回退 utils.ChangeType（既有行为不变）
	v, err := convertFieldValue("123", reflect.TypeOf(int64(0)))
	if err != nil {
		t.Error("fallback convert error:", err)
	} else if n, ok := v.(int64); !ok || n != 123 {
		t.Error("fallback convert =", v, "want int64(123)")
	}
	// 泛型注册 → 查询命中
	registerTypeHandlerTagJSON(t)
	if getTypeHandler(typ) == nil {
		t.Error("registered type should have handler")
	}
	// convertFieldValue 优先自定义 handler
	v, err = convertFieldValue(`{"label":"a","level":1}`, typ)
	if err != nil {
		t.Error("handler convert error:", err)
	} else if tag, ok := v.(TypeHandlerTag); !ok || tag.Label != "a" || tag.Level != 1 {
		t.Error("handler convert =", v, "want {a 1}")
	}
	// 覆盖注册生效
	RegisterTypeHandler(typ, func(source interface{}, targetType reflect.Type) (interface{}, error) {
		return TypeHandlerTag{Label: "override"}, nil
	})
	v, err = convertFieldValue("anything", typ)
	if err != nil {
		t.Error("override convert error:", err)
	} else if tag, ok := v.(TypeHandlerTag); !ok || tag.Label != "override" {
		t.Error("override convert =", v, "want {override 0}")
	}
}

// initTypeHandlerTest 初始化 SQLite 数据源（TypeHandler 专用表/模型/Mapper）。
func initTypeHandlerTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="TypeHandlerMapper">
  <resultMap id="BaseResultMap" type="TypeHandlerModel">
    <id column="id" jdbcType="INTEGER" property="id" />
    <result column="name" jdbcType="VARCHAR" property="name" />
    <result column="tags" jdbcType="VARCHAR" property="tags" />
  </resultMap>
  <select id="selectAll" resultMap="BaseResultMap">
    select id, name, tags from t_typehandler order by id
  </select>
  <select id="selectAuto" resultType="TypeHandlerModel">
    select id, name, tags from t_typehandler order by id
  </select>
  <select id="streamAll" resultMap="BaseResultMap">
    select id, name, tags from t_typehandler order by id
  </select>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "TypeHandlerMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return ""
	}
	dbPath := filepath.Join(dir, "test.db")
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return ""
	}
	return dir
}

// Test_SqliteTypeHandlerQuery SQLite 端到端：TEXT 列 JSON 字符串经 TypeHandler
// 自动反序列化为 struct（resultMap 与 resultType 自动映射两条路径）。
func Test_SqliteTypeHandlerQuery(t *testing.T) {
	dir := initTypeHandlerTest(t)
	if dir == "" {
		return
	}
	defer Close()

	if _, err := Execute(`CREATE TABLE t_typehandler (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		tags TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	RegisterModel(new(TypeHandlerModel))
	if err := RegisterMapper(new(TypeHandlerMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	registerTypeHandlerTagJSON(t)
	tagJSON := `{"label":"vip","level":3}`
	if _, err := Execute(`INSERT INTO t_typehandler (name, tags) VALUES (?, ?)`, "u1", tagJSON); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	mp := NewMapper("TypeHandlerMapper").(TypeHandlerMapper)
	// resultMap 路径（setColumnValuesPrepared）
	rs, err := mp.SelectAll()
	if err != nil {
		t.Errorf("select all failed: %v", err)
		return
	}
	if len(rs) != 1 {
		t.Errorf("select all count = %d, want 1", len(rs))
		return
	}
	if rs[0].Tags.Label != "vip" || rs[0].Tags.Level != 3 {
		t.Errorf("resultMap tags = %+v, want {vip 3}", rs[0].Tags)
	}
	// resultType 自动映射路径（setModelFieldValues）
	rs2, err := mp.SelectAuto()
	if err != nil {
		t.Errorf("select auto failed: %v", err)
		return
	}
	if len(rs2) != 1 {
		t.Errorf("select auto count = %d, want 1", len(rs2))
		return
	}
	if rs2[0].Tags.Label != "vip" || rs2[0].Tags.Level != 3 {
		t.Errorf("resultType tags = %+v, want {vip 3}", rs2[0].Tags)
	}
}

// Test_SqliteTypeHandlerStream SQLite 端到端：RowStream.Scan 结构体字段走 TypeHandler。
func Test_SqliteTypeHandlerStream(t *testing.T) {
	dir := initTypeHandlerTest(t)
	if dir == "" {
		return
	}
	defer Close()

	if _, err := Execute(`CREATE TABLE t_typehandler (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		tags TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	RegisterModel(new(TypeHandlerModel))
	if err := RegisterMapper(new(TypeHandlerMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	registerTypeHandlerTagJSON(t)
	if _, err := Execute(`INSERT INTO t_typehandler (name, tags) VALUES (?, ?)`, "s1", `{"label":"gold","level":9}`); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}
	mp := NewMapper("TypeHandlerMapper").(TypeHandlerMapper)
	st, err := mp.StreamAll()
	if err != nil {
		t.Errorf("stream all failed: %v", err)
		return
	}
	defer st.Close()
	var got []TypeHandlerModel
	for st.Next() {
		var m TypeHandlerModel
		if err := st.Scan(&m); err != nil {
			t.Errorf("stream scan failed: %v", err)
			return
		}
		got = append(got, m)
	}
	if err := st.Err(); err != nil {
		t.Errorf("stream err: %v", err)
		return
	}
	if len(got) != 1 {
		t.Errorf("stream count = %d, want 1", len(got))
		return
	}
	if got[0].Tags.Label != "gold" || got[0].Tags.Level != 9 {
		t.Errorf("stream tags = %+v, want {gold 9}", got[0].Tags)
	}
}
