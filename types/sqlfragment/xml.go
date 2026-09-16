package sqlfragment

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// XmlElemType XML 元素类型：静态文本节点 / 节点子节点。
type XmlElemType string

const (
	XmlTextElem XmlElemType = "text" // 静态文本节点
	XmlNodeElem XmlElemType = "node" // 节点子节点
)

// XmlElement XML 解析后的元素：文本节点 Val 为 string，节点子节点 Val 为 XmlNode。
type XmlElement struct {
	ElementType XmlElemType
	Val         interface{}
}

// XmlNode XML 节点：标签名、属性与子元素。
type XmlNode struct {
	Id       string
	Name     string
	Attrs    map[string]string
	Elements []XmlElement
}

// ParseXmlFile 解析磁盘上的 XML 文件。
func ParseXmlFile(filename string) (*XmlNode, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ParseXmlContent(content)
}

// ParseXmlContent 解析内存中的 XML 内容（embed.FS / 内存来源，无需落盘）。
func ParseXmlContent(content []byte) (*XmlNode, error) {
	return ParseXmlNode(bytes.NewReader(content))
}

// ParseXmlNode 从 Reader 解析 XML 为节点树。
func ParseXmlNode(r io.Reader) (*XmlNode, error) {
	parser := xml.NewDecoder(r)
	var root XmlNode

	st := newStack()
	for {
		token, err := parser.Token()
		if err != nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement: //tag start
			node := startElement2XmlNode(t)
			st.Push(node)

		case xml.EndElement: //tag end
			if st.Len() > 0 {
				//cur node
				n := st.Pop().(XmlNode)
				if st.Len() > 0 { //if the root XmlNode then append to XmlElement
					e := XmlElement{
						ElementType: XmlNodeElem,
						Val:         n,
					}

					pn := st.Pop().(XmlNode)
					els := pn.Elements
					els = append(els, e)
					pn.Elements = els
					st.Push(pn)
				} else { //else root = n
					root = n
				}
			}
		case xml.CharData: //tag content
			if st.Len() > 0 {
				n := charData2XmlNode(st, t)
				st.Push(n)
			}

		case xml.Comment:
		case xml.ProcInst:
		case xml.Directive:
		default:
		}
	}

	if st.Len() != 0 {
		return nil, fmt.Errorf("parse xml error, there is tag no close, please check your xml config")
	}

	return &root, nil
}

func charData2XmlNode(st *stack, t xml.Token) XmlNode {
	n := st.Pop().(XmlNode)
	bts := t.(xml.CharData)
	content := strings.TrimSpace(string(bts))
	if content != "" {
		e := XmlElement{
			ElementType: XmlTextElem,
			Val:         content,
		}
		els := n.Elements
		els = append(els, e)
		n.Elements = els
	}
	return n
}

func startElement2XmlNode(t xml.Token) XmlNode {
	elmt := t.(xml.StartElement)
	name := elmt.Name.Local
	attr := elmt.Attr
	attrMap := make(map[string]string)
	for _, val := range attr {
		attrMap[val.Name.Local] = val.Value
	}
	node := XmlNode{
		Name:     name,
		Attrs:    attrMap,
		Elements: make([]XmlElement, 0),
	}
	for _, val := range attr {
		if val.Name.Local == "id" {
			node.Id = val.Value
		}
	}
	return node
}
