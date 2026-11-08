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
  // ==================== 85-03（RBAC-01）批次 A：ops 域写码 TS 同批声明 ====================
  // 后端 pkg/permission/config.go「运维管理权限（ops）」区块的镜像声明（b 类零豁免）。
  // 其中 import/export 16 码为本批新增（此前 Excel 导入导出搭 ANY-of 组级车），
  // walls/doors/floor-plan 为既有字面量码——这些码的 sys_menu F 型菜单当前为 0 声明，
  // c 类由 check-permission-codes.mjs 的 PENDING_SEED_WHITELIST 显式豁免，
  // 87 播种 migration 落地后必须清空该白名单。
  opsBuilding: {
    query: "ops:building:query",
    spacesQuery: "ops:building:spaces:query",
    spaces3dQuery: "ops:building:spaces:3d:query",
    add: "ops:building:add",
    edit: "ops:building:edit",
    delete: "ops:building:delete",
    import: "ops:building:import",
    export: "ops:building:export",
  },
  opsFloor: {
    query: "ops:floor:query",
    add: "ops:floor:add",
    edit: "ops:floor:edit",
    delete: "ops:floor:delete",
    import: "ops:floor:import",
    export: "ops:floor:export",
  },
  opsWorkstation: {
    query: "ops:workstation:query",
    add: "ops:workstation:add",
    edit: "ops:workstation:edit",
    delete: "ops:workstation:delete",
    import: "ops:workstation:import",
    export: "ops:workstation:export",
    // 按人员反推部门（批量全库写，独立权限项不与 edit 共用）
    backfill: "ops:workstation:backfill",
  },
  opsAsset: {
    list: "ops:asset:list",
    add: "ops:asset:add",
    edit: "ops:asset:edit",
    delete: "ops:asset:delete",
    import: "ops:asset:import",
    export: "ops:asset:export",
  },
  opsInfoPoint: {
    query: "ops:infopoint:query",
    add: "ops:infopoint:add",
    edit: "ops:infopoint:edit",
    delete: "ops:infopoint:delete",
    import: "ops:infopoint:import",
    export: "ops:infopoint:export",
  },
  opsServerRoom: {
    query: "ops:serverroom:query",
    add: "ops:serverroom:add",
    edit: "ops:serverroom:edit",
    delete: "ops:serverroom:delete",
    import: "ops:serverroom:import",
    export: "ops:serverroom:export",
  },
  opsDedicatedLine: {
    query: "ops:dedicatedline:query",
    add: "ops:dedicatedline:add",
    edit: "ops:dedicatedline:edit",
    delete: "ops:dedicatedline:delete",
    import: "ops:dedicatedline:import",
    export: "ops:dedicatedline:export",
  },
  opsRoomDevice: {
    query: "ops:roomdevice:query",
    add: "ops:roomdevice:add",
    edit: "ops:roomdevice:edit",
    delete: "ops:roomdevice:delete",
    import: "ops:roomdevice:import",
    export: "ops:roomdevice:export",
  },
  opsLocationAlias: {
    list: "ops:location:alias:list",
    add: "ops:location:alias:add",
    edit: "ops:location:alias:edit",
    delete: "ops:location:alias:delete",
  },
  // FIX-13 / OQ-7：5 个读码此前只存在于后端 router.go 字面量里，
  // sys_menu 未播种、TS 未声明 → 读路由恒 403。migration_236 补种后同步声明。
  opsWall: {
    list: "ops:walls:list",
    add: "ops:walls:add",
    edit: "ops:walls:edit",
    delete: "ops:walls:delete",
  },
  opsDoor: {
    list: "ops:doors:list",
    add: "ops:doors:add",
    edit: "ops:doors:edit",
    delete: "ops:doors:delete",
  },
  opsFloorPlan: {
    list: "ops:floor-plan:list",
    query: "ops:floor-plan:query",
    save: "ops:floor-plan:save",
    edit: "ops:floor-plan:edit",
  },
  // FIX-06：工单评价创建（写）独立权限码，后端严格单一权限点挂载。
  workorderRating: {
    create: "ops:workorder:rating:create",
  },
  // ==================== 85-03（RBAC-01）批次 B：network 域写码 TS 同批声明 ====================
  // 后端「网络设备管理权限」区块的镜像声明。network 域仅 backup:add/restore/diff、
  // command:execute、discovery:add 有 sys_menu F 型授权来源（其余均进
  // PENDING_SEED_WHITELIST，87 播种后清空）。
  networkDevice: {
    list: "network:device:list",
    view: "network:device:view",
    add: "network:device:add",
    edit: "network:device:edit",
    delete: "network:device:delete",
    export: "network:device:export",
  },
  networkCredential: {
    list: "network:credential:list",
    view: "network:credential:view",
    add: "network:credential:add",
    edit: "network:credential:edit",
    delete: "network:credential:delete",
    export: "network:credential:export",
  },
  networkTemplate: {
    list: "network:template:list",
    add: "network:template:add",
    edit: "network:template:edit",
    delete: "network:template:delete",
    view: "network:template:view",
    export: "network:template:export",
  },
  networkCommand: {
    execute: "network:command:execute",
    view: "network:command:view",
    export: "network:command:export",
  },
  networkBackup: {
    list: "network:backup:list",
    add: "network:backup:add",
    delete: "network:backup:delete",
    restore: "network:backup:restore",
    diff: "network:backup:diff",
    export: "network:backup:export",
  },
  networkDiscovery: {
    add: "network:discovery:add",
    view: "network:discovery:view",
    delete: "network:discovery:delete",
    export: "network:discovery:export",
  },
  // ==================== 85-03（RBAC-01）批次 C：manager / AD 域 TS 同批声明 ====================
  // 既有字面量码升常量（migration_221 / migration_163 菜单 seed 对应按钮权限，
  // F 型非零 → 不进 PENDING_SEED_WHITELIST，87 播种后窗口期零影响面）。
  opsManagerRule: {
    list: "ops:manager:rule:list",
    edit: "ops:manager:rule:edit",
    delete: "ops:manager:rule:delete",
  },
  opsManagerOrg: {
    list: "ops:manager:org:list",
    edit: "ops:manager:org:edit",
    delete: "ops:manager:org:delete",
  },
  opsADConfig: {
    list: "ops:ad:config:list",
    add: "ops:ad:config:add",
    edit: "ops:ad:config:edit",
    delete: "ops:ad:config:delete",
    test: "ops:ad:config:test",
    sync: "ops:ad:config:sync",
  },
  opsADGroup: {
    edit: "ops:ad:group:edit",
    mappingSync: "ops:ad:group:mapping:sync",
  },
  opsADUser: {
    edit: "ops:ad:user:edit",
  },
  // ==================== Phase 144 ALRT-03：同步漂移告警 ====================
  // 与后端 pkg/permission/config.go + migration 238 的 sys_menu.perms 三方逐字一致。
  // ⚠️ 本组只放 4 个 F 型**操作码**；C 型页面本体权限 `system:adDriftAlert`
  // 按文件头既有惯例**不进本表**（校验脚本按「主前缀 + ':'」圈定 scope，
  // 若把页面本体码混进组里会把 scope 退化为 `system:` 并把所有 system:* 常量
  // 误报成 b 类漂移）。页面本体码在下方单独导出，供文档/引用核对。
  // 命名以 Lead 裁决为准（261108-v154-architecture-decisions.md §二）。
  adDriftAlert: {
    list: "system:adDriftAlert:list",
    config: "system:adDriftAlert:config",
    edit: "system:adDriftAlert:edit",
    trigger: "system:adDriftAlert:trigger",
  },
} as const;

