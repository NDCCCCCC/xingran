/**
 * 系统管理相关类型
 */

import type { Status, PageParams } from "./base";

// ==================== 用户相关 ====================

/**
 * 用户类型
 */
export interface User {
  id: string;
  username: string;
  nickname?: string;
  employeeNo?: string;
  email?: string;
  phone?: string;
  avatar?: string;
  gender: 0 | 1 | 2;
  status: Status;
  deptId?: string;
  deptName?: string;
  deptFullName?: string; // 完整部门路径（从二级开始）
  roles: string[];
  roleIds?: string[];
  permissions: string[];
  isAdmin?: boolean;
  dataScope?: string;
  loginIp?: string;
  loginTime?: string;
  createTime: string;
  updateTime: string;
}

/**
 * 用户列表查询参数
 */
export interface UserListParams extends PageParams {
  username?: string;
  nickname?: string;
  deptId?: string;
  status?: number;
  dateRange?: [string, string];
}

// ==================== 角色相关 ====================

/**
 * 数据权限范围
 */
export type DataScope = 1 | 2 | 3 | 4 | 5;

/**
 * 角色类型
 */
export interface Role {
  id: string;
  roleName: string;
  roleKey: string;
  roleSort: number;
  dataScope: DataScope;
  menuCheckStrictly: boolean;
  deptCheckStrictly: boolean;
  status: Status;
  menuIds: string[];
  deptIds?: string[];
  createTime: string;
  updateTime: string;
  remark?: string;
}

// ==================== 菜单相关 ====================

/**
 * 菜单类型：M=目录，C=菜单，F=按钮
 */
export type MenuType = "M" | "C" | "F";

/**
 * 菜单类型
 */
export interface Menu {
  id: string;
  menuName: string;
  parentId?: string | null;
  orderNum: number;
  path?: string | null;
  component?: string | null;
  menuType: MenuType;
  visible: Status;
  status: Status;
  perms?: string | null;
  icon?: string | null;
  remark?: string;
  meta?: import("./menu").RouteMeta | null;
  children?: Menu[];
  createTime: string;
  updateTime: string;
}

// ==================== 部门相关 ====================

/**
 * 部门类型
 */
export interface Department {
  id: string;
  deptName: string;
  deptCode: string;
  parentId?: string;
  ancestors: string;
  orderNum: number;
  leader?: string;
  leaderName?: string;
  leaderUsername?: string;
  phone?: string;
  email?: string;
  isExternalOrg?: Status;
  status: Status;
  children?: Department[];
  createdAt: string;
  updatedAt: string;
  remark?: string;
  accessible?: boolean;
}

// ==================== 岗位相关 ====================

/**
 * 岗位类型
 */
export interface Post {
  id: string;
  postCode: string;
  postName: string;
  postSort: number;
  status: Status;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

// ==================== 字典相关 ====================

/**
 * 字典类型
 */
export interface DictType {
  id: string;
  dictName: string;
  dictType: string;
  status: Status;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * 字典数据
 */
export interface DictData {
  id: string;
  dictSort: number;
  dictLabel: string;
  dictValue: string;
  dictType: string;
  cssClass?: string;
  listClass?: string;
  isDefault: boolean;
  status: Status;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

// ==================== 配置相关 ====================

/**
 * 配置类型：Y=是，N=否
 */
export type ConfigType = "Y" | "N";

/**
 * 参数配置
 */
export interface Config {
  id: string;
  configName: string;
  configKey: string;
  configValue: string;
  configType: ConfigType;
  isSystem: Status;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

// ==================== 通知公告 ====================

/**
 * 通知类型：1=公告，2=警告
 */
export type NoticeType = "1" | "2";

/**
 * 通知公告
 */
export interface Notice {
  id: string;
  noticeTitle: string;
  noticeType: NoticeType;
  noticeContent?: string;
  status: Status;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

// ==================== 日志相关 ====================

/**
 * 操作日志
 */
export interface OperLog {
  id: string;
  title?: string;
  businessType: number;
  method?: string;
  requestMethod?: string;
  operatorType: number;
  operatorName?: string;
  deptName?: string;
  operUrl?: string;
  operIp?: string;
  operLocation?: string;
  operParam?: string;
  jsonResult?: string;
  status: Status;
  errorMsg?: string;
  operTime: string;
  costTime: number;
}

/**
 * 登录日志
 */
export interface LoginLog {
  id: string;
  username?: string;
  ipaddr?: string;
  loginLocation?: string;
  browser?: string;
  os?: string;
  status: Status;
  msg?: string;
  loginTime: string;
}
