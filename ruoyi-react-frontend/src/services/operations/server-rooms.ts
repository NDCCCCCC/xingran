import { post } from '@/lib/api';
import type { PageData, PageParams, ServerRoom } from '@/types/operations';

export const serverRoomApi = {
  // 列表查询
  list: (params: PageParams) => post<PageData<ServerRoom>>('/ops/server-rooms/list', params),

  // 创建
  create: (data: Omit<ServerRoom, 'id' | 'createdAt' | 'updatedAt'>) =>
    post('/ops/server-rooms', data),

  // 更新
  update: (id: string, data: Partial<ServerRoom>) =>
    post(`/ops/server-rooms/${id}/update`, data),

  // 删除
  delete: (id: string) => post(`/ops/server-rooms/${id}/delete`),

  // 批量删除
  batchDelete: (ids: string[]) =>
    post('/ops/server-rooms/batch', { ids, action: 'delete' }),

  // 导出
  export: (params: PageParams) =>
    post('/ops/server-rooms/export', params),

  // 导入
  import: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return post('/ops/server-rooms/import', formData);
  },

  // 下载模板
  downloadTemplate: () =>
    post('/ops/server-rooms/template', {}),
};
