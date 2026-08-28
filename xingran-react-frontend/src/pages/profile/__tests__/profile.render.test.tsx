/**
 * Phase 88 Batch17 — profile 页 + ad/SyncMonitor 渲染
 */
import { describe, it, expect, vi } from "vitest";
import { renderPageWithEndpoints } from "@/test/utils/renderPage";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import ProfilePage from "../index";
import SyncMonitor from "@/pages/ad/SyncMonitor";

/** 直接轮询 body 文本出现 */
async function waitText(text: string, timeoutMs = 6000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (document.body.innerHTML.includes(text)) return true;
    await new Promise((r) => setTimeout(r, 150));
  }
  return false;
}

describe("pages/profile — index", () => {
  it("renders profile info tabs", async () => {
    await renderPageWithEndpoints(<ProfilePage />, {
      endpoints: {
        "/system/profile/info": {
          data: {
            id: "u1",
            username: "admin",
            nickname: "管理员",
            email: "admin@test.com",
            phone: "13800000000",
            dept: { deptName: "信息部" },
          },
        },
        "/system/settings/preferences": { data: {} },
      },
    });
    // Tab 头至少出现
    expect(
      (await waitText("个人信息")) ||
        (await waitText("基本资料")) ||
        (await waitText("admin")) ||
        (await waitText("修改密码"))
    ).toBe(true);
  });

  it("renders without profile data (error path)", async () => {
    await renderPageWithEndpoints(<ProfilePage />, {
      endpoints: {
        "/system/profile/info": { data: null },
        "/system/settings/preferences": { data: {} },
      },
    });
    await new Promise((r) => setTimeout(r, 800));
    expect(document.body.innerHTML.length).toBeGreaterThan(50);
  });
});

describe("pages/ad — SyncMonitor", () => {
  it("renders sync logs table", async () => {
    await renderPageWithEndpoints(<SyncMonitor />, {
      endpoints: {
        "/api/v1/ad/groups/sync/logs": {
          data: {
            list: [
              {
                id: "log1",
                configName: "默认同步任务",
                syncType: "full",
                status: "success",
                startTime: "2026-01-01 02:00:00",
                duration: 300,
                successCount: 120,
                failureCount: 0,
              },
            ],
            total: 1,
            current: 1,
            pageSize: 10,
          },
        },
        "/api/v1/ad/groups/sync/status": { data: { running: false } },
      },
    });
    expect(await waitText("默认同步任务")).toBe(true);
  });

  it("renders empty logs", async () => {
    await renderPageWithEndpoints(<SyncMonitor />, {
      endpoints: {
        "/api/v1/ad/groups/sync/logs": {
          data: { list: [], total: 0, current: 1, pageSize: 10 },
        },
        "/api/v1/ad/groups/sync/status": { data: { running: false } },
      },
    });
    const ok = await waitText("No data");
    expect(ok || document.querySelector(".ant-table-placeholder")).toBeTruthy();
  });
});
