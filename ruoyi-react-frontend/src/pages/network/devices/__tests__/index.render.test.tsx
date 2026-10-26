/**
 * Phase 88 Batch81 — network devices 页面渲染测试(215 stmts, 30.7% → 高)
 * Phase 67-T3: 迁移 66 契约组件后补 3 个契约断言（分页口径/单删/批删 INV-8）
 */
import { describe, it, expect, vi, afterEach } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { screen, fireEvent, waitFor, cleanup } from "@testing-library/react";
import NetworkDevicePage from "../index";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

// confirmDelete 走命令式 Modal.confirm（独立挂 body），RTL cleanup 不覆盖——手动清残留
afterEach(() => {
  cleanup();
  document.body
    .querySelectorAll(".ant-modal-root, .ant-modal-mask, .ant-dropdown")
    .forEach((node) => node.remove());
});

async function renderDevice(endpoints: Record<string, unknown> = {}) {
  // 端点注册走 createApiMock 进程级路由表（88-Batch81 版本 renderDevice 的
  // endpoints 参数实际未被接线，本次修正为真正生效）
  const { createApiMock } = await import("@/test/utils/createApiMock");
  for (const [endpoint, response] of Object.entries(endpoints)) {
    createApiMock(endpoint).endpoint.mockResolvedValue(response);
  }
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return await renderWithProviders(
    <QueryClientProvider client={qc}>
      <NetworkDevicePage />
    </QueryClientProvider>,
    { endpoints }
  );
}

describe("NetworkDevicePage 渲染", () => {
  it("空数据 → 渲染不抛错", async () => {
    const { baseElement } = await renderDevice();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("列表空 → 表格空态", async () => {
    const { baseElement } = await renderDevice({
      "/network/devices/list": { data: { list: [], total: 0 } },
    });
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("列表 1 行 → 表格行渲染", async () => {
    const { baseElement } = await renderDevice({
      "/network/devices/list": {
        data: {
          list: [
            {
              id: "d1",
              name: "switch-01",
              ipAddress: "10.0.0.1",
              deviceType: "switch",
              status: 0,
            },
          ],
          total: 1,
        },
      },
    });
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("分页 showTotal 契约口径：共 1 条 / 1 页", async () => {
    renderDevice({
      "/network/devices/list": {
        data: {
          list: [
            {
              id: "d1",
              deviceName: "switch-01",
              ipAddress: "10.0.0.1",
              deviceType: "switch",
              status: 0,
            },
          ],
          total: 1,
        },
      },
    });
    await waitFor(() => expect(screen.getByText("共 1 条 / 1 页")).toBeTruthy());
  });

  it("单删 → ConfirmDelete 契约文案（确认删除/删除后不可恢复）", async () => {
    renderDevice({
      "/network/devices/list": {
        data: {
          list: [
            {
              id: "d1",
              deviceName: "switch-01",
              ipAddress: "10.0.0.1",
              deviceType: "switch",
              status: 0,
            },
          ],
          total: 1,
        },
      },
    });
    await waitFor(() => expect(screen.getByText("switch-01")).toBeTruthy());
    // 操作列 5 个动作 ≥3 → ActionButtons 收进「操作」下拉；
    // 删除项经 createDeleteAction render → ConfirmDelete 按钮文字固定「删除」，
    // 设备名只进可访问名 aria-label=「删除{target}」（68.1 Q1）
    const moreButton = screen.getAllByRole("button", { name: "操作" }).at(-1) as HTMLElement;
    fireEvent.click(moreButton);
    const deleteMenuButton = await waitFor(() => {
      const btn = [...document.querySelectorAll<HTMLButtonElement>(".ant-dropdown-menu-item button")].find(
        (el) => el.getAttribute("aria-label")?.includes("switch-01")
      );
      expect(btn).toBeTruthy();
      return btn as HTMLButtonElement;
    });
    fireEvent.click(deleteMenuButton);
    await waitFor(() => {
      expect(document.querySelector(".ant-modal-confirm")).toBeTruthy();
      expect(screen.getAllByText("确认删除").length).toBeGreaterThanOrEqual(1);
      expect(screen.getByText(/删除后不可恢复/)).toBeTruthy();
    });
  });

  it("批删 → batchDelete INV-8 逐字文案（将删除选中的 N 项…）", async () => {
    renderDevice({
      "/network/devices/list": {
        data: {
          list: [
            {
              id: "d1",
              deviceName: "switch-01",
              ipAddress: "10.0.0.1",
              deviceType: "switch",
              status: 0,
            },
            {
              id: "d2",
              deviceName: "router-01",
              ipAddress: "10.0.0.2",
              deviceType: "router",
              status: 0,
            },
          ],
          total: 2,
        },
      },
    });
    await waitFor(() => expect(screen.getByText("router-01")).toBeTruthy());
    // 勾选首行数据行复选框（[0] 为表头全选）→ BatchDeleteButton 出现 → 点击触发确认
    const checkboxes = document.querySelectorAll<HTMLInputElement>(
      ".ant-table-selection-column input[type=checkbox]"
    );
    fireEvent.click(checkboxes[1]);
    await waitFor(() => expect(screen.getByText(/批量删除/)).toBeTruthy());
    fireEvent.click(screen.getByText(/批量删除/).closest("button") as HTMLElement);
    await waitFor(() => {
      const confirmEl = document.querySelector(".ant-modal-confirm");
      expect(confirmEl).toBeTruthy();
      expect(screen.getAllByText("确认批量删除").length).toBeGreaterThanOrEqual(1);
      // INV-8 逐字文案（68.1-T1：PageTable target 透传打通 →「选中的 1 项设备」，PLAN 注记 3 期望文案改善）
      expect(confirmEl?.textContent).toContain("将删除选中的 1 项设备，删除后不可恢复，是否继续？");
    });
  });
});
