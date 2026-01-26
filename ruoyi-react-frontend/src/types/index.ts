// 通用响应类型
export interface BaseResponse<T = any> {
  code: number;
  message: string;
  data?: T;
  timestamp: number;
  request_id: string;
}

export interface PageResponse<T> {
  list: T[];
  total: number;
  current: number;
  pageSize: number;
}

// 用户相关类型
export interface User {
  id: string;
  username: string;
  nickname?: string;
  employeeNo?: string;
  email?: string;
  phone?: string;
  avatar?: string;
  gender: 0 | 1 | 2;
  status: 0 | 1;
  deptId?: string;
  deptName?: string;
  roles: string[];
  roleIds?: string[];
  permissions: string[];
  isAdmin?: boolean; // 是否为管理员
  dataScope?: string; // 数据权限范围
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
  dataScope: 1 | 2 | 3 | 4 | 5;
  menuCheckStrictly: boolean;
  deptCheckStrictly: boolean;
  status: 0 | 1;
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
  menuType: 'M' | 'C' | 'F';
  visible: 0 | 1;
  status: 0 | 1;
  perms?: string | null;
  icon?: string | null;
  remark?: string;
  meta?: import('@/types/menu').RouteMeta | null; // 菜单元数据（JSONB）
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
  isExternalOrg?: 0 | 1;
  status: 0 | 1;
  children?: Department[];
  createdAt: string;
  updatedAt: string;
  remark?: string;
  accessible?: boolean;
}

