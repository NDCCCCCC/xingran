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

// ==================== 网络设备管理类型 ====================

// 设备厂商
export type DeviceVendor = 'huawei' | 'h3c' | 'ruijie' | 'maipu';

// 设备类型
export type DeviceType = 'router' | 'switch' | 'firewall' | 'ap' | 'loadbalancer';

// 设备状态
export type DeviceStatus = 0 | 1 | 2;  // 0:在线 1:离线 2:未知

// 协议类型（SSH 或 Telnet）
export type ProtocolType = 'ssh' | 'telnet';

// 授权凭证
export interface AuthCredential {
  id: string;
  credentialName: string;
  protocolType: ProtocolType;        // SSH 或 Telnet
  username?: string;
  password?: string;                 // 加密存储，前端不显示
  enablePassword?: string;
  snmpCommunities: string[];         // 多个 SNMP Community
  snmpVersion: 'v1' | 'v2c' | 'v3';
  description?: string;
  isDefault: boolean;
  createdAt: string;
  updatedAt: string;
}

// 网络设备
export interface NetworkDevice {
  id: string;
  deviceName: string;
  deviceType: DeviceType;
  vendor: DeviceVendor;
  model?: string;
  serialNumber?: string;     // 设备序列号（SSH采集）
  softwareVersion?: string;  // 软件版本（SSH采集）
  uptime?: string;           // 设备运行时间（SSH采集）
  ipAddress: string;
  port: number;
  snmpPort: number;
  credentialId?: string;
  credentialName?: string;
  deptId?: string;
  deptName?: string;
  location?: string;
  status: DeviceStatus;
  lastSeenAt?: string;
  lastConfigAt?: string;
  description?: string;
  createdAt: string;
  updatedAt: string;
}

// MAC地址类型
export type MACType = 'dynamic' | 'static' | 'secure';

// 设备MAC地址
export interface DeviceMACAddress {
  id: string;
  deviceId: string;
  deviceName?: string;
  macAddress: string;
  interfaceName: string;
  vlanId?: number;
  macType: MACType;
  collectedAt: string;
  createdAt: string;
}

// 802.1X端口状态
export type Dot1xPortStatus = 'authorized' | 'unauthorized' | 'unknown';

// 端口安全模式
export type PortSecurityMode = '' | 'protect' | 'restrict' | 'shutdown';

// 设备端口状态
export interface DevicePortStatus {
  id: string;
  deviceId: string;
  deviceName?: string;
  interfaceName: string;
  adminStatus: string;
  operStatus: string;
  description?: string;          // 端口描述

  // 物理属性
  vlan?: number;                 // VLAN ID
  duplex?: string;               // 双工模式
  speed?: string;                // 速率
  portType?: string;             // 端口类型

  dot1xEnabled: boolean;
  dot1xPortStatus: Dot1xPortStatus;
  portSecurityEnabled: boolean;
  portSecurityMode: PortSecurityMode;
  maxMACCount?: number;
  currentMACCount?: number;
  collectedAt: string;
  createdAt: string;
}

// 配置模板类型
export type TemplateType = 'init' | 'config' | 'backup';

// 配置模板
export interface ConfigTemplate {
  id: string;
  templateName: string;
  templateCode: string;
  templateType: TemplateType;
  vendor?: DeviceVendor;
  deviceType?: DeviceType;
  templateContent: string;
  variables?: Record<string, any>;
  description?: string;
  isSystem: boolean;
  version: number;
  createdAt: string;
  updatedAt: string;
}

// 配置执行状态
export type ExecutionStatus = 'pending' | 'running' | 'success' | 'failed';

// 配置执行类型
export type ExecutionType = 'template' | 'command';

// 配置执行记录
export interface ConfigExecution {
  id: string;
  executionName: string;
  executionType: ExecutionType;
  templateId?: string;
  templateName?: string;
  deviceIds: string[];
  status: ExecutionStatus;
  totalDevices: number;
  successCount: number;
  failureCount: number;
  commandContent?: string;
  executionStrategy: 'parallel' | 'serial';
  concurrency: number;
  timeout: number;
  startedAt?: string;
  completedAt?: string;
  errorMessage?: string;
  createdAt: string;
  updatedAt: string;
}

// 配置执行明细
export interface ConfigExecutionDetail {
  id: string;
  executionId: string;
  deviceId: string;
  deviceName: string;
  ipAddress: string;
  status: ExecutionStatus;
  commandSent?: string;
  outputReceived?: string;
  errorMessage?: string;
  startedAt?: string;
  completedAt?: string;
  duration?: number;
  createdAt: string;
  updatedAt: string;
}

// 配置备份类型
export type BackupType = 'auto' | 'manual';

// 配置备份
export interface ConfigBackup {
  id: string;
  deviceId: string;
  deviceName: string;
  backupType: BackupType;
  configHash: string;
  version: number;
  changeReason?: string;
  backupSize: number;
  filePath: string;
  createdAt: string;
  createdBy?: string;
}

// 设备发现状态
export type DiscoveryStatus = 'pending' | 'running' | 'completed' | 'failed';

// 设备发现
export interface DeviceDiscovery {
  id: string;
  taskName: string;
  discoveryType: 'snmp' | 'scan';
  ipRanges: string[];
  snmpCommunity?: string;
  snmpPort: number;
  status: DiscoveryStatus;
  totalIPs: number;
  discoveredCount: number;
  autoImport: boolean;
  deptId?: string;
  startedAt?: string;
  completedAt?: string;
  errorMessage?: string;
  createdAt: string;
  updatedAt: string;
}