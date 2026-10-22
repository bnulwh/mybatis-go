package orm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/orm/dialector"
)

// F2：|| 跨模式语义差异检测 —— 开关门控、family/name 判定、每 SQL 告警一次。

type f2CaptureLogger struct {
	warns []string
}

func (l *f2CaptureLogger) Debugf(format string, args ...interface{}) {}
func (l *f2CaptureLogger) Printf(format string, args ...interface{}) {}
func (l *f2CaptureLogger) Infof(format string, args ...interface{})  {}
func (l *f2CaptureLogger) Warnf(format string, args ...interface{}) {
	l.warns = append(l.warns, fmt.Sprintf(format, args...))
}
func (l *f2CaptureLogger) Errorf(format string, args ...interface{}) {}

func f2NewDB(t *testing.T, dbType dialector.DatabaseType) *DB {
	t.Helper()
	tp := "postgres"
	if dbType == dialector.MySqlDb {
		tp = "mysql"
	}
	cfg := newDatabaseConfig(tp, "localhost", 1, "u", "p", "db")
	d, err := dialector.NewForType(dbType, cfg)
	if err != nil {
		t.Errorf("NewForType failed: %v", err)
		return nil
	}
	return &DB{Config: &Config{Dialector: d}}
}

// Test_PortabilityCheck_MySQLFamily MySQL 族 + SQL 含 || → 告警且每条 SQL 一次。
func Test_PortabilityCheck_MySQLFamily(t *testing.T) {
	defer SetPortabilityCheck(PortabilityCheck())
	defer log.SetLogger(log.GetLogger())
	logger := &f2CaptureLogger{}
	log.SetLogger(logger)
	SetPortabilityCheck(true)

	db := f2NewDB(t, dialector.MySqlDb)
	if db == nil {
		return
	}
	db.checkPortability("SELECT a || b FROM t")
	db.checkPortability("SELECT a || b FROM t") // 重复：不再告警
	db.checkPortability("SELECT a || c FROM t")
	if len(logger.warns) != 2 {
		t.Errorf("want 2 warns (dedup per sql), got %d: %v", len(logger.warns), logger.warns)
	}
	if len(logger.warns) > 0 && !strings.Contains(logger.warns[0], "logical OR") {
		t.Errorf("mysql family warn should mention logical OR, got %q", logger.warns[0])
	}
}

// Test_PortabilityCheck_QuotedAndComment 引号/注释内的 || 不触发。
func Test_PortabilityCheck_QuotedAndComment(t *testing.T) {
	defer SetPortabilityCheck(PortabilityCheck())
	defer log.SetLogger(log.GetLogger())
	logger := &f2CaptureLogger{}
	log.SetLogger(logger)
	SetPortabilityCheck(true)

	db := f2NewDB(t, dialector.MySqlDb)
	if db == nil {
		return
	}
	db.checkPortability("SELECT 'a || b' FROM t -- c || d")
	db.checkPortability("SELECT /* x || y */ a FROM t")
	if len(logger.warns) != 0 {
		t.Errorf("quoted/comment || should not warn, got %v", logger.warns)
	}
	db.checkPortability("SELECT 'ok' || a FROM t") // 引号外的 || 触发
	if len(logger.warns) != 1 {
		t.Errorf("real || should warn once, got %d", len(logger.warns))
	}
}

// Test_PortabilityCheck_SwitchOff 开关关闭（默认）不告警。
func Test_PortabilityCheck_SwitchOff(t *testing.T) {
	defer SetPortabilityCheck(PortabilityCheck())
	defer log.SetLogger(log.GetLogger())
	logger := &f2CaptureLogger{}
	log.SetLogger(logger)
	SetPortabilityCheck(false)

	db := f2NewDB(t, dialector.MySqlDb)
	if db == nil {
		return
	}
	db.checkPortability("SELECT a || b FROM t")
	if len(logger.warns) != 0 {
		t.Errorf("switch off should not warn, got %v", logger.warns)
	}
}

// Test_PortabilityCheck_Kingbase 金仓（PG family + name=kingbase）含 || → 告警模式依赖。
func Test_PortabilityCheck_Kingbase(t *testing.T) {
	defer SetPortabilityCheck(PortabilityCheck())
	defer log.SetLogger(log.GetLogger())
	logger := &f2CaptureLogger{}
	log.SetLogger(logger)
	SetPortabilityCheck(true)

	cfg := newDatabaseConfig("kingbase", "localhost", 54321, "u", "p", "db")
	d, err := dialector.NewForType(dialector.KingbaseDb, cfg)
	if err != nil {
		t.Errorf("NewForType failed: %v", err)
		return
	}
	db := &DB{Config: &Config{Dialector: d}}
	db.checkPortability("SELECT a || b FROM kb_t")
	if len(logger.warns) != 1 {
		t.Errorf("kingbase with || should warn once, got %d", len(logger.warns))
		return
	}
	if !strings.Contains(logger.warns[0], "compatibility mode") {
		t.Errorf("kingbase warn should mention compatibility mode, got %q", logger.warns[0])
	}
}

// Test_PortabilityCheck_PostgresNoWarn 纯 PG（|| 即拼接，语义无歧义）不告警。
func Test_PortabilityCheck_PostgresNoWarn(t *testing.T) {
	defer SetPortabilityCheck(PortabilityCheck())
	defer log.SetLogger(log.GetLogger())
	logger := &f2CaptureLogger{}
	log.SetLogger(logger)
	SetPortabilityCheck(true)

	db := f2NewDB(t, dialector.PostgresDb)
	if db == nil {
		return
	}
	db.checkPortability("SELECT a || b FROM t")
	if len(logger.warns) != 0 {
		t.Errorf("postgres with || should not warn, got %v", logger.warns)
	}
}

// Test_PortabilityCheck_ConfigParse 配置 mybatis.configuration.portability-check 解析与套用。
func Test_PortabilityCheck_ConfigParse(t *testing.T) {
	defer SetPortabilityCheck(PortabilityCheck())

	cm := map[string]string{
		"spring.datasource.url":                   "jdbc:sqlite:test.db",
		"mybatis.configuration.portability-check": "true",
	}
	if err := InitializeFromSettings(cm); err != nil {
		t.Errorf("InitializeFromSettings failed: %v", err)
		return
	}
	defer Close()
	if !PortabilityCheck() {
		t.Error("portability-check=true should enable PortabilityCheck")
	}

	cm2 := map[string]string{
		"spring.datasource.url": "jdbc:sqlite:test.db",
	}
	if err := InitializeFromSettings(cm2); err != nil {
		t.Errorf("InitializeFromSettings failed: %v", err)
		return
	}
	defer Close()
	if PortabilityCheck() {
		t.Error("portability-check default should be off")
	}
}
