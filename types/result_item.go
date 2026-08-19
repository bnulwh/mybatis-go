package types

import (
	"github.com/bnulwh/mybatis-go/log"
	"reflect"
	"strings"
)

type ResultItemKind string

const (
	ResultItemKindId          ResultItemKind = "id"
	ResultItemKindResult      ResultItemKind = "result"
	ResultItemKindAssociation ResultItemKind = "association"
	ResultItemKindCollection  ResultItemKind = "collection"
)

type ResultItem struct {
	Column     string
	Type       reflect.Type
	Property   string
	PrimaryKey bool
	Kind       ResultItemKind
	JavaType   string // association/collection 的 javaType 属性
	OfType     string // collection 元素类型（ofType 属性）
	ResultMap  string // 引用的嵌套 resultMap id
}

// <id column="id" jdbcType="INTEGER" property="id" />
// <result column="created_by" jdbcType="VARCHAR" property="createdBy" />
// <association property="dept" javaType="SysDept" resultMap="deptResult" />
// <collection property="roles" javaType="java.util.List" ofType="SysRole" resultMap="RoleResult" />
func parseResultItemFromXmlNode(elem xmlElement) *ResultItem {
	log.Debugf("--parse result item from: %v", ToJson(elem))
	xn := elem.Val.(xmlNode)
	name := strings.ToLower(xn.Name)
	pro := xn.Attrs["property"]
	switch name {
	case "id":
		return &ResultItem{
			Column:     xn.Attrs["column"],
			Type:       ParseJdbcTypeFrom(xn.Attrs["jdbcType"]),
			Property:   pro,
			PrimaryKey: true,
			Kind:       ResultItemKindId,
		}
	case "result":
		return &ResultItem{
			Column:   xn.Attrs["column"],
			Type:     ParseJdbcTypeFrom(xn.Attrs["jdbcType"]),
			Property: pro,
			Kind:     ResultItemKindResult,
		}
	case "association":
		return &ResultItem{
			Property:  pro,
			Kind:      ResultItemKindAssociation,
			JavaType:  xn.Attrs["javaType"],
			ResultMap: xn.Attrs["resultMap"],
		}
	case "collection":
		return &ResultItem{
			Property:  pro,
			Kind:      ResultItemKindCollection,
			JavaType:  xn.Attrs["javaType"],
			OfType:    xn.Attrs["ofType"],
			ResultMap: xn.Attrs["resultMap"],
		}
	default:
		log.Warnf("unsupport result item: %v", name)
		return &ResultItem{
			Property: pro,
			Kind:     ResultItemKind(name),
		}
	}
}

// golangType 返回生成模型字段的 Go 类型字符串。
// 普通 result/id 沿用 jdbcType 推断；association/collection 依据
// javaType/ofType/嵌套 resultMap 建立真实关联类型（如 *SysDept / []SysRole）。
func (in *ResultItem) golangType(namedMaps map[string]*ResultMap) string {
	switch in.Kind {
	case ResultItemKindAssociation:
		return "*" + in.nestedTypeName(namedMaps)
	case ResultItemKindCollection:
		return "[]" + in.nestedTypeName(namedMaps)
	}
	return in.Type.String()
}

// nestedTypeName 解析嵌套模型短名：
// association 优先 javaType；collection 优先 ofType，其次引用的 resultMap 的 type。
func (in *ResultItem) nestedTypeName(namedMaps map[string]*ResultMap) string {
	tn := ""
	switch in.Kind {
	case ResultItemKindAssociation:
		tn = in.JavaType
	case ResultItemKindCollection:
		tn = in.OfType
		if tn == "" {
			tn = in.JavaType
		}
	}
	sname := GetShortName(tn)
	// java 容器类型不能作为元素类型（如 java.util.List）
	switch strings.ToUpper(sname) {
	case "LIST", "ARRAY", "ARRAYLIST", "LINKEDLIST", "SLICE", "SET", "HASHSET", "TREESET", "COLLECTION":
		sname = ""
	}
	if sname == "" && in.ResultMap != "" && namedMaps != nil {
		if rm, ok := namedMaps[in.ResultMap]; ok {
			sname = GetShortName(rm.TypeName)
		}
	}
	if sname == "" {
		log.Warnf("cannot resolve nested type for result item %v (kind=%v javaType=%v ofType=%v resultMap=%v)",
			in.Property, in.Kind, in.JavaType, in.OfType, in.ResultMap)
		return "interface{}"
	}
	return strings.TrimPrefix(toGolangType(sname), "models.")
}
