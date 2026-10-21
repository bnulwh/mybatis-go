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
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	parser := xml.NewDecoder(bytes.NewReader(stripXmlComments(content)))
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

var (
	cdataPrefix   = []byte("<![CDATA[")
	cdataSuffix   = []byte("]]>")
	commentPrefix = []byte("<!--")
	commentSuffix = []byte("-->")
)

// stripXmlComments 解析前移除 XML 注释（A2 前置修复）：
// 注释会把相邻文本切成多段 CharData，每段独立 TrimSpace 后拼接丢失中间空格
// （`SELECT a <!-- c --> FROM t` → `SELECT aFROM t`）。统一将注释替换为单个空格，
// 保证文本连续；CDATA 段内的 `<!--` 属于文本数据，原样保留；未闭合注释保留由解码器报错。
func stripXmlComments(content []byte) []byte {
	var out bytes.Buffer
	out.Grow(len(content))
	i := 0
	for i < len(content) {
		if bytes.HasPrefix(content[i:], cdataPrefix) {
			end := bytes.Index(content[i+len(cdataPrefix):], cdataSuffix)
			if end < 0 {
				out.Write(content[i:])
				return out.Bytes()
			}
			stop := i + len(cdataPrefix) + end + len(cdataSuffix)
			out.Write(content[i:stop])
			i = stop
			continue
		}
		if bytes.HasPrefix(content[i:], commentPrefix) {
			end := bytes.Index(content[i+len(commentPrefix):], commentSuffix)
			if end < 0 {
				out.Write(content[i:])
				return out.Bytes()
			}
			i += len(commentPrefix) + end + len(commentSuffix)
			out.WriteByte(' ')
			continue
		}
		out.WriteByte(content[i])
		i++
	}
	return out.Bytes()
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
