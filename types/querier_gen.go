package types

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/bnulwh/mybatis-go/log"
	"github.com/bnulwh/mybatis-go/types/sqlfragment"
	"github.com/bnulwh/mybatis-go/utils"
)

// S1 Querier 代码生成：扫描存量 XML Mapper，对每个「纯静态」select
// （无动态标签、无 ${} 注入，判定见 sqlfragment.ExtractStaticSQL）生成类型安全的
// Querier 接口 + 实现（普通 Go 函数，非反射代理）。生成的实现经
// (db *orm.DB) QueryToContext 执行：? 占位参数绑定、方言占位符转换 / 表前缀 /
// ctx 事务路由全部由 orm 执行层承担（可行性报告 §3）。
//
// 生成策略：
//   - 结果类型：resultMap → 模型 struct（本包内生成，db tag 带列名与 pk 标记）；
//     resultType 标量 / map → []标量 / []map[string]interface{}（与既有
//     generateDefine 的 select 返回形态一致，恒为切片）；
//   - 参数类型：parameterType 为标量（Long/String/...）→ 单参数，绑定全部占位符；
//     无 parameterType → 按占位符名逐个生成具名参数（jdbcType 可得则定型，
//     否则 interface{}）；parameterType 为 struct/map/slice → 跳过（无法静态推断）；
//   - 跳过项记入 QuerierGenStats.Skipped（含原因），不影响其余语句。

// QuerierSkip 记录一个被跳过的语句及原因。
type QuerierSkip struct {
	Id     string
	Reason string
}

// QuerierGenStats 单个 Mapper 的 Querier 生成统计。
type QuerierGenStats struct {
	Mapper    string
	File      string
	Models    int
	Functions int
	Skipped   []QuerierSkip
}

// QuerierGenShared 跨 Mapper 共享的生成状态（模型短名去重，避免同包重复定义）。
type QuerierGenShared struct {
	Models map[string]bool
}

// GenerateQuerierFiles 为全部 Mapper 生成 Querier 文件（S1）。
// 每个 Mapper 输出 <基名>_querier.go 到 dir；无静态 select 的 Mapper 不产出文件。
// 返回逐 Mapper 生成统计（含跳过原因），供调用方（cmd/sqlc）打印报告。
func (in *SqlMappers) GenerateQuerierFiles(dir, pkg string) ([]*QuerierGenStats, error) {
	if err := utils.MakeDirAll(dir); err != nil {
		return nil, err
	}
	shared := &QuerierGenShared{Models: map[string]bool{}}
	var stats []*QuerierGenStats
	for i := range in.Mappers {
		st, err := in.Mappers[i].GenerateQuerierFile(dir, pkg, shared)
		if err != nil {
			log.Warnf("mapper %v generate querier file failed: %v", in.Mappers[i].Namespace, err)
			continue
		}
		if st != nil {
			stats = append(stats, st)
		}
	}
	return stats, nil
}

// GenerateQuerierFile 为单个 Mapper 生成 Querier 文件。
// shared 为 nil 时独立生成（测试 / 单文件场景）；跨 Mapper 批量生成时传入共享实例
// 以对同名模型去重。无静态 select 时返回统计（Functions=0、File=""）且不写盘。
func (in *SqlMapper) GenerateQuerierFile(dir, pkg string, shared *QuerierGenShared) (*QuerierGenStats, error) {
	if pkg == "" {
		pkg = "querier"
	}
	if shared == nil {
		shared = &QuerierGenShared{Models: map[string]bool{}}
	}
	base := querierBaseName(in.Namespace)
	if base == "" {
		return nil, fmt.Errorf("cannot derive querier name from namespace %v", in.Namespace)
	}
	stats := &QuerierGenStats{Mapper: in.Namespace}
	ifName := UpperFirst(base) + "Querier"
	implName := lowerFirst(base) + "Querier"

	var models bytes.Buffer
	var iface bytes.Buffer
	var impl bytes.Buffer
	for _, rm := range in.Maps {
		if rm == nil || GetShortName(rm.TypeName) == "" {
			continue
		}
		sname := GetShortName(rm.TypeName)
		if shared.Models[sname] {
			continue
		}
		shared.Models[sname] = true
		models.WriteString(renderQuerierModel(rm, in.NamedMaps, shared))
		stats.Models++
	}

	for _, fn := range in.Functions {
		if fn.Type != SelectFunction {
			continue
		}
		ifaceLine, implBlock, skip := in.generateQuerierMethod(fn, implName)
		if skip != nil {
			stats.Skipped = append(stats.Skipped, *skip)
			continue
		}
		iface.WriteString(ifaceLine)
		impl.WriteString(implBlock)
		stats.Functions++
	}
	if stats.Functions == 0 {
		return stats, nil
	}
	stats.File = filepath.Join(dir, fmt.Sprintf("%s_querier.go", buildKey(base)))
	content := assembleQuerierFile(pkg, in, ifName, implName, models.String(), iface.String(), impl.String())
	if err := os.WriteFile(stats.File, content, 0640); err != nil {
		return stats, err
	}
	gofmtFile(stats.File)
	return stats, nil
}

