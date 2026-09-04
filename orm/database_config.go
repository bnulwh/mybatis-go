package orm

import (
	"errors"
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type DatabaseType string

const (
	MySqlDb           DatabaseType = "mysql"
	PostgresDb        DatabaseType = "postgres"
	KingbaseDb        DatabaseType = "kingbase"
	SqliteDb          DatabaseType = "sqlite"
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
	Schema   string // 模式：PG/Kingbase 的 schema（默认 public）；MySQL 下 schema 即数据库名，为空时等价于 Name
	Type     DatabaseType
}

type MyBatisSetting struct {
	DatabaseSetting
	MapperLocations  string
	TypeAliasPackage string
	MaxRows          int64
	TablePrefix      string            // 数据表名前缀（如 test_），SQL 执行时自动拼接到表名前，空串不启用
	TablePrefixMap   map[string]string // 前缀映射：oldprefix→newprefix（空值=移除），配置键 mybatis.table-prefix-map（2.1）
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
	DSN        string // 自定义 DSN：非空时优先于 GenerateDSN()（注入自定义连接串）
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
		PreparedStmt: true,
		SpringConfig: false,
		Dialector:    nil,
		ConnPool:     nil,
		cacheStore:   &sync.Map{},
	}
}

func (ds *DatabaseSetting) generateConn() string {
	switch ds.Type {
	case PostgresDb, KingbaseDb:
		// KingbaseES 兼容 PostgreSQL 连接串格式；
		// search_path 为 lib/pq 支持的运行时参数，连接建立后自动 SET，避免依赖库默认搜索路径
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			ds.Host, ds.Port, ds.Username, ds.Password, ds.Name)
		if ds.Schema != "" {
			dsn += " search_path=" + ds.Schema
		}
		return dsn
	case MySqlDb:
		// parseTime=true 使 DATETIME/TIMESTAMP 列直接扫描为 time.Time
		// （否则 go-sql-driver 返回 []byte 原始字节，无法 Scan 到时间字段）；
		// loc=Local 按本机时区解析，与 SQLite 的 _loc=auto 语义一致。
		if strings.Contains(ds.Name, "?") {
			return fmt.Sprintf("%s:%s@tcp(%s)/%s&parseTime=true&loc=Local",
				ds.Username, ds.Password, joinHostPort(ds.Host, ds.Port), ds.Name)
		}
		return fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&loc=Local",
			ds.Username, ds.Password, joinHostPort(ds.Host, ds.Port), ds.Name)
	case SqliteDb:
		// Name 为 sqlite 文件路径；_loc=auto 使 DATETIME 列返回 time.Time
		if strings.Contains(ds.Name, "?") {
			return ds.Name
		}
		return fmt.Sprintf("%s?_loc=auto", ds.Name)
	}
	return ""
}

// joinHostPort 拼接 host:port；IPv6 地址（含冒号）需加方括号，驱动才能正确解析
func joinHostPort(host string, port int64) string {
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("%s:%d", host, port)
}

func (ds *DatabaseSetting) getDriver() string {
	switch ds.Type {
	case MySqlDb:
		return "mysql"
	case PostgresDb:
		return "postgres"
	case KingbaseDb:
		return "kingbase"
	case SqliteDb:
		return "sqlite"
	}
	return ""
}

// effectiveSchema 返回表结构查询使用的 schema：
//   - MySQL 下 schema 即数据库名，显式配置时优先，否则回退 dbName；
//   - PG/Kingbase 未配置时默认 public（与历史硬编码行为一致）；
//   - SQLite 无 schema 概念，返回空串。
func (ds *DatabaseSetting) effectiveSchema(dbName string) string {
	if ds.Schema != "" {
		return ds.Schema
	}
	if ds.Type == MySqlDb {
		return dbName
	}
	if ds.Type == SqliteDb {
		return ""
	}
	return "public"
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
	dt, err := parseDatabaseType(tp)
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
			TablePrefix:    parseTablePrefix(m),
			TablePrefixMap: parseTablePrefixMap(m),
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

// parseSchema 解析 schema 配置，按优先级：
// spring.datasource.schema 键 > JDBC URL query 参数（currentSchema / search_path / schema）。
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

// schemaFromURL 从 JDBC URL 的 query 参数提取 schema，兼容 PostgreSQL JDBC 的
// currentSchema 与通用的 search_path / schema 两种写法（search_path 可含逗号多值）。
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

// parseTablePrefix 解析数据表名前缀，按优先级：
// mybatis.table-prefix > mybatis-plus.global-config.db-config.table-prefix > spring.datasource.table-prefix。
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

// parseTablePrefixMap 解析前缀映射：mybatis.table-prefix-map = threedb_:,app_:subsp_
// 逗号分隔多个 old:new；new 为空表示移除（剥离该前缀）；无冒号条目等价 old:（移除该前缀）；
// 旧前缀为空（如输入 ":x" / ","）则忽略。
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

// parseBool 解析布尔配置项；key 不存在或值非法时返回默认值。
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

func parseDatabaseType(tps string) (DatabaseType, error) {
	switch strings.ToLower(tps) {
	case "mysql":
		return MySqlDb, nil
	case "postgres", "postgresql":
		return PostgresDb, nil
	case "kingbase", "kingbase8", "kingbase7", "kingbase6", "kingbase5":
		return KingbaseDb, nil
	case "sqlite", "sqlite3":
		return SqliteDb, nil
	default:
		return "", fmt.Errorf("not support database type %v", tps)
	}
}
func parseAddr(m map[string]string) (string, string, int64, string, error) {
	val, ok := m["spring.datasource.url"]
	if !ok {
		return "", "", 0, "", errors.New("not found key spring.datasource.url")
	}
	val = strings.TrimSpace(val)
	if strings.HasPrefix(strings.ToLower(val), "jdbc:sqlite:") {
		// jdbc:sqlite:path 或 jdbc:sqlite:file:path
		path := val[len("jdbc:sqlite:"):]
		path = strings.TrimPrefix(path, "file:")
		return "sqlite", "", 0, path, nil
	}
	re := regexp.MustCompile(`jdbc:([\w]+)://(?:\[([^\]]+)\]|([\w.-]+)):(\d+)/([\w._-]+)`)
	matched := re.FindStringSubmatch(val)
	if len(matched) < 6 {
		return "", "", 0, "", errors.New("unsupport format of spring.datasource.url")
	}
	// matched[2] 为 IPv6（方括号内），matched[3] 为 IPv4/域名
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
