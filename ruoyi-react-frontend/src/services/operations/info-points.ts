import { post } from '@/lib/api';

// API响应包装类型
interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
  timestamp: number;
  request_id: string;
}

import type { PageData, PageParams, InfoPoint } from '@/types/operations';

export const infoPointApi = {
  // 列表查询
  list: (params: PageParams) => post<ApiResponse<PageData<InfoPoint>>>('/ops/info-points/list', params),

  // 创建
  create: (data: Omit<InfoPoint, 'id' | 'createdAt' | 'updatedAt'>) =>
    post('/ops/info-points', data),

  // 更新
  update: (id: string, data: Partial<InfoPoint>) =>
    post(`/ops/info-points/${id}/update`, data),

  // 删除
  delete: (id: string) => post(`/ops/info-points/${id}/delete`),

  // 批量删除
  batchDelete: (ids: string[]) =>
    post('/ops/info-points/batch', { ids, action: 'delete' }),
};
