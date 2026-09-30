package dialector

import "strings"

// ReservedNames 数据库保留名集合（内部统一小写，匹配大小写不敏感）：
// 表位置（FROM/JOIN/INTO/UPDATE/TABLE 等之后）出现这些名字时属于内置函数/内置对象，
// 表名前缀改写与表名提取均应跳过，避免把 json_populate_recordset(...)、dual 等
// 误当数据表加前缀。
//
// 零值与 nil 指针均安全：Has 恒返回 false（退化为无保留名行为）。
type ReservedNames struct {
	set map[string]struct{}
}

// NewReservedNames 按名称构建保留名集合；空白项忽略，其余统一转小写。
func NewReservedNames(names ...string) *ReservedNames {
	if len(names) == 0 {
		return &ReservedNames{}
	}
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		n = strings.ToLower(strings.TrimSpace(n))
		if n != "" {
			set[n] = struct{}{}
		}
	}
	return &ReservedNames{set: set}
}

// Has 判断 name 是否命中保留名（大小写不敏感；nil 接收者恒 false）。
func (r *ReservedNames) Has(name string) bool {
	if r == nil || len(r.set) == 0 {
		return false
	}
	_, ok := r.set[strings.ToLower(name)]
	return ok
}

// Merge 返回两个集合的并集；任一为 nil/空时直接返回另一个（零拷贝）。
func (r *ReservedNames) Merge(other *ReservedNames) *ReservedNames {
	if r == nil || len(r.set) == 0 {
		return other
	}
	if other == nil || len(other.set) == 0 {
		return r
	}
	merged := make(map[string]struct{}, len(r.set)+len(other.set))
	for k := range r.set {
		merged[k] = struct{}{}
	}
	for k := range other.set {
		merged[k] = struct{}{}
	}
	return &ReservedNames{set: merged}
}
