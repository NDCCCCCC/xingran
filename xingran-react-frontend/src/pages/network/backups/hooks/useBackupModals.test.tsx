/**
 * Phase 88 Batch68 — useBackupModals hook 测试(15 个 handler)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, ConfigProvider } from "antd";

vi.mock("@/lib/api", () => ({
  post: vi.fn().mockResolvedValue({ data: {} }),
}));

import { useBackupModals } from "../hooks/useBackupModals";

beforeEach(() => {
  vi.clearAllMocks();
});

const wrap = ({ children }: { children: React.ReactNode }) => (
  <ConfigProvider>
    <App>{children}</App>
  </ConfigProvider>
);

const opts = () => ({ onLoad: vi.fn() });

describe("useBackupModals — open/close 系列", () => {
  it("openBackupModal 设 backupModalVisible=true", () => {
    const { result } = renderHook(() => useBackupModals(opts()), { wrapper: wrap });
    act(() => result.current.openBackupModal());
    expect(result.current.backupModalVisible).toBe(true);
  });

  it("closeBackupModal(form) 设 false + resetFields", () => {
    const { result } = renderHook(() => useBackupModals(opts()), { wrapper: wrap });
    const form = { resetFields: vi.fn() };
    act(() => result.current.openBackupModal());
    act(() => result.current.closeBackupModal(form as any));
    expect(result.current.backupModalVisible).toBe(false);
    expect(form.resetFields).toHaveBeenCalled();
  });

  it("openRestoreModal + closeRestoreModal", () => {
    const { result } = renderHook(() => useBackupModals(opts()), { wrapper: wrap });
    act(() => result.current.openRestoreModal({ id: "b1" } as any));
    expect(result.current.restoreModalVisible).toBe(true);
    expect(result.current.selectedRestoreBackup?.id).toBe("b1");
    act(() => result.current.closeRestoreModal());
    expect(result.current.restoreModalVisible).toBe(false);
  });

  it("openContentDrawer 调 post + 设 backupContent", async () => {
    const { post } = await import("@/lib/api");
    vi.mocked(post).mockResolvedValueOnce({ data: { content: "config text" } });
    const { result } = renderHook(() => useBackupModals(opts()), { wrapper: wrap });
    await act(async () => {
      await result.current.openContentDrawer({ id: "b1" } as any);
    });
    expect(result.current.contentDrawerVisible).toBe(true);
    expect(result.current.backupContent).toBe("config text");
  });

  it("openVersionListDrawer + closeVersionListDrawer", () => {
    const { result } = renderHook(() => useBackupModals(opts()), { wrapper: wrap });
    act(() =>
      result.current.openVersionListDrawer({
        deviceId: "d1",
        backups: [],
      } as any)
    );
    expect(result.current.versionListDrawerVisible).toBe(true);
    act(() => result.current.closeVersionListDrawer());
    expect(result.current.versionListDrawerVisible).toBe(false);
  });
});

describe("useBackupModals — handle 系列", () => {
  it("handleBackup validateFields + post + onLoad", async () => {
    const onLoad = vi.fn();
    const { result } = renderHook(() => useBackupModals({ onLoad }), { wrapper: wrap });
    const form = {
      validateFields: vi.fn().mockResolvedValue({ deviceIds: ["d1"] }),
      resetFields: vi.fn(),
    } as any;
    await act(async () => {
      await result.current.handleBackup(form);
    });
    expect(onLoad).toHaveBeenCalled();
  });

  it("handleBackup validate 失败 → 不调 post", async () => {
    const form = {
      validateFields: vi.fn().mockRejectedValue({ errorFields: [] }),
      resetFields: vi.fn(),
    } as any;
    const { result } = renderHook(() => useBackupModals(opts()), { wrapper: wrap });
    await act(async () => {
      await result.current.handleBackup(form);
    });
  });

  it("handleRestore 调 post + onLoad", async () => {
    const onLoad = vi.fn();
    const { result } = renderHook(() => useBackupModals({ onLoad }), { wrapper: wrap });
    act(() => result.current.openRestoreModal({ id: "b1" } as any));
    await act(async () => {
      await result.current.handleRestore();
    });
    expect(onLoad).toHaveBeenCalled();
  });
});
