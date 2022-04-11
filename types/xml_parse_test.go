package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func Test_parseXmlFile(t *testing.T) {
	dir, _ := os.Getwd()
	fname := filepath.Join(dir, "..", "resources/mapper/UserInfoModelMapper.xml")
	//fmt.Println(os.Getwd())
	r, _ := parseXmlFile(fname)
	bs, _ := json.MarshalIndent(r, "", "    ")
	fmt.Println(string(bs))
}

func Test_parseXmlNode(t *testing.T) {
	content := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>" +
		"<!DOCTYPE mapper PUBLIC \"-//mybatis.org//DTD Mapper 3.0//EN\" \"http://mybatis.org/dtd/mybatis-3-mapper.dtd\">" +
		"<mapper namespace=\"UserInfoModelMapper\">" +
		"  <resultMap id=\"BaseResultMap\" type=\"UserInfoModel\">" +
		"    <id column=\"id\" jdbcType=\"INTEGER\" property=\"id\" />" +
		"    <result column=\"created_by\" jdbcType=\"VARCHAR\" property=\"createdBy\" />" +
		"    <result column=\"updated_by\" jdbcType=\"VARCHAR\" property=\"updatedBy\" />" +
		"    <result column=\"create_time\" jdbcType=\"TIMESTAMP\" property=\"createTime\" />" +
		"    <result column=\"update_time\" jdbcType=\"TIMESTAMP\" property=\"updateTime\" />" +
		"    <result column=\"group_id\" jdbcType=\"INTEGER\" property=\"groupId\" />" +
		"    <result column=\"username\" jdbcType=\"VARCHAR\" property=\"username\" />" +
		"    <result column=\"pass_md5\" jdbcType=\"VARCHAR\" property=\"passMd5\" />" +
		"    <result column=\"roles\" jdbcType=\"VARCHAR\" property=\"roles\" />" +
		"    <result column=\"description\" jdbcType=\"VARCHAR\" property=\"description\" />" +
		"    <result column=\"avatar\" jdbcType=\"VARCHAR\" property=\"avatar\" />" +
		"  </resultMap>" +
		"  <sql id=\"base_column_list\">" +
		"    id, created_by, updated_by, create_time, update_time, group_id, username," +
		"    pass_md5, roles, description, avatar" +
		"  </sql>" +
		"  <delete id=\"deleteByPrimaryKey\" parameterType=\"java.lang.Integer\">" +
		"    delete from user_info" +
		"    where id = #{id,jdbcType=INTEGER}" +
		"  </delete>" +
		"  <insert id=\"insert\" parameterType=\"com.chinahuik.wechat.models.UserInfoModel\">" +
		"    insert into user_info (id, created_by, updated_by, " +
		"      create_time, update_time, group_id, " +
		"      username, pass_md5, roles, " +
		"      description, avatar)" +
		"    values (#{id,jdbcType=INTEGER}, #{createdBy,jdbcType=VARCHAR}, #{updatedBy,jdbcType=VARCHAR}, " +
		"      #{createTime,jdbcType=TIMESTAMP}, #{updateTime,jdbcType=TIMESTAMP}, #{groupId,jdbcType=INTEGER}, " +
		"      #{username,jdbcType=VARCHAR}, #{passMd5,jdbcType=VARCHAR}, #{roles,jdbcType=VARCHAR}, " +
		"      #{description,jdbcType=VARCHAR}, #{avatar,jdbcType=VARCHAR})" +
		"  </insert>" +
		"  <update id=\"updateByPrimaryKey\" parameterType=\"com.chinahuik.wechat.models.UserInfoModel\">" +
		"    update user_info" +
		"    set created_by = #{createdBy,jdbcType=VARCHAR}," +
		"      updated_by = #{updatedBy,jdbcType=VARCHAR}," +
		"      create_time = #{createTime,jdbcType=TIMESTAMP}," +
		"      update_time = #{updateTime,jdbcType=TIMESTAMP}," +
		"      group_id = #{groupId,jdbcType=INTEGER}," +
		"      username = #{username,jdbcType=VARCHAR}," +
		"      pass_md5 = #{passMd5,jdbcType=VARCHAR}," +
		"      roles = #{roles,jdbcType=VARCHAR}," +
		"      description = #{description,jdbcType=VARCHAR}," +
		"      avatar = #{avatar,jdbcType=VARCHAR}" +
		"    where id = #{id,jdbcType=INTEGER}" +
		"  </update>" +
		"  <select id=\"selectByPrimaryKey\" parameterType=\"java.lang.Integer\" resultMap=\"BaseResultMap\">" +
		"    select <include refid=\"base_column_list\"/>" +
		"    from user_info" +
		"    where id = #{id,jdbcType=INTEGER}" +
		"  </select>" +
		"  <select id=\"selectAll\" resultMap=\"BaseResultMap\">" +
		"    select <include refid=\"base_column_list\"/>" +
		"    from user_info" +
		"  </select>" +
		"</mapper>"
	r := bytes.NewReader([]byte(content))
	r1, _ := parseXmlNode(r)
	bs, _ := json.MarshalIndent(r1, "", "    ")
	fmt.Println(string(bs))

}
