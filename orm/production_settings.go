package orm

import (
	"sort"
	"strings"
	"sync"

	"github.com/bnulwh/mybatis-go/log"
)

// G2：生产级开关推荐预设 —— 防护/观测项默认全关，提供一键推荐配置与启动期提示。

// RecommendedProductionSettings 返回生产环境推荐配置（safe-update / 连接池监控 /
// 慢 SQL EXPLAIN + 阈值）。使用方可合并进配置 map 后交给 InitializeFromSettings：
//
//	for k, v := range orm.RecommendedProductionSettings() {
//	    if _, ok := cm[k]; !ok { // 不覆盖已有显式配置
//	        cm[k] = v
//	    }
//	}
//	orm.InitializeFromSettings(cm)
func RecommendedProductionSettings() map[string]string {
	return map[string]string{
		"mybatis.configuration.safe-update":         "true", // UPDATE/DELETE 无 WHERE 阻止执行
		"mybatis.configuration.pool-stats-interval": "60",   // 连接池状态每 60 秒输出一次
		"mybatis.configuration.explain-slow-sql":    "true", // 慢 SELECT 自动 EXPLAIN
		"mybatis.configuration.slow-sql-threshold":  "3000", // 慢 SQL 阈值（毫秒）
	}
}

var productionHintOnce sync.Once

// UnrecommendedSettingItems 返回当前配置中未开启的生产推荐项描述（已排序）；
// 全部开启时返回空切片。供启动期提示与测试断言使用。
func UnrecommendedSettingItems(s MyBatisSetting) []string {
	var missing []string
	if !s.SafeUpdate {
		missing = append(missing, "mybatis.configuration.safe-update=true（UPDATE/DELETE 无 WHERE 阻止执行）")
	}
	if s.PoolStatsInterval <= 0 {
		missing = append(missing, "mybatis.configuration.pool-stats-interval=60（连接池状态定期日志）")
	}
	if !s.ExplainSlowSQL {
		missing = append(missing, "mybatis.configuration.explain-slow-sql=true（慢 SQL 自动 EXPLAIN）")
	}
	sort.Strings(missing)
	return missing
}

// warnUnrecommendedSettings 启动时打印当前未开启的推荐项清单（每进程一次，Warn 级）。
// 仅提示不强制；开发/测试环境可忽略，或按 RecommendedProductionSettings 补齐后消除。
func warnUnrecommendedSettings(s MyBatisSetting) {
	productionHintOnce.Do(func() {
		missing := UnrecommendedSettingItems(s)
		if len(missing) == 0 {
			return
		}
		log.Warnf("production: 以下推荐配置项未开启：\n  - %s\n可调用 orm.RecommendedProductionSettings() 获取推荐配置",
			strings.Join(missing, "\n  - "))
	})
}
