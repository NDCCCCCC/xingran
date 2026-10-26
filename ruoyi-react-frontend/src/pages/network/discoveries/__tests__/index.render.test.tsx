/**
 * Phase 88 Batch91 — network discoveries 页面渲染(89 stmts, 29.2% → 高)
 * Phase 67 T1 迁移后：走真实 PageTable 契约链路；断言保留（分页 showTotal、
 * 删除确认文案、统计卡渲染）+ 新增状态筛选接线断言（C 类 1 缺陷修复）。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor, fireEvent } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/api/networkApi", () => ({
  batchExport: vi.fn(() => Promise.resolve({})),
}));

import { createApiMock } from "@/test/utils/createApiMock";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import DeviceDiscoveryPage from "../index";

describe("DeviceDiscoveryPage 渲染（PageTable 契约链路）", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("空数据 → 渲染不抛错（antd 默认空态，无分页文案）", async () => {
    const api = createApiMock("/network/discoveries/list");
    api.endpoint.mockResolvedValue({ data: { list: [], total: 0 } });
    const { baseElement, container } = renderWithProviders(<DeviceDiscoveryPage />);
    await waitFor(() => {
      expect(container.querySelector(".ant-table")).not.toBeNull();
    });
    expect(baseElement.textContent).not.toContain("共 0 条");
  });

  it("1 行 → 表格行渲染 + 分页 showTotal 逐字口径", async () => {
    const api = createApiMock("/network/discoveries/list");
    api.endpoint.mockResolvedValue({
      data: {
        list: [
          {
            id: "d1",
            taskName: "扫描任务A",
            discoveryType: "snmp",
            ipRanges: ["10.0.0.0/24"],
            totalIPs: 256,
            discoveredCount: 5,
            status: "completed",
            startedAt: "2026-01-01 12:00:00",
            completedAt: "2026-01-01 12:05:00",
          },
        ],
        total: 1,
      },
    });
    renderWithProviders(<DeviceDiscoveryPage />);
    await waitFor(
      () => {
        expect(screen.getAllByText("扫描任务A").length).toBeGreaterThan(0);
      },
      { timeout: 5000 }
    );
    // 63 §1 逐字口径（INV-1）
    expect(screen.getByText("共 1 条 / 1 页")).not.toBeNull();
  });

  it("统计卡渲染（任务总数/待执行/扫描中/已完成/失败/发现设备）", async () => {
    const api = createApiMock("/network/discoveries/list");
    api.endpoint.mockResolvedValue({ data: { list: [], total: 0 } });
    const statsApi = createApiMock("/network/discoveries/statistics");
    statsApi.endpoint.mockResolvedValue({
      data: { total: 9, pending: 1, running: 2, completed: 3, failed: 4, totalDevices: 5 },
    });
    const { container } = renderWithProviders(<DeviceDiscoveryPage />);
    await waitFor(() => {
      expect(container.querySelector(".ant-table")).not.toBeNull();
    });
    await waitFor(() => {
      expect(screen.getByText("任务总数")).not.toBeNull();
      expect(screen.getByText("发现设备")).not.toBeNull();
    });
  });

  it("list 失败 → ErrorAlertWithRetry 错误条（request 失败仍联动统计）", async () => {
    const api = createApiMock("/network/discoveries/list");
    api.endpoint.mockRejectedValue(new Error("网络异常"));
    const statsApi = createApiMock("/network/discoveries/statistics");
    statsApi.endpoint.mockResolvedValue({ data: {} });
    const { container } = renderWithProviders(<DeviceDiscoveryPage />);
    await waitFor(() => {
      expect(container.querySelector(".ant-alert-error")).not.toBeNull();
    });
    // retry = 列表 + 统计都刷（M1-error 裁决）：失败时统计端点同样被调用
    expect(statsApi.endpoint).toHaveBeenCalled();
  });

  it("C 类 1 接线：选择状态后搜索请求参数携带 status", async () => {
    const api = createApiMock("/network/discoveries/list");
    api.endpoint.mockResolvedValue({ data: { list: [], total: 0 } });
    renderWithProviders(<DeviceDiscoveryPage />);
    await waitFor(() => {
      expect(screen.getAllByRole("combobox").length).toBeGreaterThan(0);
    });
    // 打开状态下拉并选择「扫描中」（页面唯一 combobox；列头「状态」文本会撞车，绕开 getByText）
    const combobox = screen.getAllByRole("combobox")[0];
    fireEvent.mouseDown(combobox);
    const option = await waitFor(() => {
      const el = screen.getByText("扫描中", { selector: ".ant-select-item-option-content" });
      return el;
    });
    fireEvent.click(option);
    // 点击「搜索」按钮（SearchForm 内建文案；用文本定位避免 accessible-name 差异）
    fireEvent.click(screen.getByText("搜索").closest("button")!);
    await waitFor(() => {
      const calls = api.endpoint.mock.calls as unknown as Array<[string, Record<string, unknown>]>;
      const searchCall = calls.find(([, body]) => body && "status" in body);
      expect(searchCall).toBeTruthy();
      expect(searchCall![1].status).toBe("running");
    });
  });

  it("删除入口经 ConfirmDelete（65 模板文案）", async () => {
    const api = createApiMock("/network/discoveries/list");
    api.endpoint.mockResolvedValue({
      data: {
        list: [
          {
            id: "d1",
            taskName: "扫描任务A",
            discoveryType: "snmp",
            ipRanges: ["10.0.0.0/24"],
            status: "completed",
          },
        ],
        total: 1,
      },
    });
    renderWithProviders(<DeviceDiscoveryPage />);
    await waitFor(
      () => {
        expect(screen.getAllByText("扫描任务A").length).toBeGreaterThan(0);
      },
      { timeout: 5000 }
    );
    // 删除按钮文字固定「删除」，任务名只进可访问名（68.1 Q1）
    const deleteBtn = screen.getByLabelText("删除扫描任务A");
    expect(deleteBtn).toBeTruthy();
    fireEvent.click(deleteBtn);
    await waitFor(() => {
      // confirmDelete 模板文案（65 §4 唯一入口；B 类对齐）
      expect(document.querySelector(".ant-modal-confirm")).not.toBeNull();
      expect(document.body.textContent).toContain("删除后不可恢复");
      expect(document.body.textContent).toContain("确认删除");
    });
  });
});
