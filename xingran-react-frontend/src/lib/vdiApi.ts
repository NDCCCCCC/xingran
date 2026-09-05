/**
 * VDI (Virtual Desktop Infrastructure) API 客户端
 * 深信服 VDI 集成 API 封装
 */

import { post } from "./api";
import { createResourceApi } from "./apiFactory";
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
  BindUserRequest,
  VDIServerConfig,
  VMPageResponse,
  CreateAccountRequest,
  ResetPasswordRequest,
  VDIPlatform,
  RunPosition,
  VDIStorage,
  VDINetwork,
} from "@/types/vdi";

// ==================== 虚拟机 API（完整VDI API集成）====================

const vmCrud = createResourceApi<VirtualMachine>({ basePath: "/vdi/vms" });

export const vmApi = {
  ...vmCrud,

  // 基础 CRUD — list/create 与工厂签名不同构,SPREAD+OVERRIDE 原样保留（Phase 94 D-08）:
  // - list: VMListParams 是 interface,工厂 list 的 PageParams & Record<string,unknown>
  //   参数会拒绝 interface 变量直传(VirtualMachineList/index.tsx:186-191 实锤 TS2345)
  // - create: CreateVMRequest 含 vtp_id/count 等实体外字段且缺 vm_id 必选字段,双向不可赋值
  // get/update/delete 类型兼容,来自工厂 spread
  list: async (params: VMListParams) => {
    return await post<VMPageResponse>("/vdi/vms/list", params);
  },

  create: async (data: CreateVMRequest) => {
    return await post<VirtualMachine>("/vdi/vms", data);
  },

  // VDI 操作（调用 VDI API）
  operate: async (request: VMOperateRequest) => {
    return await post<void>("/vdi/vms/operate", request);
  },

  bindUser: async (id: string, request: BindUserRequest) => {
    return await post<void>(`/vdi/vms/${id}/bind_user`, request);
  },

  unbindUser: async (id: string) => {
    return await post<void>(`/vdi/vms/${id}/unbind_user`, {});
  },

  // 同步操作
  sync: async (id: string) => {
    return await post<void>(`/vdi/vms/${id}/sync`, {});
  },

  // 资源组查询
  listResourceGroups: async (vdiServerId?: string) => {
    return await post<VDIResourceGroup[]>("/vdi/vms/resource-groups", {
      vdi_server_id: vdiServerId || "",
    });
  },

  // 资源查询（资源组下的具体计算资源）
  listResources: async (vdiServerId: string, groupId: string) => {
    return await post<VDIResource[]>("/vdi/vms/resources", {
      vdi_server_id: vdiServerId,
      group_id: groupId,
    });
  },

  // 批量操作辅助方法
  batchOperate: async (vmIds: string[], action: string) => {
    return await post<void>("/vdi/vms/operate", {
      vm_ids: vmIds,
      action,
    });
  },

  // ==================== 账号管理 API ====================

  listAccounts: async (vmId: string) => {
    return await post<{ list: VMAccount[] }>(`/vdi/vms/${vmId}/accounts`, {});
  },

  createAccount: async (vmId: string, data: CreateAccountRequest) => {
    return await post<VMAccount>(`/vdi/vms/${vmId}/accounts`, data);
  },

  resetAccountPassword: async (vmId: string, accountId: string, data: ResetPasswordRequest) => {
    return await post<void>(`/vdi/vms/${vmId}/accounts/${accountId}/reset_password`, data);
  },

  deleteAccount: async (vmId: string, accountId: string) => {
    return await post<void>(`/vdi/vms/${vmId}/accounts/${accountId}/delete`, {});
  },

  // ==================== VDI 创建虚拟机相关 API ====================

  listVTPPlatforms: async (vdiServerId: string) => {
    return await post<VDIPlatform[]>("/vdi/vms/vtp-platforms", {
      vdi_server_id: vdiServerId,
    });
  },

  listRunPositions: async (vdiServerId: string, vtpId: number) => {
    return await post<RunPosition[]>("/vdi/vms/run-positions", {
      vdi_server_id: vdiServerId,
      vtp_id: vtpId,
    });
  },

  listStorages: async (vdiServerId: string, vtpId: number) => {
    return await post<VDIStorage[]>("/vdi/vms/storages", {
      vdi_server_id: vdiServerId,
      vtp_id: vtpId,
    });
  },

  listNetworks: async (vdiServerId: string, vtpId: number) => {
    return await post<VDINetwork[]>("/vdi/vms/networks", {
      vdi_server_id: vdiServerId,
      vtp_id: vtpId,
    });
  },
};

// ==================== VDI 服务器 API ====================

// 纯 SPREAD 接入（Phase 94 D-08）:CreatePayload 排除集含 snake_case created_at/updated_at,
// VDIServerConfig 对 Partial<CreatePayload<VDIServer>> 可赋值（RESEARCH tsc 实测）
const vdiServerCrud = createResourceApi<VDIServer>({ basePath: "/vdi/servers" });

export const vdiServerApi = {
  ...vdiServerCrud,

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
  BindUserRequest,
  VDIServerConfig,
  CreateAccountRequest,
  ResetPasswordRequest,
  VDIPlatform,
  RunPosition,
  VDIStorage,
  VDINetwork,
};
