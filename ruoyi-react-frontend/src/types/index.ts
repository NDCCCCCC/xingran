// 通用响应类型
export interface BaseResponse<T = any> {
  code: number;
  message: string;
  data?: T;
  timestamp: number;
  request_id: string;
}

// 分页响应类型
export interface PageResponse<T> {
  list: T[];
  total: number;
  current: number;
  page_size: number;
}

// 用户相关类型
export interface User {
  id: string;
  username: string;
  nickname?: string;
  email?: string;
  phone?: string;
  avatar?: string;
  gender: 0 | 1 | 2;  // 0:未知 1:男 2:女
  status: 0 | 1;     // 0:启用 1:禁用
  deptId?: string;
  deptName?: string;
  roles: string[];
  permissions: string[];
  loginIp?: string;
  loginTime?: string;
  createTime: string;
  updateTime: string;
}

// 角色相关类型
export interface Role {
  id: string;
  roleName: string;
  roleKey: string;
  roleSort: number;
  dataScope: 1 | 2 | 3 | 4 | 5; // 数据范围：1:全部 2:部门 3:部门及子部门 4:个人 5:自定义
  menuCheckStrictly: boolean;
  deptCheckStrictly: boolean;
  status: 0 | 1; // 0:正常 1:停用
  menuIds: string[];
  deptIds?: string[];
  createTime: string;
  updateTime: string;
  remark?: string;
}

// 菜单相关类型
export interface Menu {
  id: string;
  menuName: string;
  parentId: string;
  orderNum: number;
  path?: string;
  component?: string;
  menuType: 'M' | 'C' | 'F'; // 目录/菜单/按钮
  visible: 0 | 1;
  status: 0 | 1; // 0:显示 1:隐藏
  perms?: string;
  icon?: string;
  children?: Menu[];
  createTime: string;
  updateTime: string;
}

// 部门相关类型
export interface Department {
  id: string;
  deptName: string;
  parentId?: string;
  ancestors: string;
  orderNum: number;
  leader?: string;
  phone?: string;
  email?: string;
  status: 0 | 1;  // 0:正常 1:停用
  children?: Department[];
  createdAt: string;
  updatedAt: string;
  remark?: string;
}

// 登录相关类型
export interface LoginRequest {
  username: string;
  password: string;
  captcha?: string;
  captchaId?: string;
}

export interface LoginResponse {
  user: User;
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  tokenType: string;
}

// 查询参数类型
export interface PageParams {
  current?: number;
  pageSize?: number;
}

export interface UserListParams extends PageParams {
  username?: string;
  nickname?: string;
  deptId?: string;
  status?: number;
  dateRange?: [string, string];
}