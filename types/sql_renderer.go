package types

type SqlRenderer interface {
	GenerateSQL(args ...interface{}) (string, error)
	//PrepareSQL(args ...interface{}) (string, []interface{}, error)
}
