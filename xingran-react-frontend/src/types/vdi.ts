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
  power_state: "pending" | "stopped" | "suspended" | "in_use";
  ip_address?: string;
  mac_address?: string;
  os_type?: string;
  cpu_number?: number; // CPU颗数
  cpu_core?: number; // 每颗CPU的核数
  cpu_per?: number; // CPU使用率
  memory?: number;
  memory_per?: number; // 内存使用率
  disk?: number;
  disk_per?: number; // 磁盘使用率
  bound_user_id?: string;
  bound_user_name?: string;
  policy_group_id?: string;
  // 网络配置信息
  ip_type?: string; // IP类型：STATIC/DHCP
  subnet_mask?: string; // 子网掩码
  default_gateway?: string; // 默认网关
  name_server?: string; // DNS服务器
  assign_ip?: string; // 分配的IP地址
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
  vdiServerId?: string;
  resourceId?: string;
  powerState?: "pending" | "stopped" | "suspended" | "in_use";
}

/**
 * 创建虚拟机请求
 */
export interface CreateVMRequest {
  name: string;
  resource_id: string;
  resource_group_id?: string; // 资源组 ID
  vdi_server_id: string;
  cpu_number?: number; // CPU颗数
  cpu_core?: number; // 每颗CPU的核数
  memory?: number;
  disk?: number;
  // VDI API specific fields
  vtp_id?: number; // VTP platform ID
  host_id?: string; // Host position ID (father_id)
  run_position_id?: string; // Run position ID (empty if id == father_id)
  disk_id?: string; // Personal disk ID
  storage_id?: string; // Storage location ID
  network_id?: string; // Network interface ID
  count?: number; // Number of VMs to create
}

/**
 * 更新虚拟机请求
 */
export interface UpdateVMRequest {
  name?: string;
  ip_address?: string;
  mac_address?: string;
}

/**
 * 虚拟机操作请求
 */
export interface VMOperateRequest {
  vm_ids: string[];
  action: "start" | "stop" | "restart" | "suspend";
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
  username: string;
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
 * VDI 资源组信息（从本地DB查询）
 */
export interface VDIResourceGroup {
  resource_group_id: string;
  name: string;
  vdi_server_id: string;
  type: string;
}

/**
 * VDI 资源信息（资源组下的具体计算资源）
 */
export interface VDIResource {
  id: number;
  name: string;
  note: string;
  grp_id: number;
}

/**
 * VDI 平台信息
 */
export interface VDIPlatform {
  id: number;
  name: string;
  tenant_id: number;
}

/**
 * VDI 运行位置
 */
export interface RunPosition {
  id: string;
  name: string;
  father_id: string;
}

/**
 * VDI 存储位置
 */
export interface VDIStorage {
  id: string;
  name: string;
  type: string;
  total: string;
  avail: string;
  shared: number;
  status: number;
}

/**
 * VDI 网络接口
 */
export interface VDINetwork {
  id: string;
  name: string;
  mode: string;
}

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
  lastSyncTime?: string;
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
  os_type: "Windows" | "Linux";
  is_admin: boolean;
  is_enabled: boolean;
  sync_status: "synced" | "pending" | "failed";
  created_at: string;
  updated_at: string;
}

/**
 * 创建账号请求
 */
export interface CreateAccountRequest {
  username: string;
  password: string;
  os_type: "Windows" | "Linux";
  is_admin?: boolean;
}

/**
 * 重置密码请求
 */
export interface ResetPasswordRequest {
  new_password: string;
}
