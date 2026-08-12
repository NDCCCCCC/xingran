package base

import (
	"fmt"
	"sort"

	"github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// BaseListRequest 通用分页+排序请求基类。
//
// 应嵌入到各模块的 XxxListRequest/Params 结构体中，使 json 顶层自动出现
// current / pageSize / orderByColumn / isAsc 四个字段。
//
// 用法示例:
//
//	type UserListRequest struct {
//	    base.BaseListRequest
//	    Username string `json:"username"`
//	    Status   *int   `json:"status,omitempty"`
//	}
//
// 嵌入后字段被提升 (promoted)，访问 userReq.Current / userReq.OrderByColumn 都可工作。
type BaseListRequest struct {
	Current       int    `json:"current"`
	PageSize      int    `json:"pageSize"`
	OrderByColumn string `json:"orderByColumn,omitempty"` // 逻辑字段名（对应 allowed map 的 key）
	IsAsc         *bool  `json:"isAsc,omitempty"`         // nil 或 true=升序，false=降序
}

// ResolveSort 是 ApplySort 的纯逻辑核心。
// 在白名单中查找 OrderByColumn 对应的 db 列名,并根据 IsAsc 决定方向。
//
// 返回:
//   - col: 数据库列名(可能带 schema/表别名,如 "sys_user.created_at")
//   - direction: "ASC" 或 "DESC"
//   - ok: true 表示找到白名单匹配,可以安全拼装 ORDER BY
//
// 无效字段返回 ok=false（不返回错误,因为非法排序应静默忽略以保证接口向前兼容）。
// OrderByColumn 为空时也返回 false（调用方应保持 db 链上的默认排序）。
//
// 关键安全设计:db_column 来自 allowed map 的 value,不是用户输入。
// 用户输入只能匹配 map key,即使恶意构造也无法注入 SQL。
func ResolveSort(req BaseListRequest, allowed map[string]string) (col, direction string, ok bool) {
	if req.OrderByColumn == "" {
		return "", "", false
	}
	dbCol, found := allowed[req.OrderByColumn]
	if !found {
		return "", "", false
	}
	dir := "ASC"
	if req.IsAsc != nil && !*req.IsAsc {
		dir = "DESC"
	}
	return dbCol, dir, true
}

// ApplySort 把用户传入的 OrderByColumn 翻译为安全的 GORM ORDER BY 子句。
//
// 行为:
//   - req.OrderByColumn 为空 → 保持 db 链默认排序（不追加 Order）
//   - allowed 为空/nil → 视为无可用排序字段,忽略 OrderByColumn
//   - OrderByColumn 不在白名单 → 打 warn 日志,db 链不变
//   - 匹配成功 → 在 db 链后追加 .Order("<col> <dir>")
//
// 用法:
//
//	allowed := map[string]string{
//	    "createdAt": "sys_user.created_at",
//	    "username":  "sys_user.username",
//	}
//	db = base.ApplySort(db, req.BaseListRequest, allowed)
func ApplySort(db *gorm.DB, req BaseListRequest, allowed map[string]string) *gorm.DB {
	if db == nil {
		return nil
	}
	col, direction, ok := ResolveSort(req, allowed)
	if !ok {
		if req.OrderByColumn != "" {
			logger.Warnf(
				"ApplySort ignored invalid orderByColumn=%q (allowed keys: %v)",
				req.OrderByColumn, sortedKeys(allowed),
			)
		}
		return db
	}
	// col 来自编译期常量/service 顶部硬编码的 map value,
	// direction 只取字面量 "ASC"/"DESC",均非外部输入 → 无 SQL 注入风险
	return db.Order(fmt.Sprintf("%s %s", col, direction))
}

// sortedKeys 返回 allowed map 的 key 列表(按字典序),仅用于日志输出。
func sortedKeys(allowed map[string]string) []string {
	keys := make([]string, 0, len(allowed))
	for k := range allowed {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
