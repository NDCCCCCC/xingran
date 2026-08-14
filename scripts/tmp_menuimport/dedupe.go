package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// dedupe.go — 去重引擎（R0-R5 规则，机械可解释）
//
// R0 软删过滤：deleted_at != nil 的行默认不进保留集；例外——若某软删 M/C
//     是保留行的祖先且其分组内无存活等价目录（即重定向后 parent 不在保留集），
//     则复活该行（生成 SQL 时 deleted_at 写 NULL）并在报告标注。
// R1 顶级目录合并：parent_id==nil 的 M 按 TrimSuffix(menu_name,"test") 分组。
//     每组选规范 id，优先级：① deleted_at IS NULL ② 存活子树节点数最多
//     ③ path 非空 ④ created_at 最早 ⑤ id 字典序（稳定 tiebreak）。其余折叠进 idMap。
// R2 重定向：idMap 求传递闭包（折叠 id 的父若也被折叠继续追），所有保留行的
//     parent_id 经闭包映射；闭包有环即违反不变量④。
// R3 同级内容去重（fixpoint）：按 (映射后parent_id, menu_name, menu_type,
//     COALESCE(perms,'')) 分组保留 1 行（优先级同 R1），折叠其余进 idMap；
//     重复 R2-R3 直到不动点。
// R4 sys_role_menu：menu_id 经 idMap 映射并去重 (role_id,menu_id)，仅用于报告
//     统计，不生成导入 SQL（旧 role id 在 dev 不存在）。
// R5 sys_role 不导入（dev 复用现有 admin/user）。

const (
	dedupSQLOut = "xingran_menus_dedup.sql"
	reportOut   = ".planning/quick/260814-ehg-dedupe-and-import-legacy-menu-data-into-/dedupe-report.md"
)

// numericCols sys_menu 19 列中非引号字面量列（version/order_num/visible/status）。
var numericCols = map[int]bool{6: true, 9: true, 13: true, 14: true}

type foldRecord struct {
	Key       string   // 分组键
	Canonical string   // 规范 id
	Reason    string   // 保留理由
	Folded    []string // 被折叠 id 列表
}

type engine struct {
	all         []*MenuRow
	byID        map[string]*MenuRow
	keep        map[string]bool
	idMap       map[string]string // folded -> canonical（单次映射，resolve 求闭包）
	resurrected map[string]bool
	r1Folds     []foldRecord
	r3Folds     []foldRecord
}

func newEngine(menus []*MenuRow) *engine {
	e := &engine{
		all:         menus,
		byID:        map[string]*MenuRow{},
		keep:        map[string]bool{},
		idMap:       map[string]string{},
		resurrected: map[string]bool{},
	}
	for _, m := range menus {
		e.byID[m.ID()] = m
		// R0：软删行默认不进保留集
		if m.DeletedAt() == nil {
			e.keep[m.ID()] = true
		}
	}
	return e
}

// resolve idMap 传递闭包（带环检测，不变量④）。
func (e *engine) resolve(id string) string {
	seen := map[string]bool{}
	cur := id
	for {
		next, ok := e.idMap[cur]
		if !ok {
			return cur
		}
		if seen[cur] {
			fail("不变量④违反: idMap 闭包有环 at " + cur)
		}
		seen[cur] = true
		cur = next
	}
}

// resolvedParent 返回映射后的 parent_id（"" 表示顶级）。
func (e *engine) resolvedParent(m *MenuRow) string {
	if m.ParentID() == nil {
		return ""
	}
	return e.resolve(*m.ParentID())
}

// aliveSubtree 存活后代节点数（基于全部解析行的 parent 链）。
func (e *engine) aliveSubtree(root string) int {
	children := map[string][]string{}
	for _, m := range e.all {
		if m.ParentID() != nil {
			children[*m.ParentID()] = append(children[*m.ParentID()], m.ID())
		}
	}
	count := 0
	var dfs func(id string)
	dfs = func(id string) {
		for _, c := range children[id] {
			if e.byID[c].DeletedAt() == nil {
				count++
			}
			dfs(c)
		}
	}
	dfs(root)
	return count
}

// less 规范 id 选择优先级：① 存活 ② 存活子树大 ③ path 非空 ④ created_at 早 ⑤ id 升序。
func (e *engine) less(a, b *MenuRow) bool {
	aAlive, bAlive := a.DeletedAt() == nil, b.DeletedAt() == nil
	if aAlive != bAlive {
		return aAlive
	}
	sa, sb := e.aliveSubtree(a.ID()), e.aliveSubtree(b.ID())
	if sa != sb {
		return sa > sb
	}
	aPath, bPath := a.Path() != nil && *a.Path() != "", b.Path() != nil && *b.Path() != ""
	if aPath != bPath {
		return aPath
	}
	if a.CreatedAt() != b.CreatedAt() {
		return a.CreatedAt() < b.CreatedAt()
	}
	return a.ID() < b.ID()
}

