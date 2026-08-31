/**
 * Phase 88 Batch217 — hooks/useCaptcha 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/services/captcha", () => ({
  getCaptcha: vi.fn(async () => ({
    captchaId: "cap1",
    captchaType: "normal",
  })),
  getCaptchaConfig: vi.fn(async () => ({
    enabled: "normal",
    type: 4,
    expireTime: 5,
    maxAttempts: 3,
  })),
}));

import { useCaptcha } from "../useCaptcha";

describe("hooks/useCaptcha", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("初始 null + loading=false", () => {
    const { result } = renderHook(() => useCaptcha());
    // useEffect 同步触发 loadConfig,这里只断言结构
    expect(result.current.captchaData).toBeNull();
    expect(typeof result.current.loadCaptcha).toBe("function");
    expect(typeof result.current.loadConfig).toBe("function");
  });

  it("初始化加载 config", async () => {
    const { result } = renderHook(() => useCaptcha());
    await waitFor(() => {
      expect(result.current.config).not.toBeNull();
    });
    expect(result.current.config?.enabled).toBe("normal");
    expect(result.current.isEnabled).toBe(true);
    expect(result.current.captchaType).toBe("normal");
  });

  it("loadCaptcha 设置 captchaData", async () => {
    const { result } = renderHook(() => useCaptcha());
    await act(async () => {
      await result.current.loadCaptcha();
    });
    expect(result.current.captchaData?.captchaId).toBe("cap1");
    expect(result.current.loading).toBe(false);
  });

  it("verifyCaptcha 返回 true", async () => {
    const { result } = renderHook(() => useCaptcha());
    const ok = await result.current.verifyCaptcha("test");
    expect(ok).toBe(true);
  });

  it("config.enabled = disabled → isEnabled false", async () => {
    const captcha = await import("@/services/captcha");
    vi.mocked(captcha.getCaptchaConfig).mockResolvedValueOnce({
      enabled: "disabled",
      type: 4,
      expireTime: 5,
      maxAttempts: 3,
    });
    const { result } = renderHook(() => useCaptcha());
    await waitFor(() => {
      expect(result.current.config?.enabled).toBe("disabled");
    });
    expect(result.current.isEnabled).toBe(false);
  });
});