// generateQuerierMethod 为单个静态 select 生成接口方法行 + 实现方法块；
// 不可静态化时返回 skip 原因。
func (in *SqlMapper) generateQuerierMethod(fn *SqlFunction, implName string) (string, string, *QuerierSkip) {
	if fn.Id == MPSelectPageID {
		return "", "", &QuerierSkip{Id: fn.Id, Reason: "分页参数 *orm.PageParam 无法静态绑定"}
	}
	sqlstr, sparams, ok := sqlfragment.ExtractStaticSQL(fn.Items)
	if !ok {
		return "", "", &QuerierSkip{Id: fn.Id, Reason: "含动态标签或 ${} 注入，无法提取静态 SQL"}
	}
	var funcParams []string
	var callArgs []string
	if fn.Param.Need && !fn.Param.AutoDerive {
		// 显式 parameterType：仅标量可静态定型，单参数绑定全部占位符（与 Base 渲染路径语义一致）
		if fn.Param.Type != BaseSqlParam {
			return "", "", &QuerierSkip{Id: fn.Id, Reason: fmt.Sprintf("parameterType=%v 无法静态推断参数类型", fn.Param.TypeName)}
		}
		slots := uniqueStaticSlotNames(sparams)
		if len(slots) == 0 {
			return "", "", &QuerierSkip{Id: fn.Id, Reason: "声明 parameterType 但语句无 #{} 占位符"}
		}
		pname := sanitizeQuerierArgName(slots[0])
		funcParams = append(funcParams, pname+" "+toGolangType(fn.Param.TypeName))
		for range sparams {
			callArgs = append(callArgs, pname)
		}
	} else {
		// 无 parameterType（或仅由占位符推导，1.1）：按占位符名逐个生成具名参数，
		// jdbcType 可得则定型，否则 interface{}；实参按占位符出现顺序展开（与 1.2 按位绑定一致）
		for _, sp := range sparams {
			pname := sanitizeQuerierArgName(sp.Name)
			if !containsString(funcParams, pname) {
				funcParams = append(funcParams, pname+" "+staticParamGoType(sp))
			}
			callArgs = append(callArgs, pname)
		}
	}
	retType := querierResultType(fn)
	method := UpperFirst(fn.Id)
	sig := fmt.Sprintf("%s(ctx context.Context%s) (%s, error)", method, querierParamSig(funcParams), retType)
	var impl bytes.Buffer
	fmt.Fprintf(&impl, "func (q *%s) %s {\n", implName, sig)
	fmt.Fprintf(&impl, "\tvar rows %s\n", retType)
	fmt.Fprintf(&impl, "\terr := q.db.QueryToContext(ctx, &rows, %s%s)\n",
		strconv.Quote(sqlstr), querierCallArgsSuffix(callArgs))
	impl.WriteString("\treturn rows, err\n}\n\n")
	return "\t" + sig + "\n", impl.String(), nil
}

// renderQuerierModel 从 resultMap 渲染模型 struct（db tag 带列名，主键带 pk 标记；
// association/collection 字段 db:"-"，由关联查询填充，不参与列扫描）。
func renderQuerierModel(rm *ResultMap, namedMaps map[string]*ResultMap, shared *QuerierGenShared) string {
	var buf bytes.Buffer
	sname := GetShortName(rm.TypeName)
	fmt.Fprintf(&buf, "// %s 源自 resultMap %v（type=%v）。\n", sname, rm.Id, rm.TypeName)
	fmt.Fprintf(&buf, "type %s struct {\n", sname)
	for _, item := range rm.Results {
		if item.Property == "" {
			continue
		}
		typ := item.golangType(namedMaps)
		tag := "-"
		switch item.Kind {
		case ResultItemKindId, ResultItemKindResult:
			tag = item.Column
			if tag == "" {
				tag = camelToSnake(item.Property)
			}
			if item.PrimaryKey {
				tag += ",pk"
			}
		case ResultItemKindAssociation, ResultItemKindCollection:
			nested := strings.TrimPrefix(strings.TrimPrefix(typ, "[]"), "*")
			if !shared.Models[nested] {
				log.Warnf("querier model %v 字段 %v 引用类型 %v 未在本批次生成（跨 Mapper 引用或未定义），产物可能需手工补齐",
					sname, item.Property, nested)
			}
		}
		fmt.Fprintf(&buf, "\t%s %s `db:\"%v\"`\n", UpperFirst(item.Property), typ, tag)
	}
	buf.WriteString("}\n\n")
	return buf.String()
}

