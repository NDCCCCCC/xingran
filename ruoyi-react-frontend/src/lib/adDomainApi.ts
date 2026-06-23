import { get, post } from "./api";
import type { BaseResponse, PageResponse } from "@/types";

// ==================== 类型定义 ====================

export interface ADConfig {
  id: string;
  configName: string;
  serverAddress: string;
  serverPort: number;
  domainName: string;
  baseDn: string;
  useSsl: boolean;
  useTls: boolean;
  syncEnabled: boolean;
  syncInterval: number;
  lastSyncAt?: string;
  status: number;
  createdAt: string;
  createdBy: string;
  updatedAt?: string;
  updatedBy?: string;
}

export interface ADOU {
  id: string;
  adConfigId: string;
  ouDn: string;
  ouName: string;
  ouPath?: string;
  parentDn?: string;
  description?: string;
  userCount: number;
  groupCount: number;
  lastSyncAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ADOUNode {
  id: string;
  name: string;
  dn: string;
  path?: string;
  children?: ADOUNode[];
}

export interface ADGroup {
  id: string;
  adConfigId: string;
  groupDn: string;
  groupName: string;
  groupScope?: string;
  groupType?: number;
  description?: string;
  memberCount: number;
  ouDn?: string;
  lastSyncAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ADUser {
  id: string;
  adConfigId: string;
  userDn: string;
  username: string;
  displayName?: string;
  email?: string;
  phone?: string;
  mobile?: string;
  title?: string;
  department?: string;
  company?: string;
  ouDn?: string;
  userAccountControl?: number;
  isEnabled: boolean;
  isLocked: boolean;
  passwordExpired: boolean;
  lastLogon?: string;
  passwordLastSet?: string;
  accountExpires?: string;
  description?: string;
  memberOf?: string;
  lastSyncAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ADSyncLog {
  id: string;
  adConfigId: string;
  syncType: string;
  syncStatus: string;
  startTime: string;
  endTime?: string;
  duration?: number;
  ouCount: number;
  groupCount: number;
  userCount: number;
  computerCount?: number;
  errorCount: number;
  errorMessage?: string;
  createdAt: string;
}

export interface ADComputer {
  id: string;
  adConfigId: string;
  computerName: string;
  distinguishedName: string;
  lastLogon?: string;
  passwordLastSet?: string;
  logonCount: number;
  ouDn?: string;
  status: number;
  originalDescription?: string;
  ipAddress?: string;
  macAddress?: string;
  managedBy?: string;
  operatingSystem?: string;
  osVersion?: string;
  cpuModel?: string;
  architecture?: string;
  memoryCapacity?: string;
  hardDiskCapacity?: string;
  lastOnlineTime?: string;
  serialNumber?: string;
  systemInfo?: string;
  lastLogonUser?: string;
  createdAt: string;
  updatedAt: string;
}

export type ADComputerDetail = ADComputer;

export interface ADSyncResult {
  ouCount: number;
  groupCount: number;
  userCount: number;
}

// ==================== 请求参数类型 ====================

export interface ADConfigListRequest {
  status?: number;
  current?: number;
  pageSize?: number;
  orderByColumn?: string;
  isAsc?: boolean;
}

export interface ADConfigCreateRequest {
  configName: string;
  serverAddress: string;
  serverPort: number;
  domainName: string;
  baseDn: string;
  useSsl?: boolean;
  useTls?: boolean;
  syncEnabled?: boolean;
  syncInterval?: number;
}

export interface ADConfigUpdateRequest {
  configName: string;
  serverAddress: string;
  serverPort: number;
  domainName: string;
  baseDn: string;
  useSsl: boolean;
  useTls: boolean;
  syncEnabled: boolean;
  syncInterval: number;
  status?: number;
}

export interface ADUserListRequest {
  configId: string;
  ouDn?: string;
  username?: string;
  isEnabled?: boolean;
  current?: number;
  pageSize?: number;
  orderByColumn?: string;
  isAsc?: boolean;
}

export interface ADGroupListRequest {
  configId: string;
  ouDn?: string;
  groupName?: string;
  current?: number;
  pageSize?: number;
  orderByColumn?: string;
  isAsc?: boolean;
}

export interface ADComputerListRequest {
  configId: string;
  ouDn?: string;
  computerName?: string;
  current?: number;
  pageSize?: number;
}

export interface ADUserUpdateRequest {
  displayName?: string;
  email?: string;
  phone?: string;
  mobile?: string;
  title?: string;
  department?: string;
  description?: string;
}

export interface ADGroupUpdateRequest {
  groupName?: string;
  description?: string;
}

// ==================== 辅助函数 ====================

const DEFAULT_PAGINATION = { current: 1, pageSize: 10 };

function withDefaultPagination<T extends { current?: number; pageSize?: number }>(
  params: T
): T {
  return { ...DEFAULT_PAGINATION, ...params };
}

// ==================== AD配置 API ====================

export function getADConfigList(params: ADConfigListRequest = {}): Promise<BaseResponse<PageResponse<ADConfig>>> {
  return post("/ad-domain/configs/list", withDefaultPagination(params));
}

export function createADConfig(data: ADConfigCreateRequest): Promise<BaseResponse<ADConfig>> {
  return post("/ad-domain/configs", data);
}

export function getADConfig(id: string): Promise<BaseResponse<ADConfig>> {
  return get(`/ad-domain/configs/${id}`);
}

export function updateADConfig(id: string, data: ADConfigUpdateRequest): Promise<BaseResponse<{ message: string }>> {
  return post(`/ad-domain/configs/${id}/update`, data);
}

export function deleteADConfig(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/ad-domain/configs/${id}/delete`);
}

export function testADConnection(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/ad-domain/configs/${id}/test`, {});
}

export function syncADData(id: string, syncType: string = "full"): Promise<BaseResponse<{ message: string; result: ADSyncResult }>> {
  return post(`/ad-domain/configs/${id}/sync`, { syncType });
}

// ==================== OU管理 API ====================

export function getADOUTree(configId: string): Promise<BaseResponse<ADOUNode[]>> {
  return post("/ad-domain/ous/tree", { configId });
}

// ==================== 用户组管理 API ====================

export function getADGroupList(params: ADGroupListRequest): Promise<BaseResponse<PageResponse<ADGroup>>> {
  return post("/ad-domain/groups/list", withDefaultPagination(params));
}

export function getADGroupDetail(id: string): Promise<BaseResponse<ADGroup>> {
  return get(`/ad-domain/groups/${id}`);
}

export function updateADGroup(id: string, configId: string, data: ADGroupUpdateRequest): Promise<BaseResponse<{ message: string }>> {
  return post(`/ad-domain/groups/${id}/update`, { configId, ...data });
}

export function getADGroupMembers(
  id: string,
  configId: string,
  params: { current?: number; pageSize?: number } = {}
): Promise<BaseResponse<PageResponse<ADUser>>> {
  return post(`/ad-domain/groups/${id}/members`, {
    configId,
    ...withDefaultPagination(params),
  });
}

export function addADGroupMember(id: string, configId: string, userDn: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/ad-domain/groups/${id}/members/add`, { configId, userDn });
}

export function removeADGroupMember(id: string, configId: string, userDn: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/ad-domain/groups/${id}/members/remove`, { configId, userDn });
}

// ==================== 用户组同步 API ====================

export interface ADGroupSyncResult {
  totalGroups: number;
  createdGroups: number;
  updatedGroups: number;
  deletedGroups: number;
  totalMembers: number;
  createdMembers: number;
  removedMembers: number;
  duration: number;
}

export interface ADGroupSyncStatus {
  configId: string;
  configName: string;
  lastSyncAt?: string;
  syncEnabled: boolean;
  totalGroups: number;
  recentlySynced: number;
  totalMemberRelations: number;
  neverSynced: number;
}

export function syncADGroups(configId: string): Promise<BaseResponse<ADGroupSyncResult>> {
  return post(`/ad-domain/groups/sync-by-config/${configId}`, {});
}

export function syncADSingleGroup(configId: string, groupDn: string): Promise<BaseResponse<{ message: string }>> {
  return post("/ad-domain/groups/sync-single", { configId, groupDn });
}

export function getADGroupSyncStatus(configId: string): Promise<BaseResponse<ADGroupSyncStatus>> {
  return post("/ad-domain/groups/sync-status", { configId });
}

// ==================== 用户管理 API ====================

export function getADUserList(params: ADUserListRequest): Promise<BaseResponse<PageResponse<ADUser>>> {
  return post("/ad-domain/users/list", withDefaultPagination(params))
}

export function getADUserIds(
  params: ADUserListRequest
): Promise<BaseResponse<string[]>> {
  return post("/ad-domain/users/ids", params)
}

export function getADUserDetail(id: string, configId: string): Promise<BaseResponse<ADUser>> {
  return post(`/ad-domain/users/${id}`, { configId })
}

export function updateADUser(id: string, configId: string, data: ADUserUpdateRequest): Promise<BaseResponse<{ message: string }>> {
  return post(`/ad-domain/users/${id}/update`, { configId, update: data });
}

export function moveADUser(id: string, configId: string, newOuDn: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/ad-domain/users/${id}/move`, { configId, move: { newOuDn } });
}

export function enableADUser(id: string, configId: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/ad-domain/users/${id}/enable`, { configId });
}

