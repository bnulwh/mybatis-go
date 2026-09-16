package orm

import (
	"github.com/bnulwh/mybatis-go/orm/dialector"
)

type ConnPool = dialector.ConnPool

type PlaceholderStyle = dialector.PlaceholderStyle

const (
	PlaceholderQuestion = dialector.PlaceholderQuestion
	PlaceholderDollar   = dialector.PlaceholderDollar
	PlaceholderColon    = dialector.PlaceholderColon
	PlaceholderAtP      = dialector.PlaceholderAtP
)

type DatabaseFamily = dialector.DatabaseFamily

const (
	FamilyPostgres = dialector.FamilyPostgres
	FamilyMySQL    = dialector.FamilyMySQL
	FamilySQLite   = dialector.FamilySQLite
	FamilyOracle   = dialector.FamilyOracle
	FamilyInformix = dialector.FamilyInformix
	FamilyMSSQL    = dialector.FamilyMSSQL
	FamilyDB2      = dialector.FamilyDB2
)

type DatabaseType = dialector.DatabaseType

const (
	MySqlDb     = dialector.MySqlDb
	PostgresDb  = dialector.PostgresDb
	KingbaseDb  = dialector.KingbaseDb
	SqliteDb    = dialector.SqliteDb
	TiDBDb      = dialector.TiDBDb
	TDSQLDb     = dialector.TDSQLDb
	PolarDBMyDb = dialector.PolarDBMyDb
	OpenGaussDb = dialector.OpenGaussDb
	GaussDBDb   = dialector.GaussDBDb
	HighGoDb    = dialector.HighGoDb
	VastbaseDb  = dialector.VastbaseDb
	OceanBaseDb      = dialector.OceanBaseDb
	OceanBaseOracleDb = dialector.OceanBaseOracleDb
	DamengDb         = dialector.DamengDb
	GBase8sDb        = dialector.GBase8sDb
	MssqlDb          = dialector.MssqlDb
	OracleDb         = dialector.OracleDb
	Db2Db            = dialector.Db2Db
)

type Dialector = dialector.Dialector

type GetDBConnector = dialector.GetDBConnector
