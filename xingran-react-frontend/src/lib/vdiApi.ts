/**
 * VDI (Virtual Desktop Infrastructure) API 客户端
 * 深信服 VDI 集成 API 封装（Phase 100 V130R-10/11 对账端态:vmApi 16 / vdiServerApi 6）
 *
 * D-100-1: createResourceApi 两站点 spread 改显式 pick——工厂注入的
 * batch/statistics/searchOptions 后端无对应路由（幽灵 404 面），不得出现在
 * 导出对象上（方法集由 apiFactory.invariants.test.ts keys 基线双向锁定）。
 * D-100-11: accounts 子资源族 4 方法已删（后端零实现,运行时 404,详情页 Tab 同删）。
 * 对账台账: .planning/phases/100-frontend-contract-fixes/RECONCILIATION.md。
 */

import { post } from "./api";
import { createResourceApi } from "./apiFactory";
import type {
  VirtualMachine,
  VDIServer,
  VDIResourceGroup,
  VDIResource,
  VMListParams,
  CreateVMRequest,
  UpdateVMRequest,
  VMOperateRequest,
  BindUserRequest,
  VDIServerConfig,
  VMPageResponse,
  VDIPlatform,
  RunPosition,
  VDIStorage,
  VDINetwork,
} from "@/types/vdi";

// ==================== 虚拟机 API（完整VDI API集成）====================

const vmCrud = createResourceApi<VirtualMachine>({ basePath: "/vdi/vms" });

export const vmApi = {
  // 基础 CRUD — list/create 手写 override（签名与工厂不同构,Phase 94 D-08 KEEP）:
  // - list: VMListParams 是 interface,工厂 list 的 PageParams & Record<string,unknown>
  //   参数会拒绝 interface 变量直传(VirtualMachineList/index.tsx:186-191 实锤 TS2345)
  // - create: CreateVMRequest 含 vtp_id/count 等实体外字段且缺 vm_id 必选字段,双向不可赋值
  // get/update/delete 类型兼容,来自工厂显式 pick（Phase 100 D-100-1 spread 改 pick,
  // 幽灵 batch/statistics/searchOptions 不再挂到导出对象）
  get: vmCrud.get,
  update: vmCrud.update,
  delete: vmCrud.delete,

  list: async (params: VMListParams) => {
    return await post<VMPageResponse>("/vdi/vms/list", params);
  },

  create: async (data: CreateVMRequest) => {
    return await post<VirtualMachine>("/vdi/vms", data);
  },

  // VDI 操作（调用 VDI API）— 路由已于 Phase 100 D-100-10 补注册
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

  // 批量操作辅助方法 — 复用 /vdi/vms/operate（D-100-10 补路由后可用）
  batchOperate: async (vmIds: string[], action: string) => {
    return await post<void>("/vdi/vms/operate", {
      vm_ids: vmIds,
      action,
    });
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

// CreatePayload 排除集含 snake_case created_at/updated_at,
// VDIServerConfig 对 Partial<CreatePayload<VDIServer>> 可赋值（RESEARCH tsc 实测）;
// Phase 100 D-100-1 spread 改显式 pick,幽灵 batch/statistics/searchOptions 不再挂载。
const vdiServerCrud = createResourceApi<VDIServer>({ basePath: "/vdi/servers" });

export const vdiServerApi = {
  list: vdiServerCrud.list,
  get: vdiServerCrud.get,
  create: vdiServerCrud.create,
  update: vdiServerCrud.update,
  delete: vdiServerCrud.delete,

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
  VMListParams,
  CreateVMRequest,
  UpdateVMRequest,
  VMOperateRequest,
  BindUserRequest,
  VDIServerConfig,
  VDIPlatform,
  RunPosition,
  VDIStorage,
  VDINetwork,
};
