import { post } from '@/lib/api';
import type { PageData, PageParams, Workstation } from '@/types/operations';

export const workstationApi = {
  // 列表查询
  list: (params: PageParams) => post<PageData<Workstation>>('/ops/workstations/list', params),

  // 创建
  create: (data: Omit<Workstation, 'id' | 'createdAt' | 'updatedAt'>) =>
    post('/ops/workstations', data),

  // 更新
  update: (id: string, data: Partial<Workstation>) =>
    post(`/ops/workstations/${id}/update`, data),

  // 删除
  delete: (id: string) => post(`/ops/workstations/${id}/delete`),

  // 批量删除
  batchDelete: (ids: string[]) =>
    post('/ops/workstations/batch', { ids, action: 'delete' }),

  // 导出
  export: (params: PageParams) =>
    post('/ops/workstations/export', params),

  // 导入
  import: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return post('/ops/workstations/import', formData);
  },

  // 下载模板
  downloadTemplate: () =>
    post('/ops/workstations/template', {}),
};
