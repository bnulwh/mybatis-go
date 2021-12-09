package main

import (
	"encoding/xml"
	"fmt"
	"github.com/beevik/etree"
	"os"
	"reflect"
)

type XmlMapper struct {
	XMLName   xml.Name `xml:"mapper"`
	Namespace string   `xml:"namespace,attr"`
	Values    []interface{}
}
type XmlResultMap struct {
	XMLName xml.Name `xml:"resultMap"`
	Id      string   `xml:"id,attr"`
	Type    string   `xml:"type,attr"`
	Values  []interface{}
}
type XmlInclude struct {
	XMLName xml.Name `xml:"include"`
	Refid   string   `xml:"refid,attr"`
}
type XmlResultId struct {
	XMLName  xml.Name `xml:"id"`
	Column   string   `xml:"column,attr"`
	JdbcType string   `xml:"jdbcType,attr"`
	Property string   `xml:"property,attr"`
}
type XmlResultItem struct {
	XMLName  xml.Name `xml:"result"`
	Column   string   `xml:"column,attr"`
	JdbcType string   `xml:"jdbcType,attr"`
	Property string   `xml:"property,attr"`
}
type XmlSql struct {
	XMLName xml.Name `xml:"sql"`
	Id      string   `xml:"id,attr"`
	Sql     string   `xml:",chardata"`
}
type XmlCharData struct {
	//*xml.Marshaler
	Sql string `xml:",chardata"`
}
type XmlFunction struct {
	XMLName       xml.Name //`xml:"delete"`
	Id            string   `xml:"id,attr"`
	ParameterType string   `xml:"parameterType,attr"`
	ResultMap     string   `xml:"resultMap,attr,omitempty"`
	ResultType    string   `xml:"resultType,attr,omitempty"`
	Values        []interface{}
}

//func (xi XmlInclude)MarshalXML(e *xml.Encoder, start xml.StartElement) error {
//	fmt.Println("XmlInclude MarshalXML called.")
//	//err := e.EncodeToken(xml.CharData([]byte(xcd.Sql)))
//	//fmt.Println(err)
//	str := fmt.Sprintf("<include refid=\"%s\"/>",xi.Refid)
//	return e.EncodeToken(xml.CharData([]byte(str)))
//	//err :=e.EncodeToken(xml.StartElement{
//	//	Name:xml.Name{Local:"include"},
//	//	Attr:[]xml.Attr{
//	//		{Name:xml.Name{Local:"refid"},Value:xi.Refid},
//	//	},
//	//})
//	//if err !=nil{
//	//	fmt.Println(err)
//	//}
//	//
//	//return e.EncodeToken(xml.EndElement{Name:xml.Name{Local:"include"}})
//}

func (xcd XmlCharData) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	fmt.Println("XmlCharData MarshalXML called.")
	//err := e.EncodeToken(xml.CharData([]byte(xcd.Sql)))
	//fmt.Println(err)
	return e.EncodeToken(xml.CharData([]byte(xcd.Sql)))
}

//func (xcd *XmlCharData) MarshalText() ([]byte, error) {
//	fmt.Println("XmlCharData MarshalText called.")
//	return []byte(xcd.Sql), nil
//}

//func (xd XmlDelete) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
//	fmt.Println("XmlDelete MarshalXML called.")
//	return nil
//}

