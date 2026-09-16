package dialector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type PlaceholderStyle int

const (
	PlaceholderQuestion PlaceholderStyle = iota
	PlaceholderDollar
	PlaceholderColon
	PlaceholderAtP
)

type DatabaseFamily string

const (
	FamilyPostgres DatabaseFamily = "postgres"
	FamilyMySQL    DatabaseFamily = "mysql"
	FamilySQLite   DatabaseFamily = "sqlite"
	FamilyOracle   DatabaseFamily = "oracle"
	FamilyInformix DatabaseFamily = "informix"
	FamilyMSSQL    DatabaseFamily = "mssql"
	FamilyDB2        DatabaseFamily = "db2"
	FamilyClickHouse DatabaseFamily = "clickhouse"
)

type DatabaseType string

const (
	MySqlDb          DatabaseType = "mysql"
	PostgresDb       DatabaseType = "postgres"
	KingbaseDb       DatabaseType = "kingbase"
	SqliteDb         DatabaseType = "sqlite"
	TiDBDb           DatabaseType = "tidb"
	TDSQLDb          DatabaseType = "tdsql"
	PolarDBMyDb      DatabaseType = "polardb"
	OpenGaussDb      DatabaseType = "opengauss"
	GaussDBDb        DatabaseType = "gaussdb"
	HighGoDb         DatabaseType = "highgo"
	VastbaseDb       DatabaseType = "vastbase"
	OceanBaseDb      DatabaseType = "oceanbase"
	OceanBaseOracleDb DatabaseType = "oceanbase-oracle"
	DamengDb         DatabaseType = "dameng"
	GBase8sDb        DatabaseType = "gbase8s"
	MssqlDb          DatabaseType = "mssql"
	OracleDb         DatabaseType = "oracle"
	Db2Db           DatabaseType = "db2"
	ClickHouseDb    DatabaseType = "clickhouse"
)

func (dt DatabaseType) Family() DatabaseFamily {
	switch dt {
	case PostgresDb, KingbaseDb, OpenGaussDb, GaussDBDb, HighGoDb, VastbaseDb:
		return FamilyPostgres
	case MySqlDb, TiDBDb, TDSQLDb, PolarDBMyDb, OceanBaseDb:
		return FamilyMySQL
	case SqliteDb:
		return FamilySQLite
	case OceanBaseOracleDb, DamengDb, OracleDb:
		return FamilyOracle
	case GBase8sDb:
		return FamilyInformix
	case MssqlDb:
		return FamilyMSSQL
	case Db2Db:
		return FamilyDB2
	case ClickHouseDb:
		return FamilyClickHouse
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
	case "opengauss", "opengauss-server":
		return OpenGaussDb, nil
	case "gaussdb", "gaussdb-pg", "gaussdb_pg":
		return GaussDBDb, nil
	case "highgo", "highgodb":
		return HighGoDb, nil
	case "vastbase", "vastbasedb":
		return VastbaseDb, nil
	case "oceanbase", "oceanbase-mysql":
		return OceanBaseDb, nil
	case "oceanbase-oracle", "oboracle":
		return OceanBaseOracleDb, nil
	case "dameng", "dm", "dm8":
		return DamengDb, nil
	case "gbase8s", "gbase", "gbase-8s":
		return GBase8sDb, nil
	case "mssql", "sqlserver", "mssql-server":
		return MssqlDb, nil
	case "oracle", "oracle-db", "oracledb":
		return OracleDb, nil
	case "db2", "ibmdb2", "db2-luw":
		return Db2Db, nil
	case "clickhouse", "click-house":
		return ClickHouseDb, nil
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
	case OpenGaussDb:
		return "opengauss"
	case GaussDBDb:
		return "gaussdb"
	case HighGoDb:
		return "highgo"
	case VastbaseDb:
		return "vastbase"
	case OceanBaseDb:
		return "oceanbase"
	case OceanBaseOracleDb:
		return "oceanbase-oracle"
	case DamengDb:
		return "dameng"
	case GBase8sDb:
		return "gbase8s"
	case MssqlDb:
		return "sqlserver"
	case OracleDb:
		return "oracle"
	case Db2Db:
		return "go_ibm_db"
	case ClickHouseDb:
		return "clickhouse"
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
	case FamilyOracle:
		return strings.ToUpper(params.Username)
	case FamilyInformix:
		return strings.ToUpper(params.Username)
	case FamilyMSSQL:
		return "dbo"
	case FamilyDB2:
		return strings.ToUpper(params.Username)
	case FamilyClickHouse:
		return params.DBName
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
	case FamilyOracle:
		return generateOracleDSN(params)
	case FamilyInformix:
		return generateInformixDSN(params)
	case FamilyMSSQL:
		return generateMssqlDSN(params)
	case FamilyDB2:
		return generateDb2DSN(params)
	case FamilyClickHouse:
		return generateClickHouseDSN(params)
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

func generateOracleDSN(p ConnectParams) string {
	return fmt.Sprintf("%s/%s@%s:%d/%s", p.Username, p.Password, p.Host, p.Port, p.DBName)
}

func generateInformixDSN(p ConnectParams) string {
	return fmt.Sprintf("%s:%s@%s:%d/%s", p.Username, p.Password, p.Host, p.Port, p.DBName)
}

func generateMssqlDSN(p ConnectParams) string {
	return fmt.Sprintf("sqlserver://%s:%s@%s?database=%s",
		url.QueryEscape(p.Username),
		url.QueryEscape(p.Password),
		joinHostPort(p.Host, p.Port),
		url.QueryEscape(p.DBName))
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
	case OpenGaussDb:
		return NewOpenGaussDialector(cfg), nil
	case GaussDBDb:
		return NewGaussDBDialector(cfg), nil
	case HighGoDb:
		return NewHighGoDialector(cfg), nil
	case VastbaseDb:
		return NewVastbaseDialector(cfg), nil
	case OceanBaseDb:
		return NewOceanBaseDialector(cfg), nil
	case OceanBaseOracleDb:
		return NewOceanBaseOracleDialector(cfg), nil
	case DamengDb:
		return NewDamengDialector(cfg), nil
	case GBase8sDb:
		return NewGBase8sDialector(cfg), nil
	case MssqlDb:
		return NewMssqlDialector(cfg), nil
	case OracleDb:
		return NewOracleDialector(cfg), nil
	case Db2Db:
		return NewDb2Dialector(cfg), nil
	case ClickHouseDb:
		return NewClickHouseDialector(cfg), nil
	default:
		return nil, ErrUnsupportedDatabase
	}
}

type GetDBConnector interface {
	GetDBConn() (*sql.DB, error)
}
