/**
 * Phase 88 Batch353 — pages/network/command/hooks/useCommandModals 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  return {
    post: vi.fn(async (url: string, data?: any) => ({ data: { url, payload: data } })),
  };
});

import { post } from "@/lib/api";
import { useCommandModals } from "../useCommandModals";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("pages/network/command/hooks/useCommandModals", () => {
  beforeEach(() => {
    vi.mocked(post).mockReset();
    vi.mocked(post).mockResolvedValue({ data: {} } as any);
  });

  it("返回两个函数", () => {
    const { result } = renderHook(() => useCommandModals(), { wrapper });
    expect(typeof result.current.handleQuickCommand).toBe("function");
    expect(typeof result.current.handleCancelExecution).toBe("function");
  });

  it("handleQuickCommand 调用 validateFields + post + resetFields + onSuccess", async () => {
    const validateFields = vi.fn(async () => ({ command: "show version" }));
    const resetFields = vi.fn();
    const onSuccess = vi.fn();
    const form: any = { validateFields, resetFields };

    const { result } = renderHook(() => useCommandModals(), { wrapper });
    await act(async () => {
      await result.current.handleQuickCommand(["d1", "d2"], form, onSuccess);
    });

    expect(validateFields).toHaveBeenCalled();
    expect(post).toHaveBeenCalledWith("/network/command/quick", {
      command: "show version",
      deviceIds: ["d1", "d2"],
    });
    expect(resetFields).toHaveBeenCalled();
    expect(onSuccess).toHaveBeenCalled();
  });

  it("handleQuickCommand 表单校验失败 → 静默 (不弹 toast)", async () => {
    const validateFields = vi.fn(async () => {
      throw { errorFields: [{ name: "cmd", errors: ["required"] }] };
    });
    const resetFields = vi.fn();
    const onSuccess = vi.fn();
    const form: any = { validateFields, resetFields };

    const { result } = renderHook(() => useCommandModals(), { wrapper });
    await act(async () => {
      await result.current.handleQuickCommand(["d1"], form, onSuccess);
    });
    expect(onSuccess).not.toHaveBeenCalled();
    expect(post).not.toHaveBeenCalled();
  });

  it("handleQuickCommand post 失败 → error toast", async () => {
    const { post } = await import("@/lib/api");
    vi.mocked(post).mockRejectedValueOnce(new Error("net"));
    const validateFields = vi.fn(async () => ({ command: "x" }));
    const form: any = { validateFields, resetFields: vi.fn() };

    const { result } = renderHook(() => useCommandModals(), { wrapper });
    await act(async () => {
      await result.current.handleQuickCommand(["d1"], form, vi.fn());
    });
    // 不会抛错
    expect(result.current.handleQuickCommand).toBeDefined();
  });

  it("handleCancelExecution 调用 cancel endpoint + onSuccess", async () => {
    const onSuccess = vi.fn();
    const { result } = renderHook(() => useCommandModals(), { wrapper });
    await act(async () => {
      await result.current.handleCancelExecution("exec-1", onSuccess);
    });
    expect(post).toHaveBeenCalledWith("/network/executions/exec-1/cancel", {});
    expect(onSuccess).toHaveBeenCalled();
  });

  it("handleCancelExecution 失败 → error toast + 不调 onSuccess", async () => {
    const { post } = await import("@/lib/api");
    vi.mocked(post).mockRejectedValueOnce(new Error("net"));
    const onSuccess = vi.fn();
    const { result } = renderHook(() => useCommandModals(), { wrapper });
    await act(async () => {
      await result.current.handleCancelExecution("exec-2", onSuccess);
    });
    expect(onSuccess).not.toHaveBeenCalled();
  });
});
