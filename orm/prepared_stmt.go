package orm

import (
	"context"
	"database/sql"
	"sync"
)

type Stmt struct {
	*sql.Stmt
	prepared   chan struct{}
	prepareErr error
}

type PreparedStmtDB struct {
	Stmts       map[string]*Stmt
	PreparedSQL []string
	Mux         *sync.RWMutex
	ConnPool
}

func (db *PreparedStmtDB) GetDBConn() (*sql.DB, error) {
	if dbConnector, ok := db.ConnPool.(GetDBConnector); ok && dbConnector != nil {
		return dbConnector.GetDBConn()
	}
	if sqldb, ok := db.ConnPool.(*sql.DB); ok {
		return sqldb, nil
	}
	return nil, ErrInvalidDB
}

func (db *PreparedStmtDB) Close() {
	db.Mux.Lock()
	defer db.Mux.Unlock()

	for _, query := range db.PreparedSQL {
		if stmt, ok := db.Stmts[query]; ok {
			delete(db.Stmts, query)
			go stmt.Close()
		}
	}
}

func (db *PreparedStmtDB) Reset() {
	db.Mux.Lock()
	defer db.Mux.Unlock()

	for query, stmt := range db.Stmts {
		delete(db.Stmts, query)
		go stmt.Close()
	}

	db.PreparedSQL = make([]string, 0, 100)
	db.Stmts = map[string]*Stmt{}
}

//func (db *PreparedStmtDB) PrepareContext(ctx context.Context,query string)(*sql.Stmt,error){
//
//}

func (db *PreparedStmtDB) prepare(ctx context.Context, conn ConnPool, query string) (Stmt, error) {
	db.Mux.RLock()
	if stmt, ok := db.Stmts[query]; ok {
		db.Mux.RUnlock()
		// wait for other goroutine prepared
		<-stmt.prepared

		if stmt.prepareErr != nil {
			return Stmt{}, stmt.prepareErr
		}
		return *stmt, nil
	}
	db.Mux.RUnlock()

	db.Mux.Lock()
	// double check
	if stmt, ok := db.Stmts[query]; ok {
		db.Mux.Unlock()
		// wait for other goroutine prepared
		<-stmt.prepared
		if stmt.prepareErr != nil {
			return Stmt{}, stmt.prepareErr
		}
		return *stmt, nil
	}
	// not found,cache preparing stmt first time
	cacheStmt := Stmt{prepared: make(chan struct{})}
	db.Stmts[query] = &cacheStmt
	db.Mux.Unlock()

	//prepared completed
	defer close(cacheStmt.prepared)

	stmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		cacheStmt.prepareErr = err
		db.Mux.Lock()
		delete(db.Stmts, query)
		db.Mux.Unlock()
		return Stmt{}, err
	}

	db.Mux.Lock()
	cacheStmt.Stmt = stmt
	db.PreparedSQL = append(db.PreparedSQL, query)
	db.Mux.Unlock()
	return cacheStmt, nil
}

func (db *PreparedStmtDB) ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	stmt, err := db.prepare(ctx, db.ConnPool, query)
	if err == nil {
		result, err = stmt.ExecContext(ctx, args...)
		if err != nil {
			db.Mux.Lock()
			defer db.Mux.Unlock()
			go stmt.Close()
			delete(db.Stmts, query)
		}
	}
	return result, err
}
func (db *PreparedStmtDB) QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	stmt, err := db.prepare(ctx, db.ConnPool, query)
	if err == nil {
		rows, err = stmt.QueryContext(ctx, args...)
		if err != nil {
			db.Mux.Lock()
			defer db.Mux.Unlock()
			go stmt.Close()
			delete(db.Stmts, query)
		}
	}
	return rows, err
}
func (db *PreparedStmtDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	stmt, err := db.prepare(ctx, db.ConnPool, query)
	if err == nil {
		return stmt.QueryRowContext(ctx, args...)
	}
	return &sql.Row{}
}

func (db *PreparedStmtDB) Stats() sql.DBStats {
	sqldb, err := db.GetDBConn()
	if err != nil {
		return sql.DBStats{}
	}
	return sqldb.Stats()
}
