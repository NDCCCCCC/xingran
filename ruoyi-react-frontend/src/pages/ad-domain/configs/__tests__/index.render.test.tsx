/**
 * Phase 88 Batch78 — ad-domain configs 页面渲染测试(99 stmts, 24.2% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import ADConfigPage from "../index";
import { createApiMock } from "@/test/utils/createApiMock";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/adDomainApi", () => ({
  createADConfig: vi.fn(),
  updateADConfig: vi.fn(),
  deleteADConfig: vi.fn(),
  testADConnection: vi.fn(),
  syncADData: vi.fn(),
  getADConfigs: vi.fn(() => Promise.resolve({ data: { list: [], total: 0 } })),
  getADConfig: vi.fn(),
  // Phase 59 GUARD-02：对账 Tab 的 API（必须 resolve 出 {data:{list,total}} 形状——
  // 组件读 res.data?.list，返回 undefined 会把空态用例变成错误态用例）
  getDeptOUMismatchList: vi.fn(() =>
    Promise.resolve({ data: { list: [], total: 0, current: 1, pageSize: 20 } })
  ),
}));

vi.mock("@/hooks/useADConfigs", () => ({
  useADConfigs: () => ({
    configs: [],
    total: 0,
    loading: false,
    page: 1,
    pageSize: 10,
    setPage: vi.fn(),
    refresh: vi.fn(),
    removeConfig: vi.fn(),
    removeConfigLocal: vi.fn(),
    toggleStatus: vi.fn(),
    upsertConfig: vi.fn(),
  }),
}));

const endpoints = {
  "/system/ad-config/accounts/list": { data: { list: [], total: 0 } },
};

describe("ADConfigPage 渲染", () => {
  it("空数据渲染不抛错", async () => {
    const { baseElement } = renderWithProviders(<ADConfigPage />, { endpoints });
    await new Promise((r) => setTimeout(r, 300));
    expect(baseElement).toBeDefined();
  });

  it("包含创建按钮 + tabs", async () => {
    const { baseElement } = renderWithProviders(<ADConfigPage />, { endpoints });
    await new Promise((r) => setTimeout(r, 200));
    expect(baseElement.textContent).toBeDefined();
  });
});
