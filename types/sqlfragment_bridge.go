package types

import (
	"io"

	"github.com/bnulwh/mybatis-go/types/sqlfragment"
)

// P0-3a：XML 解析与动态 SQL 片段引擎已整体迁移至 types/sqlfragment 独立包。
// 本文件以类型别名 + 薄包装保持 types 包内部既有调用
// （result_map / result_item / sql_mapper / mp_builtin 及测试中的 xmlNode 字面量等）零改动。

type xmlNode = sqlfragment.XmlNode
type xmlElement = sqlfragment.XmlElement
type xmlElemType = sqlfragment.XmlElemType

const (
	xmlTextElem = sqlfragment.XmlTextElem // 静态文本节点
	xmlNodeElem = sqlfragment.XmlNodeElem // 节点子节点
)

type SqlElement = sqlfragment.SqlElement

func parseXmlFile(filename string) (*xmlNode, error) {
	return sqlfragment.ParseXmlFile(filename)
}

// parseXmlContent 解析内存中的 XML 内容（embed.FS / 内存来源，无需落盘）。
func parseXmlContent(content []byte) (*xmlNode, error) {
	return sqlfragment.ParseXmlContent(content)
}

func parseXmlNode(r io.Reader) (*xmlNode, error) {
	return sqlfragment.ParseXmlNode(r)
}

func parseSqlElementFromXmlNode(node xmlNode) *SqlElement {
	return sqlfragment.ParseSqlElementFromXmlNode(node)
}
