/**
 * 网络设备管理相关类型
 */

// ==================== 设备基础类型 ====================

/**
 * 设备厂商
 */
export type DeviceVendor = "huawei" | "h3c" | "ruijie" | "maipu";

/**
 * 设备类型
 */
export type DeviceType = "router" | "switch" | "firewall" | "ap" | "loadbalancer";

/**
 * 设备状态：0=在线，1=离线，2=未知
 */
export type DeviceStatus = 0 | 1 | 2;

/**
 * 协议类型
 */
export type ProtocolType = "ssh" | "telnet";

/**
 * MAC地址类型
 */
export type MACType = "dynamic" | "static" | "secure";

/**
 * 802.1X端口状态
 */
export type Dot1xPortStatus = "authorized" | "unauthorized" | "unknown";

/**
 * 端口安全模式
 */
export type PortSecurityMode = "" | "protect" | "restrict" | "shutdown";

// ==================== 认证凭证 ====================

/**
 * SNMP版本
 */
export type SNMPVersion = "v1" | "v2c" | "v3";

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
export type TemplateType = "init" | "config" | "backup";

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
export type ExecutionStatus = "pending" | "running" | "success" | "failed";

/**
 * 配置执行类型
 */
export type ExecutionType = "template" | "command";

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
  executionStrategy: "parallel" | "serial";
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
export type BackupType = "auto" | "manual";

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
export type DiscoveryStatus = "pending" | "running" | "completed" | "failed";

/**
 * 设备发现任务
 */
export interface DeviceDiscovery {
  id: string;
  taskName: string;
  discoveryType: "snmp" | "scan";
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

// ==================== 端口写操作 (Phase 53) ====================

/**
 * 端口写操作 action 字面量联合类型 (Phase 53)
 *
 * 与后端 portcollection.PortAction 字面量集合一一对应,编译期锁定 5 个 action,
 * 防 typo 引入第 4 态导致 BulkWriteDrawer 结果分区漏判 (LANDMINE #3 / T-53-04)。
 */
export type PortWriteAction =
  | "shutdown"
  | "undo_shutdown"
  | "description"
  | "dot1x_enable"
  | "dot1x_disable"
  | "set_access_vlan" // v1.20.1 VLAN-01
  | "port_binding"; // v1.20.1 BIND-01/02

/**
 * 单端口写操作结果 (Phase 53, 镜像后端 portwrite.PortResult)
 *
 * Source: internal/services/portwrite/port_write_service.go:34-42
 *
 * status 字段用字面量联合 "succeeded" | "failed" | "skipped" (不是 string),
 * 编译期锁定三态防 typo。
 */
export interface PortResult {
  portId: string;
  action: PortWriteAction;
  status: "succeeded" | "failed" | "skipped";
  noOp: boolean;
  currentState?: string;
  error?: string;
  commandSent?: string;
}

/**
 * 批量写请求 (Phase 53, 镜像后端 portwrite.BatchWriteRequest)
 *
 * Source: internal/services/portwrite/port_write_service.go:45-50
 *
 * action 复用 PortWriteAction 联合类型;description 仅 action="description" 时使用。
 */
export interface BatchWriteRequest {
  deviceId: string;
  action: PortWriteAction;
  portIds: string[];
  description?: string;
  // v1.20.1 batch extensions (4 optional fields)
  // 与 W2 batch_orchestrator.go + W3 handler BatchWriteRequest 字段对齐。
  vlanId?: number; // action === "set_access_vlan" 时使用, 1-4094
  op?: "add" | "remove"; // action === "port_binding" 时使用
  ipAddress?: string; // action === "port_binding" 时使用, IPv4
  macAddress?: string; // action === "port_binding" 时使用, optional
  reason?: string;
}

/**
 * 修改 access VLAN 请求 (v1.20.1 VLAN-01)
 *
 * Source: internal/services/portwrite/port_write_service.go (W2 SetAccessVlan)
 *
 * vlanId: 1-4094 (前端 InputNumber min/max + 后端 service ErrVlanIdOutOfRange 二次校验)。
 */
export interface SetAccessVlanRequest {
  portId: string;
  vlanId: number; // 1-4094
  reason: string;
}

/**
 * 端口绑定请求 (v1.20.1 BIND-01/02)
 *
 * Source: internal/services/portwrite/port_write_service.go (W2 PortBinding)
 *
 * op=add 创建静态绑定; op=remove 删除已有绑定。
 * ipAddress: 严格 IPv4 regex (后端 service 二次校验)。
 * macAddress: 可选 MAC (Huawei/H3C 用, Ruijie 接受); undefined 表示仅 IP 绑定。
 */
export interface PortBindingRequest {
  portId: string;
  op: "add" | "remove";
  ipAddress: string; // IPv4 regex
  macAddress?: string; // MAC regex (optional)
  reason: string;
}

/**
 * 批量写结果 (Phase 53, 镜像后端 portwrite.BatchResult)
 *
 * Source: internal/services/portwrite/batch_orchestrator.go:15-19
 *
 * 后端保证三个切片即使为空也初始化为 []PortResult{}(非 nil),
 * JSON 序列化输出 [] 而非 null — 前端无需做 null 守卫。
 */
export interface BatchResult {
  succeeded: PortResult[];
  failed: PortResult[];
  skipped: PortResult[];
}