func (e *engine) pickCanonical(group []*MenuRow) (*MenuRow, string) {
	sort.Slice(group, func(i, j int) bool { return e.less(group[i], group[j]) })
	c := group[0]
	reason := fmt.Sprintf("alive=%v subtree=%d path=%q created=%s",
		c.DeletedAt() == nil, e.aliveSubtree(c.ID()), deref(c.Path()), c.CreatedAt())
	return c, reason
}

// runR1 顶级目录合并。
func (e *engine) runR1() {
	groups := map[string][]*MenuRow{}
	for _, m := range e.all {
		if m.ParentID() == nil && m.MenuType() == "M" {
			key := strings.TrimSuffix(m.Name(), "test")
			groups[key] = append(groups[key], m)
		}
	}
	keys := sortedKeys(groups)
	for _, k := range keys {
		g := groups[k]
		if len(g) < 2 {
			continue
		}
		canon, reason := e.pickCanonical(g)
		rec := foldRecord{Key: k, Canonical: canon.ID(), Reason: reason}
		for _, m := range g {
			if m.ID() == canon.ID() {
				continue
			}
			e.idMap[m.ID()] = canon.ID()
			delete(e.keep, m.ID())
			rec.Folded = append(rec.Folded, m.ID())
		}
		e.r1Folds = append(e.r1Folds, rec)
	}
}

// runR3Once 同级内容去重一轮，返回本轮折叠数。
func (e *engine) runR3Once() int {
	type gkey struct{ parent, name, mtype, perms string }
	groups := map[gkey][]*MenuRow{}
	for id := range e.keep {
		m := e.byID[id]
		if m.ParentID() == nil && m.MenuType() == "M" {
			continue // 顶级 M 已由 R1 处理
		}
		k := gkey{e.resolvedParent(m), m.Name(), m.MenuType(), deref(m.Perms())}
		groups[k] = append(groups[k], m)
	}
	folded := 0
	for _, g := range groups {
		if len(g) < 2 {
			continue
		}
		canon, reason := e.pickCanonical(g)
		rec := foldRecord{
			Key:       fmt.Sprintf("(%s,%s,%s,%s)", canon.Name(), canon.MenuType(), deref(canon.Perms()), e.resolvedParent(canon)[:8]),
			Canonical: canon.ID(),
			Reason:    reason,
		}
		for _, m := range g {
			if m.ID() == canon.ID() {
				continue
			}
			e.idMap[m.ID()] = canon.ID()
			delete(e.keep, m.ID())
			rec.Folded = append(rec.Folded, m.ID())
			folded++
		}
		e.r3Folds = append(e.r3Folds, rec)
	}
	return folded
}

// resurrectOrphans R0 例外：保留行重定向后的 parent 不在保留集 → 复活祖先。
// 返回复活 id 列表；循环直到不动点（复活的祖先自身可能还有软删祖父）。
func (e *engine) resurrectOrphans() []string {
	var out []string
	for {
		added := false
		for id := range e.keep {
			m := e.byID[id]
			p := e.resolvedParent(m)
			if p == "" || e.keep[p] {
				continue
			}
			// parent 被 R0 过滤且分组内无存活等价目录 → 复活
			e.keep[p] = true
			e.resurrected[p] = true
			out = append(out, p)
			added = true
		}
		if !added {
			return out
		}
	}
}

