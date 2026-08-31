/**
 * Phase 88 Batch322 — components/captcha/TextCaptcha 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

let mockCaptcha: any = null;
vi.mock("@/services/captcha", () => ({
  getCaptcha: vi.fn(async () => mockCaptcha),
}));

import TextCaptcha from "../TextCaptcha";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("components/captcha/TextCaptcha", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCaptcha = null;
  });

  it("正常返回 captcha → 渲染图片", async () => {
    mockCaptcha = {
      captchaType: "normal",
      captchaImg: "data:image/png;base64,xxx",
      captchaId: "cap-1",
    };
    render(<TextCaptcha value="" onChange={vi.fn()} />, { wrapper });
    await waitFor(() => {
      expect(screen.getByAltText("验证码")).toBeInTheDocument();
    });
  });

  it("onChange 触发回调", async () => {
    mockCaptcha = {
      captchaType: "normal",
      captchaImg: "data:image/png;base64,xxx",
      captchaId: "cap-2",
    };
    const onChange = vi.fn();
    render(<TextCaptcha value="" onChange={onChange} />, { wrapper });
    await waitFor(() => screen.getByAltText("验证码"));
    const input = screen.getByPlaceholderText("请输入验证码") as HTMLInputElement;
    fireEvent.change(input, { target: { value: "abcd" } });
    expect(onChange).toHaveBeenCalledWith("abcd", "cap-2");
  });

  it("验证码未启用 (空对象) → 静默不渲染图片", async () => {
    mockCaptcha = {};
    render(<TextCaptcha value="" onChange={vi.fn()} />, { wrapper });
    await new Promise((r) => setTimeout(r, 50));
    expect(screen.queryByAltText("验证码")).toBeNull();
  });

  it("slider 类型 → 触发 onError CAPTCHA_TYPE_MISMATCH", async () => {
    mockCaptcha = { captchaType: "slider" };
    const onError = vi.fn();
    render(<TextCaptcha value="" onChange={vi.fn()} onError={onError} />, { wrapper });
    await waitFor(() => {
      expect(onError).toHaveBeenCalledWith("CAPTCHA_TYPE_MISMATCH");
    });
  });

  it("getCaptcha 抛错 → onError 被调用", async () => {
    mockCaptcha = null;
    const { getCaptcha } = await import("@/services/captcha");
    vi.mocked(getCaptcha).mockRejectedValueOnce(new Error("net"));
    const onError = vi.fn();
    render(<TextCaptcha value="" onChange={vi.fn()} onError={onError} />, { wrapper });
    await waitFor(() => {
      expect(onError).toHaveBeenCalled();
    });
  });

  it("点击刷新按钮 → 重新调用 getCaptcha", async () => {
    mockCaptcha = {
      captchaType: "normal",
      captchaImg: "data:image/png;base64,xxx",
      captchaId: "cap-3",
    };
    const { getCaptcha } = await import("@/services/captcha");
    render(<TextCaptcha value="" onChange={vi.fn()} />, { wrapper });
    await waitFor(() => screen.getByAltText("验证码"));
    vi.mocked(getCaptcha).mockClear();
    const btn = document.querySelector(".refresh-btn") as HTMLElement;
    fireEvent.click(btn);
    await waitFor(() => {
      expect(getCaptcha).toHaveBeenCalled();
    });
  });
});
