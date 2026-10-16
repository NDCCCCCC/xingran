/**
 * Phase 88 Batch99 — services/dashboardService 测试(39 stmts, 20.5% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { dashboardService } from "../dashboardService";
import { createApiMock } from "@/test/utils/createApiMock";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

describe("dashboardService", () => {
  it("getDefaultDashboard 成功 → 返回 dashboard", async () => {
    const api = createApiMock("/system/dashboards/default");
    api.endpoint.mockResolvedValueOnce({ data: { dashboard: { id: "d1", name: "默认" } } } as any);
    const r = await dashboardService.getDefaultDashboard();
    expect(r?.id).toBe("d1");
  });

  it("getDefaultDashboard 失败 → 返回 null", async () => {
    const api = createApiMock("/system/dashboards/default");
    api.endpoint.mockResolvedValueOnce({ data: { dashboard: null } } as any);
    const r = await dashboardService.getDefaultDashboard();
    expect(r).toBeNull();
  });

  it("getDashboards → 返回 data", async () => {
    const api = createApiMock("/system/dashboards/list");
    api.endpoint.mockResolvedValueOnce({ data: { list: [], total: 0 } } as any);
    const r = await dashboardService.getDashboards({ current: 1, pageSize: 10 });
    expect(r.total).toBe(0);
  });

  it("getDashboard(id) → 返回 dashboard", async () => {
    const api = createApiMock("/system/dashboards/d1");
    api.endpoint.mockResolvedValueOnce({ data: { dashboard: { id: "d1" } } } as any);
    const r = await dashboardService.getDashboard("d1");
    expect(r.id).toBe("d1");
  });

  it("createDashboard → POST 创建", async () => {
    const api = createApiMock("/system/dashboards");
    api.endpoint.mockResolvedValueOnce({ data: { dashboard: { id: "d-new" } } } as any);
    const r = await dashboardService.createDashboard({ name: "new" } as any);
    expect(r.id).toBe("d-new");
  });

  it("updateDashboard → POST update", async () => {
    const api = createApiMock("/system/dashboards/d1/update");
    api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
    await dashboardService.updateDashboard("d1", { name: "new" } as any);
    expect(api.endpoint).toHaveBeenCalled();
  });

  it("deleteDashboard → DELETE", async () => {
    const api = createApiMock("/system/dashboards/d1");
    api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
    await dashboardService.deleteDashboard("d1");
    expect(api.endpoint).toHaveBeenCalled();
  });

  it("duplicateDashboard → 复制", async () => {
    const api = createApiMock("/system/dashboards/d1/duplicate");
    api.endpoint.mockResolvedValueOnce({ data: { dashboard: { id: "d-copy" } } } as any);
    const r = await dashboardService.duplicateDashboard("d1");
    expect(r.id).toBe("d-copy");
  });

  it("setDefaultDashboard → POST set-default", async () => {
    const api = createApiMock("/system/dashboards/d1/set-default");
    api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
    await dashboardService.setDefaultDashboard("d1");
    expect(api.endpoint).toHaveBeenCalled();
  });

  it("getTemplates → 返回 templates", async () => {
    const api = createApiMock("/system/dashboards/templates");
    api.endpoint.mockResolvedValueOnce({ data: { templates: [{ id: "t1" }] } } as any);
    const r = await dashboardService.getTemplates("global");
    expect(r).toHaveLength(1);
  });

  it("createFromTemplate → 从模板创建", async () => {
    const api = createApiMock("/system/dashboards/templates/t1/create");
    api.endpoint.mockResolvedValueOnce({ data: { dashboard: { id: "d-from-tpl" } } } as any);
    const r = await dashboardService.createFromTemplate("t1", "new");
    expect(r.id).toBe("d-from-tpl");
  });

  it("getVersions → 返回版本列表", async () => {
    const api = createApiMock("/system/dashboards/d1/versions");
    api.endpoint.mockResolvedValueOnce({ data: { versions: [{ id: "v1" }] } } as any);
    const r = await dashboardService.getVersions("d1");
    expect(r).toHaveLength(1);
  });

  it("restoreVersion → POST restore", async () => {
    const api = createApiMock("/system/dashboards/d1/versions/v1/restore");
    api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
    await dashboardService.restoreVersion("d1", "v1");
    expect(api.endpoint).toHaveBeenCalled();
  });

  it("createVersion → 创建版本", async () => {
    const api = createApiMock("/system/dashboards/d1/versions");
    api.endpoint.mockResolvedValueOnce({ data: { version: { id: "v1" } } } as any);
    const r = await dashboardService.createVersion("d1", "comment");
    expect(r.id).toBe("v1");
  });

  it("exportDashboard → 返回 config 字符串", async () => {
    const api = createApiMock("/system/dashboards/d1/export");
    api.endpoint.mockResolvedValueOnce({ data: { config: "{}" } } as any);
    const r = await dashboardService.exportDashboard("d1");
    expect(r).toBe("{}");
  });

  it("importDashboard → POST import", async () => {
    const api = createApiMock("/system/dashboards/import");
    api.endpoint.mockResolvedValueOnce({ data: { dashboard: { id: "d-imp" } } } as any);
    const r = await dashboardService.importDashboard("{}");
    expect(r.id).toBe("d-imp");
  });

  it("getWidgetData → POST widget data", async () => {
    const api = createApiMock("/system/dashboards/widgets/w1/data");
    api.endpoint.mockResolvedValueOnce({ data: { data: { v: 42 } } } as any);
    const r = await dashboardService.getWidgetData("w1");
    expect(r).toEqual({ v: 42 });
  });

  it("getBatchWidgetData → Map 结果", async () => {
    const api = createApiMock("/system/dashboards/widgets/batch-data");
    api.endpoint.mockResolvedValueOnce({
      data: { data: { w1: { v: 1 }, w2: { v: 2 } } },
    } as any);
    const r = await dashboardService.getBatchWidgetData(["w1", "w2"]);
    expect(r).toBeInstanceOf(Map);
    expect(r.get("w1")).toEqual({ v: 1 });
  });

  it("getAvailableEndpoints → 返回 categories", async () => {
    const api = createApiMock("/system/dashboards/endpoints");
    api.endpoint.mockResolvedValueOnce({ data: { categories: [] } } as any);
    const r = await dashboardService.getAvailableEndpoints();
    expect(r).toEqual([]);
  });

  it("getEndpointsByWidgetType → POST filter", async () => {
    const api = createApiMock("/system/dashboards/endpoints/filter");
    api.endpoint.mockResolvedValueOnce({ data: { categories: [] } } as any);
    const r = await dashboardService.getEndpointsByWidgetType("stat-card");
    expect(r).toEqual([]);
  });

  it("validateEndpoint → GET validate", async () => {
    const api = createApiMock("/system/dashboards/endpoints/validate");
    api.endpoint.mockResolvedValueOnce({ data: { route: "/x", method: "GET" } } as any);
    const r = await dashboardService.validateEndpoint("/x", "GET");
    expect(r.route).toBe("/x");
  });

  it("invalidateEndpointCache → POST", async () => {
    const api = createApiMock("/system/dashboards/endpoints/cache/invalidate");
    api.endpoint.mockResolvedValueOnce({ code: 0 } as any);
    await dashboardService.invalidateEndpointCache();
    expect(api.endpoint).toHaveBeenCalled();
  });
});
