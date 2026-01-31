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

// ==================== 楼宇管理 ====================

export interface BuildingListParams extends PageParams {
  name?: string;
  code?: string;
  status?: number;
  orgId?: string;  // 所属机构ID，用于按部门筛选
}

export const buildingApi = {
  // 查询楼宇列表
  list: async (params: BuildingListParams) => {
    return await post<BaseResponse<PageResponse<Building>>>('/ops/building/list', params);
  },

  // 获取楼宇详情
  get: async (id: string) => {
    return await post<BaseResponse<Building>>('/ops/building/' + id, {});
  },

  // 创建楼宇
  create: async (data: Partial<Building>) => {
    return await post('/ops/building', data);
  },

  // 更新楼宇
  update: async (id: string, data: Partial<Building>) => {
    return await post('/ops/building/' + id + '/update', data);
  },

  // 删除楼宇
  delete: async (id: string) => {
    return await post('/ops/building/' + id + '/delete', {});
  },

  // 批量操作
  batch: async (action: string, data: Record<string, unknown>) => {
    return await post('/ops/building/batch', { action, ...data });
  },
};

// ==================== 楼层管理 ====================

export interface FloorListParams extends PageParams {
  buildingId?: string;
  floorNo?: string;
  name?: string;
  status?: number;
}

export const floorApi = {
  list: async (params: FloorListParams) => {
    return await post<BaseResponse<PageResponse<Floor>>>('/ops/floor/list', params);
  },

  get: async (id: string) => {
    return await post<BaseResponse<Floor>>('/ops/floor/' + id, {});
  },

  create: async (data: Partial<Floor>) => {
    return await post('/ops/floor', data);
  },

  update: async (id: string, data: Partial<Floor>) => {
    return await post('/ops/floor/' + id + '/update', data);
  },

  delete: async (id: string) => {
    return await post('/ops/floor/' + id + '/delete', {});
  },

  // 获取楼层树
  tree: async () => {
    return await post<BaseResponse<Floor[]>>('/ops/floor/tree', {});
  },

  batch: async (action: string, data: Record<string, unknown>) => {
    return await post('/ops/floor/batch', { action, ...data });
  },
};

// ==================== 工位管理 ====================

export interface WorkstationListParams extends PageParams {
  floorCode?: string;
  workstationCode?: string;
  name?: string;
  status?: number;
}

export const workstationApi = {
  list: async (params: WorkstationListParams) => {
    return await post<BaseResponse<PageResponse<WorkstationOps>>>('/ops/workstation/list', params);
  },

  get: async (id: string) => {
    return await post<BaseResponse<WorkstationOps>>('/ops/workstation/' + id, {});
  },

  create: async (data: Partial<WorkstationOps>) => {
    return await post('/ops/workstation', data);
  },

  update: async (id: string, data: Partial<WorkstationOps>) => {
    return await post('/ops/workstation/' + id + '/update', data);
  },

  delete: async (id: string) => {
    return await post('/ops/workstation/' + id + '/delete', {});
  },

  // 批量更新位置
  updatePositions: async (items: { id: string; positionX: number; positionY: number }[]) => {
    return await post('/ops/workstation/positions', { items });
  },

  batch: async (action: string, data: Record<string, unknown>) => {
    return await post('/ops/workstation/batch', { action, ...data });
  },
};

// ==================== 机房管理 ====================

export interface ServerRoomListParams extends PageParams {
  buildingCode?: string;
  floorNo?: string;
  name?: string;
  status?: number;
}

export const serverRoomApi = {
  list: async (params: ServerRoomListParams) => {
    return await post<BaseResponse<PageResponse<ServerRoom>>>('/ops/serverRoom/list', params);
  },

  get: async (id: string) => {
    return await post<BaseResponse<ServerRoom>>('/ops/serverRoom/' + id, {});
  },

  create: async (data: Partial<ServerRoom>) => {
    return await post('/ops/serverRoom', data);
  },

  update: async (id: string, data: Partial<ServerRoom>) => {
    return await post('/ops/serverRoom/' + id + '/update', data);
  },

  delete: async (id: string) => {
    return await post('/ops/serverRoom/' + id + '/delete', {});
  },

  batch: async (action: string, data: Record<string, unknown>) => {
    return await post('/ops/serverRoom/batch', { action, ...data });
  },
};

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

