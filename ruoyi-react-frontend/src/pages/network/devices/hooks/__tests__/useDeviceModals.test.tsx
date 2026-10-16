/**
 * Phase 88 Batch295 — pages/network/devices/hooks/useDeviceModals 测试
 */
import { describe, it, expect } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { useDeviceModals } from "../useDeviceModals";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const sampleDevice: any = { id: "d1", deviceName: "Switch-01" };

describe("network/devices/hooks/useDeviceModals", () => {
  it("初始 visible=false + null", () => {
    const { result } = renderHook(() => useDeviceModals(), { wrapper });
    expect(result.current.quickCreateModalVisible).toBe(false);
    expect(result.current.detailModalVisible).toBe(false);
    expect(result.current.viewingDevice).toBeNull();
    expect(result.current.probeResult).toBeNull();
    expect(result.current.probing).toBe(false);
    expect(result.current.creating).toBe(false);
  });

  it("openDetailModal 设置 device + visible", () => {
    const { result } = renderHook(() => useDeviceModals(), { wrapper });
    act(() => result.current.openDetailModal(sampleDevice));
    expect(result.current.viewingDevice?.id).toBe("d1");
    expect(result.current.detailModalVisible).toBe(true);
  });

  it("closeQuickCreateModal 关闭 + 清 probeResult", () => {
    const { result } = renderHook(() => useDeviceModals(), { wrapper });
    act(() => result.current.setQuickCreateModalVisible(true));
    act(() => result.current.setProbeResult({ success: true }));
    act(() => result.current.closeQuickCreateModal());
    expect(result.current.quickCreateModalVisible).toBe(false);
    expect(result.current.probeResult).toBeNull();
  });

  it("set 切换函数", () => {
    const { result } = renderHook(() => useDeviceModals(), { wrapper });
    act(() => result.current.setDetailModalVisible(true));
    expect(result.current.detailModalVisible).toBe(true);
    act(() => result.current.setViewingDevice(sampleDevice));
    expect(result.current.viewingDevice?.id).toBe("d1");
    act(() => result.current.setProbing(true));
    expect(result.current.probing).toBe(true);
    act(() => result.current.setCreating(true));
    expect(result.current.creating).toBe(true);
  });
});