// checkInvariants 不变量校验（违反即 exit 1）。
func (e *engine) checkInvariants() {
	// ① 保留集顶级分组键无重复
	topKeys := map[string]string{}
	for id := range e.keep {
		m := e.byID[id]
		if m.ParentID() == nil && m.MenuType() == "M" {
			k := strings.TrimSuffix(m.Name(), "test")
			if prev, dup := topKeys[k]; dup {
				fail(fmt.Sprintf("不变量①违反: 顶级目录键 %q 重复 (%s vs %s)", k, prev, id))
			}
			topKeys[k] = id
		}
	}
	// ② 每个非顶级保留行 parent ∈ 保留集
	for id := range e.keep {
		m := e.byID[id]
		p := e.resolvedParent(m)
		if p != "" && !e.keep[p] {
			fail(fmt.Sprintf("不变量②违反: %s(%s) 的 parent %s 不在保留集", m.Name(), id[:8], p[:8]))
		}
	}
	// ③ 保留集内无 (parent,menu_name,menu_type,perms) 重复
	type gkey struct{ parent, name, mtype, perms string }
	seen := map[gkey]string{}
	for id := range e.keep {
		m := e.byID[id]
		k := gkey{e.resolvedParent(m), m.Name(), m.MenuType(), deref(m.Perms())}
		if prev, dup := seen[k]; dup {
			fail(fmt.Sprintf("不变量③违反: (%s,%s,%s) 重复 (%s vs %s)", k.name, k.mtype, k.perms, prev[:8], id[:8]))
		}
		seen[k] = id
	}
	// ④ idMap 闭包无环（resolve 内置检测）
	for id := range e.idMap {
		e.resolve(id)
	}
}

// keptRows 保留集按 order_num → created_at → id 排序。
func (e *engine) keptRows() []*MenuRow {
	var rows []*MenuRow
	for id := range e.keep {
		rows = append(rows, e.byID[id])
	}
	sort.Slice(rows, func(i, j int) bool {
		oi, _ := strconv.Atoi(deref(rows[i].Fields[9]))
		oj, _ := strconv.Atoi(deref(rows[j].Fields[9]))
		if oi != oj {
			return oi < oj
		}
		if rows[i].CreatedAt() != rows[j].CreatedAt() {
			return rows[i].CreatedAt() < rows[j].CreatedAt()
		}
		return rows[i].ID() < rows[j].ID()
	})
	return rows
}

// sqlValue 把第 idx 列字段渲染为 SQL 字面量（单引号转义 ''；nil→NULL）。
func sqlValue(idx int, v *string) string {
	if v == nil {
		return "NULL"
	}
	if numericCols[idx] {
		return *v
	}
	return "'" + strings.ReplaceAll(*v, "'", "''") + "'"
}

// menuCols 与输入一致的 19 列清单。
const menuCols = `"id", "created_at", "updated_at", "deleted_at", "created_by", "updated_by", "version", "menu_name", "parent_id", "order_num", "path", "component", "menu_type", "visible", "status", "perms", "icon", "remark", "meta"`

// generateSQL 生成幂等 INSERT（每行一条，ON CONFLICT DO NOTHING）。
func (e *engine) generateSQL(rows []*MenuRow) string {
	var b strings.Builder
	b.WriteString("-- 由 scripts/tmp_menuimport -mode=gen 生成：生产菜单数据去重后导入 dev\n")
	b.WriteString("-- 幂等：每行 ON CONFLICT DO NOTHING；parent_id 已经 idMap 闭包重定向；复活行 deleted_at 写 NULL\n\n")
	for _, m := range rows {
		vals := make([]string, 19)
		for i := 0; i < 19; i++ {
			v := m.Fields[i]
			switch i {
			case 3: // deleted_at：复活行写 NULL
				if e.resurrected[m.ID()] {
					v = nil
				}
			case 8: // parent_id：闭包重定向
				if v != nil {
					r := e.resolve(*v)
					v = &r
				}
			}
			vals[i] = sqlValue(i, v)
		}
		fmt.Fprintf(&b, "INSERT INTO \"sys_menu\" (%s) VALUES (%s) ON CONFLICT DO NOTHING;\n", menuCols, strings.Join(vals, ", "))
	}
	return b.String()
}

// r4Stats sys_role_menu 映射统计（仅报告，不生成导入 SQL）。
func (e *engine) r4Stats(roleMenus []*RoleMenuRow) (total, mappedFolded, deduped int) {
	type pair struct{ r, m string }
	seen := map[pair]bool{}
	for _, rm := range roleMenus {
		mid := e.resolve(rm.MenuID)
		if mid != rm.MenuID {
			mappedFolded++
		}
		seen[pair{rm.RoleID, mid}] = true
	}
	return len(roleMenus), mappedFolded, len(seen)
}

