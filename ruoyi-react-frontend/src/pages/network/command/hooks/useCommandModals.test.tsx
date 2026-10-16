/**
 * Phase 88 Batch64 — useCommandModals hook 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, ConfigProvider } from "antd";

vi.mock("@/lib/api", () => ({
  post: vi.fn().mockResolvedValue({ data: {} }),
}));

import { useCommandModals } from "../hooks/useCommandModals";

beforeEach(() => {
  vi.clearAllMocks();
});

const wrap = ({ children }: { children: React.ReactNode }) => (
  <ConfigProvider>
    <App>{children}</App>
  </ConfigProvider>
);

describe("useCommandModals", () => {
  it("返回 2 handler", () => {
    const { result } = renderHook(() => useCommandModals(), { wrapper: wrap });
    expect(typeof result.current.handleQuickCommand).toBe("function");
    expect(typeof result.current.handleCancelExecution).toBe("function");
  });

  it("handleQuickCommand validateFields + post + onSuccess", async () => {
    const onSuccess = vi.fn();
    const form = {
      validateFields: vi.fn().mockResolvedValue({ command: "show version" }),
      resetFields: vi.fn(),
    };
    const { result } = renderHook(() => useCommandModals(), { wrapper: wrap });
    await act(async () => {
      await result.current.handleQuickCommand(["d1"], form as any, onSuccess);
    });
    expect(onSuccess).toHaveBeenCalled();
    expect(form.resetFields).toHaveBeenCalled();
  });

  it("handleQuickCommand validate 失败 → short-circuit", async () => {
    const onSuccess = vi.fn();
    const form = {
      validateFields: vi.fn().mockRejectedValue({ errorFields: [{ name: "command" }] }),
      resetFields: vi.fn(),
    };
    const { result } = renderHook(() => useCommandModals(), { wrapper: wrap });
    await act(async () => {
      await result.current.handleQuickCommand(["d1"], form as any, onSuccess);
    });
    expect(onSuccess).not.toHaveBeenCalled();
  });

  it("handleCancelExecution 调 post + onSuccess", async () => {
    const onSuccess = vi.fn();
    const { result } = renderHook(() => useCommandModals(), { wrapper: wrap });
    await act(async () => {
      await result.current.handleCancelExecution("e1", onSuccess);
    });
    expect(onSuccess).toHaveBeenCalled();
  });

  it("handleCancelExecution 抛错 → message.error + 不调 onSuccess", async () => {
    const { post } = await import("@/lib/api");
    vi.mocked(post).mockRejectedValueOnce(new Error("cancel fail"));
    const onSuccess = vi.fn();
    const { result } = renderHook(() => useCommandModals(), { wrapper: wrap });
    await act(async () => {
      await result.current.handleCancelExecution("e1", onSuccess);
    });
    expect(onSuccess).not.toHaveBeenCalled();
  });
});
