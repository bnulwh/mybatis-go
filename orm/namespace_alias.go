package orm

import (
	"fmt"
	"strings"

	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types"
)

// RegisterNamespaceAlias 为已加载的 Mapper 注册 namespace 别名（E3）：
// alias 的各查找键与 name 解析到同一 *types.SqlMapper，用于 namespace 改名 /
// 加载前缀后保持旧引用可用（旧名结构体 RegisterMapper、NewMapper 均不受影响）。
//
// 典型用法（共享 XML 与本地 Mapper 同名冲突，namespace 改名为 QualityPlatformDisplayMapper 后）：
//
//	err := orm.RegisterNamespaceAlias("PlatformDisplayMapper", "QualityPlatformDisplayMapper")
//
// 注册后自动重绑全部已注册 Mapper 结构体（别名生效前因找不到 XML 而未绑定的旧名结构体随即补绑）。
// alias 键覆盖既有不同 Mapper 时仅告警不报错（与加载期「后者覆盖前者」语义一致）。
func RegisterNamespaceAlias(alias, name string) error {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return fmt.Errorf("namespace alias must not be empty")
	}
	gCache.mappers.mu.Lock()
	if gCache.sqls == nil {
		gCache.mappers.mu.Unlock()
		return fmt.Errorf("mapper xml definitions not loaded yet, cannot register alias %q", alias)
	}
	mp := lookupSqlMapper(gCache.sqls.NamedMappers, name)
	if mp == nil {
		gCache.mappers.mu.Unlock()
		return fmt.Errorf("mapper %q not found in loaded xml definitions", name)
	}
	for _, key := range namespaceAliasKeys(alias) {
		if old, ok := gCache.sqls.NamedMappers[key]; ok && old != mp {
			log.Warnf("namespace alias %q overrides existing mapper key (was namespace %s)", key, old.Namespace)
		}
		gCache.sqls.NamedMappers[key] = mp
	}
	gCache.mappers.mu.Unlock()
	// 旧名结构体可能在别名注册前已因找不到 XML 而未绑定，这里统一补绑
	return gCache.bindSqls()
}

// NamespaceConflicts 返回加载期检测到的「同 namespace 多文件」记录（后者覆盖前者；E3）。
// 未加载 XML 或无冲突时返回 nil。
func NamespaceConflicts() []types.NamespaceConflict {
	if gCache.sqls == nil {
		return nil
	}
	return gCache.sqls.NamespaceConflicts
}

// namespaceAliasKeys 生成名字的全部查找键（与 NamedMappers 注册规则一致：原名/小写/短名/小写短名）。
func namespaceAliasKeys(name string) []string {
	sn := types.GetShortName(name)
	return []string{name, strings.ToLower(name), sn, strings.ToLower(sn)}
}

// lookupSqlMapper 按 name 的各查找键在 NamedMappers 中解析 *types.SqlMapper。
func lookupSqlMapper(nmp map[string]*types.SqlMapper, name string) *types.SqlMapper {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	for _, key := range namespaceAliasKeys(name) {
		if mp, ok := nmp[key]; ok {
			return mp
		}
	}
	return nil
}