export function disableADUser(id: string, configId: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/ad-domain/users/${id}/disable`, { configId });
}

// ==================== 同步日志 API ====================

export function getADSyncLogs(
  configId: string | undefined,
  params: { current?: number; pageSize?: number } = {}
): Promise<BaseResponse<PageResponse<ADSyncLog>>> {
  return post("/ad-domain/logs/list", {
    configId,
    ...withDefaultPagination(params),
  });
}

// ==================== 电脑设备管理 API ====================

export function getADComputerList(params: ADComputerListRequest): Promise<BaseResponse<PageResponse<ADComputerDetail>>> {
  return post("/ad-domain/computers/list", withDefaultPagination(params));
}

export function getADComputerDetail(configId: string, computerDn: string): Promise<BaseResponse<ADComputerDetail>> {
  return post("/ad-domain/computers/detail", { configId, computerDn });
}

// ==================== 部门组映射 API ====================

export interface DeptGroupMapping {
  id: string;
  deptId: string;
  adGroupId: string;
  adConfigId: string;
  mappingType: "auto" | "manual";
  mappingStatus: "active" | "inactive";
  groupDn: string;
  groupName: string;
  syncEnabled: boolean;
  lastSyncAt?: string;
  createdBy?: string;
  updatedBy?: string;
  createdAt: string;
  updatedAt: string;
  Dept?: { id: string; deptName: string };
  ADGroup?: { id: string; groupName: string; groupDn: string };
}

export interface MappingListRequest {
  adConfigId?: string;
  deptId?: string;
  mappingType?: "auto" | "manual";
  mappingStatus?: "active" | "inactive";
  groupName?: string;
  current?: number;
  pageSize?: number;
}

export interface MappingListResponse {
  total: number;
  list: DeptGroupMapping[];
}

export interface CreateMappingRequest {
  deptId: string;
  adGroupId: string;
  adConfigId: string;
  mappingType?: "auto" | "manual";
  syncEnabled?: boolean;
}

export interface UpdateMappingRequest {
  mappingStatus?: "active" | "inactive";
  syncEnabled?: boolean;
}

export function getMappingList(params: MappingListRequest): Promise<BaseResponse<MappingListResponse>> {
  return post("/ad-domain/mappings/list", withDefaultPagination(params));
}

export function createMapping(data: CreateMappingRequest): Promise<BaseResponse<DeptGroupMapping>> {
  return post("/ad-domain/mappings", data);
}

export function getMapping(id: string): Promise<BaseResponse<DeptGroupMapping>> {
  return get(`/ad-domain/mappings/${id}`);
}

export function updateMapping(id: string, data: UpdateMappingRequest): Promise<BaseResponse<null>> {
  return post(`/ad-domain/mappings/${id}/update`, data);
}

export function deleteMapping(id: string): Promise<BaseResponse<null>> {
  return post(`/ad-domain/mappings/${id}/delete}`, {});
}

// ==================== OU 部门映射 API ====================

export interface OUDeptMappingResponse {
  hasMapping: boolean;
  message?: string;
  mapping?: {
    deptId: string;
    deptName: string;
    ouDn: string;
    ouName: string;
    syncEnabled: boolean;
    syncStatus: string;
  };
}

export function getOUDeptMapping(ouDn: string): Promise<BaseResponse<OUDeptMappingResponse>> {
  return get(`/ad-domain/ou/${encodeURIComponent(ouDn)}/dept-mapping`);
}

export function updateOUDeptMapping(
  ouDn: string,
  data: { deptId: string }
): Promise<BaseResponse<{ message: string; mapping: OUDeptMappingResponse["mapping"] }>> {
  return post(`/ad-domain/ou/${encodeURIComponent(ouDn)}/dept-mapping`, data);
}

// ==================== OU组映射 API ====================

export interface OUGroupMapping {
  id: string;
  adConfigId: string;
  ouDn: string;
  ouName: string;
  adGroupId: string;
  mappingStatus: "active" | "inactive";
  syncEnabled: boolean;
  lastSyncAt?: string;
  createdAt: string;
  updatedAt: string;
  adGroup?: {
    id: string;
    groupName: string;
    groupDn: string;
    description?: string;
  };
}

export interface OUGroupMappingListRequest {
  adConfigId?: string;
  ouDn?: string;
  groupName?: string;
  status?: string;
  current?: number;
  pageSize?: number;
}

export interface OUGroupMappingListResponse {
  list: OUGroupMapping[];
  total: number;
  current: number;
  pageSize: number;
}

export interface OUGroupMappingCreateRequest {
  adConfigId: string;
  ouDn: string;
  ouName: string;
  adGroupId: string;
  syncEnabled?: boolean;
}

export interface OUGroupMappingUpdateRequest {
  syncEnabled?: boolean;
  status?: "active" | "inactive";
}

export function getOUGroupMappings(params: OUGroupMappingListRequest): Promise<BaseResponse<OUGroupMappingListResponse>> {
  return post("/ad-domain/ou-group-mappings/list", withDefaultPagination(params));
}

export function getOUGroupMapping(id: string): Promise<BaseResponse<OUGroupMapping>> {
  return get(`/ad-domain/ou-group-mappings/${id}`);
}

export function createOUGroupMapping(data: OUGroupMappingCreateRequest): Promise<BaseResponse<OUGroupMapping>> {
  return post("/ad-domain/ou-group-mappings", data);
}

export function updateOUGroupMapping(id: string, data: OUGroupMappingUpdateRequest): Promise<BaseResponse<{ message: string }>> {
  return post(`/ad-domain/ou-group-mappings/${id}/update`, data);
}

export function deleteOUGroupMapping(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/ad-domain/ou-group-mappings/${id}/delete`, {});
}

