/**
 * Phase 88 Batch281 — pages/system/user/hooks/useUserModals 测试
 */
import { describe, it, expect } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { useUserModals } from "../useUserModals";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const sampleUser: any = { id: "u1", username: "test" };

describe("system/user/hooks/useUserModals", () => {
  it("初始 visible=false + user=null", () => {
    const { result } = renderHook(() => useUserModals(), { wrapper });
    expect(result.current.resetPasswordModalVisible).toBe(false);
    expect(result.current.resettingUser).toBeNull();
  });

  it("openResetPasswordModal 设置 user + visible", () => {
    const { result } = renderHook(() => useUserModals(), { wrapper });
    act(() => result.current.openResetPasswordModal(sampleUser));
    expect(result.current.resettingUser?.id).toBe("u1");
    expect(result.current.resetPasswordModalVisible).toBe(true);
  });

  it("closeResetPasswordModal 清状态", () => {
    const { result } = renderHook(() => useUserModals(), { wrapper });
    act(() => result.current.openResetPasswordModal(sampleUser));
    act(() => result.current.closeResetPasswordModal());
    expect(result.current.resetPasswordModalVisible).toBe(false);
    expect(result.current.resettingUser).toBeNull();
  });

  it("setResetPasswordModalVisible 切换", () => {
    const { result } = renderHook(() => useUserModals(), { wrapper });
    act(() => result.current.setResetPasswordModalVisible(true));
    expect(result.current.resetPasswordModalVisible).toBe(true);
    act(() => result.current.setResetPasswordModalVisible(false));
    expect(result.current.resetPasswordModalVisible).toBe(false);
  });

  it("setResettingUser 切换", () => {
    const { result } = renderHook(() => useUserModals(), { wrapper });
    act(() => result.current.setResettingUser(sampleUser));
    expect(result.current.resettingUser?.username).toBe("test");
    act(() => result.current.setResettingUser(null));
    expect(result.current.resettingUser).toBeNull();
  });

  it("resetPasswordForm 存在", () => {
    const { result } = renderHook(() => useUserModals(), { wrapper });
    expect(result.current.resetPasswordForm).toBeDefined();
  });
});
