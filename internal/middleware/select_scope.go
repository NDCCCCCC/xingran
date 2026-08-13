package middleware

// SelectScope 是从 scopes 集合 + 继承权限标志 + 操作类型推导限流 scope 的纯函数。
//
// 语义(Phase 61 / D-11/D-12/D-13):
//  1. inheritPerms == true → 短路返回 ("default", true),不做 action-based 检查
//     (InheritPerms=true 时 scopes 仅含细粒度 permission code,如 system:user:list,
//     不含粗粒度 read/write/admin,action-based 匹配必然失败,故短路)
//  2. 否则计算 requiredScope := getRequiredScope(action)(未知 action 默认 read)
//  3. scopes 包含 requiredScope → 精确匹配返回 (requiredScope, true)
//  4. scopes 包含 "admin" → 返回 ("admin", true)(admin 最高限额覆盖)
//  5. 都不含 → 返回 ("", false)(fail-closed,无 fallback,D-12)
//
// Phase 61 QUAL-03 引入,替代原 getScopeFromContext 直接读 context 的实现,
// 使单元测试无需 context key 中转,直接断言纯函数返回值(D-20)。
func SelectScope(scopes []string, inheritPerms bool, action string) (scope string, allowed bool) {
	// D-13: inherit_perms 短路保留 — 走 default 限额
	if inheritPerms {
		return "default", true
	}

	// D-11: action → requiredScope(未知 action 默认 read 兜底)
	required := getRequiredScope(action)

	// D-12: 精确匹配优先
	for _, s := range scopes {
		if s == required {
			return required, true
		}
	}

	// D-12: admin 最高限额覆盖
	for _, s := range scopes {
		if s == "admin" {
			return "admin", true
		}
	}

	// D-12: fail-closed,无 fallback
	return "", false
}
