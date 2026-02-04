/**
 * 网络设备管理相关类型
 */

import type { Status } from './base';

// ==================== 设备基础类型 ====================

/**
 * 设备厂商
 */
export type DeviceVendor = 'huawei' | 'h3c' | 'ruijie' | 'maipu';

/**
 * 设备类型
 */
export type DeviceType = 'router' | 'switch' | 'firewall' | 'ap' | 'loadbalancer';

/**
 * 设备状态：0=在线，1=离线，2=未知
 */
export type DeviceStatus = 0 | 1 | 2;

/**
 * 协议类型
 */
export type ProtocolType = 'ssh' | 'telnet';

/**
 * MAC地址类型
 */
export type MACType = 'dynamic' | 'static' | 'secure';

/**
 * 802.1X端口状态
 */
export type Dot1xPortStatus = 'authorized' | 'unauthorized' | 'unknown';

/**
 * 端口安全模式
 */
export type PortSecurityMode = '' | 'protect' | 'restrict' | 'shutdown';

// ==================== 认证凭证 ====================

/**
 * SNMP版本
 */
export type SNMPVersion = 'v1' | 'v2c' | 'v3';

/**
 * 授权凭证
 */
export interface AuthCredential {
  id: string;
  credentialName: string;
  protocolType: ProtocolType;
  username?: string;
  password?: string;
  enablePassword?: string;
  snmpCommunities: string[];
  snmpVersion: SNMPVersion;
  description?: string;
  isDefault: boolean;
  createdAt: string;
  updatedAt: string;
}

// ==================== 网络设备 ====================

/**
 * 网络设备
 */
export interface NetworkDevice {
  id: string;
  deviceName: string;
  deviceType: DeviceType;
  vendor: DeviceVendor;
  model?: string;
  serialNumber?: string;
  softwareVersion?: string;
  uptime?: string;
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

/**
 * 设备MAC地址
 */
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

/**
 * 设备端口状态
 */
export interface DevicePortStatus {
  id: string;
  deviceId: string;
  deviceName?: string;
  interfaceName: string;
  adminStatus: string;
  operStatus: string;
  description?: string;
  vlan?: number;
  duplex?: string;
  speed?: string;
  portType?: string;
  dot1xEnabled: boolean;
  dot1xPortStatus: Dot1xPortStatus;
  portSecurityEnabled: boolean;
  portSecurityMode: PortSecurityMode;
  maxMACCount?: number;
  currentMACCount?: number;
  collectedAt: string;
  createdAt: string;
}

// ==================== 配置管理 ====================

/**
 * 配置模板类型
 */
export type TemplateType = 'init' | 'config' | 'backup';

/**
 * 配置模板
 */
export interface ConfigTemplate {
  id: string;
  templateName: string;
  templateCode: string;
  templateType: TemplateType;
  vendor?: DeviceVendor;
  deviceType?: DeviceType;
  templateContent: string;
  variables?: Record<string, string | number | boolean>;
  description?: string;
  isSystem: boolean;
  version: number;
  createdAt: string;
  updatedAt: string;
}

/**
 * 配置执行状态
 */
export type ExecutionStatus = 'pending' | 'running' | 'success' | 'failed';

/**
 * 配置执行类型
 */
export type ExecutionType = 'template' | 'command';

/**
 * 配置执行记录
 */
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

/**
 * 配置执行明细
 */
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

/**
 * 配置备份类型
 */
export type BackupType = 'auto' | 'manual';

/**
 * 配置备份
 */
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

// ==================== 设备发现 ====================

/**
 * 设备发现状态
 */
export type DiscoveryStatus = 'pending' | 'running' | 'completed' | 'failed';

/**
 * 设备发现任务
 */
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
