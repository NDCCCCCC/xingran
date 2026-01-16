/**
 * 运维管理模块 API
 */

import { post } from './api';
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
}

export const buildingApi = {
  // 查询楼宇列表
  list: async (params: BuildingListParams) => {
    return await post<BaseResponse<PageResponse<Building>>>('/ops/buildings/list', params);
  },

  // 获取楼宇详情
  get: async (id: string) => {
    return await post<BaseResponse<Building>>('/ops/buildings/' + id, {});
  },

  // 创建楼宇
  create: async (data: Partial<Building>) => {
    return await post('/ops/buildings', data);
  },

  // 更新楼宇
  update: async (id: string, data: Partial<Building>) => {
    return await post('/ops/buildings/' + id + '/update', data);
  },

  // 删除楼宇
  delete: async (id: string) => {
    return await post('/ops/buildings/' + id + '/delete', {});
  },

  // 批量操作
  batch: async (action: string, data: Record<string, unknown>) => {
    return await post('/ops/buildings/batch', { action, ...data });
  },
};

// ==================== 楼层管理 ====================

export interface FloorListParams extends PageParams {
  buildingCode?: string;
  floorNo?: string;
  name?: string;
  status?: number;
}

export const floorApi = {
  list: async (params: FloorListParams) => {
    return await post<BaseResponse<PageResponse<Floor>>>('/ops/floors/list', params);
  },

  get: async (id: string) => {
    return await post<BaseResponse<Floor>>('/ops/floors/' + id, {});
  },

  create: async (data: Partial<Floor>) => {
    return await post('/ops/floors', data);
  },

  update: async (id: string, data: Partial<Floor>) => {
    return await post('/ops/floors/' + id + '/update', data);
  },

  delete: async (id: string) => {
    return await post('/ops/floors/' + id + '/delete', {});
  },

  // 获取楼层树
  tree: async () => {
    return await post<BaseResponse<Floor[]>>('/ops/floors/tree', {});
  },

  batch: async (action: string, data: Record<string, unknown>) => {
    return await post('/ops/floors/batch', { action, ...data });
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
    return await post<BaseResponse<PageResponse<WorkstationOps>>>('/ops/workstations/list', params);
  },

  get: async (id: string) => {
    return await post<BaseResponse<WorkstationOps>>('/ops/workstations/' + id, {});
  },

  create: async (data: Partial<WorkstationOps>) => {
    return await post('/ops/workstations', data);
  },

  update: async (id: string, data: Partial<WorkstationOps>) => {
    return await post('/ops/workstations/' + id + '/update', data);
  },

  delete: async (id: string) => {
    return await post('/ops/workstations/' + id + '/delete', {});
  },

  // 批量更新位置
  updatePositions: async (items: { id: string; positionX: number; positionY: number }[]) => {
    return await post('/ops/workstations/positions', { items });
  },

  batch: async (action: string, data: Record<string, unknown>) => {
    return await post('/ops/workstations/batch', { action, ...data });
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
    return await post<BaseResponse<PageResponse<ServerRoom>>>('/ops/server-rooms/list', params);
  },

  get: async (id: string) => {
    return await post<BaseResponse<ServerRoom>>('/ops/server-rooms/' + id, {});
  },

  create: async (data: Partial<ServerRoom>) => {
    return await post('/ops/server-rooms', data);
  },

  update: async (id: string, data: Partial<ServerRoom>) => {
    return await post('/ops/server-rooms/' + id + '/update', data);
  },

  delete: async (id: string) => {
    return await post('/ops/server-rooms/' + id + '/delete', {});
  },

  batch: async (action: string, data: Record<string, unknown>) => {
    return await post('/ops/server-rooms/batch', { action, ...data });
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

    const token = localStorage.getItem('token');
    const response = await fetch('/api/v1/ops/rooms/photos/upload', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
      },
      body: formData,
    });

    if (!response.ok) {
      throw new Error('上传失败');
    }

    return await response.json();
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
    return await post<BaseResponse<PageResponse<RoomDevice>>>('/ops/room-devices/list', params);
  },

  get: async (id: string) => {
    return await post<BaseResponse<RoomDevice>>('/ops/room-devices/' + id, {});
  },

  create: async (data: Partial<RoomDevice>) => {
    return await post('/ops/room-devices', data);
  },

  update: async (id: string, data: Partial<RoomDevice>) => {
    return await post('/ops/room-devices/' + id + '/update', data);
  },

  delete: async (id: string) => {
    return await post('/ops/room-devices/' + id + '/delete', {});
  },

  batch: async (action: string, data: Record<string, unknown>) => {
    return await post('/ops/room-devices/batch', { action, ...data });
  },
};

// ==================== 专线管理 ====================

export interface DedicatedLineListParams extends PageParams {
  lineCode?: string;
  lineType?: string;
  isp?: string;
  status?: number;
}

export const dedicatedLineApi = {
  list: async (params: DedicatedLineListParams) => {
    return await post<BaseResponse<PageResponse<DedicatedLine>>>('/ops/dedicated-lines/list', params);
  },

  get: async (id: string) => {
    return await post<BaseResponse<DedicatedLine>>('/ops/dedicated-lines/' + id, {});
  },

  create: async (data: Partial<DedicatedLine>) => {
    return await post('/ops/dedicated-lines', data);
  },

  update: async (id: string, data: Partial<DedicatedLine>) => {
    return await post('/ops/dedicated-lines/' + id + '/update', data);
  },

  delete: async (id: string) => {
    return await post('/ops/dedicated-lines/' + id + '/delete', {});
  },

  batch: async (action: string, data: Record<string, unknown>) => {
    return await post('/ops/dedicated-lines/batch', { action, ...data });
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
    return await post<BaseResponse<PageResponse<InfoPoint>>>('/ops/info-points/list', params);
  },

  get: async (id: string) => {
    return await post<BaseResponse<InfoPoint>>('/ops/info-points/' + id, {});
  },

  create: async (data: Partial<InfoPoint>) => {
    return await post('/ops/info-points', data);
  },

  update: async (id: string, data: Partial<InfoPoint>) => {
    return await post('/ops/info-points/' + id + '/update', data);
  },

  delete: async (id: string) => {
    return await post('/ops/info-points/' + id + '/delete', {});
  },

  batch: async (action: string, data: Record<string, unknown>) => {
    return await post('/ops/info-points/batch', { action, ...data });
  },
};

// ==================== Excel导入导出 ====================

export const excelApi = {
  // 下载模板
  downloadTemplate: async (entityType: string) => {
    const token = localStorage.getItem('token');
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

    const token = localStorage.getItem('token');
    const response = await fetch(`/api/v1/ops/${entityType}/import`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
      },
      body: formData,
    });

    if (!response.ok) {
      throw new Error('导入失败');
    }

    return await response.json();
  },

  // 导出数据
  export: async (entityType: string, params: Record<string, unknown> = {}) => {
    const token = localStorage.getItem('token');
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
