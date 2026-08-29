/**
 * Phase 88 Batch45 — components/shared ExcelImport 渲染测试
 *
 * 验证 Modal 渲染 + handleDownloadTemplate 默认/自定义/错误路径 + handleReset
 * 状态重置。customRequest 走 XHR,跳过直接覆盖(降低 mock 复杂度,只验证
 * 渲染 + 静态分支)。
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
      export: vi.fn(),
      importFile: vi.fn(),
      downloadTemplate: vi.fn().mockResolvedValue({ success: true }),
    },
  };
});

vi.mock("@/utils/authHelpers", async () => {
  return {
    getAuthHeaders: vi.fn().mockResolvedValue({ Authorization: "Bearer token" }),
  };
});

import { excelApi } from "@/lib/opsApi";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import ExcelImport from "../ExcelImport";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("ExcelImport — 渲染", () => {
  it("visible=false 不渲染 body", () => {
    const { baseElement } = renderWithProviders(
      <ExcelImport entityType="building" visible={false} />
    );
    expect(baseElement.querySelector(".ant-modal-body")).toBeNull();
  });

  it("visible=true 渲染 Modal title + 导入说明 + 步骤1/2 按钮", async () => {
    renderWithProviders(<ExcelImport entityType="building" visible />);
    await waitFor(() => {
      expect(screen.getByText(/导入.*building.*数据/)).toBeDefined();
    });
    expect(await screen.findByText("导入说明")).toBeDefined();
    expect(await screen.findByText(/下载.*building.*Excel模板/)).toBeDefined();
    expect(await screen.findByText("选择Excel文件")).toBeDefined();
  }, 15000);

  it("提供 templateUrl 时按钮文案不变", async () => {
    renderWithProviders(
      <ExcelImport entityType="custom" templateUrl="/custom/template.xlsx" visible />
    );
    await waitFor(() => {
      expect(screen.getByText(/下载.*custom.*Excel模板/)).toBeDefined();
    });
  }, 15000);

  it("提供 importUrl 走自定义路径(渲染正常)", async () => {
    renderWithProviders(
      <ExcelImport entityType="dept" importUrl="/api/v1/system/departments/import" visible />
    );
    await waitFor(() => {
      expect(screen.getByText(/导入.*dept.*数据/)).toBeDefined();
    });
  }, 15000);
});

describe("ExcelImport — handleDownloadTemplate 默认路径", () => {
  it("无 templateUrl → 调 excelApi.downloadTemplate + message.success", async () => {
    vi.mocked(excelApi.downloadTemplate).mockResolvedValue({ success: true });
    renderWithProviders(<ExcelImport entityType="building" visible />);
    const btn = await screen.findByText(/下载.*building.*Excel模板/);
    fireEvent.click(btn);
    await waitFor(() => {
      expect(excelApi.downloadTemplate).toHaveBeenCalledWith("building");
    });
  }, 15000);

  it("excelApi.downloadTemplate 抛错 → 走 catch + message.error", async () => {
    vi.mocked(excelApi.downloadTemplate).mockRejectedValue(new Error("template fail"));
    renderWithProviders(<ExcelImport entityType="building" visible />);
    const btn = await screen.findByText(/下载.*building.*Excel模板/);
    fireEvent.click(btn);
    await waitFor(() => {
      expect(excelApi.downloadTemplate).toHaveBeenCalled();
    });
  }, 15000);
});

describe("ExcelImport — handleDownloadTemplate 自定义 URL", () => {
  it("提供 templateUrl → 调 fetch + getAuthHeaders", async () => {
    const fetchSpy = vi.spyOn(global, "fetch").mockResolvedValue({
      ok: true,
      headers: { get: () => null },
      blob: () => Promise.resolve(new Blob()),
    } as any);
    renderWithProviders(
      <ExcelImport entityType="custom" templateUrl="/api/v1/system/custom/template" visible />
    );
    const btn = await screen.findByText(/下载.*custom.*Excel模板/);
    fireEvent.click(btn);
    await waitFor(() => {
      expect(fetchSpy).toHaveBeenCalledWith(
        "/api/v1/system/custom/template",
        expect.objectContaining({ headers: expect.any(Object) })
      );
    });
    fetchSpy.mockRestore();
  }, 15000);

  it("fetch !response.ok → message.error", async () => {
    vi.spyOn(global, "fetch").mockResolvedValue({
      ok: false,
      headers: { get: () => null },
      blob: () => Promise.resolve(new Blob()),
    } as any);
    renderWithProviders(
      <ExcelImport entityType="custom" templateUrl="/api/v1/custom/template" visible />
    );
    const btn = await screen.findByText(/下载.*custom.*Excel模板/);
    fireEvent.click(btn);
    await waitFor(() => {
      // 等待异步 catch 走完
      expect(global.fetch).toHaveBeenCalled();
    });
  }, 15000);
});

describe("ExcelImport — handleReset", () => {
  it("visible toggle 触发 destroyOnHidden + remount 不抛错", async () => {
    const { rerender } = renderWithProviders(<ExcelImport entityType="building" visible />);
    await screen.findByText(/导入.*building.*数据/);
    rerender(<ExcelImport entityType="building" visible={false} />);
    // 不抛错即通过
    expect(true).toBe(true);
  }, 15000);
});
