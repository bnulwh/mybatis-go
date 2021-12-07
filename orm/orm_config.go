package orm

import (
	"errors"
	"fmt"
	log "github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/utils"
	"regexp"
	"strconv"
	"strings"
)

type DatabaseType string

const (
	MySqlDb    DatabaseType = "mysql"
	PostgresDb DatabaseType = "postgres"
)

type DatabaseConfig struct {
	Host     string
	Port     int64
	Username string
	Password string
	DbName   string
	DbType   DatabaseType
}

type MyBatisConfig struct {
	DbConfig         DatabaseConfig
	MapperLocations  string
	TypeAliasPackage string
	MaxRows          int64
}

func NewConfig(filename string) *MyBatisConfig {
	cm := utils.LoadSettings(filename)
	return NewConfigFromSettings(cm)
}
func NewConfigFromSettings(cm map[string]string) *MyBatisConfig {
	dbc := parseDatabaseConfig(cm)
	ml := cm["mybatis.mapper-locations"]
	return &MyBatisConfig{
		DbConfig:        *dbc,
		MapperLocations: ml,
	}
}

func (in *DatabaseConfig) GenerateConn() (string, string) {
	switch in.DbType {
	case PostgresDb:
		return "postgres", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			in.Host, in.Port, in.Username, in.Password, in.DbName)
	case MySqlDb:
		return "mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			in.Username, in.Password, in.Host, in.Port, in.DbName)
	default:
		panic(fmt.Sprintf("not support database type %v", in.DbType))
	}
}

func parseDatabaseConfig(m map[string]string) *DatabaseConfig {
	tp, h, P, d, err := parseAddr(m)
	if err != nil {
		log.Errorf("parse postgres addr failed: %v", err)
		panic(err)
	}
	u, ok := m["spring.datasource.username"]
	if !ok {
		log.Errorf("get database username failed.")
		panic("get database username failed.")
	}
	p, ok := m["spring.datasource.password"]
	if !ok {
		log.Errorf("get database password failed.")
		panic("get database password failed.")
	}
	return &DatabaseConfig{
		Host:     h,
		Port:     P,
		Username: u,
		Password: p,
		DbName:   d,
		DbType:   parseDatabaseType(tp),
	}
}
func parseDatabaseType(tps string) DatabaseType {
	switch strings.ToLower(tps) {
	case "mysql":
		return MySqlDb
	case "postgres", "postgresql":
		return PostgresDb
	default:
		panic(fmt.Sprintf("not support database type %v", tps))
	}
}
func parseAddr(m map[string]string) (string, string, int64, string, error) {
	val, ok := m["spring.datasource.url"]
	if !ok {
		return "", "", 0, "", errors.New("not found key spring.datasource.url")
	}
	re := regexp.MustCompile(`jdbc:([\w]+)://([\w-\\.]+):([\d]+)/([\w_-]+)`)
	matched := re.FindStringSubmatch(val)
	if len(matched) < 5 {
		return "", "", 0, "", errors.New("unsupport formate of spring.datasource.url")
	}
	i, _ := strconv.Atoi(matched[3])
	return matched[1], matched[2], int64(i), matched[4], nil
}
