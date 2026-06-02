/**
 * VDI (Virtual Desktop Infrastructure) API 客户端
 * 深信服 VDI 集成 API 封装
 */

import { post } from './api';
import type {
  BaseResponse,
  PageResponse,
} from '@/types';
import type {
  VirtualMachine,
  VDIServer,
  VDIResourceGroup,
  VDIResource,
  VMAccount,
  VMListParams,
  CreateVMRequest,
  UpdateVMRequest,
  VMOperateRequest,
  VMIPConfigRequest,
  BindUserRequest,
  VDIServerConfig,
  VMPageResponse,
  CreateAccountRequest,
  ResetPasswordRequest,
  VDIPlatform,
  RunPosition,
  VDIStorage,
  VDINetwork,
} from '@/types/vdi';

// ==================== 虚拟机 API（完整VDI API集成）====================

export const vmApi = {
  // 基础 CRUD
  list: async (params: VMListParams) => {
    return await post<VMPageResponse>('/vdi/vm/list', params);
  },

  get: async (id: string) => {
    return await post<VirtualMachine>(`/vdi/vm/${id}`, {});
  },

  create: async (data: CreateVMRequest) => {
    return await post<VirtualMachine>('/vdi/vm', data);
  },

  update: async (id: string, data: UpdateVMRequest) => {
    return await post<void>(`/vdi/vm/${id}/update`, data);
  },

  delete: async (id: string) => {
    return await post<void>(`/vdi/vm/${id}/delete`, {});
  },

  // VDI 操作（调用 VDI API）
  operate: async (request: VMOperateRequest) => {
    return await post<void>('/vdi/vm/operate', request);
  },

  bindUser: async (id: string, request: BindUserRequest) => {
    return await post<void>(`/vdi/vm/${id}/bind_user`, request);
  },

  unbindUser: async (id: string) => {
    return await post<void>(`/vdi/vm/${id}/unbind_user`, {});
  },

  // 同步操作
  sync: async (id: string) => {
    return await post<void>(`/vdi/vm/${id}/sync`, {});
  },

  // 资源组查询
  listResourceGroups: async (vdiServerId?: string) => {
    return await post<VDIResourceGroup[]>('/vdi/vm/resource-groups', {
      vdi_server_id: vdiServerId || '',
    });
  },

  // 资源查询（资源组下的具体计算资源）
  listResources: async (vdiServerId: string, groupId: string) => {
    return await post<VDIResource[]>('/vdi/vm/resources', {
      vdi_server_id: vdiServerId,
      group_id: groupId,
    });
  },

  // 批量操作辅助方法
  batchOperate: async (vmIds: string[], action: string) => {
    return await post<void>('/vdi/vm/operate', {
      vm_ids: vmIds,
      action,
    });
  },

  // ==================== 账号管理 API ====================

  listAccounts: async (vmId: string) => {
    return await post<{ list: VMAccount[] }>(`/vdi/vm/${vmId}/accounts`, {});
  },

  createAccount: async (vmId: string, data: CreateAccountRequest) => {
    return await post<VMAccount>(`/vdi/vm/${vmId}/accounts`, data);
  },

  resetAccountPassword: async (vmId: string, accountId: string, data: ResetPasswordRequest) => {
    return await post<void>(`/vdi/vm/${vmId}/accounts/${accountId}/reset_password`, data);
  },

  deleteAccount: async (vmId: string, accountId: string) => {
    return await post<void>(`/vdi/vm/${vmId}/accounts/${accountId}/delete`, {});
  },

  // ==================== VDI 创建虚拟机相关 API ====================

  listVTPPlatforms: async (vdiServerId: string) => {
    return await post<VDIPlatform[]>('/vdi/vm/vtp-platforms', {
      vdi_server_id: vdiServerId,
    });
  },

  listRunPositions: async (vdiServerId: string, vtpId: number) => {
    return await post<RunPosition[]>('/vdi/vm/run-positions', {
      vdi_server_id: vdiServerId,
      vtp_id: vtpId,
    });
  },

  listStorages: async (vdiServerId: string, vtpId: number) => {
    return await post<VDIStorage[]>('/vdi/vm/storages', {
      vdi_server_id: vdiServerId,
      vtp_id: vtpId,
    });
  },

  listNetworks: async (vdiServerId: string, vtpId: number) => {
    return await post<VDINetwork[]>('/vdi/vm/networks', {
      vdi_server_id: vdiServerId,
      vtp_id: vtpId,
    });
  },
};

// ==================== VDI 服务器 API ====================

export const vdiServerApi = {
  list: async (params: { current: number; pageSize: number }) => {
    return await post<PageResponse<VDIServer>>('/vdi/servers/list', params);
  },

  get: async (id: string) => {
    return await post<VDIServer>(`/vdi/servers/${id}`, {});
  },

  create: async (data: VDIServerConfig) => {
    return await post<VDIServer>('/vdi/servers', data);
  },

  update: async (id: string, data: Partial<VDIServerConfig>) => {
    return await post<void>(`/vdi/servers/${id}/update`, data);
  },

  delete: async (id: string) => {
    return await post<void>(`/vdi/servers/${id}/delete`, {});
  },

  testConnection: async (id: string) => {
    return await post<void>(`/vdi/servers/${id}/test`, {});
  },
};

// ==================== 类型导出 ====================

export type {
  VirtualMachine,
  VDIServer,
  VDIResourceGroup,
  VDIResource,
  VMAccount,
  VMListParams,
  CreateVMRequest,
  UpdateVMRequest,
  VMOperateRequest,
  VMIPConfigRequest,
  BindUserRequest,
  VDIServerConfig,
  CreateAccountRequest,
  ResetPasswordRequest,
  VDIPlatform,
  RunPosition,
  VDIStorage,
  VDINetwork,
};
