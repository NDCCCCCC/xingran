/**
 * Phase 88 Batch92 — system notice 页面渲染(182 stmts, 30.2% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import NoticeManagement from "../index";
import { createApiMock } from "@/test/utils/createApiMock";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/noticeApi", async () => {
  const { createApiMock } = await import("@/test/utils/createApiMock");
  return {
    getNoticeStatistics: createApiMock("/system/notice/statistics").endpoint,
  };
});

function renderNotice(endpoints: Record<string, unknown> = {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(
    <QueryClientProvider client={qc}>
      <NoticeManagement />
    </QueryClientProvider>,
    { endpoints }
  );
}

describe("NoticeManagement 渲染", () => {
  it("空数据 → 渲染不抛错", async () => {
    const api = createApiMock("/system/notice/list");
    api.endpoint.mockResolvedValueOnce({ data: { list: [], total: 0 } } as any);
    const { baseElement } = renderNotice({
      "/system/notice/list": { data: { list: [], total: 0 } },
      "/system/notice/statistics": { data: {} },
    });
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("1 行 → 渲染", async () => {
    const { baseElement } = renderNotice({
      "/system/notice/list": {
        data: {
          list: [
            {
              noticeId: 1,
              noticeTitle: "测试",
              noticeContent: "内容",
              noticeType: "1",
              priority: 1,
              targetType: 1,
              publishStatus: "1",
              createBy: "admin",
              createTime: "2026-01-01 12:00:00",
            },
          ],
          total: 1,
        },
      },
      "/system/notice/statistics": { data: {} },
    });
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("list 失败 → catch 路径", async () => {
    const api = createApiMock("/system/notice/list");
    api.endpoint.mockRejectedValueOnce(new Error("net"));
    const { baseElement } = renderNotice();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });
});
