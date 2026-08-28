/**
 * Phase 88 Batch36 — system/apikeys/LogsModal 渲染测试
 *
 * 桩掉 @/api/apikey + renderPageWithEndpoints 模式
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/api/apikey", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/api/apikey")>();
  return {
    ...actual,
    listUsageLogs: vi.fn().mockResolvedValue({
      data: {
        list: [
          {
            id: "log-1",
            method: "POST",
            path: "/system/users",
            statusCode: 200,
            ipAddress: "127.0.0.1",
            createdAt: "2026-08-28T10:00:00Z",
          },
          {
            id: "log-2",
            method: "POST",
            path: "/system/users",
            statusCode: 500,
            ipAddress: "127.0.0.1",
            createdAt: "2026-08-28T10:05:00Z",
          },
        ],
        total: 2,
      },
    }),
    getUsageSummary: vi.fn().mockResolvedValue({
      total: 100,
      success: 80,
      failed: 20,
      last24h: 12,
    }),
  };
});

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import LogsModal from "../LogsModal";
import { screen } from "@testing-library/react";

describe("LogsModal 渲染", () => {
  it("visible=false 不渲染内容", () => {
    const { baseElement } = renderWithProviders(
      <LogsModal visible={false} apiKeyId="k1" onClose={vi.fn()} />
    );
    // AntD Modal 在 visible=false 时不渲染 .ant-modal-body
    expect(baseElement.querySelector(".ant-modal-body")).toBeNull();
  });

  it("visible=true 渲染表格行 + 状态 tag", async () => {
    renderWithProviders(
      <LogsModal visible apiKeyId="k1" onClose={vi.fn()} />
    );
    // 2 条记录 → 至少 2 处 POST 文本
    expect((await screen.findAllByText("POST")).length).toBeGreaterThanOrEqual(2);
    expect((await screen.findAllByText("/system/users")).length).toBeGreaterThanOrEqual(2);
    // 状态码列 Tag(200 = success,500 = error) → ≥ 2 tag
    await vi.waitFor(
      () => {
        expect(document.querySelectorAll(".ant-tag").length).toBeGreaterThanOrEqual(2);
      },
      { timeout: 5000 }
    );
  }, 15000);

  it("apiKeyId 未传不 crash", () => {
    const { baseElement } = renderWithProviders(
      <LogsModal visible apiKeyId="" onClose={vi.fn()} />
    );
    expect(baseElement).not.toBeNull();
  });
});
