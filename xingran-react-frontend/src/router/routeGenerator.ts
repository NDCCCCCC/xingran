/**
 * 路由生成器
 * 将后端菜单数据转换为前端路由配置
 * 包含安全验证：防止路径遍历攻击、XSS 注入
 */

import type { Menu } from "@/types";
import type { MenuRouteConfig, RouteMeta } from "@/types/menu";

const DANGEROUS_PATH_PATTERNS = [/\.\./, /\\/, /\.html$/i, /\.js$/i];

const XSS_PATTERNS = [/<script/i, /javascript:/i, /onerror=/i, /onload=/i, /onclick=/i];

const ALLOWED_COMPONENT_PREFIXES = ["pages/", "components/"];

/**
 * 路由生成器
 * 安全地将菜单数据转换为路由配置
 */
export class RouteGenerator {
  static generate(menus: Menu[]): MenuRouteConfig[] {
    if (!menus || menus.length === 0) {
      return [];
    }

    const routes: MenuRouteConfig[] = [];

    const processMenu = (menu: Menu, parentPath = "") => {
      if (!this.validateMenu(menu)) {
        return;
      }

      const fullPath = this.resolvePath(menu.path || "", parentPath);

      if (menu.menuType === "C") {
        const componentPath = this.resolveComponent(menu.component || "", fullPath);

        const route: MenuRouteConfig = {
          path: fullPath,
          component: componentPath,
          meta: this.buildRouteMeta(menu),
          children: [],
        };

        routes.push(route);
      }

      if (menu.children && menu.children.length > 0) {
        for (const child of menu.children) {
          processMenu(child, fullPath);
        }
      }
    };

    for (const menu of menus) {
      processMenu(menu);
    }

    return routes;
  }

  private static validateMenu(menu: Menu): boolean {
    if (!menu.id || !menu.menuName) {
      console.warn("[RouteGenerator] Invalid menu: missing id or menuName", menu);
      return false;
    }

    if (menu.path) {
      for (const pattern of DANGEROUS_PATH_PATTERNS) {
        if (pattern.test(menu.path)) {
          console.warn("[RouteGenerator] Invalid path: contains dangerous characters", {
            path: menu.path,
            pattern,
          });
          return false;
        }
      }
    }

    if (menu.meta && typeof menu.meta === "object") {
      const meta = menu.meta as RouteMeta;
      if (meta.title) {
        for (const pattern of XSS_PATTERNS) {
          if (pattern.test(meta.title)) {
            console.warn("[RouteGenerator] Invalid meta.title: potential XSS", {
              title: meta.title,
            });
            return false;
          }
        }
      }
    }

    return true;
  }

  private static buildRouteMeta(menu: Menu): RouteMeta {
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

  private static resolvePath(menuPath: string, parentPath: string): string {
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

  private static resolveComponent(component: string, path: string): string {
    if (path === "dashboard") {
      return "pages/dashboard-system/index";
    }

    if (!component) {
      return `pages/${path}/index`;
    }

    let normalizedComponent = component;

    if (normalizedComponent.startsWith("/")) {
      normalizedComponent = normalizedComponent.slice(1);
    }

    if (normalizedComponent.startsWith("pages/")) {
      normalizedComponent = normalizedComponent.slice(6);
    }

    if (
      !normalizedComponent.endsWith("/index") &&
      !normalizedComponent.includes("/index.") &&
      !normalizedComponent.match(/\/index\.\w+$/)
    ) {
      if (!normalizedComponent.match(/\.\w+$/)) {
        normalizedComponent = `${normalizedComponent}/index`;
      }
    }

    if (
      !normalizedComponent.startsWith("pages/") &&
      !normalizedComponent.startsWith("components/")
    ) {
      normalizedComponent = `pages/${normalizedComponent}`;
    }

    return normalizedComponent;
  }

  static getComponentPattern(): string {
    return "/src/pages/**/{index,detail}.tsx";
  }
}