// 登录相关类型
export interface LoginRequest {
  username: string;
  password: string;
  encryptedPassword?: boolean;
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
  status: 0 | 1;
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

// 岗位
export interface Post {
  id: string;
  postCode: string;
  postName: string;
  postSort: number;
  status: 0 | 1;  // 0:正常 1:停用
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
  configType: 'Y' | 'N';
  isSystem: 0 | 1;
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
  details?: ConfigExecutionDetail[];
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
  ipAddress: string;
  backupType: BackupType;
  configHash: string;
  version: number;
  changeReason?: string;
  backupSize: number;
  filePath: string;
  createdAt: string;
  updatedAt: string;
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

// ==================== 个人信息相关类型 ====================

// 用户详细信息
export interface UserProfile extends User {
  remark?: string;
  pwdUpdateTime?: string;
}

// 个人信息更新请求
export interface UpdateProfileRequest {
  nickname?: string;
  email?: string;
  phone?: string;
  gender: 0 | 1 | 2;
  remark?: string;
}

// 修改密码请求
export interface ChangePasswordRequest {
  oldPassword: string;
  newPassword: string;
}

// ==================== 系统设置相关类型 ====================

// 用户个人设置
export interface UserPreferences {
  theme: 'light' | 'dark';
  language: 'zh-CN' | 'en-US';
  pageSize: number;
  sidebarCollapsed: boolean;
}

// ==================== 运维管理相关类型 ====================

// 楼宇状态
export type BuildingStatus = 0 | 1;  // 0:正常 1:停用

// 楼宇
export interface Building {
  id: string;
  name: string;
  code: string;
  address?: string;
  totalFloors?: number;
  totalArea?: number;
  description?: string;
  status: BuildingStatus;
  createdAt: string;
  updatedAt: string;
}

// 楼层状态
export type FloorStatus = 0 | 1;  // 0:正常 1:停用

// 楼层
export interface Floor {
  id: string;
  buildingCode: string;
  code: string;
  buildingName?: string;
  floorNo: string;
  name?: string;
  area?: number;
  description?: string;
  status: FloorStatus;
  createdAt: string;
  updatedAt: string;
}

// 工位类型
export type WorkstationOpsType = 0 | 1 | 2;  // 0:固定工位 1:灵活工位 2:管理工位

// 工位状态
export type WorkstationOpsStatus = 0 | 1 | 2;  // 0:空闲可分配 1:占用已分配 2:维护中不可用

// 工位（运维管理）
export interface WorkstationOps {
  id: string;
  floorId: string;
  floorName?: string;
  name: string;
  type: WorkstationOpsType;
  status: WorkstationOpsStatus;
  deptId?: string;        // 所属部门ID
  deptName?: string;      // 所属部门名称
  userId?: string;        // 所属用户ID
  userName?: string;      // 所属用户名称
  positionX?: number;
  positionY?: number;
  rotation?: number;      // 平面图旋转角度（度），0-360，默认0
  description?: string;
  createdAt: string;
  updatedAt: string;
}

// 机房状态
export type RoomStatus = 0 | 1;  // 0:正常 1:停用

// 机房
export interface ServerRoom {
  id: string;
  buildingId: string;     // 所属楼宇ID
  buildingName?: string;
  floorId: string;        // 所属楼层ID
  floorName?: string;
  floorNo?: string;       // 楼层号（显示用）
  name: string;
  area?: number;
  rackCount?: number;
  powerCapacity?: number;
  remark?: string;
  status: RoomStatus;
  createdAt: string;
  updatedAt: string;
}

// 机房照片
export interface RoomPhoto {
  id: string;
  roomId: string;
  fileId: string;
  fileName?: string;
  fileUrl: string;
  sortOrder: number;
  isPrimary: boolean;
  description?: string;
  createdAt: string;
}

// 设备类型
export type RoomDeviceType = 'server' | 'storage' | 'ups' | 'pdu' | 'firewall' | 'kvm' | 'cabinet' | 'other';

// 机房设备状态
export type RoomDeviceStatus = 0 | 1 | 2;  // 0:正常 1:故障 2:报废

// 机房设备
export interface RoomDevice {
  id: string;
  roomCode: string;
  roomName?: string;
  deviceCode: string;
  deviceName: string;
  deviceType: RoomDeviceType;
  vendor?: string;
  model?: string;
  serialNumber?: string;
  positionDesc?: string;
  positionU?: number;
  heightU?: number;
  powerConsumption?: number;
  purchaseDate?: string;
  warrantyDate?: string;
  status: RoomDeviceStatus;
  description?: string;
  createdAt: string;
  updatedAt: string;
}

// 专线状态
export type LineStatus = 0 | 1 | 2;  // 0:正常 1:故障 2:停用

// 专线
export interface DedicatedLine {
  id: string;
  name: string;                // 专线名称
  lineType: string;            // 专线类型（数据字典）
  bandwidth?: string;
  isp: string;
  sourceDeviceId?: string;
  sourceDeviceName?: string;
  sourcePort?: string;
  sourceRoomId?: string;       // 源机房ID
  sourceRoomName?: string;     // 源机房名称
  destDeviceId?: string;
  destDeviceName?: string;
  destPort?: string;
  destRoomId?: string;         // 目的机房ID
  destRoomName?: string;       // 目的机房名称
  ipAddress?: string;
  subnetMask?: string;
  vlan?: string;
  carrierContactName?: string;
  carrierContactPhone?: string;
  monthlyFee?: number;
  status: LineStatus;
  remark?: string;             // 备注（与后端字段保持一致）
  createdAt: string;
  updatedAt: string;
}

// 信息点类型（使用字典 ops_info_point_type）
export type InfoPointType = string;

// 信息点状态
export type InfoPointStatus = 0 | 1 | 2;  // 0:正常 1:故障 2:停用

// 信息点
export interface InfoPoint {
  id: string;
  workstationId: string;
  workstationName?: string;
  name: string;
  infoPointType: InfoPointType;
  deviceId?: string;
  deviceName?: string;
  portId?: string;
  portName?: string;
  status: InfoPointStatus;
  description?: string;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

// ==================== 值班管理相关类型 ====================

// 我的值班统计
export interface MyDutyStats {
  isOnDutyToday: boolean;          // 今日是否值班
  thisMonthCount: number;          // 本月值班次数
  totalCount: number;              // 累计值班次数
  nextDutyDate?: string;           // 下次值班日期
  nextDutyPoolName?: string;       // 下次值班班组名称
  todayDutyRecords?: TodayDutyRecord[]; // 今日值班记录
}

// 今日值班记录
export interface TodayDutyRecord {
  poolName: string;                // 班组名称
  dutyType: 'weekday' | 'weekend' | 'holiday'; // 值班类型
}