package orm

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// C2：goroutine 作用域标记 —— 嵌套计数、跨 goroutine 隔离、panic 路径平衡。
func Test_TxScopeGoroutineTracking(t *testing.T) {
	if inTxScope() {
		t.Error("inTxScope should be false before entering scope")
	}
	enterTxScope()
	if !inTxScope() {
		t.Error("inTxScope should be true after enterTxScope")
	}
	// 嵌套：深度累加
	enterTxScope()
	if !inTxScope() {
		t.Error("inTxScope should be true with nested scope")
	}
	leaveTxScope()
	if !inTxScope() {
		t.Error("inTxScope should remain true after leaving inner scope")
	}
	leaveTxScope()
	if inTxScope() {
		t.Error("inTxScope should be false after leaving all scopes")
	}

	// 跨 goroutine：作用域标记不串扰
	enterTxScope()
	defer leaveTxScope()
	done := make(chan bool)
	go func() {
		done <- inTxScope() // 其他 goroutine 不在本作用域内
	}()
	if otherIn := <-done; otherIn {
		t.Error("inTxScope should be false on a different goroutine")
	}
}

// C2：未加入事务的语句告警 —— 作用域外不告警、作用域内告警一次、带事务 ctx 不告警、
// 开关关闭时不告警。
func Test_WarnNonCtxInTxScope(t *testing.T) {
	ctx := context.Background()
	// 作用域外：不告警
	warnNonCtxInTxScope(ctx, "TxWarnTest", "outside")
	if _, ok := nonCtxTxWarned.Load("TxWarnTest.outside"); ok {
		t.Error("should not warn outside tx scope")
	}
	// 作用域内：告警一次
	enterTxScope()
	defer leaveTxScope()
	warnNonCtxInTxScope(ctx, "TxWarnTest", "inside")
	if _, ok := nonCtxTxWarned.Load("TxWarnTest.inside"); !ok {
		t.Error("should warn once inside tx scope")
	}
	// 带 WithTx 事务的 ctx：不告警
	txCtx := context.WithValue(ctx, txContextKey{}, &Transaction{})
	warnNonCtxInTxScope(txCtx, "TxWarnTest", "withTx")
	if _, ok := nonCtxTxWarned.Load("TxWarnTest.withTx"); ok {
		t.Error("should not warn for ctx carrying tx")
	}
	// 开关关闭：不告警
	SetTxScopeWarn(false)
	defer SetTxScopeWarn(true)
	warnNonCtxInTxScope(ctx, "TxWarnTest", "disabled")
	if _, ok := nonCtxTxWarned.Load("TxWarnTest.disabled"); ok {
		t.Error("should not warn when tx scope warn disabled")
	}
}

// C2 端到端：WithTx 内调用未声明 ctx 的 Mapper 方法 —— 该语句不加入事务
// （回滚后仍然留存），且告警恰好记录一次。
func Test_WithTxNonCtxMapperBypass(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "mapper")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Errorf("create mapper dir failed: %v", err)
		return
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="TxScopeTestMapper">
  <insert id="insertNoCtx" parameterType="TxScopeTestModel">
    insert into t_txscope (name) values (#{name})
  </insert>
</mapper>`
	if err := os.WriteFile(filepath.Join(xmlDir, "TxScopeTestMapper.xml"), []byte(xml), 0644); err != nil {
		t.Errorf("write mapper xml failed: %v", err)
		return
	}
	dbPath := filepath.Join(dir, "txscope.db")
	cm := map[string]string{
		"spring.datasource.url":    "jdbc:sqlite:" + dbPath,
		"mybatis.mapper-locations": xmlDir,
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("initialize sqlite failed: %v", err)
		return
	}
	defer Close()
	if _, err := Execute(`CREATE TABLE t_txscope (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Errorf("create table failed: %v", err)
		return
	}
	RegisterModel(new(TxScopeTestModel))
	if err := RegisterMapper(new(TxScopeTestMapper)); err != nil {
		t.Errorf("register mapper failed: %v", err)
		return
	}
	mp := NewMapper("TxScopeTestMapper").(TxScopeTestMapper)

	sentinel := errors.New("rollback for test")
	// WithTx 内调用无 ctx 方法 → 语句走独立连接，不随回滚撤销（记录 C2 缺陷行为）
	err := WithTx(context.Background(), func(ctx context.Context) error {
		if _, err := mp.InsertNoCtx(TxScopeTestModel{Name: "bypassed"}); err != nil {
			return err
		}
		return sentinel // 触发回滚
	})
	if err != sentinel {
		t.Errorf("WithTx should return fn's error, got %v", err)
		return
	}
	// 语句未加入事务：回滚后行仍在
	res, err := Query(`SELECT count(*) AS c FROM t_txscope`)
	if err != nil {
		t.Errorf("query failed: %v", err)
		return
	}
	if len(res) != 1 || res[0]["c"] != int64(1) {
		t.Errorf("non-ctx statement should persist after WithTx rollback (C2 pitfall), got %v", res)
	}
	// C2 告警恰好记录一次
	if _, ok := nonCtxTxWarned.Load("TxScopeTestMapper.insertNoCtx"); !ok {
		t.Error("expected C2 warn record for TxScopeTestMapper.insertNoCtx")
	}
}

// C2：配置解析 —— mybatis.configuration.tx-scope-warn（默认 true）。
func Test_ParseTxScopeWarn(t *testing.T) {
	cm := map[string]string{
		"spring.datasource.url":               "jdbc:sqlite:test.db",
		"mybatis.configuration.tx-scope-warn": "false",
	}
	cfg := NewConfigFromSettings(cm)
	if cfg.Setting.TxScopeWarn {
		t.Error("tx-scope-warn=false should parse to false")
	}
	cm2 := map[string]string{
		"spring.datasource.url": "jdbc:sqlite:test.db",
	}
	cfg2 := NewConfigFromSettings(cm2)
	if !cfg2.Setting.TxScopeWarn {
		t.Error("default tx-scope-warn should be true")
	}
}

// TxScopeTestMapper C2 测试用 Mapper：方法刻意不声明 context.Context 参数。
type TxScopeTestMapper struct {
	BaseMapper
	InsertNoCtx func(model TxScopeTestModel) (int64, error)
}

type TxScopeTestModel struct {
	Id   int64  `db:"id,pk"`
	Name string `db:"name"`
}