export function getOUGroupMappingsByOU(ouDn: string): Promise<BaseResponse<OUGroupMapping[]>> {
  return get(`/ad-domain/ou-group-mappings/ou/${encodeURIComponent(ouDn)}`);
}

// ==================== 批量用户同步 API ====================

export interface BatchSyncUsersRequest {
  configId: string;
  groupId?: string;
  userDns: string[];
  defaultRoleId?: string;
}

export interface BatchSyncResult {
  total: number;
  success: number;
  failed: number;
  skipped: number;
  errors?: Array<{
    username: string;
    error: string;
  }>;
}

// 直接批量同步AD用户（不依赖用户组）
export function batchSyncADUsersDirect(
  data: BatchSyncUsersRequest
): Promise<BaseResponse<BatchSyncResult>> {
  return post("/ad-domain/users/batch-sync", data);
}

// ==================== Phase 36: AD 服务账号池 ====================

export interface ADServiceAccount {
  id: string;
  configId: string;
  username: string;
  status: number; // 0=可用, 1=停用, 2=熔断
  failureCount: number;
  circuitBreakerUntil?: string;
  lastSuccessAt?: string;
  lastFailureAt?: string;
  lastFailureReason?: string;
  manualUnlockReason?: string;
  manualUnlockedBy?: string;
  manualUnlockedAt?: string;
  remark?: string;
  createdAt: string;
  updatedAt?: string;
}

