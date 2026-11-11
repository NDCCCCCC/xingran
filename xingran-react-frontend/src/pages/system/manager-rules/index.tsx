/**
 * 经理同步 · 同步规则（旧路由薄壳）
 *
 * 该页已合并进「系统设置 → 经理同步」分类区块（见
 * @/pages/system/settings/manager-sync，内含「机构定义」+「同步规则」两个 Tab）。
 *
 * 保留本薄壳的原因：历史 sys_menu 仍可能指向 component=system/manager-rules/index，
 * 转发到合并组件后旧路由依然能打开（菜单入口已隐藏，不再有独立入口）。
 */

import ManagerSyncSettings from "@/pages/system/settings/manager-sync";

export default ManagerSyncSettings;
