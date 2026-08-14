/**
 * 组件动态加载器
 * 包含安全措施：组件路径白名单验证
 *
 * 使用 Vite 的 import.meta.glob 实现组件懒加载
 */

import { lazy, type ComponentType } from "react";

/**
 * 组件加载器
 * 负责动态加载和缓存组件
 */
export class ComponentLoader {
  private cache = new Map<string, ComponentType>();

  // 允许的组件路径前缀（白名单）
  private static readonly ALLOWED_PREFIXES = ["pages/", "components/"];

  // 危险字符模式
  private static readonly DANGEROUS_PATTERNS = [
    /\.\./, // 路径遍历
    /\\/, // 反斜杠
    /\.html$/i, // HTML 文件
    /\.js$/i, // JS 文件
    /\.json$/i, // JSON 文件
  ];

  // 使用 Vite 的 glob 导入所有页面组件。
  // 排除 login 页面：它在 DynamicRoutes.tsx 中静态导入（顶层路由，必须随首屏加载），
  // 不需要走 glob 的懒加载机制，否则 Vite 会报 "dynamically imported but also statically imported"。
  // 同时匹配 index.tsx（列表页）和 detail.tsx（详情子页）—— 详情页路径由 DynamicRoutes.tsx
  // 的静态 Route 显式注册（例如 /system/notice/:id、/my-notices/:id），不走菜单动态生成。
  public static componentModules = import.meta.glob<{ default: ComponentType }>(
    [
      "/src/pages/**/{index,detail}.tsx",
      "!**/login/**",
      // 以下两个详情页在 DynamicRoutes.tsx 中静态导入（无对应 sys_menu 节点，走静态 Route 兜底，
      // 见 DynamicRoutes.tsx 的 /system/notice/:id 与 /my-notices/:id），需从 glob 懒加载排除，
      // 否则 Vite 报 "dynamically imported but also statically imported"（与 login 同理）
      "!/src/pages/system/notice/detail.tsx",
      "!/src/pages/my-notices/detail.tsx",
    ],
    {
      eager: false,
    }
  );

  /**
   * 加载组件
   * @param componentPath 组件路径（如 'pages/system/user/index'）
   * @returns 组件类型
   */
  async load(componentPath: string): Promise<ComponentType> {
    // 标准化路径
    const normalizedPath = this.normalizePath(componentPath);

    // 路径白名单验证
    if (!this.isValidComponentPath(normalizedPath)) {
      console.error(`[ComponentLoader] Invalid component path: ${normalizedPath}`);
      throw new Error(`Invalid component path: ${normalizedPath}`);
    }

    // 检查缓存
    if (this.cache.has(normalizedPath)) {
      return this.cache.get(normalizedPath)!;
    }

    try {
      const component = await this.importComponent(normalizedPath);
      this.cache.set(normalizedPath, component);
      return component;
    } catch (error) {
      console.error(`[ComponentLoader] Failed to load component: ${normalizedPath}`, error);
      throw error;
    }
  }

  /**
   * 验证组件路径是否在白名单内
   */
  private isValidComponentPath(path: string): boolean {
    // 检查是否包含危险字符
    for (const pattern of ComponentLoader.DANGEROUS_PATTERNS) {
      if (pattern.test(path)) {
        console.warn("[ComponentLoader] Path contains dangerous characters", { path });
        return false;
      }
    }

    // 检查白名单
    const isValid = ComponentLoader.ALLOWED_PREFIXES.some((prefix) => path.startsWith(prefix));

    if (!isValid) {
      console.warn("[ComponentLoader] Path not in whitelist", { path });
    }

    return isValid;
  }

  /**
   * 标准化组件路径
   */
  private normalizePath(path: string): string {
    let normalized = path;

    // 去掉前导斜杠
    if (normalized.startsWith("/")) {
      normalized = normalized.slice(1);
    }

    // 确保 .tsx 扩展名
    if (!normalized.endsWith(".tsx")) {
      normalized += ".tsx";
    }

    return normalized;
  }

  /**
   * 导入组件
   */
  private async importComponent(path: string): Promise<ComponentType> {
    // 构建 Vite glob 导入的路径
    // import.meta.glob 返回的路径是相对于项目根目录的
    const vitePath = `/src/${path}`;

    // 查找对应的模块
    const moduleLoader = ComponentLoader.componentModules[vitePath];

    if (!moduleLoader) {
      throw new Error(`Component not found: ${vitePath}`);
    }

    const module = await moduleLoader();
    return module.default;
  }

  /**
   * 获取错误组件
   */
  private getErrorComponent(): ComponentType {
    return () => (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <h2 className="text-xl font-semibold text-red-600 mb-2">页面加载失败</h2>
          <p className="text-gray-600">请检查路由配置或联系管理员</p>
        </div>
      </div>
    );
  }

  /**
   * 清空缓存
   */
  clearCache(): void {
    this.cache.clear();
  }

  /**
   * 获取缓存大小
   */
  getCacheSize(): number {
    return this.cache.size;
  }
}

// 导出单例
export const componentLoader = new ComponentLoader();

/**
 * 创建懒加载组件
 * @param componentPath 组件路径（如 'system/user/index' 或 'pages/system/user/index'）
 *                     约定：数据库中存储不带 'pages/' 前缀的路径
 *                     前端自动添加 'pages/' 前缀
 * @returns 懒加载组件
 */
export function createLazyComponent(componentPath: string): ComponentType {
  // 标准化路径
  let normalizedPath = componentPath;

  // 去掉前导斜杠
  if (normalizedPath.startsWith("/")) {
    normalizedPath = normalizedPath.slice(1);
  }

  // 如果路径已经包含 pages/ 前缀，去掉它（统一处理）
  if (normalizedPath.startsWith("pages/")) {
    normalizedPath = normalizedPath.slice(6);
  }

  // 自动添加 pages/ 前缀
  if (!normalizedPath.startsWith("pages/")) {
    normalizedPath = `pages/${normalizedPath}`;
  }

  // 确保有 .tsx 扩展名
  if (!normalizedPath.endsWith(".tsx")) {
    normalizedPath += ".tsx";
  }

  // 构建完整路径（Vite glob 格式）
  const fullPath = `/src/${normalizedPath}`;

  // 检查模块是否存在
  const moduleLoader = ComponentLoader.componentModules[fullPath];

  if (!moduleLoader) {
    console.error(`[createLazyComponent] Module not found: ${fullPath}`);
    // 返回错误组件（非 lazy）
    return () => (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <h2 className="text-xl font-semibold text-red-600 mb-2">页面加载失败</h2>
          <p className="text-gray-600">组件路径: {componentPath}</p>
          <p className="text-gray-500 text-sm mt-2">未找到模块: {fullPath}</p>
          <p className="text-gray-400 text-xs mt-1">请确保文件路径正确</p>
        </div>
      </div>
    );
  }

  // 直接使用 moduleLoader 创建 lazy 组件
  return lazy(moduleLoader);
}

/**
 * 批量预加载组件
 * @param componentPaths 组件路径数组
 */
export async function preloadComponents(componentPaths: string[]): Promise<void> {
  await Promise.all(componentPaths.map((path) => componentLoader.load(path)));
}