export interface ADServiceAccountListResponse {
  list: ADServiceAccount[];
  total: number;
  current: number;
  pageSize: number;
}

export interface ADServiceAccountStats {
  total: number;
  available: number;
  disabled: number;
  circuitBroken: number;
  currentAccount?: string;
}

// 列表（带分页）
export function listADServiceAccounts(
  data: { configId: string; page?: number; pageSize?: number; status?: number }
): Promise<BaseResponse<ADServiceAccountListResponse>> {
  return post("/ad-domain/accounts/list", data);
}

// 新增
export function createADServiceAccount(
  data: { configId: string; username: string; password: string; remark?: string }
): Promise<BaseResponse<{ id: string }>> {
  return post("/ad-domain/accounts/create", data);
}

// 更新
export function updateADServiceAccount(
  data: { id: string; username?: string; password?: string; remark?: string }
): Promise<BaseResponse<{ ok: boolean }>> {
  return post("/ad-domain/accounts/update", data);
}

// 删除
export function deleteADServiceAccount(id: string): Promise<BaseResponse<{ ok: boolean }>> {
  return post("/ad-domain/accounts/delete", { id });
}

// 启用
export function enableADServiceAccount(id: string): Promise<BaseResponse<{ ok: boolean }>> {
  return post("/ad-domain/accounts/enable", { id });
}

// 停用
export function disableADServiceAccount(id: string): Promise<BaseResponse<{ ok: boolean }>> {
  return post("/ad-domain/accounts/disable", { id });
}

// 立即解锁（reason ≥10 字符）
export function unlockADServiceAccount(
  data: { id: string; reason: string }
): Promise<BaseResponse<{ ok: boolean }>> {
  return post("/ad-domain/accounts/unlock", data);
}

// 统计
export function getADServiceAccountStats(
  configId: string
): Promise<BaseResponse<ADServiceAccountStats>> {
  return post("/ad-domain/accounts/stats", { configId });
}
