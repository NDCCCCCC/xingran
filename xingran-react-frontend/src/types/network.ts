/**
 * 网络设备管理相关类型
 */

// ==================== 设备基础类型 ====================

/**
 * 设备厂商
 */
export type DeviceVendor = "huawei" | "h3c" | "ruijie" | "maipu" | "cisco";

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
  | "port_binding" // v1.20.1 BIND-01/02
  | "trunk_set_native" // v1.51 Phase 124 PWX-01
  | "trunk_set_allowed" // v1.51 Phase 124 PWX-01
  | "trunk_set_hybrid" // v1.51 Phase 124 PWX-01
  | "portsec_enable" // v1.51 Phase 125 PWX-03
  | "portsec_disable" // v1.51 Phase 125 PWX-03
  | "portsec_max_mac" // v1.51 Phase 125 PWX-03
  | "portsec_sticky" // v1.51 Phase 125 PWX-03
  | "portsec_static_bind"; // v1.51 Phase 125 PWX-03

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
  /**
   * Phase 123 (commit 6cfdca54) 扩展: 写失败的恢复语义。
   * - rollback="rolled_back": 失败后已自动回滚
   * - rollback="rollback_failed": 回滚也失败（UI 必须常驻告警, 引导查看审计页）
   */
  extra?: {
    /** no_rollback: trunk 动作无 DB 原值可恢复, 快照留痕（区别于 rollback_failed 告警） */
    rollback?: "rolled_back" | "rollback_failed" | "no_rollback";
    rollbackError?: string;
  };
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
  /**
   * Phase 123 (commit 6cfdca54): 管理员强制跳过预检。
   * 契约已就绪; UI 开关待权限真相源确认（Phase 86 门控语义）后接入, 当前仅透传层支持。
   */
  skipPreflight?: boolean;
  // ---- v1.51 Phase 124 (commit 9835291c) trunk/hybrid 扩展, 按 action 携带 ----
  /** action === "trunk_set_native" 时使用, 1-4094 */
  nativeVlan?: number;
  /** action === "trunk_set_allowed" 时使用 */
  allowedVlans?: number[];
  /** 仅 trunk_set_allowed: add | remove | replace（默认 replace; replace 仅锐捷原生支持） */
  mode?: "add" | "remove" | "replace";
  /** action === "trunk_set_hybrid" 时使用; 与 hybridUntagged 不可重叠（后端 400 兜底） */
  hybridTagged?: number[];
  /** action === "trunk_set_hybrid" 时使用 */
  hybridUntagged?: number[];
  // ---- v1.51 Phase 125 (commit 006ece28) portsec 扩展, 按 action 携带 ----
  /** action === "portsec_max_mac" 时使用, 1-4096 */
  maxMacCount?: number;
  // portsec_static_bind 复用上方 macAddress（必填）与 vlanId（必填, 1-4094）
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
  /** Phase 122 (ece40502): 与 WS port_write_* 事件同值, 供进度订阅关联与断线对账 */
  batchId?: string;
  succeeded: PortResult[];
  failed: PortResult[];
  skipped: PortResult[];
}

/**
 * 端口写预览命令 (v1.51 Phase 121 PORTW-02, 镜像后端 portwrite preview)
 *
 * Source: POST /network/ports/write/preview 契约 (portwriter, commit 8e158388)
 *
 * critical=true 的命令可能影响管理面/链路 (如 shutdown), UI 需醒目标注。
 */
export interface PortWritePreviewCommand {
  index: number;
  command: string;
  riskLevel: "critical" | "medium" | "low";
  critical: boolean;
  note?: string;
}

/**
 * 端口写预览端口条目
 *
 * status="skipped" → noOp=true 且 commands=[]（pre-state 已匹配目标态, 下发时会被跳过）。
 */
export interface PortWritePreviewPort {
  portId: string;
  interfaceName: string;
  status: "preview" | "skipped";
  noOp: boolean;
  commands: PortWritePreviewCommand[];
}

/**
 * 端口写预览响应 (dry-run, 不下发任何设备命令)
 */
export interface PortWritePreviewResult {
  dryRun: boolean;
  deviceId: string;
  deviceName: string;
  vendor: string;
  model: string;
  action: PortWriteAction;
  riskLevel: "critical" | "medium" | "low";
  summary: string;
  ports: PortWritePreviewPort[];
}

/**
 * 端口写审计行 (v1.51 Phase 127 PWX-04, 镜像 models.PortWriteAudit)
 *
 * Source: internal/models/port_write_audit.go:26-40 (append-only 真相源)
 *
 * beforeValue/afterValue 为任意 JSON (pre-state / post-state 快照),
 * 前端以格式化 JSON 对比展示; failureReason/operLogId nullable。
 */
export interface PortWriteAuditRow {
  id: string;
  deviceId: string;
  portId: string;
  action: string;
  beforeValue?: unknown;
  afterValue?: unknown;
  commandSent: string;
  deviceResponse: string;
  /** 契约 30a80c94: 值域仅 succeeded|failed（回滚结果在 snapshot 表, 不在 audit 行） */
  status: "succeeded" | "failed";
  failureReason?: string | null;
  operator: string;
  operLogId?: string | null;
  createdAt: string;
}

/** 端口写审计分页查询请求（契约 30a80c94: current 从 1 起, pageSize 上限 100, 时间 RFC3339） */
export interface PortWriteAuditListRequest {
  current: number;
  pageSize: number;
  deviceId?: string;
  portId?: string;
  operator?: string;
  action?: string;
  status?: string;
  startTime?: string;
  endTime?: string;
  /** 契约 9a1ea4d7: 等值过滤——PORTW-03 断线对账精确拉取该批次审计行 */
  batchId?: string;
}

/** 端口写审计分页响应 (项目分页约定: list/total/current/pageSize) */
export interface PortWriteAuditListResult {
  list: PortWriteAuditRow[];
  total: number;
  current: number;
  pageSize: number;
}
