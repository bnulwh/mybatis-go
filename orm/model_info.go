package orm

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
)

// struct tag 元数据解析（P0-2）：
//
//	type SysUserModel struct {
//	    UserId   int64  `db:"user_id,pk"`
//	    UserName string `db:"user_name"`
//	    IsRemoved bool  `db:"is_removed,logic"` // 显式逻辑删除列
//	    Version   int   `db:"version,version"`  // 乐观锁版本列
//	    CreatedAt time.Time `db:"created_at,fill:insert"`
//	    Ignored   string `db:"-"`                // 不映射
//	}
//
// 表名：实现 TableName() string 方法 → 显式表名；否则短名去 Model 后缀 camelToSnake。
// 元数据在 RegisterModel 时解析入缓存，经 SetModelStructureProvider 注入 types 包，
// 供 MP 内置 CRUD 生成时优先于 resultMap 推导（表名/主键/逻辑删除列以 tag 为准）。

const (
	FillInsert       = "insert"
	FillUpdate       = "update"
	FillInsertUpdate = "insert_update"
)

// FieldInfo 模型字段元数据（db tag 解析结果）。
type FieldInfo struct {
	Name    string       // Go 字段名
	Column  string       // 列名（tag 指定或 camelToSnake 推导）
	Type    reflect.Type // Go 字段类型
	Primary bool         // db:"...,pk"
	Logic   bool         // db:"...,logic"
	Version bool         // db:"...,version"
	Fill    string       // db:"...,fill:insert|update|insert_update"
}

// ModelInfo 模型元数据（RegisterModel 时解析入缓存）。
type ModelInfo struct {
	GoType        reflect.Type
	TableName     string
	ExplicitTable bool
	Fields        []*FieldInfo
	PrimaryColumn string
	PrimaryField  *FieldInfo
	LogicColumn   string
	VersionColumn string
}

// hasExplicitMeta 是否携带会改变 MP 内置 CRUD 行为的元数据
// （显式表名 / 显式主键 / 显式逻辑删除列），仅列改名 / fill 不触发 provider 覆盖。
func (in *ModelInfo) hasExplicitMeta() bool {
	return in.ExplicitTable || in.PrimaryColumn != "" || in.LogicColumn != ""
}

// FillColumns 返回指定填充时机（insert/update）的字段；fill:insert_update 两时机均返回。
func (in *ModelInfo) FillColumns(when string) []*FieldInfo {
	var out []*FieldInfo
	for _, f := range in.Fields {
		if f.Fill == when || f.Fill == FillInsertUpdate {
			out = append(out, f)
		}
	}
	return out
}

// ParseModelInfo 解析模型 struct 的 db tag 元数据。
// inPtr 须为非 nil struct 指针；无任何 db tag 时返回 (nil, nil)。
func ParseModelInfo(inPtr interface{}) (*ModelInfo, error) {
	val := reflect.Indirect(reflect.ValueOf(inPtr))
	if !val.IsValid() || val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("<orm.ParseModelInfo> expect non-nil struct ptr, got %v", reflect.TypeOf(inPtr))
	}
	return parseModelInfoFromType(val.Type())
}

// parseModelInfoFromType 按类型解析元数据；无任何 db tag 返回 (nil, nil)，未知选项返回 error。
func parseModelInfoFromType(typ reflect.Type) (*ModelInfo, error) {
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("<orm.ParseModelInfo> expect struct type, got %v", typ)
	}
	info := &ModelInfo{GoType: typ, Fields: []*FieldInfo{}}
	// TableName() string 方法 → 显式表名（New().Elem() 可寻址，值/指针接收者均可）
	if m := reflect.New(typ).Elem().MethodByName("TableName"); m.IsValid() {
		mt := m.Type()
		if mt.NumIn() == 0 && mt.NumOut() == 1 && mt.Out(0).Kind() == reflect.String {
			info.TableName = m.Call(nil)[0].String()
			info.ExplicitTable = true
		}
	}
	if !info.ExplicitTable {
		info.TableName = camelToSnake(strings.TrimSuffix(typ.Name(), "Model"))
	}
	hasTag := false
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.PkgPath != "" {
			continue // 未导出字段
		}
		tag, ok := f.Tag.Lookup("db")
		if !ok {
			// 无 tag 的导出字段：默认列名 camelToSnake
			info.Fields = append(info.Fields, &FieldInfo{Name: f.Name, Column: camelToSnake(f.Name), Type: f.Type})
			continue
		}
		hasTag = true
		parts := strings.Split(tag, ",")
		col := strings.TrimSpace(parts[0])
		if col == "-" {
			continue // db:"-" 不映射
		}
		fi := &FieldInfo{Name: f.Name, Type: f.Type}
		if col == "" {
			col = camelToSnake(f.Name)
		}
		fi.Column = col
		for _, opt := range parts[1:] {
			switch strings.TrimSpace(opt) {
			case "pk":
				fi.Primary = true
			case "logic":
				fi.Logic = true
			case "version":
				fi.Version = true
			case "fill:" + FillInsert:
				fi.Fill = FillInsert
			case "fill:" + FillUpdate:
				fi.Fill = FillUpdate
			case "fill:" + FillInsertUpdate:
				fi.Fill = FillInsertUpdate
			default:
				return nil, fmt.Errorf("<orm.ParseModelInfo> field %s.%s: unknown db tag option `%s`", typ.Name(), f.Name, opt)
			}
		}
		info.Fields = append(info.Fields, fi)
		if fi.Primary && info.PrimaryColumn == "" {
			info.PrimaryColumn = fi.Column
			info.PrimaryField = fi
		}
		if fi.Logic && info.LogicColumn == "" {
			info.LogicColumn = fi.Column
		}
		if fi.Version && info.VersionColumn == "" {
			info.VersionColumn = fi.Column
		}
	}
	if !hasTag {
		return nil, nil
	}
	return info, nil
}

