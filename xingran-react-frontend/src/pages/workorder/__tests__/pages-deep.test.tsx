/**
 * Phase 88 Batch18 — workorder 深测: orders 主表 + categories + statistics
 * mock 字段 camelCase 匹配 dataIndex。
 */
import { describe, it, expect, vi } from "vitest";
import { renderPageWithEndpoints } from "@/test/utils/renderPage";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import WorkOrderPage from "../orders";
import CategoriesPage from "../categories";
import StatisticsPage from "../statistics";

async function waitText(text: string, timeoutMs = 6000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (document.body.innerHTML.includes(text)) return true;
    await new Promise((r) => setTimeout(r, 150));
  }
  return false;
}

const ordersList = {
  data: {
    list: [
      {
        id: "wo1",
        workOrderNo: "WO-2026-0001",
        title: "核心交换机端口故障",
        type: "fault",
        priority: "high",
        status: "pending",
        creatorName: "zhangsan",
        handlerName: "lisi",
        createdAt: "2026-01-03 10:00:00",
        updatedAt: "2026-01-03 11:00:00",
      },
    ],
    total: 1,
    current: 1,
    pageSize: 10,
  },
};

const categoriesArr = {
  data: [
    { id: "c1", categoryName: "网络故障", parentId: null, sortOrder: 0, status: 0, children: [] },
    { id: "c2", categoryName: "硬件维修", parentId: null, sortOrder: 1, status: 0, children: [] },
  ],
};

const categoriesList = {
  data: {
    list: categoriesArr.data,
    total: 2,
    current: 1,
    pageSize: 10,
  },
};

describe("pages/workorder — orders", () => {
  it("renders orders table rows", async () => {
    await renderPageWithEndpoints(<WorkOrderPage />, {
      endpoints: {
        "/workorder/orders/list": ordersList,
        "/workorder/orders/status-statistics": {
          data: {
            total: 1,
            pending: 1,
            processing: 0,
            completed: 0,
            closed: 0,
            byPriority: {},
            byType: {},
          },
        },
        "/workorder/categories/list": categoriesArr,
        "/workorder/categories/enabled": categoriesArr,
        "/system/users/list": { data: [] },
        "/system/departments/tree": { data: [] },
      },
    });
    expect(await waitText("WO-2026-0001")).toBe(true);
    expect(document.body.innerHTML).toContain("核心交换机端口故障");
  });

  it("renders empty orders state", async () => {
    await renderPageWithEndpoints(<WorkOrderPage />, {
      endpoints: {
        "/workorder/orders/list": {
          data: { list: [], total: 0, current: 1, pageSize: 10 },
        },
        "/workorder/orders/status-statistics": {
          data: {
            total: 1,
            pending: 1,
            processing: 0,
            completed: 0,
            closed: 0,
            byPriority: {},
            byType: {},
          },
        },
        "/workorder/categories/list": { data: [] },
        "/workorder/categories/enabled": { data: [] },
        "/system/users/list": { data: [] },
        "/system/departments/tree": { data: [] },
      },
    });
    const ok = await waitText("No data");
    expect(ok || document.querySelector(".ant-table-placeholder")).toBeTruthy();
  });
});

describe("pages/workorder — categories", () => {
  it("renders category rows", async () => {
    await renderPageWithEndpoints(<CategoriesPage />, {
      endpoints: {
        "/workorder/categories/list": categoriesArr,
      },
    });
    expect(await waitText("网络故障")).toBe(true);
    expect(document.body.innerHTML).toContain("硬件维修");
  });

  it("renders empty categories (Tree empty)", async () => {
    await renderPageWithEndpoints(<CategoriesPage />, {
      endpoints: {
        "/workorder/categories/list": { data: [] },
      },
    });
    await new Promise((r) => setTimeout(r, 800));
    // 空分类: 页面 chrome 存在且不渲染分类文本
    expect(document.body.innerHTML.length).toBeGreaterThan(100);
    expect(document.body.innerHTML.includes("网络故障")).toBe(false);
  });
});

describe("pages/workorder — statistics", () => {
  it("renders statistics page chrome", async () => {
    await renderPageWithEndpoints(<StatisticsPage />, {
      endpoints: {
        "/workorder/statistics": {
          data: {
            total: 16,
            pending: 3,
            processing: 2,
            completed: 10,
            closed: 1,
            byPriority: { high: 5, low: 11 },
            byCategory: { 网络故障: 9, 硬件维修: 7 },
          },
        },
        "/workorder/orders/list": ordersList,
      },
    });
    await new Promise((r) => setTimeout(r, 1000));
    expect(document.body.innerHTML.length).toBeGreaterThan(100);
  });
});
