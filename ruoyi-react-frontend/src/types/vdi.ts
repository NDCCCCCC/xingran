/**
 * VDI (Virtual Desktop Infrastructure) 类型定义
 * 深信服 VDI 集成相关类型
 */

// ==================== 虚拟机类型 ====================

/**
 * 虚拟机信息
 */
export interface VirtualMachine {
  id: string;
  vm_id: string;
  name: string;
  resource_id: string;
  status: number;
  power_state: 'running' | 'stopped' | 'suspended';
  ip_address?: string;
  mac_address?: string;
  os_type?: string;
  cpu?: number;
  memory?: number;
  disk?: number;
  bound_user_id?: string;
  bound_user_name?: string;
  policy_group_id?: string;
  last_sync_at?: string;
  vdi_server_id: string;
  created_at: string;
  updated_at: string;
}

/**
 * 虚拟机列表查询参数
 */
export interface VMListParams {
  current: number;
  pageSize: number;
  name?: string;
  status?: number;
  vdiServerId?: string;
  resourceId?: string;
  powerState?: 'running' | 'stopped' | 'suspended';
}

/**
 * 创建虚拟机请求
 */
export interface CreateVMRequest {
  name: string;
  resource_id: string;
  vdi_server_id: string;
  cpu?: number;
  memory?: number;
  disk?: number;
}

/**
 * 更新虚拟机请求
 */
export interface UpdateVMRequest {
  name?: string;
  status?: number;
  ip_address?: string;
  mac_address?: string;
}

/**
 * 虚拟机操作请求
 */
export interface VMOperateRequest {
  vm_ids: string[];
  action: 'start' | 'stop' | 'restart' | 'suspend';
}

/**
 * IP配置请求
 */
export interface VMIPConfigRequest {
  vm_id: string;
  ip_address: string;
  netmask: string;
  gateway: string;
}

/**
 * 重命名请求
 */
export interface RenameVMRequest {
  new_name: string;
}

/**
 * 绑定用户请求
 */
export interface BindUserRequest {
  user_id: string;
}

/**
 * 分页响应
 */
export interface VMPageResponse {
  list: VirtualMachine[];
  total: number;
  current: number;
  pageSize: number;
}

// ==================== VDI 服务器类型 ====================

/**
 * VDI 服务器配置
 */
export interface VDIServer {
  id: string;
  name: string;
  endpoint: string;
  username: string;
  tenant_id: number;
  status: number;
  token_expiry?: string;
  created_at: string;
  updated_at: string;
}

/**
 * VDI 服务器配置请求
 */
export interface VDIServerConfig {
  name: string;
  endpoint: string;
  username: string;
  password: string;
  tenant_id: number;
  status: number;
}

// ==================== VM 账号类型 ====================

/**
 * VM 账号信息
 */
export interface VMAccount {
  id: string;
  vm_id: string;
  account_id: string;
  username: string;
  account_type: string;
  os_type: 'Windows' | 'Linux';
  is_admin: boolean;
  is_enabled: boolean;
  sync_status: 'synced' | 'pending' | 'failed';
  created_at: string;
  updated_at: string;
}

/**
 * 创建账号请求
 */
export interface CreateAccountRequest {
  username: string;
  password: string;
  os_type: 'Windows' | 'Linux';
  is_admin?: boolean;
}

/**
 * 重置密码请求
 */
export interface ResetPasswordRequest {
  new_password: string;
}
