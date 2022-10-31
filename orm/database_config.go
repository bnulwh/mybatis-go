package orm

import (
	"errors"
	"fmt"
	log "github.com/bnulwh/logrus"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type DatabaseType string

const (
	MySqlDb           DatabaseType = "mysql"
	PostgresDb        DatabaseType = "postgres"
	DefaultMaxIdle                 = 100
	DefaultMaxOpen                 = 100
	DefaultMaxTimeout              = 300
)

type DatabaseSetting struct {
	Host     string
	Port     int64
	Username string
	Password string
	Name     string
	Type     DatabaseType
}

type MyBatisSetting struct {
	DatabaseSetting
	MapperLocations  string
	TypeAliasPackage string
	MaxRows          int64
}

type Config struct {
	Setting      MyBatisSetting
	MaxIdle      int
	MaxOpen      int
	MaxTimeout   int
	PreparedStmt bool
	SpringConfig bool
	Dialector
	ConnPool   ConnPool
	cacheStore *sync.Map
}

func NewConfig(filename string) *Config {
	cm := LoadSettings(filename)
	return NewConfigFromSettings(cm)
}
func NewConfigFromSettings(cm map[string]string) *Config {
	cfg := parseDatabaseConfig(cm)
	ml := cm["mybatis.mapper-locations"]
	cfg.Setting.MapperLocations = ml
	return cfg
}
func newDatabaseConfig(dbType, host string, port int, user, pwd, dbName string) *Config {
	dt, err := parseDatabaseType(dbType)
	if err != nil {
		log.Errorf("parse datbase type failed.")
		panic("parse datbase type failed.")
	}
	return &Config{
		Setting: MyBatisSetting{
			DatabaseSetting: DatabaseSetting{
				Host:     host,
				Port:     int64(port),
				Username: user,
				Password: pwd,
				Name:     dbName,
				Type:     dt,
			},
		},
		MaxOpen:      DefaultMaxOpen,
		MaxIdle:      DefaultMaxIdle,
		MaxTimeout:   DefaultMaxTimeout,
		SpringConfig: false,
		Dialector:    nil,
		ConnPool:     nil,
		cacheStore:   &sync.Map{},
	}
}

func (ds *DatabaseSetting) generateConn() string {
	switch ds.Type {
	case PostgresDb:
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			ds.Host, ds.Port, ds.Username, ds.Password, ds.Name)
	case MySqlDb:
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			ds.Username, ds.Password, ds.Host, ds.Port, ds.Name)
	}
	return ""
}

func (ds *DatabaseSetting) getDriver() string {
	switch ds.Type {
	case MySqlDb:
		return "mysql"
	case PostgresDb:
		return "postgres"
	}
	return ""
}

func (in *Config) GenerateDSN() string {
	return in.Setting.generateConn()
}

func (in *Config) DriverName() string {
	return in.Setting.getDriver()
}

func parseDatabaseConfig(m map[string]string) *Config {
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
	return &Config{
		Setting: MyBatisSetting{
			DatabaseSetting: DatabaseSetting{
				Host:     h,
				Port:     P,
				Username: u,
				Password: p,
				Name:     d,
				Type:     dt,
			},
		},
		MaxIdle:      int(ic),
		MaxOpen:      oc,
		MaxTimeout:   mt,
		SpringConfig: true,
		Dialector:    nil,
		ConnPool:     nil,
		cacheStore:   &sync.Map{},
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
