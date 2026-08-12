/**
 * 菜单路由类型定义
 * 与后端 models/menu.go 中的 MenuMeta 结构体保持一致
 */

/**
 * 路由元数据（与后端一致）
 * 统一管理路由相关配置：标题、图标、权限、缓存等
 */
export interface RouteMeta {
  // 基础信息
  title: string;           // 页面标题（必填，中文）
  icon?: string;           // 菜单图标（如 UserOutlined）

  // 布局控制
  hidden?: boolean;        // 是否隐藏菜单项（但路由可访问）
  affix?: boolean;         // 是否固定标签页（不可关闭）
  keepAlive?: boolean;     // 是否缓存组件（keep-alive）

  // 权限控制
  permissions?: string[];  // 需要的权限标识（如 ['system:user:list']）
  roles?: string[];        // 允许的角色（如 ['admin', 'operator']）

  // 国际化（预留）
  i18nKey?: string;        // 国际化 key

  // 其他扩展
  noCache?: boolean;       // 是否禁用缓存
  link?: string;           // 外链跳转
}

/**
 * 菜单路由配置
 * 与后端菜单数据结构对应
 */
export interface MenuRouteConfig {
  path: string;                    // 路由路径（如 'system/user'）
  component: string;               // 组件路径（如 'pages/system/user/index'）
  meta: RouteMeta;                 // 路由元数据
  redirect?: string;               // 重定向路径（可选）
  children?: MenuRouteConfig[];    // 子路由配置
}

/**
 * 面包屑路径项
 */
export interface BreadcrumbItem {
  path: string;      // 路径
  title: string;     // 标题
}

/**
 * 路由权限检查结果
 */
export interface RoutePermissionCheck {
  hasPermission: boolean;    // 是否有权限
  missingPermissions?: string[];  // 缺少的权限列表
}
