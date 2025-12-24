import { post } from './api';
import type { Menu, BaseResponse } from '@/types';

/**
 * 获取当前用户的菜单树
 */
export async function getUserMenus(): Promise<Menu[]> {
  const response = await post<BaseResponse<Menu[]>>('/system/menus/user-menus', {});
  return response.data || [];
}

/**
 * 获取用户权限列表
 */
export async function getUserPermissions(): Promise<string[]> {
  const response = await post<BaseResponse<string[]>>('/system/menus/user-permissions', {});
  return response.data || [];
}
