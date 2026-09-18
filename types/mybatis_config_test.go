package types

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeCfgFixture 在 base 下写一个最小可加载的 Mapper XML，返回相对路径。
func writeCfgFixture(t *testing.T, base, rel string) string {
	t.Helper()
	body := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="com.cfg.system.mapper.` + strings.TrimSuffix(filepath.Base(rel), ".xml") + `">
	<resultMap id="BaseResultMap" type="CfgUser">
		<id column="user_id" jdbcType="BIGINT" property="userId"/>
		<result column="user_name" jdbcType="VARCHAR" property="userName"/>
		<result column="deleted" jdbcType="INTEGER" property="deleted"/>
	</resultMap>
	<select id="SelectUserByUserName" parameterType="string" resultMap="BaseResultMap">
		select * from cfg_user where user_name = #{userName}
	</select>
</mapper>
`
	return writeCfgFile(t, base, rel, body)
}

// writeCfgFile 在 base 下写文本文件（自动创建父目录），返回绝对路径。
func writeCfgFile(t *testing.T, base, rel, content string) string {
	t.Helper()
	p := filepath.Join(base, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Errorf("mkdir %v failed: %v", filepath.Dir(p), err)
		return ""
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Errorf("write %v failed: %v", p, err)
		return ""
	}
	return p
}

func Test_LoadMappersFromMyBatisConfig_Properties(t *testing.T) {
	base := t.TempDir()
	writeCfgFixture(t, base, "mapper/SysUserMapper.xml")
	// classpath*: 前缀 + 逗号多值 + 驼峰键（relaxed binding）
	cfg := writeCfgFile(t, base, "application.properties",
		"mybatis-plus.mapperLocations=classpath*:mapper/*.xml\n")
	mps, info, err := LoadMappersFromMyBatisConfig(cfg)
	if err != nil {
		t.Errorf("load from properties failed: %v", err)
		return
	}
	if len(info.XmlFiles) != 1 {
		t.Errorf("expect 1 xml file, got %v (%v)", len(info.XmlFiles), info.XmlFiles)
	}
	if len(mps.Mappers) != 1 || mps.Mappers[0].Namespace != "com.cfg.system.mapper.SysUserMapper" {
		t.Errorf("expect 1 mapper SysUserMapper, got %+v", mps.Mappers)
	}
}

func Test_LoadMappersFromMyBatisConfig_Yaml(t *testing.T) {
	base := t.TempDir()
	writeCfgFixture(t, base, "mapper/system/SysUserMapper.xml")
	writeCfgFixture(t, base, "mapper/SysRoleMapper.xml")
	// 块列表 + ** 跨目录 + classpath: 前缀
	cfg := writeCfgFile(t, base, "application.yml",
		"mybatis-plus:\n"+
			"  mapper-locations:\n"+
			"    - classpath:mapper/**/*.xml\n")
	mps, info, err := LoadMappersFromMyBatisConfig(cfg)
	if err != nil {
		t.Errorf("load from yaml failed: %v", err)
		return
	}
	if len(info.XmlFiles) != 2 {
		t.Errorf("expect 2 xml files (recursive **), got %v (%v)", len(info.XmlFiles), info.XmlFiles)
	}
	if len(mps.Mappers) != 2 {
		t.Errorf("expect 2 mappers, got %v", len(mps.Mappers))
	}
}

func Test_LoadMappersFromMyBatisConfig_MyBatisConfigXml(t *testing.T) {
	base := t.TempDir()
	writeCfgFixture(t, base, "mybatis/mappers/SysUserMapper.xml")
	abs := writeCfgFixture(t, base, "mapper/SysRoleMapper.xml")
	// resource（classpath 相对）+ url（file: 绝对路径）
	fileURL := "file:///" + strings.ReplaceAll(filepath.ToSlash(abs), " ", "%20")
	cfg := writeCfgFile(t, base, "mybatis-config.xml",
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"+
			"<configuration>\n"+
			"	<mappers>\n"+
			"		<mapper resource=\"mybatis/mappers/SysUserMapper.xml\"/>\n"+
			"		<mapper url=\""+fileURL+"\"/>\n"+
			"	</mappers>\n"+
			"</configuration>\n")
	mps, info, err := LoadMappersFromMyBatisConfig(cfg)
	if err != nil {
		t.Errorf("load from mybatis-config.xml failed: %v", err)
		return
	}
	if len(info.XmlFiles) != 2 {
		t.Errorf("expect 2 xml files, got %v (%v)", len(info.XmlFiles), info.XmlFiles)
	}
	if len(mps.Mappers) != 2 {
		t.Errorf("expect 2 mappers, got %v", len(mps.Mappers))
	}
}

func Test_LoadMappersFromMyBatisConfig_SpringXml(t *testing.T) {
	base := t.TempDir()
	writeCfgFixture(t, base, "mapper/SysUserMapper.xml")
	// MybatisSqlSessionFactoryBean 的 mapperLocations：<list><value> 形态
	cfg := writeCfgFile(t, base, "spring-datasource.xml",
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"+
			"<beans>\n"+
			"	<bean id=\"sqlSessionFactory\" class=\"com.baomidou.mybatisplus.extension.spring.MybatisSqlSessionFactoryBean\">\n"+
			"		<property name=\"mapperLocations\">\n"+
			"			<list>\n"+
			"				<value>classpath*:mapper/*.xml</value>\n"+
			"			</list>\n"+
			"		</property>\n"+
			"	</bean>\n"+
			"</beans>\n")
	mps, info, err := LoadMappersFromMyBatisConfig(cfg)
	if err != nil {
		t.Errorf("load from spring xml failed: %v", err)
		return
	}
	if len(info.XmlFiles) != 1 || len(mps.Mappers) != 1 {
		t.Errorf("expect 1 xml / 1 mapper, got %v xml / %v mappers", len(info.XmlFiles), len(mps.Mappers))
	}
}

func Test_LoadMappersFromMyBatisConfig_ConfigLocationChain(t *testing.T) {
	base := t.TempDir()
	writeCfgFixture(t, base, "mybatis/mappers/SysUserMapper.xml")
	// properties 无 mapper-locations，只有 config-location → 链式进入 mybatis-config.xml；
	// 其 resource 以 classpath 根（= properties 所在目录）解析，而非 mybatis-config.xml 所在目录
	writeCfgFile(t, base, "mybatis/mybatis-config.xml",
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"+
			"<configuration>\n"+
			"	<mappers>\n"+
			"		<mapper resource=\"mybatis/mappers/SysUserMapper.xml\"/>\n"+
			"	</mappers>\n"+
			"</configuration>\n")
	cfg := writeCfgFile(t, base, "application.properties",
		"mybatis.config-location=classpath:mybatis/mybatis-config.xml\n")
	mps, info, err := LoadMappersFromMyBatisConfig(cfg)
	if err != nil {
		t.Errorf("load via config-location chain failed: %v", err)
		return
	}
	if len(info.XmlFiles) != 1 || len(mps.Mappers) != 1 {
		t.Errorf("expect 1 xml / 1 mapper, got %v xml / %v mappers", len(info.XmlFiles), len(mps.Mappers))
	}
	if len(info.Locations) != 1 || info.Locations[0] != "mybatis/mappers/SysUserMapper.xml" {
		t.Errorf("expect chained location mybatis/mappers/SysUserMapper.xml, got %v", info.Locations)
	}
}

func Test_LoadMappersFromMyBatisConfig_Errors(t *testing.T) {
	base := t.TempDir()
	// 不支持的扩展名
	bad := writeCfgFile(t, base, "config.json", "{}")
	if _, _, err := LoadMappersFromMyBatisConfig(bad); err == nil {
		t.Error("expect error for unsupported .json config")
	}
	// 配置存在但解析不到任何 XML
	empty := writeCfgFile(t, base, "application.yml", "spring:\n  application:\n    name: demo\n")
	if _, _, err := LoadMappersFromMyBatisConfig(empty); err == nil {
		t.Error("expect error when no mapper xml resolved")
	}
	// 文件不存在
	if _, _, err := LoadMappersFromMyBatisConfig(filepath.Join(base, "no-such.yml")); err == nil {
		t.Error("expect error for missing config file")
	}
}

func Test_ExpandMapperLocations_FallbackBases(t *testing.T) {
	base := t.TempDir()
	writeCfgFixture(t, base, "mapper/SysUserMapper.xml")
	// 配置在 mybatis/ 子目录、XML 在资源根：相对模式未命中时回退父目录
	sub := filepath.Join(base, "mybatis")
	files := ExpandMapperLocations(sub, []string{"classpath*:mapper/*.xml"})
	if len(files) != 1 {
		t.Errorf("expect fallback to parent dir finds 1 xml, got %v (%v)", len(files), files)
	}
}
