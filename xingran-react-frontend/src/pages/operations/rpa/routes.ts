/**
 * W7 / UI-FIX-01：RPA 模块路由常量（唯一事实来源）
 *
 * 为什么需要这个文件（真实浏览器暴露的 P0 缺陷）：
 *   向导页 `wizard/index.tsx` 三处写死了 `/operations/rpa/tasks` —— 这个路径
 *   **不存在任何路由**。RPA 页面不由前端静态注册，而是由 sys_menu seed
 *   （migration_256 / 255 / 258）播种 component，再由 RouteGenerator.resolvePath
 *   把 sys_menu 的 path 逐级拼接成真实前端路由。
 *
 *   PG 实证（2026-11-26 查询生产 10.62.10.34 `sys_menu`）：
 *     770e8400-...440001  虚拟机      parent=NULL  path=vdi
 *     af1451a4-...3920    RPA 管理     parent=770e8400 path=rpa   component=pages/operations/rpa/index
 *     a1b2c3d4-0255-...   RPA 实时画面  parent=af1451a4 path=live  component=pages/operations/rpa/live/index
 *     a1b2c3d4-0256-...   新建自动化任务 parent=af1451a4 path=wizard component=pages/operations/rpa/wizard/index
 *     a1b2c3d4-0258-...   定时调度     parent=af1451a4 path=schedules
 *
 *   ⇒ 真实路由是 `/vdi/rpa`（不是 `/operations/rpa`）。写错路径时 SPA 落到
 *   catch-all → dashboard，用户「点返回任务列表」直接跳仪表盘。
 *
 * 纪律：**本文件之外不允许再出现 `/vdi/rpa...` 或 `/operations/rpa...` 字符串**
 * （测试 `routes.test.ts` 会扫源码拦截回归）。
 */

/** RPA 管理父级菜单的 path（= 任务列表/执行记录/Worker 监控三个 Tab 的宿主页） */
const RPA_BASE = "/vdi/rpa";

/** 左侧菜单里已播种的 RPA 子页（拼在 RPA_BASE 后） */
export const RPA_ROUTES = {
  /** RPA 管理首页：任务管理 / 执行记录 / Worker 监控 三个 Tab */
  taskList: RPA_BASE,
  /** 实时画面（?executionId= 定位） */
  live: `${RPA_BASE}/live`,
  /** 新建自动化任务向导 */
  wizard: `${RPA_BASE}/wizard`,
  /** 定时调度列表 */
  schedules: `${RPA_BASE}/schedules`,
} as const;

/**
 * 把逻辑路由拼到应用 base（Vite `import.meta.env.BASE_URL`）。
 *
 * 项目可能以子路径部署（`/xingran/`）；整页跳转（window.location）必须带
 * base，而 react-router 的 navigate 走 basename 自动补，**不要**再加 base。
 */
export function withBase(path: string): string {
  const base = import.meta.env.BASE_URL;
  const prefix = base && base !== "/" ? base.replace(/\/$/, "") : "";
  return `${prefix}${path}`;
}

/** 实时画面整页跳转 URL（带 ?executionId=） */
export function rpaLiveUrl(executionId: string): string {
  return `${withBase(RPA_ROUTES.live)}?executionId=${encodeURIComponent(executionId)}`;
}
