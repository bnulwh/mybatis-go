package orm

import (
	"errors"
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/orm/dialector"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultMaxIdle    = dialector.DefaultMaxIdle
	DefaultMaxOpen    = dialector.DefaultMaxOpen
	DefaultMaxTimeout = dialector.DefaultMaxTimeout
)

type DatabaseSetting struct {
	Host     string
	Port     int64
	Username string
	Password string
	Name     string
	Schema   string
	Type     DatabaseType
}

type MyBatisSetting struct {
	DatabaseSetting
	MapperLocations    string
	TypeAliasPackage   string
	MaxRows            int64
	TablePrefix        string
	TablePrefixMap     map[string]string
	TablePrefixSetTTL  time.Duration
	TableStructureTTL  time.Duration
	SafeUpdate         bool
	IdType             string
	SnowflakeWorkerID  int64
	CacheEnabled       bool
	LocalCacheSize     int
	LocalCacheTTL      time.Duration
	PrettySQL          bool
	ExplainSlowSQL     bool
	SlowSQLThreshold   time.Duration
	PoolStatsInterval  time.Duration
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
	DSN        string
	cacheStore *sync.Map
}

func (c *Config) GetDSN() string                      { return c.DSN }
func (c *Config) GetConnPool() dialector.ConnPool      { return c.ConnPool }
func (c *Config) GetMaxTimeout() int                   { return c.MaxTimeout }
func (c *Config) GetMaxOpen() int                      { return c.MaxOpen }
func (c *Config) GetConnectParams() dialector.ConnectParams {
	return dialector.ConnectParams{
		Host:     c.Setting.Host,
		Port:     c.Setting.Port,
		Username: c.Setting.Username,
		Password: c.Setting.Password,
		DBName:   c.Setting.Name,
		Schema:   c.Setting.Schema,
		Type:     c.Setting.Type,
	}
}

func (c *Config) GenerateDSN() string {
	return dialector.GenerateDSN(c.GetConnectParams())
}

func (c *Config) DriverName() string {
	return dialector.GetDriverName(c.Setting.Type)
}

func (c *Config) EffectiveSchema() string {
	return dialector.EffectiveSchema(c.GetConnectParams())
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
	dt, err := dialector.ParseDatabaseType(dbType)
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
		PreparedStmt: true,
		SpringConfig: false,
		Dialector:    nil,
		ConnPool:     nil,
		cacheStore:   &sync.Map{},
	}
}

func parseDatabaseConfig(m map[string]string) *Config {
	tp, h, P, d, err := parseAddr(m)
	if err != nil {
		log.Errorf("parse postgres addr failed: %v", err)
		panic(err)
	}
	dt, err := dialector.ParseDatabaseType(tp)
	if err != nil {
		log.Errorf("parse datbase type failed.")
		panic("parse datbase type failed.")
	}
	var u, p string
	if dt != SqliteDb {
		var ok bool
		u, ok = m["spring.datasource.username"]
		if !ok {
			log.Errorf("get database username failed.")
			panic("get database username failed.")
		}
		p, ok = m["spring.datasource.password"]
		if !ok {
			log.Errorf("get database password failed.")
			panic("get database password failed.")
		}
	}
	ic := parseInt(m, "spring.datasource.max-idle", DefaultMaxIdle)
	oc := parseInt(m, "spring.datasource.max-open", DefaultMaxOpen)
	mt := parseInt(m, "spring.datasource.max-timeout", DefaultMaxTimeout)
	ps := parseBool(m, "spring.datasource.prepared-stmt", true)

	return &Config{
		Setting: MyBatisSetting{
			DatabaseSetting: DatabaseSetting{
				Host:     h,
				Port:     P,
				Username: u,
				Password: p,
				Name:     d,
				Schema:   parseSchema(m),
				Type:     dt,
			},
			TablePrefix:       parseTablePrefix(m),
			TablePrefixMap:    parseTablePrefixMap(m),
			TablePrefixSetTTL: parseTablePrefixSetTTL(m),
			TableStructureTTL: parseTableStructureTTL(m),
			SafeUpdate:        parseBool(m, "mybatis.configuration.safe-update", false),
			IdType:            strings.TrimSpace(m["mybatis.configuration.id-type"]),
			SnowflakeWorkerID: parseInt64(m, "mybatis.configuration.snowflake-worker-id", 1),
			CacheEnabled:      parseBool(m, "mybatis.configuration.cache-enabled", false),
			LocalCacheSize:    parseInt(m, "mybatis.configuration.local-cache-size", 1024),
			LocalCacheTTL:     parseDuration(m, "mybatis.configuration.local-cache-ttl", 3600),
			PrettySQL:         parseBool(m, "mybatis.configuration.pretty-sql", false),
			ExplainSlowSQL:    parseBool(m, "mybatis.configuration.explain-slow-sql", false),
			SlowSQLThreshold:  parseDuration(m, "mybatis.configuration.slow-sql-threshold", 3000),
			PoolStatsInterval: parseDuration(m, "mybatis.configuration.pool-stats-interval", 0),
		},
		MaxIdle:      int(ic),
		MaxOpen:      oc,
		MaxTimeout:   mt,
		PreparedStmt: ps,
		SpringConfig: true,
		Dialector:    nil,
		ConnPool:     nil,
		cacheStore:   &sync.Map{},
	}
}

