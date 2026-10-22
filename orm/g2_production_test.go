package orm

import (
	"strings"
	"testing"
	"time"
)

// G2：生产级开关推荐预设。

// Test_RecommendedProductionSettings 推荐配置项齐全且值有效。
func Test_RecommendedProductionSettings(t *testing.T) {
	rec := RecommendedProductionSettings()
	want := map[string]string{
		"mybatis.configuration.safe-update":         "true",
		"mybatis.configuration.pool-stats-interval": "60",
		"mybatis.configuration.explain-slow-sql":    "true",
		"mybatis.configuration.slow-sql-threshold":  "3000",
	}
	if len(rec) != len(want) {
		t.Errorf("want %d recommended items, got %d: %v", len(want), len(rec), rec)
	}
	for k, v := range want {
		got, ok := rec[k]
		if !ok {
			t.Errorf("recommended settings missing %s", k)
			continue
		}
		if got != v {
			t.Errorf("recommended %s = %q, want %q", k, got, v)
		}
	}
}

// Test_UnrecommendedSettingItems 推荐项缺省检测：默认全关 → 3 项缺失；全开 → 空。
func Test_UnrecommendedSettingItems(t *testing.T) {
	def := MyBatisSetting{}
	missing := UnrecommendedSettingItems(def)
	if len(missing) != 3 {
		t.Errorf("default setting should miss 3 recommended items, got %d: %v", len(missing), missing)
	}
	for _, m := range missing {
		if !strings.Contains(m, "mybatis.configuration.") {
			t.Errorf("missing item should reference config key, got %q", m)
		}
	}

	full := MyBatisSetting{
		SafeUpdate:        true,
		PoolStatsInterval: 60 * time.Second,
		ExplainSlowSQL:    true,
	}
	if got := UnrecommendedSettingItems(full); len(got) != 0 {
		t.Errorf("all recommended enabled should have no missing items, got %v", got)
	}

	partial := MyBatisSetting{SafeUpdate: true}
	if got := UnrecommendedSettingItems(partial); len(got) != 2 {
		t.Errorf("partial setting should miss 2 items, got %d: %v", len(got), got)
	}
}

// Test_RecommendedMergeEndToEnd 推荐配置合并进 settings map 后 InitializeFromSettings，
// 缺失项清零（端到端：解析 → 套用 → 复检）。
func Test_RecommendedMergeEndToEnd(t *testing.T) {
	cm := map[string]string{
		"spring.datasource.url": "jdbc:sqlite:test.db",
	}
	for k, v := range RecommendedProductionSettings() {
		if _, ok := cm[k]; !ok {
			cm[k] = v
		}
	}
	cfg := NewConfigFromSettings(cm)
	if got := UnrecommendedSettingItems(cfg.Setting); len(got) != 0 {
		t.Errorf("after merging recommended settings, missing items should be empty, got %v", got)
	}
	if !cfg.Setting.SafeUpdate || !cfg.Setting.ExplainSlowSQL || cfg.Setting.PoolStatsInterval <= 0 {
		t.Errorf("recommended settings not applied: %+v", cfg.Setting)
	}
}
