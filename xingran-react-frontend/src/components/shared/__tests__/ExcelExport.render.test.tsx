/**
 * Phase 88 Batch44 — components/shared ExcelExport 渲染测试
 *
 * 验证 Modal 渲染 + filters 分支(全部 vs 条件)+ handleExport success/error
 * 两条路径(excelApi.export 默认 + fetch 自定义 URL)。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/opsApi", async () => {
  return {
    excelApi: {
      export: vi.fn().mockResolvedValue({ success: true }),
      importFile: vi.fn(),
      downloadTemplate: vi.fn(),
    },
  };
});

vi.mock("@/utils/authHelpers", async () => {
  return {
    getAuthHeaders: vi.fn().mockResolvedValue({ Authorization: "Bearer token" }),
  };
});

import { excelApi } from "@/lib/opsApi";
import { getAuthHeaders } from "@/utils/authHelpers";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import ExcelExport from "../ExcelExport";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("ExcelExport — 渲染", () => {
  it("visible=false 不渲染 body", () => {
    const { baseElement } = renderWithProviders(
      <ExcelExport entityType="building" visible={false} />
    );
    expect(baseElement.querySelector(".ant-modal-body")).toBeNull();
  });

  it("visible=true 渲染 Modal title + 立即导出按钮", async () => {
    renderWithProviders(<ExcelExport entityType="building" visible />);
    await screen.findByText(/导出.*building.*数据/);
    expect(await screen.findByText("立即导出")).toBeDefined();
  }, 15000);

  it("无 filters 走 '将导出全部数据' warning Alert", async () => {
    renderWithProviders(<ExcelExport entityType="building" visible />);
    await waitFor(() => {
      expect(screen.getByText("将导出全部数据")).toBeDefined();
    });
  }, 15000);

  it("有 filters 走 '将导出以下筛选条件的数据' info Alert + Descriptions", async () => {
    renderWithProviders(
      <ExcelExport
        entityType="building"
        visible
        filters={{ name: "测试", status: 0, orgId: "org-1" }}
      />
    );
    await waitFor(() => {
      expect(screen.getByText("将导出以下筛选条件的数据")).toBeDefined();
    });
    // Descriptions 渲染了 label
    expect(await screen.findByText("名称")).toBeDefined();
  }, 15000);
});

describe("ExcelExport — handleExport 默认路径(excelApi.export)", () => {
  it("点立即导出 → excelApi.export + onClose", async () => {
    vi.mocked(excelApi.export).mockResolvedValue({ success: true });
    const onClose = vi.fn();
    renderWithProviders(
      <ExcelExport entityType="building" visible onClose={onClose} filters={{ name: "x" }} />
    );
    const btn = await screen.findByText("立即导出");
    fireEvent.click(btn);
    await waitFor(() => {
      expect(excelApi.export).toHaveBeenCalledWith("building", { name: "x" });
    });
    await waitFor(() => {
      expect(onClose).toHaveBeenCalled();
    });
  }, 15000);

  it("excelApi.export 抛错 → message.error 不调 onClose", async () => {
    vi.mocked(excelApi.export).mockRejectedValue(new Error("export fail"));
    const onClose = vi.fn();
    renderWithProviders(<ExcelExport entityType="building" visible onClose={onClose} />);
    const btn = await screen.findByText("立即导出");
    fireEvent.click(btn);
    await waitFor(() => {
      expect(excelApi.export).toHaveBeenCalled();
    });
    expect(onClose).not.toHaveBeenCalled();
  }, 15000);
});

describe("ExcelExport — handleExport 自定义 URL 路径(fetch)", () => {
  it("提供 exportUrl 时调 fetch + onClose", async () => {
    const onClose = vi.fn();
    const fetchSpy = vi.spyOn(global, "fetch").mockResolvedValue({
      ok: true,
      headers: { get: () => "attachment; filename=test.xlsx" },
      blob: () => Promise.resolve(new Blob()),
    } as any);
    renderWithProviders(
      <ExcelExport
        entityType="custom"
        exportUrl="/api/v1/custom/export"
        visible
        onClose={onClose}
        filters={{ id: "1" }}
      />
    );
    const btn = await screen.findByText("立即导出");
    fireEvent.click(btn);
    await waitFor(() => {
      expect(getAuthHeaders).toHaveBeenCalled();
      expect(fetchSpy).toHaveBeenCalledWith(
        "/api/v1/custom/export",
        expect.objectContaining({ method: "POST" })
      );
    });
    fetchSpy.mockRestore();
  }, 15000);

  it("fetch !response.ok → message.error", async () => {
    const onClose = vi.fn();
    vi.spyOn(global, "fetch").mockResolvedValue({
      ok: false,
      headers: { get: () => null },
      blob: () => Promise.resolve(new Blob()),
    } as any);
    renderWithProviders(
      <ExcelExport
        entityType="custom"
        exportUrl="/api/v1/custom/export"
        visible
        onClose={onClose}
      />
    );
    const btn = await screen.findByText("立即导出");
    fireEvent.click(btn);
    await waitFor(() => {
      expect(onClose).not.toHaveBeenCalled();
    });
  }, 15000);
});

describe("ExcelExport — 取消按钮", () => {
  it("点取消调 onClose", async () => {
    const onClose = vi.fn();
    const { baseElement } = renderWithProviders(
      <ExcelExport entityType="building" visible onClose={onClose} />
    );
    // Modal footer 在 portal, 用 baseElement 检索
    await waitFor(() => {
      const btns = baseElement.querySelectorAll(".ant-modal-footer .ant-btn");
      expect(btns.length).toBe(2);
    });
    const btns = Array.from(baseElement.querySelectorAll(".ant-modal-footer .ant-btn"));
    // 第一个按钮是"取消",第二个是"立即导出"(Antd Button 在两个字之间插空格,去空白比较)
    const cancelBtn = btns[0] as HTMLElement;
    expect(cancelBtn.textContent?.replace(/\s+/g, "")).toBe("取消");
    fireEvent.click(cancelBtn);
    expect(onClose).toHaveBeenCalled();
  }, 15000);
});
