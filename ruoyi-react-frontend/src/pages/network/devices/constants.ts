/**
 * NetworkDevice 常量定义
 */

export interface SelectOption {
  label: string;
  value: string | number;
}

// 设备厂商选项
export const VENDOR_OPTIONS: SelectOption[] = [
  { label: "Huawei", value: "huawei" },
  { label: "H3C", value: "h3c" },
  { label: "Ruijie", value: "ruijie" },
  { label: "Maipu", value: "maipu" },
  { label: "Cisco", value: "cisco" },
];

// 设备类型选项（Phase 69 DICT-03: 降级为静态 fallback——搜索/编辑下拉与表格 label 已迁
// useDict("network_device_type")，字典空态/接口异常时回退本数组；导出保留不删除）
export const DEVICE_TYPE_OPTIONS: SelectOption[] = [
  { label: "路由器", value: "router" },
  { label: "交换机", value: "switch" },
  { label: "防火墙", value: "firewall" },
  { label: "无线AP", value: "ap" },
  { label: "负载均衡器", value: "loadbalancer" },
];

// 设备状态选项
export const STATUS_OPTIONS: SelectOption[] = [
  { label: "在线", value: 0 },
  { label: "离线", value: 1 },
  { label: "未知", value: 2 },
];

// 状态颜色映射
export const STATUS_COLOR_MAP: Record<number, string> = {
  0: "success",
  1: "error",
  2: "default",
};

// 设备类型标签颜色
export const DEVICE_TYPE_TAG_COLOR = "blue";

// 厂商标签颜色
export const VENDOR_TAG_COLOR = "geekblue";

/**
 * 网络设备模块的权限码（v1.24 HYG-04 / #16 前端侧）。
 *
 * 声明位置说明（**刻意不放在 `src/constants/permissionCodes.ts`**）：
 *   那个文件是 PERM-06 的「模块 × 操作 → 权限码」全局单一真相源，
 *   其双向校验脚本 `scripts/check-permission-codes.mjs` 会逐模块比对
 *   「TS 声明 ↔ 后端常量（pkg/permission/config.go）↔ migrations 的 sys_menu 种子」，
 *   其中一条判据是「TS 声明了但 sys_menu 未播种 → 该权限永远不会被授予 → 报错」。
 *   而 network:device 的**动作级**权限码在 sys_menu 中**尚未播种 F 型按钮**
 *   （`menu_catalog_seed.sql` 只播种了 C 型页面菜单，perms='network:device'），
 *   登记进全局表会直接击穿该门禁 ⇒ 本 phase 在页内域声明，待后端补播种后再登记进全局表。
 *
 * 码的来源：
 *   - `page`：C 型页面菜单 `network/devices/index` 的 perms（sys_menu 已播种）——
 *     能进入设备管理页面的用户都持有它。
 *   - `add` / `edit` / `remove`：后端 `pkg/permission/config.go` 已定义同名常量
 *     （NetworkDeviceAdd / NetworkDeviceEdit / NetworkDeviceDelete），
 *     但 sys_menu 尚无 F 型按钮 ⇒ 当前**没有授权来源**。
 */
export const DEVICE_PERMISSION_CODES = {
  /** 页面级（C 型菜单）权限码 */
  page: "network:device",
  /** 动作级：新增 */
  add: "network:device:add",
  /** 动作级：编辑 */
  edit: "network:device:edit",
  /** 动作级：删除 */
  remove: "network:device:delete",
} as const;