func main() {
	marshalerType := reflect.TypeOf((*xml.Marshaler)(nil)).Elem()

	a := XmlMapper{
		Namespace: "test",
		Values: []interface{}{
			XmlResultMap{
				Id:   "BaseResultMap",
				Type: "UserInfoModel",
				Values: []interface{}{
					XmlResultId{
						Column:   "id",
						JdbcType: "INTEGER",
						Property: "id",
					},
					XmlResultItem{
						Column:   "created_by",
						JdbcType: "VARCHAR",
						Property: "createdBy",
					},
				},
			},
			XmlSql{
				Id: "base_column_list",
				Sql: "id, created_by, updated_by, create_time, update_time, group_id, username," +
					"pass_md5, roles, description, avatar",
			},
			XmlFunction{
				XMLName:       xml.Name{Local: "delete"},
				Id:            "deleteByPrimaryKey",
				ParameterType: "java.lang.Integer",
				Values: []interface{}{
					XmlCharData{
						Sql: "delete from user_info where id = #{id,jdbcType=INTEGER}",
					},
				},
			},
			XmlFunction{
				XMLName:       xml.Name{Local: "insert"},
				Id:            "insert",
				ParameterType: "com.chinahuik.wechat.models.UserInfoModel",
				Values: []interface{}{
					XmlCharData{
						Sql: "insert into user_info (id, created_by, updated_by, " +
							"create_time, update_time, group_id," +
							"username, pass_md5, roles," +
							"description, avatar)" +
							"values (#{id,jdbcType=INTEGER}, #{createdBy,jdbcType=VARCHAR}, #{updatedBy,jdbcType=VARCHAR}," +
							"#{createTime,jdbcType=TIMESTAMP}, #{updateTime,jdbcType=TIMESTAMP}, #{groupId,jdbcType=INTEGER}," +
							"#{username,jdbcType=VARCHAR}, #{passMd5,jdbcType=VARCHAR}, #{roles,jdbcType=VARCHAR}," +
							"#{description,jdbcType=VARCHAR}, #{avatar,jdbcType=VARCHAR})",
					},
				},
			},
			XmlFunction{
				XMLName:       xml.Name{Local: "update"},
				Id:            "updateByPrimaryKey",
				ParameterType: "com.chinahuik.wechat.models.UserInfoModel",
				Values: []interface{}{
					XmlCharData{
						Sql: "update user_info " +
							"set created_by = #{createdBy,jdbcType=VARCHAR}," +
							"updated_by = #{updatedBy,jdbcType=VARCHAR}," +
							"create_time = #{createTime,jdbcType=TIMESTAMP}," +
							"update_time = #{updateTime,jdbcType=TIMESTAMP}," +
							"group_id = #{groupId,jdbcType=INTEGER}," +
							"username = #{username,jdbcType=VARCHAR}," +
							"pass_md5 = #{passMd5,jdbcType=VARCHAR}," +
							"roles = #{roles,jdbcType=VARCHAR}," +
							"description = #{description,jdbcType=VARCHAR}," +
							"avatar = #{avatar,jdbcType=VARCHAR} " +
							"where id = #{id,jdbcType=INTEGER}",
					},
				},
			},
			XmlFunction{
				XMLName:       xml.Name{Local: "select"},
				Id:            "selectByPrimaryKey",
				ParameterType: "java.lang.Integer",
				ResultMap:     "BaseResultMap",
				Values: []interface{}{
					XmlCharData{
						Sql: "select ",
					},
					XmlInclude{
						Refid: "base_column_list",
					},
					XmlCharData{
						Sql: " from user_info " +
							"where id = #{id,jdbcType=INTEGER}",
					},
				},
			},
			XmlFunction{
				XMLName:   xml.Name{Local: "select"},
				Id:        "selectAll",
				ResultMap: "BaseResultMap",
				Values: []interface{}{
					XmlCharData{
						Sql: "select ",
					},
					XmlInclude{
						Refid: "base_column_list",
					},
					XmlCharData{
						Sql: " from user_info ",
					},
				},
			},
		},
	}

	output, _ := xml.MarshalIndent(a, "", "    ")

	fmt.Println(string(output))

	output2, _ := xml.Marshal(a)
	fmt.Println(string(output2))

	b := XmlFunction{
		XMLName: xml.Name{
			Local: "update",
		},
		Id:            "deleteByPrimaryKey",
		ParameterType: "java.lang.Integer",
		Values: []interface{}{
			&XmlCharData{
				Sql: "delete from user_info where id = #{id,jdbcType=INTEGER}",
			},
			//"delete from user_info where id = #{id,jdbcType=INTEGER}",
		},
	}
	typ := reflect.TypeOf(b)
	kind := reflect.TypeOf(b).Kind()
	fmt.Println(kind)
	fmt.Println(typ.Implements(marshalerType))
	output3, _ := xml.MarshalIndent(b, "", "    ")

	fmt.Println(string(output3))
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)
	doc.CreateDirective(`DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd"`)
	mapper := doc.CreateElement("mapper")
	mapper.CreateAttr("namespace", "test")
	mapper.CreateText("test hello")
	xi := mapper.CreateElement("include")
	xi.CreateAttr("refid", "base_column_list")
	mapper.CreateText(" from xxx")
	//doc.Indent(2)
	doc.IndentTabs()
	doc.WriteTo(os.Stdout)

}
