package orm

import (
	"context"
	"testing"
	"time"
)

// P4-2：QueryStream 基础流式遍历 —— 行 map 内容与 Count 正确
func Test_QueryStreamBasic(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	if _, err := Execute(`CREATE TABLE t_stream (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	for i := 1; i <= 5; i++ {
		if _, err := Execute(`INSERT INTO t_stream (name) VALUES (?)`, "row_"+string(rune('0'+i))); err != nil {
			t.Errorf("insert failed: %v", err)
			return
		}
	}

	st, err := QueryStream(context.Background(), `SELECT id, name FROM t_stream ORDER BY id`)
	if err != nil {
		t.Errorf("QueryStream failed: %v", err)
		return
	}
	defer st.Close()
	var rows []map[string]interface{}
	for st.Next() {
		rows = append(rows, st.Row())
	}
	if err := st.Err(); err != nil {
		t.Errorf("stream Err failed: %v", err)
	}
	if len(rows) != 5 {
		t.Errorf("stream row count failed, got %d want 5", len(rows))
		return
	}
	if st.Count() != 5 {
		t.Errorf("stream Count failed, got %d want 5", st.Count())
	}
	for i, r := range rows {
		if r["name"] != "row_"+string(rune('0'+i+1)) {
			t.Errorf("row %d name failed, got %v", i, r["name"])
		}
	}
}

// P4-2：QueryStream 与 Query 结果一致性（同一 SQL 逐行与全量结果相同）
func Test_QueryStreamParity(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	if _, err := Execute(`CREATE TABLE t_stream (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	for i := 0; i < 8; i++ {
		Execute(`INSERT INTO t_stream (name) VALUES (?)`, "p"+string(rune('a'+i)))
	}

	all, err := Query(`SELECT id, name FROM t_stream ORDER BY id`)
	if err != nil {
		t.Errorf("Query failed: %v", err)
		return
	}
	st, err := QueryStream(context.Background(), `SELECT id, name FROM t_stream ORDER BY id`)
	if err != nil {
		t.Errorf("QueryStream failed: %v", err)
		return
	}
	defer st.Close()
	var streamRows []map[string]interface{}
	for st.Next() {
		streamRows = append(streamRows, st.Row())
	}
	if len(streamRows) != len(all) {
		t.Errorf("stream/query count mismatch: %d vs %d", len(streamRows), len(all))
		return
	}
	for i := range all {
		if streamRows[i]["id"] != all[i]["id"] || streamRows[i]["name"] != all[i]["name"] {
			t.Errorf("row %d mismatch: stream=%v query=%v", i, streamRows[i], all[i])
		}
	}
}

// P4-2：Scan 填充结构体（下划线列名 → 驼峰字段）与 map
func Test_QueryStreamScan(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	if _, err := Execute(`CREATE TABLE t_stream (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, create_time DATETIME)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	tm := time.Date(2026, 8, 13, 17, 0, 0, 0, time.UTC)
	if _, err := Execute(`INSERT INTO t_stream (name, create_time) VALUES (?, ?)`, "scan_me", tm); err != nil {
		t.Errorf("insert failed: %v", err)
		return
	}

	st, err := QueryStream(context.Background(), `SELECT id, name, create_time FROM t_stream`)
	if err != nil {
		t.Errorf("QueryStream failed: %v", err)
		return
	}
	defer st.Close()

	// Scan 到 map
	var mp map[string]interface{}
	if !st.Next() {
		t.Errorf("stream has no row: %v", st.Err())
		return
	}
	if err := st.Scan(&mp); err != nil {
		t.Errorf("scan to map failed: %v", err)
		return
	}
	if mp["name"] != "scan_me" {
		t.Errorf("map scan name failed, got %v", mp["name"])
	}

	// Scan 到结构体（字段名 Id/Name/CreateTime 与列 id/name/create_time 匹配）
	st2, err := QueryStream(context.Background(), `SELECT id, name, create_time FROM t_stream`)
	if err != nil {
		t.Errorf("QueryStream failed: %v", err)
		return
	}
	defer st2.Close()
	if !st2.Next() {
		t.Errorf("stream has no row: %v", st2.Err())
		return
	}
	var m SqliteTestModel
	if err := st2.Scan(&m); err != nil {
		t.Errorf("scan to struct failed: %v", err)
		return
	}
	if m.Name != "scan_me" || m.Id == 0 || !m.CreateTime.Equal(tm) {
		t.Errorf("struct scan failed, got %+v", m)
	}

	// Scan 错误分支：新流未调用 Next 就 Scan
	st3, err := QueryStream(context.Background(), `SELECT id, name, create_time FROM t_stream`)
	if err != nil {
		t.Errorf("QueryStream failed: %v", err)
		return
	}
	defer st3.Close()
	var m2 SqliteTestModel
	if err := st3.Scan(&m2); err == nil {
		t.Error("expected error scanning without Next on fresh stream")
	}
}

// P4-2：提前 Close 释放连接（未读完即退出），后续查询不受影响；Close 幂等
func Test_QueryStreamEarlyClose(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	if _, err := Execute(`CREATE TABLE t_stream (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	for i := 0; i < 10; i++ {
		Execute(`INSERT INTO t_stream (name) VALUES (?)`, "x")
	}

	st, err := QueryStream(context.Background(), `SELECT id, name FROM t_stream`)
	if err != nil {
		t.Errorf("QueryStream failed: %v", err)
		return
	}
	// 只读第一行就关闭
	if !st.Next() {
		t.Errorf("stream first Next failed: %v", st.Err())
		return
	}
	if err := st.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
		return
	}
	if st.Next() {
		t.Error("expected Next false after Close")
	}
	// Close 幂等
	if err := st.Close(); err != nil {
		t.Errorf("second Close should be no-op, got %v", err)
	}

	// 连接已释放：后续查询正常
	res, err := Query(`SELECT COUNT(*) AS c FROM t_stream`)
	if err != nil {
		t.Errorf("query after close failed: %v", err)
		return
	}
	if len(res) != 1 || res[0]["c"] != int64(10) {
		t.Errorf("query after close mismatch: %v", res)
	}
}

// P4-2：流式读取遵循全局行数上限（P4-3），上限在打开时快照
func Test_QueryStreamRowLimit(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()
	old := DefaultRowLimit()
	defer SetDefaultRowLimit(old)

	if _, err := Execute(`CREATE TABLE t_stream (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	for i := 0; i < 25; i++ {
		Execute(`INSERT INTO t_stream (name) VALUES (?)`, "x")
	}

	// 上限 2 → 只流式读到 2 行
	SetDefaultRowLimit(2)
	st, err := QueryStream(context.Background(), `SELECT id, name FROM t_stream`)
	if err != nil {
		t.Errorf("QueryStream failed: %v", err)
		return
	}
	defer st.Close()
	for st.Next() {
	}
	if st.Count() != 2 {
		t.Errorf("stream count with limit 2 failed, got %d", st.Count())
	}

	// 上限 0 → 不返回任何行
	SetDefaultRowLimit(0)
	st0, err := QueryStream(context.Background(), `SELECT id, name FROM t_stream`)
	if err != nil {
		t.Errorf("QueryStream failed: %v", err)
		return
	}
	defer st0.Close()
	if st0.Next() {
		t.Error("expected no rows with limit 0")
	}

	// 负数 → 全部 25 行
	SetDefaultRowLimit(-1)
	stAll, err := QueryStream(context.Background(), `SELECT id, name FROM t_stream`)
	if err != nil {
		t.Errorf("QueryStream failed: %v", err)
		return
	}
	defer stAll.Close()
	for stAll.Next() {
	}
	if stAll.Count() != 25 {
		t.Errorf("stream count with negative limit failed, got %d", stAll.Count())
	}
}

// P4-2：SQL 错误在 QueryStream 打开时返回；正常遍历结束后 Err() 为 nil
func Test_QueryStreamErr(t *testing.T) {
	dir := initSqliteTest(t)
	if dir == "" {
		return
	}
	defer Close()

	// 不存在的表 → 立即报错
	if _, err := QueryStream(context.Background(), `SELECT * FROM no_such_table`); err == nil {
		t.Error("expected error for missing table")
	}

	if _, err := Execute(`CREATE TABLE t_stream (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	Execute(`INSERT INTO t_stream (name) VALUES (?)`, "a")

	// 空结果集 → 正常结束，Err() 为 nil
	st, err := QueryStream(context.Background(), `SELECT id, name FROM t_stream WHERE name = 'none'`)
	if err != nil {
		t.Errorf("QueryStream failed: %v", err)
		return
	}
	defer st.Close()
	if st.Next() {
		t.Error("expected no rows for empty result")
	}
	if st.Err() != nil {
		t.Errorf("expected nil Err for empty result, got %v", st.Err())
	}
	if st.Count() != 0 {
		t.Errorf("expected count 0, got %d", st.Count())
	}
}
