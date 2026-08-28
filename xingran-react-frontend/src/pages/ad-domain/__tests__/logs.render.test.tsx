/**
 * Phase 88 Batch28 — ad-domain logs 页渲染测试(mock adDomainApi)
 *
 * adDomainApi 是独立 api 模块(非 @/lib/api),直接 vi.mock 模块函数。
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/adDomainApi", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/adDomainApi")>();
  return {
    ...actual,
    getADConfigList: vi.fn().mockResolvedValue({
      code: 0,
      data: { list: [{ id: "cfg-1", name: "主域" }], total: 1 },
    }),
    getADSyncLogs: vi.fn().mockResolvedValue({
      code: 0,
      data: {
        list: [
          {
            id: "log-1",
            configId: "cfg-1",
            configName: "主域",
            status: "success",
            startTime: "2026-08-28T10:00:00Z",
            endTime: "2026-08-28T10:05:00Z",
            totalProcessed: 100,
            createdCount: 10,
            updatedCount: 80,
            errorCount: 0,
          },
          {
            id: "log-2",
            configId: "cfg-1",
            configName: "主域",
            status: "failed",
            errorMessage: "连接超时",
            startTime: "2026-08-27T10:00:00Z",
            endTime: "2026-08-27T10:01:00Z",
          },
        ],
        total: 2,
      },
    }),
  };
});

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import ADSyncLogsPage from "../logs";
import { getADConfigList, getADSyncLogs } from "@/lib/adDomainApi";

describe("ad-domain logs 页渲染", () => {
  it("挂载拉取 configs+logs 渲染表格", async () => {
    const { container } = renderWithProviders(<ADSyncLogsPage />, { route: "/ad-domain/logs" });

    await vi.waitFor(
      () => {
        expect(container.querySelector(".ant-table")).not.toBeNull();
      },
      { timeout: 8000 }
    );
    expect(getADConfigList).toHaveBeenCalled();
    expect(getADSyncLogs).toHaveBeenCalled();
  }, 15000);
});
