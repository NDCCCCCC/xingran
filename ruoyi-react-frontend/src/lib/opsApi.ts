/**
 * 运维管理模块 API
 */

import axios from 'axios';
import type { AxiosInstance, AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import { post, get, postFormData } from './api';
import { getAccessToken } from '@/utils/authHelpers';
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
} from '@/types';

// ==================== 通用 CRUD 工厂函数 ====================

interface CrudApiConfig {
  basePath: string;
}

function createCrudApi<T>(config: CrudApiConfig) {
  const { basePath } = config;

  return {
    list: async (params: PageParams & Record<string, unknown>) => {
      return await post<PageResponse<T>>(`${basePath}/list`, params);
    },

    get: async (id: string) => {
      return await post<T>(`${basePath}/${id}`, {});
    },

    create: async (data: Partial<T>) => {
      return await post(basePath, data);
    },

    update: async (id: string, data: Partial<T>) => {
      return await post(`${basePath}/${id}/update`, data);
    },

    delete: async (id: string) => {
      return await post(`${basePath}/${id}/delete`, {});
    },

    batch: async (action: string, data: Record<string, unknown>) => {
      return await post(`${basePath}/batch`, { action, ...data });
    },
  };
}

// ==================== 楼宇管理 ====================

export interface BuildingListParams extends PageParams {
  name?: string;
  code?: string;
  status?: number;
  orgId?: string;  // 所属机构ID，用于按部门筛选
}

export const buildingApi = createCrudApi<Building>({ basePath: '/ops/building' });

// ==================== 楼层管理 ====================

export interface FloorListParams extends PageParams {
  buildingId?: string;
  floorNo?: string;
  name?: string;
  status?: number;
  orgId?: string;  // 所属机构ID，用于按部门筛选
}

const floorCrudApi = createCrudApi<Floor>({ basePath: '/ops/floor' });

export const floorApi = {
  ...floorCrudApi,

  tree: async () => {
    return await post<Floor[]>('/ops/floor/tree', {});
  },
};

// ==================== 工位管理 ====================

export interface WorkstationListParams extends PageParams {
  floorCode?: string;
  workstationCode?: string;
  name?: string;
  status?: number;
  orgId?: string;  // 所属机构ID，用于按部门筛选
}

const workstationCrudApi = createCrudApi<WorkstationOps>({ basePath: '/ops/workstation' });

export const workstationApi = {
  ...workstationCrudApi,

  updatePositions: async (items: { id: string; positionX: number; positionY: number }[]) => {
    return await post('/ops/workstation/positions', { items });
  },
};

// ==================== 机房管理 ====================

export interface ServerRoomListParams extends PageParams {
  buildingCode?: string;
  floorNo?: string;
  name?: string;
  status?: number;
}

export const serverRoomApi = createCrudApi<ServerRoom>({ basePath: '/ops/serverRoom' });

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
    formData.append('roomId', roomId);
    formData.append('primaryIndex', primaryIndex.toString());
    files.forEach(file => {
      formData.append('files', file);
    });

    return await postFormData<BaseResponse<RoomPhoto[]>>('/ops/rooms/photos/upload', formData);
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
    return await post('/ops/rooms/photos/batch-delete', { ids: photoIds });
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

export const roomDeviceApi = createCrudApi<RoomDevice>({ basePath: '/ops/roomDevice' });

// ==================== 专线管理 ====================

export interface DedicatedLineListParams extends PageParams {
  lineType?: string;
  isp?: string;
  status?: number;
}

export const dedicatedLineApi = createCrudApi<DedicatedLine>({ basePath: '/ops/dedicatedLine' });

// ==================== 信息点管理 ====================

export interface InfoPointListParams extends PageParams {
  workstationId?: string;
  code?: string;
  name?: string;
  infoPointType?: string;
  status?: number;
}

export const infoPointApi = createCrudApi<InfoPoint>({ basePath: '/ops/infoPoint' });

// ==================== Excel导入导出 ====================

// 专用于文件下载的 axios 实例：
// - 复用与主 API 客户端相同的 baseURL（去掉硬编码 /api/v1/ 前缀）
// - 不挂载响应拦截器，因为响应是 Blob 二进制流，无法 JSON 解析
// - 请求拦截器只做 Token 注入，行为与其他 CRUD 保持一致
const blobAxios: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://10.62.10.33:9000/api/v1',
  timeout: 30000,
});

