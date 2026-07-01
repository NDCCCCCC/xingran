/**
 * 路由路径常量
 * 统一管理所有路由路径，避免硬编码
 */

// ============================================
// 用户中心相关路由
// ============================================
export const USER_PROFILE = "/user/profile"; // 个人中心
export const USER_SETTINGS = "/user/settings"; // 系统设置
export const USER_NOTICES = "/user/my-notices"; // 我的通知

// ============================================
// 主要页面路由
// ============================================
export const DASHBOARD = "/dashboard"; // 系统仪表盘（可配置仪表盘系统）
export const MONITOR_DASHBOARD = "/monitor/dashboard"; // 监控仪表盘（服务器性能监控）
export const LOGIN = "/login"; // 登录页

// ============================================
// 系统管理相关路由
// ============================================
export const SYSTEM_USER = "/system/user"; // 用户管理
export const SYSTEM_ROLE = "/system/role"; // 角色管理
export const SYSTEM_MENU = "/system/menu"; // 菜单管理
export const SYSTEM_DEPT = "/system/dept"; // 部门管理
export const SYSTEM_DICT = "/system/dict"; // 字典管理
export const SYSTEM_NOTICE = "/system/notice"; // 通知公告

// ============================================
// 网络设备相关路由
// ============================================
export const NETWORK_DEVICES = "/network/devices"; // 设备管理
export const NETWORK_PORTS = "/network/ports"; // 端口管理
export const NETWORK_DISCOVERIES = "/network/discoveries"; // 设备发现

// ============================================
// 工单管理相关路由
// ============================================
export const WORKORDER_ORDERS = "/workorder/orders"; // 工单管理
export const WORKORDER_CATEGORIES = "/workorder/categories"; // 工单分类
export const WORKORDER_STATISTICS = "/workorder/statistics"; // 工单统计

// ============================================
// 运维管理相关路由
// ============================================
export const OPS_BUILDINGS = "/ops/operations/buildings"; // 楼宇管理

// ============================================
// 值班管理相关路由
// ============================================
export const DUTY_MY_DUTY = "/duty/my-duty"; // 我的值班