export const roomDeviceApi = {
  list: async (params: RoomDeviceListParams) => {
    return await post<BaseResponse<PageResponse<RoomDevice>>>('/ops/roomDevice/list', params);
  },

  get: async (id: string) => {
    return await post<BaseResponse<RoomDevice>>('/ops/roomDevice/' + id, {});
  },

  create: async (data: Partial<RoomDevice>) => {
    return await post('/ops/roomDevice', data);
  },

  update: async (id: string, data: Partial<RoomDevice>) => {
    return await post('/ops/roomDevice/' + id + '/update', data);
  },

  delete: async (id: string) => {
    return await post('/ops/roomDevice/' + id + '/delete', {});
  },

  batch: async (action: string, data: Record<string, unknown>) => {
    return await post('/ops/roomDevice/batch', { action, ...data });
  },
};

// ==================== 专线管理 ====================

export interface DedicatedLineListParams extends PageParams {
  lineType?: string;
  isp?: string;
  status?: number;
}

export const dedicatedLineApi = {
  list: async (params: DedicatedLineListParams) => {
    return await post<BaseResponse<PageResponse<DedicatedLine>>>('/ops/dedicatedLine/list', params);
  },

  get: async (id: string) => {
    return await post<BaseResponse<DedicatedLine>>('/ops/dedicatedLine/' + id, {});
  },

  create: async (data: Partial<DedicatedLine>) => {
    return await post('/ops/dedicatedLine', data);
  },

  update: async (id: string, data: Partial<DedicatedLine>) => {
    return await post('/ops/dedicatedLine/' + id + '/update', data);
  },

  delete: async (id: string) => {
    return await post('/ops/dedicatedLine/' + id + '/delete', {});
  },

  batch: async (action: string, data: Record<string, unknown>) => {
    return await post('/ops/dedicatedLine/batch', { action, ...data });
  },
};

// ==================== 信息点管理 ====================

export interface InfoPointListParams extends PageParams {
  workstationId?: string;
  code?: string;
  name?: string;
  infoPointType?: string;
  status?: number;
}

export const infoPointApi = {
  list: async (params: InfoPointListParams) => {
    return await post<BaseResponse<PageResponse<InfoPoint>>>('/ops/infoPoint/list', params);
  },

  get: async (id: string) => {
    return await post<BaseResponse<InfoPoint>>('/ops/infoPoint/' + id, {});
  },

  create: async (data: Partial<InfoPoint>) => {
    return await post('/ops/infoPoint', data);
  },

  update: async (id: string, data: Partial<InfoPoint>) => {
    return await post('/ops/infoPoint/' + id + '/update', data);
  },

  delete: async (id: string) => {
    return await post('/ops/infoPoint/' + id + '/delete', {});
  },

  batch: async (action: string, data: Record<string, unknown>) => {
    return await post('/ops/infoPoint/batch', { action, ...data });
  },
};

// ==================== Excel导入导出 ====================

