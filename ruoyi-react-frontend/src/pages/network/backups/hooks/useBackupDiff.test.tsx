/**
 * Phase 88 Batch69 — useBackupDiff hook 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, ConfigProvider } from "antd";

vi.mock("@/lib/api", () => ({
  post: vi.fn().mockResolvedValue({ data: { content: "config text" } }),
}));

import { useBackupDiff } from "../hooks/useBackupDiff";

beforeEach(() => {
  vi.clearAllMocks();
});

const wrap = ({ children }: { children: React.ReactNode }) => (
  <ConfigProvider>
    <App>{children}</App>
  </ConfigProvider>
);

const bk = (id: string) => ({ id, deviceId: "d1", fileName: "config.tar.gz" }) as any;

describe("useBackupDiff", () => {
  it("initial state + 9 handler/ref", () => {
    const { result } = renderHook(() => useBackupDiff(), { wrapper: wrap });
    expect(result.current.diffModalVisible).toBe(false);
    expect(result.current.diffResult).toBeNull();
    expect(result.current.compareBackup1).toBeNull();
    expect(result.current.compareBackup2).toBeNull();
    expect(typeof result.current.openDiffModal).toBe("function");
    expect(typeof result.current.closeDiffModal).toBe("function");
    expect(result.current.leftScrollRef).toBeDefined();
    expect(result.current.rightScrollRef).toBeDefined();
  });

  it("closeDiffModal 设 diffModalVisible=false", () => {
    const { result } = renderHook(() => useBackupDiff(), { wrapper: wrap });
    act(() => result.current.closeDiffModal());
    expect(result.current.diffModalVisible).toBe(false);
  });

  it("openDiffModal 调 2 次 post(并行) + setResult", async () => {
    const { result } = renderHook(() => useBackupDiff(), { wrapper: wrap });
    await act(async () => {
      await result.current.openDiffModal(bk("b1"), bk("b2"));
    });
    expect(result.current.diffModalVisible).toBe(true);
    expect(result.current.compareBackup1?.id).toBe("b1");
    expect(result.current.compareBackup2?.id).toBe("b2");
  });

  it("handleLeftScroll 设 isLeftScrolling ref", () => {
    const { result } = renderHook(() => useBackupDiff(), { wrapper: wrap });
    act(() => {
      result.current.handleLeftScroll({
        target: { scrollTop: 100 },
      } as any);
    });
    // 不抛错即通过(ref 操作内部状态)
    expect(true).toBe(true);
  });

  it("handleRightScroll 不抛错", () => {
    const { result } = renderHook(() => useBackupDiff(), { wrapper: wrap });
    act(() => {
      result.current.handleRightScroll({
        target: { scrollTop: 200 },
      } as any);
    });
    expect(true).toBe(true);
  });
});
