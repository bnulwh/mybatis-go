package types

import (
	"bytes"
	"encoding/xml"
	"io"
	"io/ioutil"
	"strings"
)

type XmlElemType string

const (
	xmlTextElem XmlElemType = "text" // 静态文本节点
	xmlNodeElem XmlElemType = "node" // 节点子节点
)
type xmlElement struct {
	ElementType XmlElemType
	Val         interface{}
}

type xmlNode struct {
	Id       string
	Name     string
	Attrs    map[string]xml.Attr
	Elements []xmlElement
}

func parseXmlFile(filename string) *xmlNode{
	content,err := ioutil.ReadFile(filename)
	if err != nil{
		return nil
	}
	r := bytes.NewReader(content)
	return parseXmlNode(r)
}

func parseXmlNode(r io.Reader) *xmlNode {
	parser := xml.NewDecoder(r)
	var root xmlNode

	st := NewStack()
	for {
		token, err := parser.Token()
		if err != nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement: //tag start
			elmt := xml.StartElement(t)
			name := elmt.Name.Local
			attr := elmt.Attr
			attrMap := make(map[string]xml.Attr)
			for _, val := range attr {
				attrMap[val.Name.Local] = val
			}
			node := xmlNode{
				Name:     name,
				Attrs:    attrMap,
				Elements: make([]xmlElement, 0),
			}
			for _, val := range attr {
				if val.Name.Local == "id" {
					node.Id = val.Value
				}
			}
			st.Push(node)

		case xml.EndElement: //tag end
			if st.Len() > 0 {
				//cur node
				n := st.Pop().(xmlNode)
				if st.Len() > 0 { //if the root xmlNode then append to xmlElement
					e := xmlElement{
						ElementType: xmlNodeElem,
						Val:         n,
					}

					pn := st.Pop().(xmlNode)
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
				n := st.Pop().(xmlNode)

				bts := xml.CharData(t)
				content := strings.TrimSpace(string(bts))
				if content != "" {
					e := xmlElement{
						ElementType: xmlTextElem,
						Val:         content,
					}
					els := n.Elements
					els = append(els, e)
					n.Elements = els
				}

				st.Push(n)
			}

		case xml.Comment:
		case xml.ProcInst:
		case xml.Directive:
		default:
		}
	}

	if st.Len() != 0 {
		panic("Parse xml error, there is tag no close, please check your xml config!")
	}

	return &root
}
