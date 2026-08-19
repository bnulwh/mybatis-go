package types

import (
	"strings"
	"testing"
)

func Test_parseResultItemFromXmlNode(t *testing.T) {
	e1 := xmlElement{
		ElementType: xmlNodeElem,
		Val: xmlNode{
			Name: "id",
			Attrs: map[string]string{
				"column":   "",
				"jdbcType": "",
				"property": "",
			},
		},
	}
	parseResultItemFromXmlNode(e1)
}

func Test_parseResultItemFromXmlNode_Association(t *testing.T) {
	e := xmlElement{
		ElementType: xmlNodeElem,
		Val: xmlNode{
			Name: "association",
			Attrs: map[string]string{
				"property":  "dept",
				"javaType":  "SysDept",
				"resultMap": "deptResult",
			},
		},
	}
	item := parseResultItemFromXmlNode(e)
	if item.Kind != ResultItemKindAssociation {
		t.Error("association kind =", item.Kind, "want", ResultItemKindAssociation)
	}
	if item.JavaType != "SysDept" {
		t.Error("association javaType =", item.JavaType, "want SysDept")
	}
	if item.ResultMap != "deptResult" {
		t.Error("association resultMap =", item.ResultMap, "want deptResult")
	}
	if item.Column != "" {
		t.Error("association column =", item.Column, "want empty")
	}
}

func Test_parseResultItemFromXmlNode_Collection(t *testing.T) {
	e := xmlElement{
		ElementType: xmlNodeElem,
		Val: xmlNode{
			Name: "collection",
			Attrs: map[string]string{
				"property":  "roles",
				"javaType":  "java.util.List",
				"ofType":    "SysRole",
				"resultMap": "RoleResult",
			},
		},
	}
	item := parseResultItemFromXmlNode(e)
	if item.Kind != ResultItemKindCollection {
		t.Error("collection kind =", item.Kind, "want", ResultItemKindCollection)
	}
	if item.OfType != "SysRole" {
		t.Error("collection ofType =", item.OfType, "want SysRole")
	}
}

// Test_ResultItemGolangType_Samples 使用 samples/（RuoYi Mapper）真实文件回归：
// SysUserResult 中 association/collection 必须生成真实关联类型，而非 string 假字段（S-06）。
func Test_ResultItemGolangType_Samples(t *testing.T) {
	mps := NewSqlMappers("../samples")
	if mps == nil || len(mps.Mappers) == 0 {
		t.Error("load samples failed")
		return
	}
	var userMap *ResultMap
	var namedMaps map[string]*ResultMap
	for i := range mps.Mappers {
		mp := &mps.Mappers[i]
		if mp.Namespace == "" || !strings.HasSuffix(mp.Namespace, "SysUserMapper") {
			continue
		}
		namedMaps = mp.NamedMaps
		for _, rm := range mp.Maps {
			if rm.Id == "SysUserResult" {
				userMap = rm
			}
		}
	}
	if userMap == nil {
		t.Error("SysUserResult not found in samples")
		return
	}
	if namedMaps == nil {
		t.Error("SysUserMapper named maps not found in samples")
		return
	}
	var dept, roles *ResultItem
	for _, item := range userMap.Results {
		switch item.Property {
		case "dept":
			dept = item
		case "roles":
			roles = item
		}
	}
	if dept == nil {
		t.Error("dept association not found")
	} else {
		if dept.Kind != ResultItemKindAssociation {
			t.Error("dept kind =", dept.Kind, "want association")
		}
		if g := dept.golangType(namedMaps); g != "*SysDept" {
			t.Error("dept golangType =", g, "want *SysDept")
		}
	}
	if roles == nil {
		t.Error("roles collection not found")
	} else {
		if roles.Kind != ResultItemKindCollection {
			t.Error("roles kind =", roles.Kind, "want collection")
		}
		if g := roles.golangType(namedMaps); g != "[]SysRole" {
			t.Error("roles golangType =", g, "want []SysRole")
		}
	}
}

// Test_ResultMapGenerateContent_Samples 验证生成的模型结构体不出现 Dept string/Roles string 假字段（S-06）。
func Test_ResultMapGenerateContent_Samples(t *testing.T) {
	mps := NewSqlMappers("../samples")
	if mps == nil || len(mps.Mappers) == 0 {
		t.Error("load samples failed")
		return
	}
	for i := range mps.Mappers {
		mp := &mps.Mappers[i]
		if mp.Namespace == "" || !strings.HasSuffix(mp.Namespace, "SysUserMapper") {
			continue
		}
		for _, rm := range mp.Maps {
			if rm.Id != "SysUserResult" {
				continue
			}
			content := string(rm.generateContent("src", mp.NamedMaps))
			if !strings.Contains(content, "Dept \t*SysDept") {
				t.Error("generated SysUser model missing `Dept *SysDept`, content:\n", content)
			}
			if !strings.Contains(content, "Roles \t[]SysRole") {
				t.Error("generated SysUser model missing `Roles []SysRole`, content:\n", content)
			}
			if strings.Contains(content, "Dept \tstring") || strings.Contains(content, "Roles \tstring") {
				t.Error("generated SysUser model still has fake string fields:\n", content)
			}
		}
	}
}
