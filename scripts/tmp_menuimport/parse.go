package main

import (
	"fmt"
	"strings"
)

// parse.go — 字符级 SQL INSERT 解析器（多行 VALUES）
//
// 解析规则（禁止行正则切 tuple，remark/meta 内含括号逗号）：
//  1. 以 `INSERT INTO "表名"` 定位块起点，`VALUES` 之后进入 tuple 扫描，
//     直到块级 `ON CONFLICT` 关键字（引号外）结束。
//  2. 字符级状态机：inQuote 布尔（' 进入，' 退出，'' 视为转义不切换状态）
//     + 括号深度。仅在深度 0 且非引号内的 '(' 开始 tuple；深度回到 0 的 ')'
//     结束 tuple；深度 1 且非引号内的 ',' 切字段。
//  3. 字段去首尾空白后：裸字面量 NULL → nil；被单引号包围 → 去引号并把 ''
//     还原为 '；其余（数字等）原样保留为字符串指针。
//     注意 '' 是空字符串（≠ NULL）。

// parseInserts 解析整份 SQL，返回 表名 → 行集合（每行 = 字段切片，nil 表示 NULL）。
func parseInserts(sql string) map[string][][]*string {
	out := map[string][][]*string{}
	i := 0
	n := len(sql)
	for i < n {
		// 定位下一个 INSERT INTO
		idx := strings.Index(sql[i:], `INSERT INTO "`)
		if idx < 0 {
			break
		}
		i += idx + len(`INSERT INTO "`)
		endQ := strings.IndexByte(sql[i:], '"')
		if endQ < 0 {
			break
		}
		table := sql[i : i+endQ]
		i += endQ + 1
		// 定位 VALUES 关键字
		vidx := strings.Index(sql[i:], "VALUES")
		if vidx < 0 {
			break
		}
		i += vidx + len("VALUES")
		// 从 VALUES 之后扫描 tuple，直到引号外的 ON CONFLICT
		rows, next := parseValuesSection(sql, i)
		out[table] = append(out[table], rows...)
		i = next
	}
	return out
}

// parseValuesSection 从 pos 开始解析 `(...),(...),...` 直到遇到引号外的
// "ON CONFLICT"（或 ';'），返回解析出的行与结束位置。
func parseValuesSection(s string, pos int) ([][]*string, int) {
	var rows [][]*string
	n := len(s)
	i := pos
	inQuote := false
	depth := 0
	var cur []string      // 当前 tuple 的原始字段文本
	var field strings.Builder

	flushField := func() {
		cur = append(cur, strings.TrimSpace(field.String()))
		field.Reset()
	}
	flushTuple := func() {
		flushField()
		rows = append(rows, convertFields(cur))
		cur = nil
	}

	for i < n {
		c := s[i]
		if inQuote {
			field.WriteByte(c)
			if c == '\'' {
				// '' 转义：两个连续单引号不退出引号状态
				if i+1 < n && s[i+1] == '\'' {
					field.WriteByte(s[i+1])
					i += 2
					continue
				}
				inQuote = false
			}
			i++
			continue
		}
		// 非引号内
		switch {
		case c == '\'':
			inQuote = true
			field.WriteByte(c)
			i++
		case c == '(':
			if depth == 0 {
				cur = nil
				field.Reset()
			} else {
				field.WriteByte(c)
			}
			depth++
			i++
		case c == ')':
			depth--
			if depth == 0 {
				flushTuple()
			} else {
				field.WriteByte(c)
			}
			i++
		case c == ',' && depth == 1:
			flushField()
			i++
		case c == '-' && i+1 < n && s[i+1] == '-' && depth == 0:
			// 行注释（tuple 间隙理论上没有，防御性跳过）
			for i < n && s[i] != '\n' {
				i++
			}
		case depth == 0 && (c == ';' || strings.HasPrefix(s[i:], "ON CONFLICT")):
			// 块结束
			if strings.HasPrefix(s[i:], "ON CONFLICT") {
				i += len("ON CONFLICT")
			} else {
				i++
			}
			return rows, i
		default:
			if depth > 0 {
				field.WriteByte(c)
			}
			i++
		}
	}
	return rows, i
}

// convertFields 把原始字段文本转为 *string：NULL→nil，引号包围→去引号+''→'，其余原样。
func convertFields(raw []string) []*string {
	out := make([]*string, len(raw))
	for i, f := range raw {
		if f == "NULL" {
			out[i] = nil
			continue
		}
		if len(f) >= 2 && f[0] == '\'' && f[len(f)-1] == '\'' {
			v := strings.ReplaceAll(f[1:len(f)-1], "''", "'")
			out[i] = &v
			continue
		}
		v := f
		out[i] = &v
	}
	return out
}

// ---------- 行结构 ----------

// sys_menu 列序（19 列）：
// 0 id, 1 created_at, 2 updated_at, 3 deleted_at, 4 created_by, 5 updated_by,
// 6 version, 7 menu_name, 8 parent_id, 9 order_num, 10 path, 11 component,
// 12 menu_type, 13 visible, 14 status, 15 perms, 16 icon, 17 remark, 18 meta
type MenuRow struct {
	Fields []*string
}

func (r *MenuRow) ID() string        { return deref(r.Fields[0]) }
func (r *MenuRow) CreatedAt() string { return deref(r.Fields[1]) }
func (r *MenuRow) DeletedAt() *string { return r.Fields[3] }
func (r *MenuRow) Name() string      { return deref(r.Fields[7]) }
func (r *MenuRow) ParentID() *string { return r.Fields[8] }
func (r *MenuRow) Path() *string     { return r.Fields[10] }
func (r *MenuRow) MenuType() string  { return deref(r.Fields[12]) }
func (r *MenuRow) Perms() *string    { return r.Fields[15] }

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// RoleMenuRow sys_role_menu 列序（4 列）：0 id, 1 role_id, 2 menu_id, 3 created_at
type RoleMenuRow struct {
	ID     string
	RoleID string
	MenuID string
}

func toMenuRows(raw [][]*string) []*MenuRow {
	rows := make([]*MenuRow, 0, len(raw))
	for _, f := range raw {
		if len(f) != 19 {
			fail(fmt.Sprintf("sys_menu 行列数异常: got %d want 19", len(f)))
		}
		rows = append(rows, &MenuRow{Fields: f})
	}
	return rows
}

func toRoleMenuRows(raw [][]*string) []*RoleMenuRow {
	rows := make([]*RoleMenuRow, 0, len(raw))
	for _, f := range raw {
		if len(f) != 4 {
			fail(fmt.Sprintf("sys_role_menu 行列数异常: got %d want 4", len(f)))
		}
		rows = append(rows, &RoleMenuRow{ID: deref(f[0]), RoleID: deref(f[1]), MenuID: deref(f[2])})
	}
	return rows
}
