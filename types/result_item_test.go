package types

import (
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
