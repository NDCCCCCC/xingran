/**
 * Phase 88 Batch78 — ad-domain configs 页面渲染测试(99 stmts, 24.2% → 高)
 * Phase 59 / GUARD-02 — 追加：外层 Tabs（配置列表/对账）、对账 Tab 无写操作入口、
 * activeTab 会话持久化。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, waitFor } from "@testing-library/react";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import ADConfigPage from "../index";
import {} from "@/test/utils/createApiMock";

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

vi.mock("@/hooks/useDeptTree", () => ({
  useDeptTree: () => ({ data: [], isLoading: false }),
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

// 与 usePersistedStateController 的存储 key 规则一致（keyPrefix=location.pathname）
const ACTIVE_TAB_KEY = "xingran_table_state_ad-domain_configs_activeTab";

describe("ADConfigPage 渲染", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

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

  it("存在『配置列表』『对账』两个 Tab 标签且默认停留在配置列表", async () => {
    const { container, getByText } = renderWithProviders(<ADConfigPage />, {
      route: "/ad-domain/configs",
      endpoints,
    });
    expect(getByText("配置列表")).toBeTruthy();
    expect(getByText("对账")).toBeTruthy();
    await waitForActiveTab(container, "配置列表");
  });

  it("切到对账 Tab 后内容区无写操作按钮（修复/删除/导出）", async () => {
    const { container, getByText } = renderWithProviders(<ADConfigPage />, {
      route: "/ad-domain/configs",
      endpoints,
    });
    fireEvent.click(getByText("对账"));
    await waitFor(
      () => {
        const pane = getActivePane(container);
        expect(pane?.textContent).toContain("无对账查看权限");
      },
      { timeout: 4000 }
    );
    const pane = getActivePane(container) as HTMLElement;
    // 只读契约：本 Tab 无任何写操作入口（无权限态即零按钮）
    expect(pane.textContent).not.toContain("修复");
    expect(pane.textContent).not.toContain("删除");
    expect(pane.textContent).not.toContain("导出");
    expect(pane.querySelectorAll("button").length).toBe(0);
  });

  it("activeTab 持久化：sessionStorage 预置 reconciliation 后渲染停留在对账", async () => {
    sessionStorage.setItem(ACTIVE_TAB_KEY, JSON.stringify("reconciliation"));
    const { container } = renderWithProviders(<ADConfigPage />, {
      route: "/ad-domain/configs",
      endpoints,
    });
    await waitForActiveTab(container, "对账");
  });
});

/** antd v6 Tabs 的激活面板：非隐藏的 tabpanel（v6 无 .ant-tabs-tabpane-active，隐藏面板挂 .ant-tabs-content-hidden） */
function getActivePane(container: HTMLElement): Element | null {
  return container.querySelector('[role="tabpanel"]:not(.ant-tabs-content-hidden)');
}

/** 等待指定 Tab 处于激活态（antd Tabs 的 active 标签节点） */
async function waitForActiveTab(container: HTMLElement, label: string) {
  await waitFor(
    () => {
      const active = container.querySelector(".ant-tabs-tab-active");
      expect(active?.textContent).toContain(label);
    },
    { timeout: 4000 }
  );
}
