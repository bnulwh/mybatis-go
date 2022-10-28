package orm

import (
	"errors"
	"fmt"
	log "github.com/bnulwh/logrus"
	"regexp"
	"strconv"
	"strings"
)

type DatabaseType string

const (
	MySqlDb           DatabaseType = "mysql"
	PostgresDb        DatabaseType = "postgres"
	DefaultMaxIdle                 = 100
	DefaultMaxOpen                 = 100
	DefaultMaxTimeout              = 300
)

type DatabaseConfig struct {
	Host       string
	Port       int64
	Username   string
	Password   string
	DbName     string
	DbType     DatabaseType
	MaxIdle    int
	MaxOpen    int
	MaxTimeout int
}

type MyBatisConfig struct {
	DbConfig         *DatabaseConfig
	MapperLocations  string
	TypeAliasPackage string
	MaxRows          int64
}

func NewConfig(filename string) *MyBatisConfig {
	cm := LoadSettings(filename)
	return NewConfigFromSettings(cm)
}
func NewConfigFromSettings(cm map[string]string) *MyBatisConfig {
	dbc := parseDatabaseConfig(cm)
	ml := cm["mybatis.mapper-locations"]
	return &MyBatisConfig{
		DbConfig:        dbc,
		MapperLocations: ml,
	}
}
func newDatabaseConfig(dbType, host string, port int, user, pwd, dbName string) *DatabaseConfig {
	dt, err := parseDatabaseType(dbType)
	if err != nil {
		log.Errorf("parse datbase type failed.")
		panic("parse datbase type failed.")
	}
	return &DatabaseConfig{
		Host:       host,
		Port:       int64(port),
		Username:   user,
		Password:   pwd,
		DbName:     dbName,
		DbType:     dt,
		MaxOpen:    DefaultMaxOpen,
		MaxIdle:    DefaultMaxIdle,
		MaxTimeout: DefaultMaxTimeout,
	}
}

func (in *DatabaseConfig) generateConn() string {
	switch in.DbType {
	case PostgresDb:
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			in.Host, in.Port, in.Username, in.Password, in.DbName)
	case MySqlDb:
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			in.Username, in.Password, in.Host, in.Port, in.DbName)
	}
	return ""
}

func (in *DatabaseConfig) getDriver() string {
	switch in.DbType {
	case MySqlDb:
		return "mysql"
	case PostgresDb:
		return "postgres"
	}
	return ""
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
	ic := parseInt(m, "spring.datasource.max-idle", DefaultMaxIdle)
	oc := parseInt(m, "spring.datasource.max-open", DefaultMaxOpen)
	mt := parseInt(m, "spring.datasource.max-timeout", DefaultMaxTimeout)

	dt, err := parseDatabaseType(tp)
	if err != nil {
		log.Errorf("parse datbase type failed.")
		panic("parse datbase type failed.")
	}
	return &DatabaseConfig{
		Host:       h,
		Port:       P,
		Username:   u,
		Password:   p,
		DbName:     d,
		DbType:     dt,
		MaxIdle:    int(ic),
		MaxOpen:    oc,
		MaxTimeout: mt,
	}
}
func parseDatabaseType(tps string) (DatabaseType, error) {
	switch strings.ToLower(tps) {
	case "mysql":
		return MySqlDb, nil
	case "postgres", "postgresql":
		return PostgresDb, nil
	default:
		return "", fmt.Errorf("not support database type %v", tps)
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
		return "", "", 0, "", errors.New("unsupport format of spring.datasource.url")
	}
	i, _ := strconv.Atoi(matched[3])
	return matched[1], matched[2], int64(i), matched[4], nil
}

func parseInt(m map[string]string, key string, def int64) int {
	val, ok := m[key]
	if !ok {
		val = fmt.Sprint(def)
	}
	nval, err := strconv.ParseInt(val, 10, 0)
	if err != nil {
		nval = int64(def)
	}
	return int(nval)
}
