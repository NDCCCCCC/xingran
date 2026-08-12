import { post } from "./api";
import type { Menu, BaseResponse } from "@/types";

/**
 * 获取当前用户的菜单树（不包含隐藏菜单，用于导航栏渲染）
 */
export async function getUserMenus(): Promise<Menu[]> {
  const response = await post<Menu[]>("/system/my-menus", {});
  return response.data || [];
}

/**
 * 获取用户所有菜单（包含隐藏菜单，用于标签页标题和路由生成）
 */
export async function getAllUserMenus(): Promise<Menu[]> {
  const response = await post<Menu[]>("/system/my-menus/all", {});
  return response.data || [];
}

/**
 * 获取用户权限列表
 */
export async function getUserPermissions(): Promise<string[]> {
  const response = await post<string[]>("/system/my-menus/permissions", {});
  return response.data || [];
}
