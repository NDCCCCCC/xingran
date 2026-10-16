/**
 * 运维管理模块 API
 */

import { post, get, postFormData } from "./api";
import { createResourceApi } from "./apiFactory";
import { downloadFile, downloadFilePost } from "./download";
import type {
  BaseResponse,
  Building,
  Floor,
  WorkstationOps,
  ServerRoom,
  RoomPhoto,
  RoomDevice,
  DedicatedLine,
  InfoPoint,
  PageParams,
  PageResponse,
  Asset,
  AssetListParams,
  WorkstationDevice,
  DeviceFormData,
} from "@/types";

// ==================== 通用类型 ====================

/**
 * 通用下拉选项 — 定义已上移 src/lib/apiFactory.ts（Phase 94 D-03 单一权威）,
 * 此处 re-export 兜底既有 import 路径（当前零外部消费方,防新增误引直接落工厂权威）。
 */
export type { DropdownOption } from "./apiFactory";

// ==================== 楼宇管理 ====================

export interface BuildingListParams extends PageParams {
  name?: string;
  code?: string;
  status?: number;
  orgId?: string; // 所属机构ID，用于按部门筛选
}

export const buildingApi = createResourceApi<Building>({ basePath: "/ops/building" });

// ==================== 楼层管理 ====================

export interface FloorListParams extends PageParams {
  buildingId?: string;
  floorNo?: string;
  name?: string;
  status?: number;
  orgId?: string; // 所属机构ID，用于按部门筛选
}

const floorCrudApi = createResourceApi<Floor>({ basePath: "/ops/floor" });

export const floorApi = {
  ...floorCrudApi,

  tree: async () => {
    return await post<Floor[]>("/ops/floor/tree", {});
  },
};

// ==================== 工位管理 ====================

export interface WorkstationListParams extends PageParams {
  floorCode?: string;
  workstationCode?: string;
  name?: string;
  status?: number;
  orgId?: string; // 所属机构ID，用于按部门筛选
}

const workstationCrudApi = createResourceApi<WorkstationOps>({ basePath: "/ops/workstation" });

// Phase 39: 工位部门下拉选项(后端 union: orgId 子孙 + alias 映射)
// JSON tag 与后端 Plan 39-03 的 DeptOption 完全对齐 (deptId / deptName / isAlias)
export interface DeptOption {
  deptId: string;
  deptName: string;
  isAlias: boolean;
}

export const workstationApi = {
  ...workstationCrudApi,

  updatePositions: async (items: { id: string; positionX: number; positionY: number }[]) => {
    return await post("/ops/workstation/positions", { items });
  },

  // Phase 39: 工位编辑"所属部门"下拉数据源(orgId 子孙 + alias union)
  // 返回 DeptOption[], isAlias=true 表示该条目是 alias 映射而来,前端应追加 [映射] 后缀
  deptOptions: async (orgId: string): Promise<DeptOption[]> => {
    const res = await post<DeptOption[]>("/ops/workstation/dept-options", { orgId });
    return res.data ?? [];
  },

  // V130R-09 D-03-6: 楼层全部工位(CAD/3D/平面图专用,无分页,替代 pageSize:1000 List 反模式)
  getFloorWorkstationsAll: async (floorId: string): Promise<WorkstationOps[]> => {
    const res = await get<{ list: WorkstationOps[]; total: number }>(
      `/ops/workstation/${floorId}/workstations-all`
    );
    return res.data?.list ?? [];
  },

  // Sprint 1 Fix #4 (Vercel async-parallel): 批量楼层工位计数,
  // 替代 BuildingView3D 原来的 1+N workstationApi.list(pageSize:1) 模式
  countsByFloor: async (floorIds: string[]): Promise<{ counts: Record<string, number> }> => {
    const res = await post<{ counts: Record<string, number> }>("/ops/workstation/counts-by-floor", {
      floorIds,
    });
    return { counts: res.data?.counts ?? {} };
  },
};

// ==================== 工位部门物理位置映射 (Phase 39) ====================