export const excelApi = {
  // 下载模板
  downloadTemplate: async (entityType: string) => {
    const token = localStorage.getItem('accessToken');
    const response = await fetch(`/api/v1/ops/${entityType}/template`, {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      throw new Error('下载模板失败');
    }

    const blob = await response.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${entityType}_template.xlsx`;
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
  },

  // 导入数据
  import: async (entityType: string, file: File) => {
    const formData = new FormData();
    formData.append('file', file);

    return await postFormData<BaseResponse<Record<string, unknown>>>(`/ops/${entityType}/import`, formData);
  },

  // 导出数据
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
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;

    // 从响应头获取文件名
    const contentDisposition = response.headers.get('content-disposition');
    let filename = `${entityType}_${Date.now()}.xlsx`;
    if (contentDisposition) {
      const match = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
      if (match && match[1]) {
        filename = decodeURIComponent(match[1].replace(/['"]/g, ''));
      }
    }

    a.download = filename;
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
  },

  // 获取状态选项
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

export const wallApi = {
  list: async (params: WallListParams) => {
    return await post<BaseResponse<PageResponse<Wall>>>('/ops/walls/list', params);
  },

  get: async (id: string) => {
    return await post<BaseResponse<Wall>>('/ops/walls/' + id, {});
  },

  create: async (data: Partial<Wall>) => {
    return await post('/ops/walls', data);
  },

  update: async (id: string, data: Partial<Wall>) => {
    return await post('/ops/walls/' + id + '/update', data);
  },

  delete: async (id: string) => {
    return await post('/ops/walls/' + id + '/delete', {});
  },

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

export const doorApi = {
  list: async (params: DoorListParams) => {
    return await post<BaseResponse<PageResponse<Door>>>('/ops/doors/list', params);
  },

  get: async (id: string) => {
    return await post<BaseResponse<Door>>('/ops/doors/' + id, {});
  },

  create: async (data: Partial<Door>) => {
    return await post('/ops/doors', data);
  },

  update: async (id: string, data: Partial<Door>) => {
    return await post('/ops/doors/' + id + '/update', data);
  },

  delete: async (id: string) => {
    return await post('/ops/doors/' + id + '/delete', {});
  },

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

export const floorPlanTextApi = {
  list: async (params: FloorPlanTextListParams) => {
    return await post<BaseResponse<PageResponse<FloorPlanText>>>('/ops/floor-plan-texts/list', params);
  },

  get: async (id: string) => {
    return await post<BaseResponse<FloorPlanText>>('/ops/floor-plan-texts/' + id, {});
  },

  create: async (data: Partial<FloorPlanText>) => {
    return await post('/ops/floor-plan-texts', data);
  },

  update: async (id: string, data: Partial<FloorPlanText>) => {
    return await post('/ops/floor-plan-texts/' + id + '/update', data);
  },

  delete: async (id: string) => {
    return await post('/ops/floor-plan-texts/' + id + '/delete', {});
  },

  batch: async (action: string, ids: string[]) => {
    return await post('/ops/floor-plan-texts/batch', { action, ids });
  },
};

// ==================== 地理编码（带缓存） ====================

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

import { getBaiduMapCache } from '@/utils/baiduMapCache';
import { cacheStats } from '@/utils/cacheStats';

/**
 * 带缓存的地理编码
 * 优先从缓存获取，缓存未命中时调用后端API
 *
 * @param address 地址字符串
 * @param maxAge 缓存最大有效期（毫秒），默认7天
 * @returns 地理编码结果或null
 */
export const geocodeWithCache = async (
  address: string,
  maxAge = 7 * 24 * 60 * 60 * 1000
): Promise<GeocodeResult | null> => {
  if (!address || address.trim() === '') {
    return null;
  }

  const cacheKey = `geocode_${address}`;
  const baiduCache = getBaiduMapCache<GeocodeResult>();

  // 1. 尝试从缓存获取
  const cached = baiduCache.get(cacheKey);
  if (cached) {
    cacheStats.recordStorageHit();
    console.log('✅ 地理编码缓存命中:', address);
    return cached;
  }

  // 2. 调用后端API
  cacheStats.recordAPICall();
  console.log('🌐 调用后端地理编码API:', address);

  try {
    const result = await post<BaseResponse<GeocodeResult>>('/ops/building/geocode', { address });

    if (result.code === 0 && result.data) {
      // 3. 存入缓存
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
 *
 * @param addresses 地址数组
 * @returns 地址到坐标的映射
 */
export const batchGeocodeWithCache = async (
  addresses: string[]
): Promise<Map<string, GeocodeResult>> => {
  const results = new Map<string, GeocodeResult>();

  // 并发处理所有地址
  const promises = addresses.map(async (address) => {
    const result = await geocodeWithCache(address);
    if (result) {
      results.set(address, result);
    }
  });

  await Promise.all(promises);
  return results;
};

/**
 * 获取地理编码缓存统计
 */
export const getGeocodeStats = () => {
  const baiduCache = getBaiduMapCache<any>();
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
  const baiduCache = getBaiduMapCache<any>();

  if (address) {
    const cacheKey = `geocode_${address}`;
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
