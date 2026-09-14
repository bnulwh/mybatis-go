package orm

import (
	"context"
	"fmt"
	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
	"reflect"
	"strings"
	"sync"
	"time"
)

const tableStructureCacheKey = "tableStructure"

type tableStructureCache struct {
	mu        sync.RWMutex
	tables    map[string]*types.TableStructure
	done      bool
	fetchedAt time.Time
	ttl       time.Duration
}

func (c *tableStructureCache) get() (map[string]*types.TableStructure, bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.getLocked()
}

func (c *tableStructureCache) getLocked() (map[string]*types.TableStructure, bool, bool) {
	stale := c.ttl > 0 && !c.fetchedAt.IsZero() && time.Since(c.fetchedAt) > c.ttl
	return c.tables, c.done, stale
}

func (c *tableStructureCache) markDone() {
	c.done = true
	c.fetchedAt = time.Now()
}

func (db *DB) tableStructureSetTTL() time.Duration {
	if db != nil && db.Config != nil && db.Setting.TableStructureTTL > 0 {
		return db.Setting.TableStructureTTL
	}
	return 0
}

func (db *DB) tableStructureSet() map[string]*types.TableStructure {
	if db == nil || db.cacheStore == nil {
		return nil
	}
	v, ok := db.cacheStore.Load(tableStructureCacheKey)
	if !ok {
		tc := &tableStructureCache{ttl: db.tableStructureSetTTL()}
		actual, _ := db.cacheStore.LoadOrStore(tableStructureCacheKey, tc)
		v = actual
	}
	cache := v.(*tableStructureCache)
	tables, done, stale := cache.get()
	if done && !stale {
		return tables
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	tables, done, stale = cache.getLocked()
	if done && !stale {
		return tables
	}
	fresh, err := db.fetchTableStructures()
	if err != nil {
		if done {
			log.Warnf("refresh table structures failed, keep stale set: %v", err)
			return tables
		}
		log.Warnf("fetch table structures from schema failed: %v", err)
		cache.tables = map[string]*types.TableStructure{}
		cache.markDone()
		return cache.tables
	}
	cache.tables = fresh
	cache.markDone()
	return cache.tables
}

func (db *DB) fetchTableStructures() (map[string]*types.TableStructure, error) {
	if db == nil || db.ConnPool == nil || db.Dialector == nil {
		return nil, nil
	}
	sqlStr := db.Dialector.TableListSQL()
	if sqlStr == "" {
		return map[string]*types.TableStructure{}, nil
	}
	ctx, cancel := withExecTimeout(context.Background())
	defer cancel()
	rows, err := db.ConnPool.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tableNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tableNames = append(tableNames, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := map[string]*types.TableStructure{}
	for _, table := range tableNames {
		ts, err := db.fetchSingleTableStructure(table)
		if err != nil {
			log.Warnf("fetch table %s structure failed: %v", table, err)
			continue
		}
		result[strings.ToLower(table)] = ts
	}
	return result, nil
}

func (db *DB) fetchSingleTableStructure(table string) (*types.TableStructure, error) {
	if db == nil || db.Dialector == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	sqlStr := db.Dialector.TableStructureSQL(table)
	if sqlStr == "" {
		return nil, fmt.Errorf("unsupport database type %v", db.Setting.Type)
	}
	ctx, cancel := withExecTimeout(context.Background())
	defer cancel()
	rows, err := db.ConnPool.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []map[string]interface{}
	colTypes, _ := rows.ColumnTypes()
	for rows.Next() {
		if len(res) == 0 {
			ptrs := prepareColumns(colTypes)
			converters := buildConvertersBasic(colTypes)
			err := rows.Scan(ptrs...)
			if err != nil {
				log.Warnf("scan error: %v", err)
				continue
			}
			mp := createMapWithConverters(ptrs, colTypes, converters)
			res = append(res, mp)
			continue
		}
		ptrs := prepareColumns(colTypes)
		converters := buildConvertersBasic(colTypes)
		if err := rows.Scan(ptrs...); err != nil {
			log.Warnf("scan error: %v", err)
			continue
		}
		mp := createMapWithConverters(ptrs, colTypes, converters)
		res = append(res, mp)
	}
	return types.NewTableStruct(table, res)
}

func (db *DB) invalidateTableStructures() {
	if db == nil || db.cacheStore == nil {
		return
	}
	db.cacheStore.Delete(tableStructureCacheKey)
}

type columnSchemaHint struct {
	goType reflect.Type
}

func (db *DB) lookupColumnSchema(tableName, colName string) *columnSchemaHint {
	if db == nil {
		return nil
	}
	tables := db.tableStructureSet()
	if tables == nil {
		return nil
	}
	ts, ok := tables[strings.ToLower(tableName)]
	if !ok {
		return nil
	}
	if cs, ok := ts.ColumnMap[colName]; ok {
		return &columnSchemaHint{goType: cs.Type}
	}
	lower := strings.ToLower(colName)
	for _, cs := range ts.Columns {
		if strings.EqualFold(cs.Name, colName) || strings.ToLower(cs.Name) == lower {
			return &columnSchemaHint{goType: cs.Type}
		}
	}
	return nil
}

func extractTableNamesFromSQL(query string) []string {
	tokens := tokenizeSQL(query)
	var names []string
	expectTable := false
	commaList := false
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		switch t.typ {
		case tkWord:
			lower := strings.ToLower(t.text)
			if expectTable {
				if lower == "if" || lower == "not" || lower == "exists" {
					continue
				}
				j := nextSig(tokens, i)
				if j >= 0 && tokens[j].typ == tkPunct && tokens[j].text == "." {
					k := nextSig(tokens, j)
					if k >= 0 && (tokens[k].typ == tkWord || tokens[k].typ == tkQuoted) {
						if isPrefixableSchema(identContent(t)) {
							names = append(names, identContent(tokens[k]))
						}
						i = k
					}
				} else {
					if !isSystemTable(identContent(t)) && !isClauseBreakWord(lower) {
						names = append(names, identContent(t))
					}
				}
				expectTable = false
				commaList = true
				continue
			}
			switch lower {
			case "from", "join", "into", "update", "table", "using":
				expectTable = true
				commaList = false
			default:
				if clauseBreakWords[lower] {
					commaList = false
				}
			}
		case tkQuoted:
			if expectTable {
				j := nextSig(tokens, i)
				if j >= 0 && tokens[j].typ == tkPunct && tokens[j].text == "." {
					k := nextSig(tokens, j)
					if k >= 0 && (tokens[k].typ == tkWord || tokens[k].typ == tkQuoted) {
						if isPrefixableSchema(identContent(t)) {
							names = append(names, identContent(tokens[k]))
						}
						i = k
					}
				} else {
					name := identContent(t)
					if !isSystemTable(name) {
						names = append(names, name)
					}
				}
				expectTable = false
				commaList = true
			}
		case tkPunct:
			switch t.text {
			case "(", ")":
				expectTable = false
				commaList = false
			case ",":
				if commaList {
					expectTable = true
				}
			}
		}
	}
	return names
}

func isClauseBreakWord(lower string) bool {
	return clauseBreakWords[lower]
}