// Phase 39: alias 表记录(deptId ↔ locationId 映射, scope 限定场景)
export interface LocationAlias {
  id: string;
  deptId: string;
  locationId: string;
  scope: string;
  remark?: string;
  createdAt?: string;
  updatedAt?: string;
  /** 原组织部门名(后端 List JOIN sys_dept 产出,仅列表接口返回) */
  originDeptName?: string;
  /** 物理位置部门名(后端 List JOIN sys_dept 产出,仅列表接口返回) */
  locationDeptName?: string;
}

// Phase 39: 工位部门物理位置映射管理 (Drawer UI 使用)
export const locationAliasApi = {
  list: async (params: { pageNum?: number; pageSize?: number } = {}) => {
    return await post<PageResponse<LocationAlias>>("/ops/location-alias/list", {
      pageNum: params.pageNum ?? 1,
      pageSize: params.pageSize ?? 10,
    });
  },

  create: async (data: { deptId: string; locationId: string; scope?: string; remark?: string }) => {
    return await post<LocationAlias>("/ops/location-alias", {
      ...data,
      scope: data.scope ?? "workstation",
    });
  },

  update: async (id: string, data: { deptId?: string; locationId?: string; remark?: string }) => {
    return await post(`/ops/location-alias/${id}/update`, data);
  },

  delete: async (id: string) => {
    return await post(`/ops/location-alias/${id}/delete`, {});
  },
};

// ==================== 机房管理 ====================

export interface ServerRoomListParams extends PageParams {
  buildingCode?: string;
  floorNo?: string;
  name?: string;
  status?: number;
}

export const serverRoomApi = createResourceApi<ServerRoom>({ basePath: "/ops/serverRoom" });

// ==================== 机房照片管理 ====================

export const roomPhotoApi = {
  // 获取机房照片列表
  list: async (roomId: string) => {
    return await post<RoomPhoto[]>(`/ops/rooms/photos/list/${roomId}`, {});
  },

  // 获取照片详情
  get: async (photoId: string) => {
    return await post<RoomPhoto>(`/ops/rooms/photos/${photoId}`, {});
  },

  // 上传照片
  upload: async (roomId: string, files: File[], primaryIndex = 0) => {
    const formData = new FormData();
    formData.append("roomId", roomId);
    formData.append("primaryIndex", primaryIndex.toString());
    files.forEach((file) => {
      formData.append("files", file);
    });

    return await postFormData<BaseResponse<RoomPhoto[]>>("/ops/rooms/photos/upload", formData);
  },

  // 设置主图
  setPrimary: async (photoId: string) => {
    return await post(`/ops/rooms/photos/${photoId}/primary`, {});
  },

  // 更新描述
  updateDescription: async (photoId: string, description: string) => {
    return await post(`/ops/rooms/photos/${photoId}/description`, { description });
  },

  // 删除照片
  delete: async (photoId: string) => {
    return await post(`/ops/rooms/photos/${photoId}`, {});
  },

  // 批量删除
  batchDelete: async (photoIds: string[]) => {
    return await post("/ops/rooms/photos/batch-delete", { ids: photoIds });
  },

  // 获取主图
  getPrimary: async (roomId: string) => {
    return await post<RoomPhoto>(`/ops/rooms/photos/primary?roomId=${roomId}`, {});
  },

  // 统计数量
  count: async (roomId: string) => {
    return await post<{ count: number }>(`/ops/rooms/photos/count?roomId=${roomId}`, {});
  },
};

// ==================== 机房设备管理 ====================

export interface RoomDeviceListParams extends PageParams {
  roomCode?: string;
  deviceCode?: string;
  deviceName?: string;
  deviceType?: string;
  status?: number;
}

export const roomDeviceApi = createResourceApi<RoomDevice>({ basePath: "/ops/roomDevice" });

// ==================== 专线管理 ====================

export interface DedicatedLineListParams extends PageParams {
  lineType?: string;
  isp?: string;
  status?: number;
}

export const dedicatedLineApi = createResourceApi<DedicatedLine>({
  basePath: "/ops/dedicatedLine",
});

// ==================== 信息点管理 ====================

export interface InfoPointListParams extends PageParams {
  workstationId?: string;
  code?: string;
  name?: string;
  infoPointType?: string;
  status?: number;
}

export const infoPointApi = createResourceApi<InfoPoint>({ basePath: "/ops/infoPoint" });

