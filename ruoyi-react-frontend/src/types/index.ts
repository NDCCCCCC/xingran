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
  roles: string[];   // 角色名称数组
  roleIds?: string[]; // 角色ID数组
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
  parentId?: string | null;
  orderNum: number;
  path?: string | null;
  component?: string | null;
  menuType: 'M' | 'C' | 'F'; // M:目录 C:菜单 F:按钮
  visible: 0 | 1; // 0:隐藏 1:显示
  status: 0 | 1; // 0:正常 1:停用
  perms?: string | null;
  icon?: string | null;
  remark?: string;
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
  leaderName?: string;
  leaderUsername?: string;
  phone?: string;
  email?: string;
  status: 0 | 1;  // 0:正常 1:停用
  children?: Department[];
  createdAt: string;
  updatedAt: string;
  remark?: string;
  accessible?: boolean; // 用户是否真正有权限访问该部门
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

// 字典类型
export interface DictType {
  id: string;
  dictName: string;
  dictType: string;
  status: 0 | 1;  // 0:启用 1:禁用
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

// 字典数据
export interface DictData {
  id: string;
  dictSort: number;
  dictLabel: string;
  dictValue: string;
  dictType: string;
  cssClass?: string;
  listClass?: string;
  isDefault: boolean;
  status: 0 | 1;  // 0:启用 1:禁用
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

// 参数配置
export interface Config {
  id: string;
  configName: string;
  configKey: string;
  configValue: string;
  configType: 'Y' | 'N';  // Y:系统内置 N:非系统内置
  isSystem: 0 | 1;  // 0:否 1:是
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

// 通知公告
export interface Notice {
  id: string;
  noticeTitle: string;
  noticeType: '1' | '2';  // 1:公告 2:警告
  noticeContent?: string;
  status: 0 | 1;  // 0:正常 1:关闭
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

// 操作日志
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
  status: 0 | 1;  // 0:成功 1:失败
  errorMsg?: string;
  operTime: string;
  costTime: number;
}

// 登录日志
export interface LoginLog {
  id: string;
  username?: string;
  ipaddr?: string;
  loginLocation?: string;
  browser?: string;
  os?: string;
  status: 0 | 1;  // 0:成功 1:失败
  msg?: string;
  loginTime: string;
}

// 工位类型
export type WorkstationType = 0 | 1 | 2;  // 0:固定工位 1:灵活工位 2:管理工位
export type WorkstationStatus = 0 | 1;  // 0:启用 1:禁用

// 工位
export interface Workstation {
  id: string;
  workstationCode: string;
  workstationName: string;
  deptId?: string;
  deptName?: string;
  location?: string;
  floor?: string;
  workstationType: WorkstationType;
  status: WorkstationStatus;
  capacity: number;
  description?: string;
  userId?: string;
  userName?: string;
  createdAt: string;
  updatedAt: string;
}