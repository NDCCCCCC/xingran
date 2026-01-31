/**
 * 运维管理模块 API
 */

import { post, postFormData } from './api';
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
} from '@/types';

// ==================== 通用 CRUD 工厂函数 ====================

interface CrudApiConfig {
  basePath: string;
}

function createCrudApi<T>(config: CrudApiConfig) {
  const { basePath } = config;

  return {
    list: async (params: PageParams & Record<string, unknown>) => {
      return await post<BaseResponse<PageResponse<T>>>(`${basePath}/list`, params);
    },

    get: async (id: string) => {
      return await post<BaseResponse<T>>(`${basePath}/${id}`, {});
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
}

const floorCrudApi = createCrudApi<Floor>({ basePath: '/ops/floor' });

export const floorApi = {
  ...floorCrudApi,

  tree: async () => {
    return await post<BaseResponse<Floor[]>>('/ops/floor/tree', {});
  },
};

// ==================== 工位管理 ====================

export interface WorkstationListParams extends PageParams {
  floorCode?: string;
  workstationCode?: string;
  name?: string;
  status?: number;
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
    return await post<BaseResponse<RoomPhoto[]>>(`/ops/rooms/photos/list/${roomId}`, {});
  },

  // 获取照片详情
  get: async (photoId: string) => {
    return await post<BaseResponse<RoomPhoto>>(`/ops/rooms/photos/${photoId}`, {});
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
    return await post<BaseResponse<RoomPhoto>>(`/ops/rooms/photos/primary?roomId=${roomId}`, {});
  },

  // 统计数量
  count: async (roomId: string) => {
    return await post<BaseResponse<{ count: number }>>(`/ops/rooms/photos/count?roomId=${roomId}`, {});
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

// 通用文件下载函数
async function downloadFile(url: string, filename: string): Promise<void> {
  const token = localStorage.getItem('accessToken');
  const response = await fetch(url, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error(`下载失败: ${filename}`);
  }

  const blob = await response.blob();
  const blobUrl = window.URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = blobUrl;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  window.URL.revokeObjectURL(blobUrl);
  document.body.removeChild(a);
}

// 从响应头提取文件名
function extractFilenameFromResponse(response: Response, defaultFilename: string): string {
  const contentDisposition = response.headers.get('content-disposition');
  if (!contentDisposition) {
    return defaultFilename;
  }

  const match = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
  if (match && match[1]) {
    return decodeURIComponent(match[1].replace(/['"]/g, ''));
  }

  return defaultFilename;
}

export const excelApi = {
  downloadTemplate: async (entityType: string) => {
    await downloadFile(`/api/v1/ops/${entityType}/template`, `${entityType}_template.xlsx`);
  },

  import: async (entityType: string, file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return await postFormData<BaseResponse<Record<string, unknown>>>(`/ops/${entityType}/import`, formData);
  },

  export: async (entityType: string, params: Record<string, unknown> = {}) => {
    const token = localStorage.getItem('accessToken');
    const response = await fetch(`/api/v1/ops/${entityType}/export`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(params),
    });

    if (!response.ok) {
      throw new Error('导出失败');
    }

    const blob = await response.blob();
    const filename = extractFilenameFromResponse(response, `${entityType}_${Date.now()}.xlsx`);
    const blobUrl = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = blobUrl;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(blobUrl);
    document.body.removeChild(a);
  },

  getStatusOptions: async (entityType: string) => {
    return await post<BaseResponse<Array<{ label: string; value: number }>>>(`/ops/${entityType}/status-options`, {});
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

import { getBaiduMapCache } from '@/utils/baiduMapCache';
import { cacheStats } from '@/utils/cacheStats';

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
  const baiduCache = getBaiduMapCache<GeocodeResult>();

  // 尝试从缓存获取
  const cached = baiduCache.get(cacheKey);
  if (cached) {
    cacheStats.recordStorageHit();
    console.log('✅ 地理编码缓存命中:', address);
    return cached;
  }

  // 调用后端API
  cacheStats.recordAPICall();
  console.log('🌐 调用后端地理编码API:', address);

  try {
    const result = await post<BaseResponse<GeocodeResult>>('/ops/building/geocode', { address });

    if (result.code === 0 && result.data) {
      baiduCache.set(cacheKey, result.data);
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
  const baiduCache = getBaiduMapCache<unknown>();
  const stats = cacheStats.getStats();
  const cacheInfo = baiduCache.getStats();

  return {
    ...stats,
    cacheSize: cacheInfo.totalSize,
    memoryCacheSize: cacheInfo.memorySize,
    storageCacheSize: cacheInfo.storageSize,
  };
};

/**
 * 清除地理编码缓存
 */
export const clearGeocodeCache = (address?: string) => {
  const baiduCache = getBaiduMapCache<unknown>();

  if (address) {
    const cacheKey = `${GEOCODE_CACHE_PREFIX}${address}`;
    baiduCache.delete(cacheKey);
    console.log('🗑️ 已清除地址缓存:', address);
  } else {
    baiduCache.clear();
    console.log('🗑️ 已清除所有地理编码缓存');
  }
};

/**
 * 重置地理编码统计
 */
export const resetGeocodeStats = () => {
  cacheStats.reset();
  console.log('🔄 已重置地理编码统计');
};