// ==================== Excel导入导出 ====================
//
// blob 下载基建（blobAxios / extractFilenameFromBlobResponse / triggerBrowserDownload /
// downloadFile）已上移 src/lib/download.ts（Phase 94 D-04 单一权威）,本文件仅消费。

export const excelApi = {
  downloadTemplate: async (entityType: string) => {
    await downloadFile(`/ops/${entityType}/template`, `${entityType}_template.xlsx`);
  },

  import: async (entityType: string, file: File) => {
    const formData = new FormData();
    formData.append("file", file);
    return await postFormData<BaseResponse<Record<string, unknown>>>(
      `/ops/${entityType}/import`,
      formData
    );
  },

  export: async (entityType: string, params: Record<string, unknown> = {}) => {
    await downloadFilePost(`/ops/${entityType}/export`, params, `${entityType}_${Date.now()}.xlsx`);
  },

  getStatusOptions: async (entityType: string) => {
    return await post<Array<{ label: string; value: number }>>(
      `/ops/${entityType}/status-options`,
      {}
    );
  },
};

// ==================== 部门映射表下载 (quick 260713-df0) ====================
//
// 工位导入辅助端点: 后端 GET /ops/workstation/dept-mapping-template 返回
// sys_dept 全部启用部门的 dept_name | dept_code 映射表, 用户在 Excel 中填写
// 部门代码时不必再翻部门列表查 dept_code。
//
// 复用 excelApi 同款 downloadFile 链路(blobAxios + triggerBrowserDownload),
// 无需新写 fetch/blob/URL.createObjectURL。
export const deptApi = {
  exportMapping: async () => {
    await downloadFile("/ops/workstation/dept-mapping-template", `dept_mapping_${Date.now()}.xlsx`);
  },
};

// ==================== 墙体管理（CAD平面图） ====================

export interface Wall {
  id: string;
  floorId: string;
  type: "straight" | "curved" | "l_shaped" | "polyline";
  points: string;
  thickness: number;
  height: number;
  color: string;
  name?: string;
  remark?: string;
}

export interface WallListParams extends PageParams {
  floorId?: string;
  type?: string;
  name?: string;
}

const wallCrudApi = createResourceApi<Wall>({ basePath: "/ops/walls" });

export const wallApi = {
  ...wallCrudApi,

  batch: async (action: string, ids: string[]) => {
    return await post("/ops/walls/batch", { action, ids });
  },
};

// ==================== 门管理（CAD平面图） ====================

export interface Door {
  id: string;
  floorId: string;
  wallId?: string;
  position: string;
  angle: number;
  type: "single" | "double" | "sliding" | "revolving" | "emergency";
  direction: "left" | "right" | "double" | "sliding";
  width: number;
  length: number;
  color: string;
  name?: string;
  remark?: string;
}

export interface DoorListParams extends PageParams {
  floorId?: string;
  wallId?: string;
  type?: string;
  name?: string;
}

const doorCrudApi = createResourceApi<Door>({ basePath: "/ops/doors" });

export const doorApi = {
  ...doorCrudApi,

  batch: async (action: string, ids: string[]) => {
    return await post("/ops/doors/batch", { action, ids });
  },
};

// ==================== 平面图文本管理（CAD平面图） ====================

export interface FloorPlanText {
  id: string;
  floorId: string;
  position: string;
  content: string;
  fontSize: number;
  color: string;
  fontFamily?: string;
  fontWeight?: string;
  fontStyle?: string;
  angle?: number;
  remark?: string;
}

export interface FloorPlanTextListParams extends PageParams {
  floorId?: string;
  content?: string;
}

const floorPlanTextCrudApi = createResourceApi<FloorPlanText>({
  basePath: "/ops/floor-plan-texts",
});

export const floorPlanTextApi = {
  ...floorPlanTextCrudApi,

  batch: async (action: string, ids: string[]) => {
    return await post("/ops/floor-plan-texts/batch", { action, ids });
  },
};

// ==================== 地理编码（带缓存） ====================

import { getDualLevelCache } from "@/utils/dualLevelCache";

export interface GeocodeResult {
  lng: number;
  lat: number;
  formattedAddress?: string;
  addressComponent?: {
    province: string;
    city: string;
    district: string;
    street: string;
  };
  precise?: number;
  level?: string;
}