// writeReport 生成 dedupe-report.md。
func (e *engine) writeReport(rows []*MenuRow, resurrected []string, r4total, r4folded, r4deduped int) {
	var b strings.Builder
	b.WriteString("# 菜单数据去重映射报告（260814-ehg）\n\n")
	fmt.Fprintf(&b, "输入: sys_menu 386 条（含软删）→ 保留集 %d 条（复活 %d 条）\n\n", len(rows), len(resurrected))

	// R0 统计：软删过滤行数与值得注意的消除项
	r0Count := 0
	for _, m := range e.all {
		if m.DeletedAt() != nil && !e.keep[m.ID()] {
			r0Count++
		}
	}
	fmt.Fprintf(&b, "## R0 软删过滤\n\n- 软删行合计 %d 条（含被 R1 折叠的软删目录）\n", r0Count)
	for _, m := range e.all {
		if m.DeletedAt() != nil && m.ParentID() == nil && m.MenuType() == "M" {
			fmt.Fprintf(&b, "- 顶级目录 `%s`（%s）：R0 消除/R1 折叠候选\n", m.ID(), m.Name())
		}
	}
	b.WriteString("\n")

	b.WriteString("## R1 顶级目录折叠组\n\n")
	if len(e.r1Folds) == 0 {
		b.WriteString("（无）\n\n")
	}
	for _, f := range e.r1Folds {
		fmt.Fprintf(&b, "### %s\n\n- 规范 id（保留）: `%s` — %s\n- 折叠 id:\n", f.Key, f.Canonical, f.Reason)
		for _, id := range f.Folded {
			m := e.byID[id]
			fmt.Fprintf(&b, "  - `%s`（name=%s deleted_at=%v）\n", id, m.Name(), m.DeletedAt() != nil)
		}
		b.WriteString("\n")
	}

	b.WriteString("## R3 同级内容折叠组（fixpoint）\n\n")
	if len(e.r3Folds) == 0 {
		b.WriteString("（无）\n\n")
	}
	r3FoldedTotal := 0
	for _, f := range e.r3Folds {
		r3FoldedTotal += len(f.Folded)
		fmt.Fprintf(&b, "- `%s` ← 折叠 %d 条: %s（键 %s）\n", f.Canonical, len(f.Folded), strings.Join(shortIDs(f.Folded), ", "), f.Key)
	}
	fmt.Fprintf(&b, "\nR3 合计折叠 %d 行。\n\n", r3FoldedTotal)

	b.WriteString("## R0 复活行（软删祖先，deleted_at 写 NULL 导入）\n\n")
	if len(resurrected) == 0 {
		b.WriteString("（无）\n\n")
	}
	for _, id := range resurrected {
		m := e.byID[id]
		fmt.Fprintf(&b, "- `%s`（%s / %s）\n", id, m.Name(), m.MenuType())
	}
	b.WriteString("\n")

	fmt.Fprintf(&b, "## R4 sys_role_menu 映射统计（不导入）\n\n- 输入 %d 条；menu_id 命中折叠映射 %d 条；映射+去重后 (role_id,menu_id) 唯一对 %d 条\n\n", r4total, r4folded, r4deduped)
	b.WriteString("## R5 sys_role\n\n不导入（dev 复用现有 admin/user 角色）。\n\n")

	// 保留集顶级目录清单
	b.WriteString("## 保留集顶级目录\n\n")
	for _, m := range rows {
		if m.ParentID() == nil && m.MenuType() == "M" {
			fmt.Fprintf(&b, "- `%s` %s\n", m.ID(), m.Name())
		}
	}

	if err := os.WriteFile(reportOut, []byte(b.String()), 0644); err != nil {
		fail("write report: " + err.Error())
	}
}

func shortIDs(ids []string) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = "`" + id[:8] + "`"
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// runGen Task 2：去重 + 生成干净 SQL + 映射报告。
func runGen() {
	menus, roleMenus, _ := loadAll()
	e := newEngine(menus)

	// R1 顶级目录合并
	e.runR1()
	// R2-R3 fixpoint：闭包重定向 + 同级内容去重直到不动点
	for {
		if e.runR3Once() == 0 {
			break
		}
	}
	// R0 例外：复活软删祖先
	resurrected := e.resurrectOrphans()
	// 不变量校验
	e.checkInvariants()
	fmt.Println("INVARIANTS PASS")

	rows := e.keptRows()
	r4total, r4folded, r4deduped := e.r4Stats(roleMenus)

	if err := os.WriteFile(dedupSQLOut, []byte(e.generateSQL(rows)), 0644); err != nil {
		fail("write dedup sql: " + err.Error())
	}
	e.writeReport(rows, resurrected, r4total, r4folded, r4deduped)

	fmt.Printf("GEN PASS kept=%d resurrected=%d r1_groups=%d r3_groups=%d r4: %d->%d (folded refs %d)\n",
		len(rows), len(resurrected), len(e.r1Folds), len(e.r3Folds), r4total, r4deduped, r4folded)
	fmt.Printf("output: %s + %s\n", dedupSQLOut, reportOut)
}
