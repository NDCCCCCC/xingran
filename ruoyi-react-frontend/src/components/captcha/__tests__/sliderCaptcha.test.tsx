/**
 * Phase 88 Batch31 — SliderCaptcha 组件测试(原 2/91)
 *
 * mock @/services/captcha 控制加载分支。
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/services/captcha", () => ({
  getCaptcha: vi.fn(),
  verifySliderCaptcha: vi.fn(),
}));

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import SliderCaptcha from "../SliderCaptcha";
import { getCaptcha } from "@/services/captcha";

describe("SliderCaptcha", () => {
  it("active=true 挂载即拉取验证码", async () => {
    (getCaptcha as any).mockResolvedValue({
      captchaId: "c1",
      captchaType: "slider",
      backgroundImage: "data:image/png;base64,x",
      sliderImage: "data:image/png;base64,y",
      sliderY: 50,
    });
    const onVerified = vi.fn();
    const { container } = renderWithProviders(<SliderCaptcha onVerified={onVerified} active />);

    await vi.waitFor(() => {
      expect(getCaptcha).toHaveBeenCalled();
    });
    expect(container).not.toBeNull();
  }, 15000);

  it("captchaType 非 slider 触发 onError", async () => {
    (getCaptcha as any).mockResolvedValue({ captchaId: "c2", captchaType: "text" });
    const onError = vi.fn();
    renderWithProviders(<SliderCaptcha onVerified={vi.fn()} onError={onError} active />);

    await vi.waitFor(() => {
      expect(onError).toHaveBeenCalledWith("CAPTCHA_TYPE_MISMATCH");
    });
  }, 15000);

  it("空 captchaType 静默(未启用验证码)", async () => {
    (getCaptcha as any).mockResolvedValue({});
    const onError = vi.fn();
    const { container } = renderWithProviders(
      <SliderCaptcha onVerified={vi.fn()} onError={onError} active />
    );
    await vi.waitFor(() => {
      expect(getCaptcha).toHaveBeenCalled();
    });
    expect(onError).not.toHaveBeenCalled();
    expect(container).not.toBeNull();
  }, 15000);

  it("getCaptcha throw 静默", async () => {
    (getCaptcha as any).mockRejectedValue(new Error("boom"));
    const { container } = renderWithProviders(<SliderCaptcha onVerified={vi.fn()} active />);
    await vi.waitFor(() => {
      expect(getCaptcha).toHaveBeenCalled();
    });
    expect(container).not.toBeNull();
  }, 15000);

  it("active=false 不拉取", () => {
    (getCaptcha as any).mockClear();
    (getCaptcha as any).mockResolvedValue({});
    renderWithProviders(<SliderCaptcha onVerified={vi.fn()} active={false} />);
    expect((getCaptcha as any).mock.calls.length).toBe(0);
  });
});
