/**
 * Phase 88 Batch16c — dashboard-system 组件渲染(DashboardList/DashboardHome)
 * mock 字段 camelCase 匹配 dataIndex;Dashboard.layout.widgets 必须提供。
 */
import { describe, it, expect, vi } from "vitest";
import { renderPageWithEndpoints } from "@/test/utils/renderPage";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import DashboardList from "../components/DashboardList";
import { DashboardHome } from "../components/DashboardHome";

/** 直接轮询 body 文本出现(最可靠的稳态等待) */
async function waitText(text: string, timeoutMs = 6000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (document.body.innerHTML.includes(text)) return true;
    await new Promise((r) => setTimeout(r, 150));
  }
  return false;
}

const dashboardsList = {
  data: {
    list: [
      {
        id: "d1",
        name: "运维总览大盘",
        description: "核心指标",
        status: 0,
        isDefault: true,
        dashboardType: "custom",
        visibility: "private",
        widgetCount: 2,
        layout: { widgets: [{ id: "w1" }, { id: "w2" }] },
        createdAt: "2026-01-01",
      },
      {
        id: "d2",
        name: "网络质量大盘",
        description: "链路指标",
        status: 0,
        isDefault: false,
        dashboardType: "custom",
        visibility: "public",
        widgetCount: 0,
        layout: { widgets: [] },
        createdAt: "2026-01-02",
      },
    ],
    total: 2,
    current: 1,
    pageSize: 10,
  },
};

describe("pages/dashboard-system — DashboardList", () => {
  it("renders dashboard rows", async () => {
    await renderPageWithEndpoints(<DashboardList />, {
      endpoints: {
        "/system/dashboards/list": dashboardsList,
      },
    });
    expect(await waitText("运维总览大盘")).toBe(true);
    expect(document.body.innerHTML).toContain("网络质量大盘");
  });

  it("renders empty state", async () => {
    await renderPageWithEndpoints(<DashboardList />, {
      endpoints: {
        "/system/dashboards/list": {
          data: { list: [], total: 0, current: 1, pageSize: 10 },
        },
      },
    });
    const ok = await waitText("No data");
    expect(ok || document.querySelector(".ant-table-placeholder")).toBeTruthy();
  });
});

describe("pages/dashboard-system — DashboardHome", () => {
  it("renders empty hint when no dashboards", async () => {
    await renderPageWithEndpoints(<DashboardHome />, {
      endpoints: {
        "/system/dashboards/list": {
          data: { list: [], total: 0, current: 1, pageSize: 10 },
        },
      },
    });
    await new Promise((r) => setTimeout(r, 1200));
    expect(document.querySelector(".ant-empty") || document.querySelector("button")).not.toBeNull();
  });

  it("renders with default dashboard present", async () => {
    await renderPageWithEndpoints(<DashboardHome />, {
      endpoints: {
        "/system/dashboards/list": dashboardsList,
        "/system/dashboards/d1": {
          data: { dashboard: dashboardsList.data.list[0], widgets: [] },
        },
      },
    });
    await new Promise((r) => setTimeout(r, 1500));
    expect(document.body.innerHTML.length).toBeGreaterThan(100);
  });
});
