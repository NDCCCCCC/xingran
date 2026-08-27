/**
 * Phase 84 84-02b — TextCaptcha mock getCaptcha 静态测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import TextCaptcha from "../TextCaptcha";

vi.mock("@/services/captcha", () => ({
  getCaptcha: vi.fn(() =>
    Promise.resolve({
      captchaType: "normal",
      captchaId: "abc-123",
      captchaImage: "data:image/png;base64,iVBORw0KGgo=",
    })
  ),
}));

describe("TextCaptcha", () => {
  it("renders without crash", () => {
    const { container } = renderWithProviders(<TextCaptcha onChange={vi.fn()} />);
    expect(container).not.toBeNull();
  });

  it("invokes onError callback prop without error", () => {
    const onError = vi.fn();
    expect(() =>
      renderWithProviders(<TextCaptcha onChange={vi.fn()} onError={onError} />)
    ).not.toThrow();
  });
});