// 缓存键前缀
const GEOCODE_CACHE_PREFIX = "geocode_";
const DEFAULT_CACHE_MAX_AGE = 7 * 24 * 60 * 60 * 1000; // 7天

/**
 * 带缓存的地理编码
 * 优先从缓存获取，缓存未命中时调用后端API
 */
export const geocodeWithCache = async (
  address: string,
  _maxAge = DEFAULT_CACHE_MAX_AGE
): Promise<GeocodeResult | null> => {
  if (!address?.trim()) {
    return null;
  }

  const cacheKey = `${GEOCODE_CACHE_PREFIX}${address}`;
  const cache = getDualLevelCache<GeocodeResult>();

  // 尝试从缓存获取
  const cached = cache.get(cacheKey);
  if (cached) {
    return cached;
  }

  try {
    const result = await post<GeocodeResult>("/ops/building/geocode", { address });

    if (result.code === 0 && result.data) {
      cache.set(cacheKey, result.data);
      return result.data;
    }

    return null;
  } catch (error) {
    console.error("❌ 地理编码失败:", error);
    return null;
  }
};

/**
 * 批量地理编码（带缓存）
 */
export const batchGeocodeWithCache = async (
  addresses: string[]
): Promise<Map<string, GeocodeResult>> => {
  const results = new Map<string, GeocodeResult>();

  await Promise.all(
    addresses.map(async (address) => {
      const result = await geocodeWithCache(address);
      if (result) {
        results.set(address, result);
      }
    })
  );

  return results;
};

/**
 * 获取地理编码缓存统计
 */
export const getGeocodeStats = () => {
  const cache = getDualLevelCache<unknown>();
  const cacheInfo = cache.getStats();

  return {
    // 直接使用 DualLevelCache 的统计
    ...cacheInfo,
    // 兼容旧字段名（apiCalls = misses）
    apiCalls: cacheInfo.misses,
    // 保留字段名兼容性
    cacheSize: cacheInfo.totalSize,
    memoryCacheSize: cacheInfo.memorySize,
    storageCacheSize: cacheInfo.storageSize,
  };
};

/**
 * 清除地理编码缓存
 */
export const clearGeocodeCache = (address?: string) => {
  const cache = getDualLevelCache<unknown>();

  if (address) {
    const cacheKey = `${GEOCODE_CACHE_PREFIX}${address}`;
    cache.delete(cacheKey);
  } else {
    cache.clear();
  }
};

/**
 * 重置地理编码统计
 */
export const resetGeocodeStats = () => {
  const cache = getDualLevelCache<unknown>();
  cache.resetStats();
};

// ==================== 资产管理 ====================

const assetCrudApi = createResourceApi<Asset>({ basePath: "/ops/asset" });

export const assetApi = {
  ...assetCrudApi,

  // 根据序列号查询资产信息
  searchBySerial: async (serial: string) => {
    return await get<Asset>(`/ops/asset/search-by-serial/${serial}`, {});
  },

  // 获取设备类型列表（动态数据）
  getDeviceTypes: async () => {
    return await post<{ value: string; count: number }[]>("/ops/asset/device-types", {});
  },

  // 获取设备种类列表（动态数据）
  getDeviceCategories: async () => {
    return await post<{ value: string; count: number }[]>("/ops/asset/device-categories", {});
  },

  // 获取状态列表（动态数据）
  getStatusValues: async () => {
    return await post<{ value: string; count: number }[]>("/ops/asset/status-values", {});
  },

  // 资产统计(专用 COUNT 端点,真实 status/nbf_status 计数)
  statistics: async () => {
    const res = await post<{ total: number; normal: number; stopped: number; nbf: number }>(
      "/ops/asset/statistics",
      {}
    );
    return res.data;
  },
  // Excel导入导出通过 SetupExcelRouter 注册在 /ops/asset 路由组下：
  // - GET  /ops/asset/template  (下载模板)
  // - POST /ops/asset/import    (上传导入)
  // - POST /ops/asset/export    (导出数据)

  excel: {
    // v129-recheck WR-05: 后端 SetupExcelRouter 只注册 GET /template，旧实现发
    // POST 必 405（潜伏死方法，唯一引用为 mock 测试）——对齐 downloadTemplate
    // 的 GET 下载链
    template: async () => {
      await downloadFile("/ops/asset/template", "asset_template.xlsx");
    },

    import: async (file: File) => {
      const formData = new FormData();
      formData.append("file", file);
      return await postFormData("/ops/asset/import", formData);
    },

    export: async (params: AssetListParams & Record<string, unknown>) => {
      await downloadFilePost("/ops/asset/export", params, `资产列表_${Date.now()}.xlsx`);

      return { code: 0, message: "导出成功", data: null };
    },
  },
};