/**
 * Phase 144 ALRT-03：同步漂移告警 **C 型页面本体**权限码。
 *
 * 与 PLAN-01-backend 的 `pkg/permission/config.go` 常量 + migration 238 播种的
 * sys_menu.perms 逐字一致。按文件头惯例不进 `PERMISSION_CODES` 操作码表
 * （否则 check-permission-codes.mjs 的 scope 圈定会被污染），单独导出供页面/文档核对。
 *
 * 5 个权限码逐字清单（三方一致）：
 *   - system:adDriftAlert           （C 型页面本体，本常量）
 *   - system:adDriftAlert:list      （读：status/logs/config）
 *   - system:adDriftAlert:config    （读：配置）
 *   - system:adDriftAlert:edit      （写：config/update）
 *   - system:adDriftAlert:trigger   （写：trigger）
 */
export const SYSTEM_AD_DRIFT_ALERT_PERM = "system:adDriftAlert";

/** 模块键名（如 systemUser） */
export type PermissionModuleKey = keyof typeof PERMISSION_CODES;

/** 单个权限码字面量类型 */
export type PermissionCodeValue =
  (typeof PERMISSION_CODES)[PermissionModuleKey][keyof (typeof PERMISSION_CODES)[PermissionModuleKey]];

/** 系统用户模块的权限码（页面内最常用的一组） */
export const SYSTEM_USER_PERMS = PERMISSION_CODES.systemUser;
