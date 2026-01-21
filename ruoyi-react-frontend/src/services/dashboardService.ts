/**
 * 仪表盘 API 服务
 * Dashboard API Service
 *
 * 封装所有仪表盘相关的 API 调用
 */

import { post, get, del } from '@/lib/api';
import type {
	Dashboard,
	CreateDashboardRequest,
	UpdateDashboardRequest,
	DashboardListParams,
	DashboardListResponse,
	DashboardVersion,
	EndpointCategory,
	EndpointDetail,
	EndpointsResponse,
} from '@/types/dashboard';

/**
 * 仪表盘 API 服务类
 */
class DashboardService {
	// ========== 仪表盘基础路径 ==========
	private readonly basePath = '/system/dashboards';

	// ========== 仪表盘 CRUD 操作 ==========

	/**
	 * 获取仪表盘列表
	 * 响应拦截器返回的是 { code, message, data: {...}, timestamp, request_id }
	 * 需要提取 .data 获取实际的列表数据
	 */
	async getDashboards(params: DashboardListParams): Promise<DashboardListResponse> {
		const response = await post<{ data: DashboardListResponse }>(
			`${this.basePath}/list`,
			params
		);
		return response.data;
	}

	/**
	 * 获取仪表盘详情
	 */
	async getDashboard(id: string): Promise<Dashboard> {
		const response = await get<{ data: { dashboard: Dashboard } }>(
			`${this.basePath}/${id}`
		);
		return response.data.dashboard;
	}

	/**
	 * 创建仪表盘
	 */
	async createDashboard(data: CreateDashboardRequest): Promise<Dashboard> {
		const response = await post<{ data: { dashboard: Dashboard } }>(
			this.basePath,
			data
		);
		return response.data.dashboard;
	}

	/**
	 * 更新仪表盘
	 */
	async updateDashboard(id: string, data: UpdateDashboardRequest): Promise<void> {
		await post(`${this.basePath}/${id}/update`, data);
	}

	/**
	 * 删除仪表盘
	 */
	async deleteDashboard(id: string): Promise<void> {
		await del(`${this.basePath}/${id}`);
	}

	/**
	 * 复制仪表盘
	 */
	async duplicateDashboard(id: string): Promise<Dashboard> {
		const response = await post<{ data: { dashboard: Dashboard } }>(
			`${this.basePath}/${id}/duplicate`
		);
		return response.data.dashboard;
	}

	/**
	 * 设置为默认仪表盘
	 */
	async setDefaultDashboard(id: string): Promise<void> {
		await post(`${this.basePath}/${id}/set-default`);
	}

	// ========== 仪表盘模板操作 ==========

	/**
	 * 获取仪表盘模板列表
	 */
	async getTemplates(scope?: 'global' | 'dept' | 'personal'): Promise<Dashboard[]> {
		const response = await post<{ data: { templates: Dashboard[] } }>(
			`${this.basePath}/templates`,
			{ scope }
		);
		return response.data.templates;
	}

	/**
	 * 从模板创建仪表盘
	 */
	async createFromTemplate(templateId: string, name: string): Promise<Dashboard> {
		const response = await post<{ data: { dashboard: Dashboard } }>(
			`${this.basePath}/templates/${templateId}/create`,
			{ name }
		);
		return response.data.dashboard;
	}

	// ========== 仪表盘版本操作 ==========

	/**
	 * 获取仪表盘版本历史
	 */
	async getVersions(dashboardId: string): Promise<DashboardVersion[]> {
		const response = await get<{ data: { versions: DashboardVersion[] } }>(
			`${this.basePath}/${dashboardId}/versions`
		);
		return response.data.versions;
	}

	/**
	 * 从版本恢复仪表盘
	 */
	async restoreVersion(dashboardId: string, versionId: string): Promise<void> {
		await post(`${this.basePath}/${dashboardId}/versions/${versionId}/restore`);
	}

	/**
	 * 创建版本快照
	 */
	async createVersion(
		dashboardId: string,
		comment?: string
	): Promise<DashboardVersion> {
		const response = await post<{ data: { version: DashboardVersion } }>(
			`${this.basePath}/${dashboardId}/versions`,
			{ comment }
		);
		return response.data.version;
	}

	// ========== 仪表盘导入导出 ==========

	/**
	 * 导出仪表盘配置（JSON）
	 */
	async exportDashboard(id: string): Promise<string> {
		const response = await get<{ data: { config: string } }>(
			`${this.basePath}/${id}/export`
		);
		return response.data.config;
	}

	/**
	 * 导入仪表盘配置
	 */
	async importDashboard(config: string): Promise<Dashboard> {
		const response = await post<{ data: { dashboard: Dashboard } }>(
			`${this.basePath}/import`,
			{ config }
		);
		return response.data.dashboard;
	}

	// ========== Widget 数据获取 ==========

	/**
	 * 获取 Widget 数据
	 */
	async getWidgetData(widgetId: string): Promise<unknown> {
		const response = await post<{ data: { data: unknown } }>(
			`${this.basePath}/widgets/${widgetId}/data`
		);
		return response.data.data;
	}

	/**
	 * 批量获取 Widget 数据
	 */
	async getBatchWidgetData(widgetIds: string[]): Promise<Map<string, unknown>> {
		const response = await post<{ data: { data: Record<string, unknown> } }>(
			`${this.basePath}/widgets/batch-data`,
			{ widgetIds }
		);
		return new Map(Object.entries(response.data.data));
	}

	// ========== API 端点元数据 ==========

	/**
	 * 获取可用的API端点列表（根据用户权限过滤）
	 */
	async getAvailableEndpoints(): Promise<EndpointCategory[]> {
		const response = await get<{ data: EndpointsResponse }>(
			`${this.basePath}/endpoints`
		);
		return response.data.categories;
	}

	/**
	 * 根据Widget类型过滤获取端点列表
	 */
	async getEndpointsByWidgetType(widgetType: string): Promise<EndpointCategory[]> {
		const response = await get<{ data: EndpointsResponse }>(
			`${this.basePath}/endpoints/filter`,
			{ params: { widgetType } }
		);
		return response.data.categories;
	}

	/**
	 * 验证API端点配置
	 */
	async validateEndpoint(route: string, method: string): Promise<EndpointDetail> {
		const response = await get<{ data: EndpointDetail }>(
			`${this.basePath}/endpoints/validate`,
			{ params: { route, method } }
		);
		return response.data;
	}

	/**
	 * 清除用户端点缓存
	 */
	async invalidateEndpointCache(): Promise<void> {
		await post(`${this.basePath}/endpoints/cache/invalidate`);
	}
}

// ========== 导出单例 ==========

export const dashboardService = new DashboardService();
