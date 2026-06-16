// 映射类型
export type MappingType = "manual" | "auto";

// 映射状态
export type MappingStatus = "active" | "inactive";

// 部门-组映射
export interface DeptGroupMapping {
  id: string;
  deptId: string;
  deptName: string;
  adConfigId: string;
  adConfigName: string;
  adGroupId: string;
  adGroupName: string;
  adGroupDN: string;
  memberOUDN: string;
  mappingType: MappingType;
  mappingStatus: MappingStatus;
  syncEnabled: boolean;
  lastSyncAt?: string;
  lastSyncStatus?: string;
  memberCount: number;
  createdBy: string;
  updatedBy: string;
  createdAt: string;
  updatedAt: string;
}

// 映射列表请求
export interface ListMappingsRequest {
  current?: number;
  pageSize?: number;
  adConfigId?: string;
  deptId?: string;
  mappingType?: MappingType;
  mappingStatus?: MappingStatus;
}

// 映射列表响应
export interface ListMappingsResponse {
  list: DeptGroupMapping[];
  total: number;
  current: number;
  pageSize: number;
}

// 创建映射请求
export interface CreateMappingRequest {
  deptId: string;
  adGroupId: string;
  adConfigId: string;
  mappingType?: MappingType;
  syncEnabled?: boolean;
}

// 更新映射请求
export interface UpdateMappingRequest {
  mappingStatus?: MappingStatus;
  syncEnabled?: boolean;
}