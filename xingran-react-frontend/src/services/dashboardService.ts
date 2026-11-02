/**
 * 仪表盘 API 服务
 * Dashboard API Service
 *
 * 封装所有仪表盘相关的 API 调用
 */

import { post, get, del } from "@/lib/api";
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
} from "@/types/dashboard";

/**
 * 仪表盘 API 服务类
 */
class DashboardService {
  // ========== 仪表盘基础路径 ==========
  private readonly basePath = "/system/dashboards";

  // ========== 仪表盘 CRUD 操作 ==========

  async getDefaultDashboard(): Promise<Dashboard | null> {
    const response = await get<{ dashboard: Dashboard | null }>(`${this.basePath}/default`);
    return response.data?.dashboard ?? null;
  }

  async getDashboards(params: DashboardListParams): Promise<DashboardListResponse> {
    const response = await post<DashboardListResponse>(`${this.basePath}/list`, params);
    return response.data!;
  }

  async getDashboard(id: string): Promise<Dashboard> {
    const response = await get<{ dashboard: Dashboard }>(`${this.basePath}/${id}`);
    return response.data!.dashboard;
  }

  async createDashboard(data: CreateDashboardRequest): Promise<Dashboard> {
    const response = await post<{ dashboard: Dashboard }>(this.basePath, data);
    return response.data!.dashboard;
  }

  async updateDashboard(id: string, data: UpdateDashboardRequest): Promise<void> {
    await post(`${this.basePath}/${id}/update`, data);
  }

  async deleteDashboard(id: string): Promise<void> {
    await del(`${this.basePath}/${id}`);
  }

  async duplicateDashboard(id: string): Promise<Dashboard> {
    const response = await post<{ dashboard: Dashboard }>(`${this.basePath}/${id}/duplicate`);
    return response.data!.dashboard;
  }

  async setDefaultDashboard(id: string): Promise<void> {
    await post(`${this.basePath}/${id}/set-default`);
  }

  async getTemplates(scope?: "global" | "dept" | "personal"): Promise<Dashboard[]> {
    const response = await post<{ templates: Dashboard[] }>(`${this.basePath}/templates`, {
      scope,
    });
    return response.data?.templates ?? [];
  }

  async createFromTemplate(templateId: string, name: string): Promise<Dashboard> {
    const response = await post<{ dashboard: Dashboard }>(
      `${this.basePath}/templates/${templateId}/create`,
      { name }
    );
    return response.data!.dashboard;
  }

  async getVersions(dashboardId: string): Promise<DashboardVersion[]> {
    const response = await get<{ versions: DashboardVersion[] }>(
      `${this.basePath}/${dashboardId}/versions`
    );
    return response.data?.versions ?? [];
  }

  async restoreVersion(dashboardId: string, versionId: string): Promise<void> {
    await post(`${this.basePath}/${dashboardId}/versions/${versionId}/restore`);
  }

  async createVersion(dashboardId: string, comment?: string): Promise<DashboardVersion> {
    const response = await post<{ version: DashboardVersion }>(
      `${this.basePath}/${dashboardId}/versions`,
      { comment }
    );
    return response.data!.version;
  }

  async exportDashboard(id: string): Promise<string> {
    const response = await get<{ config: string }>(`${this.basePath}/${id}/export`);
    return response.data!.config;
  }

  async importDashboard(config: string): Promise<Dashboard> {
    const response = await post<{ dashboard: Dashboard }>(`${this.basePath}/import`, { config });
    return response.data!.dashboard;
  }

  async getWidgetData(widgetId: string): Promise<unknown> {
    const response = await post<{ data: unknown }>(`${this.basePath}/widgets/${widgetId}/data`);
    return response.data!.data;
  }

  async getBatchWidgetData(widgetIds: string[]): Promise<Map<string, unknown>> {
    const response = await post<{ data: Record<string, unknown> }>(
      `${this.basePath}/widgets/batch-data`,
      { widgetIds }
    );
    return new Map(Object.entries(response.data!.data));
  }

  async getAvailableEndpoints(): Promise<EndpointCategory[]> {
    const response = await get<EndpointsResponse>(`${this.basePath}/endpoints`);
    return response.data?.categories ?? [];
  }

  async getEndpointsByWidgetType(widgetType: string): Promise<EndpointCategory[]> {
    const response = await get<EndpointsResponse>(`${this.basePath}/endpoints/filter`, {
      params: { widgetType },
    });
    return response.data?.categories ?? [];
  }

  async validateEndpoint(route: string, method: string): Promise<EndpointDetail> {
    const response = await get<EndpointDetail>(`${this.basePath}/endpoints/validate`, {
      params: { route, method },
    });
    return response.data!;
  }

  async invalidateEndpointCache(): Promise<void> {
    await post(`${this.basePath}/endpoints/cache/invalidate`);
  }
}

// ========== 导出单例 ==========

export const dashboardService = new DashboardService();