blobAxios.interceptors.request.use(async (config: InternalAxiosRequestConfig) => {
  const token = await getAccessToken();
  if (token && config.headers) {
    config.headers.set('Authorization', `Bearer ${token}`);
  }
  return config;
});

// 从响应头提取文件名
function extractFilenameFromBlobResponse(
  response: AxiosResponse<Blob>,
  defaultFilename: string
): string {
  const contentDisposition: string | undefined = response.headers['content-disposition'];
  if (!contentDisposition) {
    return defaultFilename;
  }

  const match = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
  if (match && match[1]) {
    return decodeURIComponent(match[1].replace(/['"]/g, ''));
  }

  return defaultFilename;
}

// 触发浏览器下载 Blob
function triggerBrowserDownload(blob: Blob, filename: string): void {
  const blobUrl = window.URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = blobUrl;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  window.URL.revokeObjectURL(blobUrl);
  document.body.removeChild(a);
}

// 通用文件下载函数
async function downloadFile(url: string, filename: string): Promise<void> {
  const response = await blobAxios.get<Blob>(url, { responseType: 'blob' });

  if (response.status < 200 || response.status >= 300) {
    throw new Error(`下载失败: ${filename}`);
  }

  triggerBrowserDownload(response.data, filename);
}

export const excelApi = {
  downloadTemplate: async (entityType: string) => {
    await downloadFile(`/ops/${entityType}/template`, `${entityType}_template.xlsx`);
  },

  import: async (entityType: string, file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return await postFormData<BaseResponse<Record<string, unknown>>>(`/ops/${entityType}/import`, formData);
  },

  export: async (entityType: string, params: Record<string, unknown> = {}) => {
    const response = await blobAxios.post<Blob>(
      `/ops/${entityType}/export`,
      params,
      { responseType: 'blob' }
    );

    if (response.status < 200 || response.status >= 300) {
      throw new Error('导出失败');
    }

    const filename = extractFilenameFromBlobResponse(
      response,
      `${entityType}_${Date.now()}.xlsx`
    );
    triggerBrowserDownload(response.data, filename);
  },

  getStatusOptions: async (entityType: string) => {
    return await post<Array<{ label: string; value: number }>>(`/ops/${entityType}/status-options`, {});
  },
};

// ==================== 墙体管理（CAD平面图） ====================

export interface Wall {
  id: string;
  floorId: string;
  type: 'straight' | 'curved' | 'l_shaped' | 'polyline';
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

const wallCrudApi = createCrudApi<Wall>({ basePath: '/ops/walls' });

export const wallApi = {
  ...wallCrudApi,

  batch: async (action: string, ids: string[]) => {
    return await post('/ops/walls/batch', { action, ids });
  },
};

// ==================== 门管理（CAD平面图） ====================

export interface Door {
  id: string;
  floorId: string;
  wallId?: string;
  position: string;
  angle: number;
  type: 'single' | 'double' | 'sliding' | 'revolving' | 'emergency';
  direction: 'left' | 'right' | 'double' | 'sliding';
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

const doorCrudApi = createCrudApi<Door>({ basePath: '/ops/doors' });

export const doorApi = {
  ...doorCrudApi,

  batch: async (action: string, ids: string[]) => {
    return await post('/ops/doors/batch', { action, ids });
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

const floorPlanTextCrudApi = createCrudApi<FloorPlanText>({ basePath: '/ops/floor-plan-texts' });

export const floorPlanTextApi = {
  ...floorPlanTextCrudApi,

  batch: async (action: string, ids: string[]) => {
    return await post('/ops/floor-plan-texts/batch', { action, ids });
  },
};

// ==================== 地理编码（带缓存） ====================

import { getDualLevelCache } from '@/utils/dualLevelCache';

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
const GEOCODE_CACHE_PREFIX = 'geocode_';
const DEFAULT_CACHE_MAX_AGE = 7 * 24 * 60 * 60 * 1000; // 7天

/**
 * 带缓存的地理编码
 * 优先从缓存获取，缓存未命中时调用后端API
 */
export const geocodeWithCache = async (
  address: string,
  maxAge = DEFAULT_CACHE_MAX_AGE
): Promise<GeocodeResult | null> => {
  if (!address?.trim()) {
    return null;
  }

  const cacheKey = `${GEOCODE_CACHE_PREFIX}${address}`;
  const cache = getDualLevelCache<GeocodeResult>();

  // 尝试从缓存获取
  const cached = cache.get(cacheKey);
  if (cached) {
    console.log('✅ 地理编码缓存命中:', address);
    return cached;
  }

  // 调用后端API
  console.log('🌐 调用后端地理编码API:', address);

  try {
    const result = await post<GeocodeResult>('/ops/building/geocode', { address });

    if (result.code === 0 && result.data) {
      cache.set(cacheKey, result.data);
      console.log('💾 地理编码结果已缓存:', address);
      return result.data;
    }

    return null;
  } catch (error) {
    console.error('❌ 地理编码失败:', error);
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
    console.log('🗑️ 已清除地址缓存:', address);
  } else {
    cache.clear();
    console.log('🗑️ 已清除所有地理编码缓存');
  }
};

/**
 * 重置地理编码统计
 */
export const resetGeocodeStats = () => {
  const cache = getDualLevelCache<unknown>();
  cache.resetStats();
  console.log('🔄 已重置地理编码统计');
};

// ==================== 资产管理 ====================

const assetCrudApi = createCrudApi<Asset>({ basePath: '/ops/asset' });

export const assetApi = {
  ...assetCrudApi,

  // 根据序列号查询资产信息
  searchBySerial: async (serial: string) => {
    return await get<Asset>(`/ops/asset/search-by-serial/${serial}`, {});
  },

  // 获取设备类型列表（动态数据）
  getDeviceTypes: async () => {
    return await post<{ value: string; count: number }[]>('/ops/asset/device-types', {});
  },

  // 获取设备种类列表（动态数据）
  getDeviceCategories: async () => {
    return await post<{ value: string; count: number }[]>('/ops/asset/device-categories', {});
  },

  // 获取状态列表（动态数据）
  getStatusValues: async () => {
    return await post<{ value: string; count: number }[]>('/ops/asset/status-values', {});
  },
  // Excel导入导出通过 SetupExcelRouter 注册在 /ops/asset 路由组下：
  // - GET  /ops/asset/template  (下载模板)
  // - POST /ops/asset/import    (上传导入)
  // - POST /ops/asset/export    (导出数据)

  excel: {
    template: async () => {
      return await post('/ops/asset/template', {});
    },

    import: async (file: File) => {
      const formData = new FormData();
      formData.append('file', file);
      return await postFormData('/ops/asset/import', formData);
    },

    export: async (params: AssetListParams & Record<string, unknown>) => {
      const response = await blobAxios.post<Blob>(
        '/ops/asset/export',
        params,
        { responseType: 'blob' }
      );

      if (response.status < 200 || response.status >= 300) {
        throw new Error('导出失败');
      }

      const filename = `资产列表_${Date.now()}.xlsx`;
      triggerBrowserDownload(response.data, filename);

      return { code: 0, message: '导出成功', data: null };
    },
  },
};

// ==================== 工位设备关联 ====================

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

  // 查询工位下全部设备（手动持久化部分），与 getManual 等价；保留为兼容旧调用
  getByWorkstation: async (workstationId: string) => {
    return await post<WorkstationDevice[]>(`/ops/workstation-device/${workstationId}`, {});
  },

  // 手动添加设备
  addManual: async (data: DeviceFormData) => {
    return await post<WorkstationDevice>('/ops/workstation-device/manual', data);
  },

  // 同步 AD 设备到数据库（保留旧接口，新流程使用 getAD 实时查询）
  syncAD: async (workstationId: string) => {
    return await post('/ops/workstation-device/sync-ad', { workstation_id: workstationId });
  },

  // 同步资产设备到数据库（保留旧接口，新流程使用 getAsset 实时查询）
  syncAsset: async (workstationId: string) => {
    return await post('/ops/workstation-device/sync-asset', { workstation_id: workstationId });
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