// ==================== 工位设备关联 ====================

// Phase 48 Wave 3: 从属组件清单 read-only API(D-07 前端展示)。
// 后端 GET /ops/asset/components?parentAssetId=<uuid> 在 router.go:691-697
// asset 组内联注册,复用 ops:asset:list 组级 middleware。
export const componentApi = {
  // 列出某父交换机下的所有从属组件(板卡/引擎/电源/风扇/光模块)。
  // 返回 Asset[] 数组(后端已加 component_type IS NOT NULL 过滤)。
  list: async (parentAssetId: string) => {
    const res = await get<{ list: Asset[]; total: number }>("/ops/asset/components", {
      parentAssetId,
    });
    return res;
  },
};

export const workstationDeviceApi = {
  // 查询工位下手动持久化的设备
  getManual: async (workstationId: string) => {
    return await post<WorkstationDevice[]>(`/ops/workstation-device/${workstationId}`, {});
  },

  // 实时查询工位关联的域控设备
  getAD: async (workstationId: string) => {
    return await post<WorkstationDevice[]>(`/ops/workstation-device/${workstationId}/ad`, {});
  },

  // 实时查询工位关联的资产设备
  getAsset: async (workstationId: string) => {
    return await post<WorkstationDevice[]>(`/ops/workstation-device/${workstationId}/asset`, {});
  },

  // 实时查询工位关联的物理链路设备(Phase 45 R5)
  // MAC→port→infoPoint→workstation 反推链,资产字段优先、AD 字段补全
  getPhysical: async (workstationId: string) => {
    return await post<WorkstationDevice[]>(`/ops/workstation-device/${workstationId}/physical`, {});
  },

  // 查询工位下全部设备（手动持久化部分），与 getManual 等价；保留为兼容旧调用
  getByWorkstation: async (workstationId: string) => {
    return await post<WorkstationDevice[]>(`/ops/workstation-device/${workstationId}`, {});
  },

  // 手动添加设备
  addManual: async (data: DeviceFormData) => {
    return await post<WorkstationDevice>("/ops/workstation-device/manual", data);
  },

  // 同步 AD 设备到数据库（保留旧接口，新流程使用 getAD 实时查询）
  syncAD: async (workstationId: string) => {
    return await post("/ops/workstation-device/sync-ad", { workstation_id: workstationId });
  },

  // 同步资产设备到数据库（保留旧接口，新流程使用 getAsset 实时查询）
  syncAsset: async (workstationId: string) => {
    return await post("/ops/workstation-device/sync-asset", { workstation_id: workstationId });
  },

  // 将 AD/资产设备设为主设备并保存为手动设备
  // 注：后端在保存前会按 device_serial 自动合并 AD + 资产实时数据，无需前端额外传参
  setPrimaryAndSave: async (
    deviceId: string,
    data: {
      workstationId: string;
      deviceSerial: string;
      deviceName: string;
      deviceModel?: string;
      deviceType?: string;
      macAddress?: string;
      ipAddress?: string;
      responsibleUser?: string;
    }
  ) => {
    return await post(`/ops/workstation-device/${deviceId}/set-primary-and-save`, data);
  },

  // 更新设备
  update: async (id: string, data: Partial<DeviceFormData>) => {
    return await post(`/ops/workstation-device/${id}/update`, data);
  },

  // 删除设备
  delete: async (id: string) => {
    return await post(`/ops/workstation-device/${id}/delete`, {});
  },

  // 设置主设备（仅对已持久化的手动设备生效）
  setPrimary: async (id: string) => {
    return await post(`/ops/workstation-device/${id}/set-primary`, {});
  },
};
