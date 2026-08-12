/**
 * 路由配置管理器
 * 统一管理所有路由配置和 meta 信息
 */

import type { Menu } from "@/types";
import type {
  MenuRouteConfig,
  RouteMeta,
  BreadcrumbItem,
  RoutePermissionCheck,
} from "@/types/menu";

interface MenuPathInfo {
  route: MenuRouteConfig;
  fullPath: string;
  level: number;
  parentPath?: string;
}

// 路径翻译映射表
const PATH_TRANSLATIONS: Record<string, string> = {
  "ops": "运维管理",
  "operations": "运维",
  "workorder": "工单",
  "orders": "工单管理",
  "categories": "工单分类",
  "statistics": "工单统计",
  "periodic": "周期性工单",
  "templates": "模板",
  "knowledge": "知识库",
  "articles": "知识库文章",
  "view": "查看",
  "system": "系统管理",
  "user": "用户",
  "users": "用户管理",
  "role": "角色",
  "roles": "角色管理",
  "menu": "菜单",
  "menus": "菜单管理",
  "dept": "部门",
  "depts": "部门管理",
  "buildings": "楼宇管理",
  "floors": "楼层管理",
  "workstations": "工位管理",
  "server-rooms": "机房管理",
  "dedicated-lines": "专线管理",
  "room-devices": "机房设备",
  "building-spaces": "楼宇空间",
  "building-spaces-3d": "楼宇空间3D",
  "info-points": "信息点管理",
  "network": "网络",
  "devices": "设备",
  "ports": "端口",
  "heatmap": "热力图",
  "monitor": "监控",
  "dashboard": "仪表盘",
  "settings": "设置",
  "profile": "个人中心",
  "duty": "值班",
}

export class RouteConfigManager {
  private routeMap = new Map<string, MenuPathInfo>();
  private initialized = false;

  initialize(menus: Menu[]): void {
    this.routeMap = this.buildRouteMap(menus);
    this.initialized = true;
  }

  isInitialized(): boolean {
    return this.initialized;
  }

  getRouteByPath(path: string): MenuRouteConfig | undefined {
    const normalizedPath = this.normalizePath(path);
    return this.routeMap.get(normalizedPath)?.route;
  }

  getRouteMeta(path: string): RouteMeta | undefined {
    return this.getRouteByPath(path)?.meta;
  }

  getRouteTitle(path: string): string {
    const meta = this.getRouteMeta(path);
    return meta?.title || this.extractTitleFromPath(path);
  }

  hasPermission(path: string, userPermissions: string[]): RoutePermissionCheck {
    const meta = this.getRouteMeta(path);

    if (!meta?.permissions || meta.permissions.length === 0) {
      return { hasPermission: true };
    }

    const hasPermission = meta.permissions.some(p => userPermissions.includes(p));

    if (hasPermission) {
      return { hasPermission: true };
    }

    const missingPermissions = meta.permissions.filter(p => !userPermissions.includes(p));
    return {
      hasPermission: false,
      missingPermissions,
    };
  }

  buildBreadcrumb(path: string): BreadcrumbItem[] {
    const normalizedPath = this.normalizePath(path);
    const pathInfo = this.routeMap.get(normalizedPath);

    if (!pathInfo) {
      return this.fallbackBreadcrumb(path);
    }

    const breadcrumb: BreadcrumbItem[] = [];
    const segments = normalizedPath.split("/").filter(Boolean);

    let currentPath = "";
    for (const segment of segments) {
      currentPath += (currentPath ? "/" : "") + segment;
      const currentInfo = this.routeMap.get(currentPath);

      if (currentInfo?.route.meta.title) {
        breadcrumb.push({
          path: `/${currentPath}`,
          title: currentInfo.route.meta.title,
        });
      }
    }

    return breadcrumb.length > 0 ? breadcrumb : this.fallbackBreadcrumb(path);
  }

  getAllRoutes(): MenuRouteConfig[] {
    return Array.from(this.routeMap.values()).map(info => info.route);
  }

  clear(): void {
    this.routeMap.clear();
    this.initialized = false;
  }

  // 私有方法

  private buildRouteMap(menus: Menu[]): Map<string, MenuPathInfo> {
    const map = new Map<string, MenuPathInfo>();

    const traverse = (
      menuList: Menu[],
      parentPath: string = "",
      level: number = 1
    ) => {
      for (const menu of menuList) {
        if (menu.menuType === "F") continue;

        const menuPath = menu.path || "";
        const fullPath = this.resolvePath(menuPath, parentPath);

        const route: MenuRouteConfig = {
          path: fullPath,
          component: menu.component || "",
          meta: this.buildRouteMeta(menu),
        };

        if (fullPath) {
          const normalized = this.normalizePath(fullPath);
          const pathInfo: MenuPathInfo = {
            route,
            fullPath,
            level,
            parentPath: parentPath || undefined,
          };

          map.set(normalized, pathInfo);
          map.set(`/${normalized}`, pathInfo);
        }

        if ((menu.children?.length ?? 0) > 0) {
          traverse(menu.children ?? [], fullPath, level + 1);
        }
      }
    };

    traverse(menus);
    return map;
  }

  private buildRouteMeta(menu: Menu): RouteMeta {
    if (menu.meta && typeof menu.meta === "object") {
      return menu.meta as RouteMeta;
    }

    return {
      title: menu.menuName || "",
      icon: menu.icon || undefined,
      hidden: menu.visible !== 1,
      keepAlive: false,
      affix: menu.path === "dashboard",
    };
  }

  private resolvePath(menuPath: string, parentPath: string): string {
    if (!menuPath) {
      return parentPath || "";
    }

    if (menuPath.startsWith("/")) {
      return menuPath.slice(1);
    }

    if (!parentPath) {
      return menuPath;
    }

    return `${parentPath}/${menuPath}`;
  }

  private normalizePath(path: string): string {
    return path.startsWith("/") ? path.slice(1) : path;
  }

  private extractTitleFromPath(path: string): string {
    const segments = path.split("/").filter(Boolean);
    return segments[segments.length - 1] || "页面";
  }

  private fallbackBreadcrumb(path: string): BreadcrumbItem[] {
    const segments = path.split("/").filter(Boolean)

    if (segments.length === 0) {
      return [{ path: "/", title: "首页" }]
    }

    const breadcrumb: BreadcrumbItem[] = []
    let currentPath = ""

    for (const segment of segments) {
      currentPath += `/${segment}`
      breadcrumb.push({
        path: currentPath,
        title: PATH_TRANSLATIONS[segment] || segment,
      })
    }

    return breadcrumb
  }
}

export const routeConfigManager = new RouteConfigManager();