// assembleQuerierFile 组装生成文件：头部注释 + imports + 模型 + 接口 + 构造器 + 实现。
func assembleQuerierFile(pkg string, mp *SqlMapper, ifName, implName, models, iface, impl string) []byte {
	var buf bytes.Buffer
	buf.WriteString("// Code generated by mybatis-go cmd/sqlc from XML mapper static selects. DO NOT EDIT.\n")
	fmt.Fprintf(&buf, "// 源: %v (namespace: %v)\n\n", filepath.Base(mp.Filename), mp.Namespace)
	fmt.Fprintf(&buf, "package %v\n\n", pkg)
	buf.WriteString("import (\n\t\"context\"\n")
	if strings.Contains(models, "time.Time") {
		buf.WriteString("\t\"time\"\n")
	}
	buf.WriteString("\n\t\"github.com/bnulwh/mybatis-go/orm\"\n)\n\n")
	buf.WriteString(models)
	fmt.Fprintf(&buf, "// %v 静态 select 生成的类型安全 Querier。\n", ifName)
	fmt.Fprintf(&buf, "type %v interface {\n%v}\n\n", ifName, iface)
	fmt.Fprintf(&buf, "type %v struct {\n\tdb *orm.DB\n}\n\n", implName)
	fmt.Fprintf(&buf, "// New%v 构造 Querier；db 为 nil 时使用 orm 当前活跃数据源。\n", ifName)
	fmt.Fprintf(&buf, "func New%v(db *orm.DB) %v {\n", ifName, ifName)
	buf.WriteString("\tif db == nil {\n\t\tdb = orm.GetActiveDataSource()\n\t}\n")
	fmt.Fprintf(&buf, "\treturn &%v{db: db}\n}\n\n", implName)
	buf.WriteString(impl)
	return buf.Bytes()
}

// querierResultType 推导 select 返回类型（恒为切片，与既有 generateDefine 语义一致）。
func querierResultType(fn *SqlFunction) string {
	if fn.Result.ResultM != nil {
		return "[]" + GetShortName(fn.Result.ResultM.TypeName)
	}
	if fn.Result.ResultT != nil && fn.Result.ResultT.Kind() == reflect.Map {
		return "[]map[string]interface{}"
	}
	if fn.Result.ResultT == nil {
		return "[]int64"
	}
	return "[]" + toGolangType(fn.Result.ResultT.String())
}

// querierParamSig 拼接方法参数签名后缀（非空时以 ", " 开头）。
func querierParamSig(params []string) string {
	if len(params) == 0 {
		return ""
	}
	return ", " + strings.Join(params, ", ")
}

// querierCallArgsSuffix 拼接调用实参后缀（非空时以 ", " 开头）。
func querierCallArgsSuffix(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return ", " + strings.Join(args, ", ")
}

// staticParamGoType 由占位符 jdbcType 推导参数 Go 类型；缺失时 interface{}。
func staticParamGoType(sp sqlfragment.StaticParam) string {
	if sp.JdbcType == "" {
		return "interface{}"
	}
	return ParseJdbcTypeFrom(sp.JdbcType).String()
}

// uniqueStaticSlotNames 按首现序去重占位符名。
func uniqueStaticSlotNames(params []sqlfragment.StaticParam) []string {
	seen := map[string]bool{}
	var out []string
	for _, sp := range params {
		if seen[sp.Name] {
			continue
		}
		seen[sp.Name] = true
		out = append(out, sp.Name)
	}
	return out
}

// containsString 切片包含判定（参数去重用）。
func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// sanitizeQuerierArgName 占位符名 → 合法 Go 形参名：
// 非 [A-Za-z0-9_] 字符替换为 _、数字开头加 p 前缀、关键字加 _ 后缀。
func sanitizeQuerierArgName(name string) string {
	var buf strings.Builder
	for i, r := range name {
		switch {
		case r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z':
			buf.WriteRune(r)
		case r >= '0' && r <= '9':
			if i == 0 {
				buf.WriteByte('p')
			}
			buf.WriteRune(r)
		default:
			buf.WriteByte('_')
		}
	}
	s := strings.Trim(buf.String(), "_")
	if s == "" {
		return "arg"
	}
	if s[0] >= '0' && s[0] <= '9' {
		s = "p" + s
	}
	switch s {	case "break", "case", "chan", "const", "continue", "default", "defer", "else",
		"fallthrough", "for", "func", "go", "goto", "if", "import", "interface",
		"map", "package", "range", "return", "select", "struct", "switch", "type", "var":
		return s + "_"
	}
	return s
}

// querierBaseName 从 namespace 短名推导 Querier 基名（去 Mapper 后缀）。
func querierBaseName(namespace string) string {
	sname := GetShortName(namespace)
	if strings.HasSuffix(strings.ToLower(sname), "mapper") {
		trimmed := sname[:len(sname)-len("mapper")]
		if trimmed != "" {
			return trimmed
		}
	}
	return sname
}

// lowerFirst 首字母小写。
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[0:1]) + s[1:]
}

// ParseQuerierFileSyntax 用 go/parser 校验生成文件语法（测试用）。
func ParseQuerierFileSyntax(filename string) error {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, filename, nil, parser.SkipObjectResolution)
	return err
}
