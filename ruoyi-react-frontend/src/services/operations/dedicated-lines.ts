import { post } from '@/lib/api';
import type { PageParams } from '@/types/base';
import type { PageData, DedicatedLine } from '@/types/operations';

// API响应包装类型
interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
  timestamp: number;
  request_id: string;
}

export const dedicatedLineApi = {
  // 列表查询
  list: (params: PageParams) => post<ApiResponse<PageData<DedicatedLine>>>('/ops/dedicated-lines/list', params),

  // 创建
  create: (data: Omit<DedicatedLine, 'id' | 'createdAt' | 'updatedAt'>) =>
    post('/ops/dedicated-lines', data),

  // 更新
  update: (id: string, data: Partial<DedicatedLine>) =>
    post(`/ops/dedicated-lines/${id}/update`, data),

  // 删除
  delete: (id: string) => post(`/ops/dedicated-lines/${id}/delete`),

  // 批量删除
  batchDelete: (ids: string[]) =>
    post('/ops/dedicated-lines/batch', { ids, action: 'delete' }),

  // 导出
  export: (params: PageParams) =>
    post('/ops/dedicated-lines/export', params),

  // 导入
  import: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return post('/ops/dedicated-lines/import', formData);
  },

  // 下载模板
  downloadTemplate: () =>
    post('/ops/dedicated-lines/template', {}),
};
