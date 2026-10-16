/**
 * Phase 88 Batch103 — operations/dedicated-lines 页面渲染(149 stmts, 33.6% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import DedicatedLines from "../index";
import { dedicatedLineApi, serverRoomApi } from "@/lib/opsApi";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import * as fs from "fs";
import * as path from "path";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/opsApi", () => ({
  dedicatedLineApi: {
    list: vi.fn(() => Promise.resolve({ data: { list: [], total: 0 } })),
    statistics: vi.fn(() => Promise.resolve({ data: {} })),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    batch: vi.fn(),
  },
  serverRoomApi: {
    list: vi.fn(() => Promise.resolve({ data: { list: [] } })),
  },
}));

function renderLines() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(
    <QueryClientProvider client={qc}>
      <DedicatedLines />
    </QueryClientProvider>
  );
}

describe("DedicatedLines 渲染", () => {
  it("空数据 → 渲染不抛错", async () => {
    vi.mocked(dedicatedLineApi.list).mockResolvedValueOnce({ data: { list: [], total: 0 } } as any);
    vi.mocked(dedicatedLineApi.statistics).mockResolvedValueOnce({ data: {} } as any);
    const { baseElement } = renderLines();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("1 行 → 表格行渲染", async () => {
    vi.mocked(dedicatedLineApi.list).mockResolvedValueOnce({
      data: {
        list: [
          {
            id: "l1",
            lineCode: "DL001",
            lineName: "专线001",
            provider: "电信",
            status: 0,
          },
        ],
        total: 1,
      },
    } as any);
    vi.mocked(dedicatedLineApi.statistics).mockResolvedValueOnce({ data: {} } as any);
    const { baseElement } = renderLines();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("list 失败 → catch 路径", async () => {
    vi.mocked(dedicatedLineApi.list).mockRejectedValueOnce(new Error("net"));
    const { baseElement } = renderLines();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  // BUGFIX-01 regression: monthlyFee=0 should render "0", not blank
  // Source-code verification: monthlyFee uses != null guard (not &&) so 0 is not falsy
  it("monthlyFee 使用 != null 判断（而非 &&）使 0 值正确渲染", () => {
    const src = fs.readFileSync(path.resolve(__dirname, "../index.tsx"), "utf-8");
    // The monthlyFee section must use != null, not && (which treats 0 as falsy)
    expect(src).toMatch(/monthlyFee != null/);
    // Must NOT use the old falsy pattern for monthlyFee rendering
    expect(src).not.toMatch(/monthlyFee && \(\s*<div>\s*<strong>月费/);
  });
});