// GetModelInfo 查询已注册模型的元数据；未注册或无 db tag 时现场解析，失败返回 nil。
func GetModelInfo(inPtr interface{}) *ModelInfo {
	val := reflect.Indirect(reflect.ValueOf(inPtr))
	if !val.IsValid() || val.Kind() != reflect.Struct {
		return nil
	}
	fn := getFullName(val.Type())
	gCache.models.mu.RLock()
	info := gCache.models.Infos[fn]
	gCache.models.mu.RUnlock()
	if info != nil {
		return info
	}
	info, err := ParseModelInfo(inPtr)
	if err != nil {
		log.Warnf("parse model info for %s failed: %v", fn, err)
		return nil
	}
	return info
}

// goTypeToDbType Go 类型 → JDBC 类型（缺省 VARCHAR）。
func goTypeToDbType(typ reflect.Type) string {
	if typ == reflect.TypeOf(time.Time{}) {
		return "TIMESTAMP"
	}
	switch typ.Kind() {
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return "INTEGER"
	case reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "BIGINT"
	case reflect.Float32:
		return "FLOAT"
	case reflect.Float64:
		return "DOUBLE"
	case reflect.String:
		return "VARCHAR"
	}
	return "VARCHAR"
}

// buildTableStructure 模型元数据 → types.TableStructure（无主键返回 nil）。
func buildTableStructure(info *ModelInfo) *types.TableStructure {
	if info == nil || info.PrimaryColumn == "" {
		return nil
	}
	ts := &types.TableStructure{
		Table:     info.TableName,
		ModelName: getFullName(info.GoType),
		Columns:   []*types.ColumnStructure{},
		ColumnMap: map[string]*types.ColumnStructure{},
	}
	for _, f := range info.Fields {
		cs := &types.ColumnStructure{
			Name:    f.Column,
			Type:    f.Type,
			DbType:  goTypeToDbType(f.Type),
			Primary: f.Column == info.PrimaryColumn,
			Fill:    f.Fill,
		}
		ts.Columns = append(ts.Columns, cs)
		ts.ColumnMap[cs.Name] = cs
		if cs.Primary && ts.PrimaryColumn == nil {
			ts.PrimaryColumn = cs
		}
		if info.LogicColumn != "" && f.Column == info.LogicColumn {
			ts.LogicColumn = cs
		}
		if info.VersionColumn != "" && f.Column == info.VersionColumn {
			ts.VersionColumn = cs
		}
	}
	if ts.PrimaryColumn == nil {
		return nil
	}
	return ts
}

// modelStructureProviderCallback types 包回调：按 resultMap type 名查缓存，
// 命中且携带显式元数据（表名/主键/逻辑删除列）时返回 tag 推导的表结构。
func modelStructureProviderCallback(typeName string) *types.TableStructure {
	gCache.models.mu.RLock()
	info := gCache.models.Infos[typeName]
	gCache.models.mu.RUnlock()
	if info == nil || !info.hasExplicitMeta() {
		return nil
	}
	return buildTableStructure(info)
}

func init() {
	types.SetModelStructureProvider(modelStructureProviderCallback)
	types.SetUpsertSQLProvider(upsertSQLProviderCallback)
	types.SetUpsertSQLByFamily(upsertSQLByFamilyCallback)
}
