package dialector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type PlaceholderStyle int

const (
	PlaceholderQuestion PlaceholderStyle = iota
	PlaceholderDollar
	PlaceholderColon
)

type DatabaseFamily string

const (
	FamilyPostgres DatabaseFamily = "postgres"
	FamilyMySQL    DatabaseFamily = "mysql"
	FamilySQLite   DatabaseFamily = "sqlite"
	FamilyOracle   DatabaseFamily = "oracle"
	FamilyInformix DatabaseFamily = "informix"
)

type DatabaseType string

const (
	MySqlDb      DatabaseType = "mysql"
	PostgresDb   DatabaseType = "postgres"
	KingbaseDb   DatabaseType = "kingbase"
	SqliteDb     DatabaseType = "sqlite"
	TiDBDb       DatabaseType = "tidb"
	TDSQLDb      DatabaseType = "tdsql"
	PolarDBMyDb  DatabaseType = "polardb"
)

func (dt DatabaseType) Family() DatabaseFamily {
	switch dt {
	case PostgresDb, KingbaseDb:
		return FamilyPostgres
	case MySqlDb, TiDBDb, TDSQLDb, PolarDBMyDb:
		return FamilyMySQL
	case SqliteDb:
		return FamilySQLite
	default:
		return DatabaseFamily("")
	}
}

func ParseDatabaseType(tps string) (DatabaseType, error) {
	switch strings.ToLower(tps) {
	case "mysql":
		return MySqlDb, nil
	case "postgres", "postgresql":
		return PostgresDb, nil
	case "kingbase", "kingbase8", "kingbase7", "kingbase6", "kingbase5":
		return KingbaseDb, nil
	case "sqlite", "sqlite3":
		return SqliteDb, nil
	case "tidb":
		return TiDBDb, nil
	case "tdsql":
		return TDSQLDb, nil
	case "polardb", "polardb-mysql", "polardb_mysql":
		return PolarDBMyDb, nil
	default:
		return "", fmt.Errorf("not support database type %v", tps)
	}
}

func GetDriverName(dbType DatabaseType) string {
	switch dbType {
	case PostgresDb:
		return "postgres"
	case KingbaseDb:
		return "kingbase"
	case MySqlDb:
		return "mysql"
	case SqliteDb:
		return "sqlite"
	case TiDBDb:
		return "tidb"
	case TDSQLDb:
		return "tdsql"
	case PolarDBMyDb:
		return "polardb"
	default:
		return string(dbType)
	}
}

type ConnectParams struct {
	Host     string
	Port     int64
	Username string
	Password string
	DBName   string
	Schema   string
	Type     DatabaseType
}

func EffectiveSchema(params ConnectParams) string {
	if params.Schema != "" {
		return params.Schema
	}
	switch params.Type.Family() {
	case FamilyMySQL:
		return params.DBName
	case FamilySQLite:
		return ""
	default:
		return "public"
	}
}

func GenerateDSN(params ConnectParams) string {
	switch params.Type.Family() {
	case FamilyPostgres:
		return generatePostgresDSN(params)
	case FamilyMySQL:
		return generateMySQLDSN(params)
	case FamilySQLite:
		return generateSQLiteDSN(params)
	}
	return ""
}

func generatePostgresDSN(p ConnectParams) string {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		p.Host, p.Port, p.Username, p.Password, p.DBName)
	if p.Schema != "" {
		dsn += " search_path=" + p.Schema
	}
	return dsn
}

func generateMySQLDSN(p ConnectParams) string {
	if strings.Contains(p.DBName, "?") {
		return fmt.Sprintf("%s:%s@tcp(%s)/%s&parseTime=true&loc=Local",
			p.Username, p.Password, joinHostPort(p.Host, p.Port), p.DBName)
	}
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&loc=Local",
		p.Username, p.Password, joinHostPort(p.Host, p.Port), p.DBName)
}

func generateSQLiteDSN(p ConnectParams) string {
	if strings.Contains(p.DBName, "?") {
		return p.DBName
	}
	return fmt.Sprintf("%s?_loc=auto", p.DBName)
}

func joinHostPort(host string, port int64) string {
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("%s:%d", host, port)
}

var ErrUnsupportedDatabase = errors.New("unsupported database type")

type ConnPool interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	Stats() sql.DBStats
}

type Dialector interface {
	Name() string
	Initialize() (ConnPool, error)
	FormatPrepareSQL(src string) string
	PlaceholderStyle() PlaceholderStyle
	NeedsReturning() bool
	Family() DatabaseFamily
	ApplyPagination(sql string, limit, offset int) string
	SystemTablePrefixes() []string
	TableStructureSQL(table string) string
	TableListSQL() string
	DefaultMaxIdle() int
}

type ConfigProvider interface {
	GetDSN() string
	GetConnPool() ConnPool
	GetMaxTimeout() int
	GetMaxOpen() int
	GetConnectParams() ConnectParams
}

func NewForType(dbType DatabaseType, cfg ConfigProvider) (Dialector, error) {
	switch dbType {
	case PostgresDb:
		return NewPostgresDialector(cfg), nil
	case MySqlDb:
		return NewMySqlDialector(cfg), nil
	case SqliteDb:
		return NewSqliteDialector(cfg), nil
	case KingbaseDb:
		return NewKingbaseDialector(cfg), nil
	case TiDBDb:
		return NewTiDBDialector(cfg), nil
	case TDSQLDb:
		return NewTDSQLDialector(cfg), nil
	case PolarDBMyDb:
		return NewPolarDBDialector(cfg), nil
	default:
		return nil, ErrUnsupportedDatabase
	}
}

type GetDBConnector interface {
	GetDBConn() (*sql.DB, error)
}