func parseSchema(m map[string]string) string {
	if v, ok := m["spring.datasource.schema"]; ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	if raw, ok := m["spring.datasource.url"]; ok {
		if s := schemaFromURL(raw); s != "" {
			return s
		}
	}
	return ""
}

func schemaFromURL(rawURL string) string {
	q := rawURL
	if i := strings.IndexByte(q, '?'); i >= 0 {
		q = q[i+1:]
	} else {
		return ""
	}
	for _, pair := range strings.Split(q, "&") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(kv[0])) {
		case "currentschema", "search_path", "schema":
		default:
			continue
		}
		v, err := url.QueryUnescape(strings.TrimSpace(kv[1]))
		if err != nil {
			v = strings.TrimSpace(kv[1])
		}
		if v != "" {
			return v
		}
	}
	return ""
}

func parseTablePrefix(m map[string]string) string {
	for _, key := range []string{
		"mybatis.table-prefix",
		"mybatis-plus.global-config.db-config.table-prefix",
		"spring.datasource.table-prefix",
	} {
		if v, ok := m[key]; ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parseTablePrefixMap(m map[string]string) map[string]string {
	raw := strings.TrimSpace(m["mybatis.table-prefix-map"])
	if raw == "" {
		return nil
	}
	out := map[string]string{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		kv := strings.SplitN(item, ":", 2)
		oldp := strings.TrimSpace(kv[0])
		newp := ""
		if len(kv) == 2 {
			newp = strings.TrimSpace(kv[1])
		}
		if oldp == "" {
			continue
		}
		out[oldp] = newp
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseTablePrefixSetTTL(m map[string]string) time.Duration {
	raw := strings.TrimSpace(m["mybatis.table-prefix-set-ttl"])
	if raw == "" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Warnf("bad table-prefix-set-ttl %q: %v", raw, err)
		return 0
	}
	return d
}

func parseTableStructureTTL(m map[string]string) time.Duration {
	raw := strings.TrimSpace(m["mybatis.table-structure-ttl"])
	if raw == "" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Warnf("bad table-structure-ttl %q: %v", raw, err)
		return 0
	}
	return d
}

func parseBool(m map[string]string, key string, def bool) bool {
	v, ok := m[key]
	if !ok {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	}
	return def
}

func parseAddr(m map[string]string) (string, string, int64, string, error) {
	val, ok := m["spring.datasource.url"]
	if !ok {
		return "", "", 0, "", errors.New("not found key spring.datasource.url")
	}
	val = strings.TrimSpace(val)
	if strings.HasPrefix(strings.ToLower(val), "jdbc:sqlite:") {
		path := val[len("jdbc:sqlite:"):]
		path = strings.TrimPrefix(path, "file:")
		return "sqlite", "", 0, path, nil
	}
	re := regexp.MustCompile(`jdbc:([\w-]+)://(?:\[([^\]]+)\]|([\w.-]+)):(\d+)/([\w._-]+)`)
	matched := re.FindStringSubmatch(val)
	if len(matched) < 6 {
		return "", "", 0, "", errors.New("unsupport format of spring.datasource.url")
	}
	host := matched[2]
	if host == "" {
		host = matched[3]
	}
	i, _ := strconv.Atoi(matched[4])
	return matched[1], host, int64(i), matched[5], nil
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

func parseInt64(m map[string]string, key string, def int64) int64 {
	val, ok := m[key]
	if !ok {
		return def
	}
	nval, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
	if err != nil {
		return def
	}
	return nval
}

func parseDuration(m map[string]string, key string, defSeconds int64) time.Duration {
	val, ok := m[key]
	if !ok {
		return time.Duration(defSeconds) * time.Second
	}
	s := strings.TrimSpace(val)
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Duration(n) * time.Second
	}
	return time.Duration(defSeconds) * time.Second
}
