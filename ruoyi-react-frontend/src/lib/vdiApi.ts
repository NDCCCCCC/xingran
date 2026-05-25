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
  VMAccount,
  VMListParams,
  CreateVMRequest,
  UpdateVMRequest,
  VMOperateRequest,
  VMIPConfigRequest,
  RenameVMRequest,
  BindUserRequest,
  VDIServerConfig,
  VMPageResponse,
  CreateAccountRequest,
  ResetPasswordRequest,
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

  configIP: async (configs: VMIPConfigRequest[]) => {
    return await post<void>('/vdi/vm/config_ip', configs);
  },

  rename: async (id: string, request: RenameVMRequest) => {
    return await post<void>(`/vdi/vm/${id}/rename`, request);
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
  VMAccount,
  VMListParams,
  CreateVMRequest,
  UpdateVMRequest,
  VMOperateRequest,
  VMIPConfigRequest,
  RenameVMRequest,
  BindUserRequest,
  VDIServerConfig,
  CreateAccountRequest,
  ResetPasswordRequest,
};
