/**
 * Phase 88 Batch123 — components/captcha/CaptchaModal 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/components/captcha", () => ({
  SliderCaptcha: ({ active, onVerified, onError }: any) => (
    <div data-testid="slider-captcha">
      <button onClick={() => onVerified?.("token-1", "captcha-id-1")}>verify-slider</button>
      <button onClick={() => onError?.("CAPTCHA_TYPE_MISMATCH")}>trigger-mismatch</button>
      <span data-testid="active">{String(active)}</span>
    </div>
  ),
  TextCaptcha: ({ value, onChange, onError }: any) => (
    <div data-testid="text-captcha">
      <input
        data-testid="captcha-input"
        value={value}
        onChange={(e) => onChange?.(e.target.value, "id-1")}
      />
      <button onClick={() => onError?.("CAPTCHA_TYPE_MISMATCH")}>trigger-mismatch</button>
    </div>
  ),
}));

import CaptchaModal from "../CaptchaModal";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("CaptchaModal", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("visible=true + slider → 渲染 SliderCaptcha + active", () => {
    const onSuccess = vi.fn();
    const { baseElement } = render(
      <CaptchaModal visible captchaType="slider" onSuccess={onSuccess} onCancel={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.querySelector('[data-testid="slider-captcha"]')).toBeTruthy();
    expect(baseElement.querySelector('[data-testid="active"]')?.textContent).toBe("true");
  });

  it("slider 验证成功 → 500ms 后调用 onSuccess", () => {
    const onSuccess = vi.fn();
    const { baseElement, getByText } = render(
      <CaptchaModal visible captchaType="slider" onSuccess={onSuccess} onCancel={vi.fn()} />,
      { wrapper }
    );
    fireEvent.click(getByText("verify-slider"));
    expect(onSuccess).not.toHaveBeenCalled();
    act(() => {
      vi.advanceTimersByTime(500);
    });
    expect(onSuccess).toHaveBeenCalledWith({
      captchaId: "captcha-id-1",
      verified: true,
    });
  });

  it("CAPTCHA_TYPE_MISMATCH 错误 + onError 提供 → 调用 onError", () => {
    const onError = vi.fn();
    const { getByText } = render(
      <CaptchaModal
        visible
        captchaType="slider"
        onSuccess={vi.fn()}
        onCancel={vi.fn()}
        onError={onError}
      />,
      { wrapper }
    );
    fireEvent.click(getByText("trigger-mismatch"));
    expect(onError).toHaveBeenCalledWith("CAPTCHA_TYPE_MISMATCH");
  });

  it("CAPTCHA_TYPE_MISMATCH 错误 + 无 onError → 调用 onCancel", () => {
    const onCancel = vi.fn();
    const { getByText } = render(
      <CaptchaModal visible captchaType="slider" onSuccess={vi.fn()} onCancel={onCancel} />,
      { wrapper }
    );
    fireEvent.click(getByText("trigger-mismatch"));
    expect(onCancel).toHaveBeenCalled();
  });

  it("visible=true + normal → 渲染 TextCaptcha + 确定按钮", () => {
    const { baseElement, getByText } = render(
      <CaptchaModal visible captchaType="normal" onSuccess={vi.fn()} onCancel={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.querySelector('[data-testid="text-captcha"]')).toBeTruthy();
    expect(baseElement.textContent).toContain("确定");
  });

  it("normal + 空验证码点确定 → warning 提示", () => {
    const onSuccess = vi.fn();
    const { getByText } = render(
      <CaptchaModal visible captchaType="normal" onSuccess={onSuccess} onCancel={vi.fn()} />,
      { wrapper }
    );
    fireEvent.click(getByText("确定"));
    expect(onSuccess).not.toHaveBeenCalled();
  });

  it("normal + 输入验证码点确定 → 500ms 后 onSuccess", async () => {
    const onSuccess = vi.fn();
    const { baseElement, getByText } = render(
      <CaptchaModal visible captchaType="normal" onSuccess={onSuccess} onCancel={vi.fn()} />,
      { wrapper }
    );
    const input = baseElement.querySelector('[data-testid="captcha-input"]') as HTMLInputElement;
    await act(async () => {
      fireEvent.change(input, { target: { value: "abcd" } });
    });
    fireEvent.click(getByText("确定"));
    act(() => {
      vi.advanceTimersByTime(500);
    });
    expect(onSuccess).toHaveBeenCalledWith({
      captchaId: "id-1",
      captcha: "abcd",
    });
  });
});
