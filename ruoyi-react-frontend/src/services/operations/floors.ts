import { post } from '@/lib/api';
import type { PageData, PageParams, Floor } from '@/types/operations';

export const floorApi = {
  // 列表查询
  list: (params: PageParams) => post<PageData<Floor>>('/ops/floors/list', params),

  // 获取楼宇-楼层树
  getTree: () => post<any[]>('/ops/floors/tree'),

  // 创建
  create: (data: Omit<Floor, 'id' | 'createdAt' | 'updatedAt'>) =>
    post('/ops/floors', data),

  // 更新
  update: (id: string, data: Partial<Floor>) =>
    post(`/ops/floors/${id}/update`, data),

  // 删除
  delete: (id: string) => post(`/ops/floors/${id}/delete`),

  // 批量删除
  batchDelete: (ids: string[]) =>
    post('/ops/floors/batch', { ids, action: 'delete' }),

  // 导出
  export: (params: PageParams) =>
    post('/ops/floors/export', params),

  // 导入
  import: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return post('/ops/floors/import', formData);
  },

  // 下载模板
  downloadTemplate: () =>
    post('/ops/floors/template', {}),
};
