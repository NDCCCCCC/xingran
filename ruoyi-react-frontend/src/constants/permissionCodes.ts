/**
 * 「模块 × 操作 → 权限码」单一真相源（v1.24 PERM-06）。
 *
 * 为什么要单一真相源：权限码在后端（pkg/permission/config.go 常量 +
 * migrations 的 sys_menu.perms 菜单种子）与前端（按钮门控）各有一份声明，
 * 历史上已发生过命名漂移（migration_159 把 ops:buildings:* 改成 ops:building:*）。
 * 前后端各写各的字符串迟早会再次漂移 —— 本文件是前端侧的**唯一**声明点。
 *
 * 校验：`node scripts/check-permission-codes.mjs` 做**双向**校验
 *   - TS 声明了但后端常量没有 → 报（前端会永远拿不到权限）
 *   - TS 声明了但 sys_menu 未声明 → 报（该权限永远不会被授予）
 *   - 后端常量有但 TS 没声明 → 报（前端漏了按钮门控）
 *   - sys_menu 声明了 F 型按钮但 TS 没声明 → 报（同上）
 * 任何一项命中 → exit 1（漂移）。
 *
 * 命名约定：一个模块的权限码共享同一前缀（`system:user` → `system:user:list` …），
 * 前缀本身（`system:user`）是**页面本体**权限（C 型菜单），不是操作码，
 * 不进本表（校验脚本按「前缀 + ':'」圈定操作码作用域）。
 */

export const PERMISSION_CODES = {
  systemUser: {
    list: "system:user:list",
    view: "system:user:view",
    add: "system:user:add",
    edit: "system:user:edit",
    remove: "system:user:remove",
    resetPwd: "system:user:resetPwd",
    import: "system:user:import",
    export: "system:user:export",
  },
  // 85-02（RBAC-04）：system 4 组写操作逐动作拆分，TS 侧提前声明（b 类零豁免）。
  // ⚠️ 这些码的 sys_menu F 型菜单当前为 0 声明（Phase 87 播种后才有授权来源），
  // c 类由 check-permission-codes.mjs 的 PENDING_SEED_WHITELIST 显式豁免；
  // 87 播种 migration 落地后必须清空该白名单。
  systemRole: {
    list: "system:role:list",
    view: "system:role:view",
    add: "system:role:add",
    edit: "system:role:edit",
    remove: "system:role:remove",
    export: "system:role:export",
  },
  systemMenu: {
    list: "system:menu:list",
    view: "system:menu:view",
    add: "system:menu:add",
    edit: "system:menu:edit",
    remove: "system:menu:remove",
  },
  systemDept: {
    list: "system:dept:list",
    view: "system:dept:view",
    add: "system:dept:add",
    edit: "system:dept:edit",
    remove: "system:dept:remove",
    import: "system:dept:import",
    export: "system:dept:export",
  },
  systemPost: {
    list: "system:post:list",
    view: "system:post:view",
    add: "system:post:add",
    edit: "system:post:edit",
    remove: "system:post:remove",
  },
} as const;

/** 模块键名（如 systemUser） */
export type PermissionModuleKey = keyof typeof PERMISSION_CODES;

/** 单个权限码字面量类型 */
export type PermissionCodeValue = (typeof PERMISSION_CODES)[PermissionModuleKey][keyof (typeof PERMISSION_CODES)[PermissionModuleKey]];

/** 系统用户模块的权限码（页面内最常用的一组） */
export const SYSTEM_USER_PERMS = PERMISSION_CODES.systemUser;
