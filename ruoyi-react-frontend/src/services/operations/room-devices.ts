import { post } from '@/lib/api';

// API响应包装类型
interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
  timestamp: number;
  request_id: string;
}
import type { PageData, PageParams, RoomDevice } from '@/types/operations';

// API响应包装类型
interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
  timestamp: number;
  request_id: string;
}

export const roomDeviceApi = {
	// 列表查询
	list: (params: PageParams) => post<ApiResponse<PageData<RoomDevice>>>('/ops/room-devices/list', params),

	// 创建
	create: (data: Omit<RoomDevice, 'id' | 'createdAt' | 'updatedAt'>) =>
		post('/ops/room-devices', data),

	// 更新
	update: (id: string, data: Partial<RoomDevice>) =>
		post(`/ops/room-devices/${id}/update`, data),

	// 删除
	delete: (id: string) => post(`/ops/room-devices/${id}/delete`),

	// 批量删除
	batchDelete: (ids: string[]) =>
		post('/ops/room-devices/batch', { ids, action: 'delete' }),

	// 导出
	export: (params: PageParams) =>
		post('/ops/room-devices/export', params),

	// 导入
	import: (file: File) => {
		const formData = new FormData();
		formData.append('file', file);
		return post('/ops/room-devices/import', formData);
	},

	// 下载模板
	downloadTemplate: () =>
		post('/ops/room-devices/template', {}),
};
