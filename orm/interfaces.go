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
)

type DatabaseFamily = dialector.DatabaseFamily

const (
	FamilyPostgres = dialector.FamilyPostgres
	FamilyMySQL    = dialector.FamilyMySQL
	FamilySQLite   = dialector.FamilySQLite
	FamilyOracle   = dialector.FamilyOracle
	FamilyInformix = dialector.FamilyInformix
)

type DatabaseType = dialector.DatabaseType

const (
	MySqlDb    = dialector.MySqlDb
	PostgresDb = dialector.PostgresDb
	KingbaseDb = dialector.KingbaseDb
	SqliteDb   = dialector.SqliteDb
)

type Dialector = dialector.Dialector

type GetDBConnector = dialector.GetDBConnector
