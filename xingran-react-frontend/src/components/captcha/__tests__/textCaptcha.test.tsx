/**
 * Phase 88 Batch226 — components/captcha/TextCaptcha 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/services/captcha", () => ({
  getCaptcha: vi.fn(async () => ({
    captchaId: "c1",
    captchaType: "normal",
    captchaImg: "data:image/png;base64,xxx",
  })),
}));

import TextCaptcha from "../TextCaptcha";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("captcha/TextCaptcha", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("渲染 input + 初始加载", async () => {
    const onChange = vi.fn();
    render(<TextCaptcha onChange={onChange} />, { wrapper });
    const input = screen.getByPlaceholderText("请输入验证码");
    expect(input).toBeInTheDocument();
    await waitFor(() => {
      expect(onChange).toHaveBeenCalledWith("", "c1");
    });
  });

  it("显示 captchaImg", async () => {
    render(<TextCaptcha onChange={vi.fn()} />, { wrapper });
    await waitFor(() => {
      expect(screen.getByAltText("验证码")).toBeInTheDocument();
    });
  });

  it("input change 触发 onChange", async () => {
    const onChange = vi.fn();
    render(<TextCaptcha onChange={onChange} />, { wrapper });
    await waitFor(() => {
      expect(onChange).toHaveBeenCalledWith("", "c1");
    });
    const input = screen.getByPlaceholderText("请输入验证码");
    fireEvent.change(input, { target: { value: "abcd" } });
    expect(onChange).toHaveBeenCalledWith("abcd", "c1");
  });

  it("reload 按钮 → 重新加载", async () => {
    render(<TextCaptcha onChange={vi.fn()} />, { wrapper });
    await waitFor(() => {
      expect(screen.getByAltText("验证码")).toBeInTheDocument();
    });
    const captcha = await import("@/services/captcha");
    vi.mocked(captcha.getCaptcha).mockClear();
    const reloadBtn = screen.getByRole("button");
    fireEvent.click(reloadBtn);
    await waitFor(() => {
      expect(captcha.getCaptcha).toHaveBeenCalled();
    });
  });

  it("click image 重新加载", async () => {
    render(<TextCaptcha onChange={vi.fn()} />, { wrapper });
    await waitFor(() => {
      expect(screen.getByAltText("验证码")).toBeInTheDocument();
    });
    const captcha = await import("@/services/captcha");
    vi.mocked(captcha.getCaptcha).mockClear();
    fireEvent.click(screen.getByAltText("验证码"));
    await waitFor(() => {
      expect(captcha.getCaptcha).toHaveBeenCalled();
    });
  });

  it("slider 类型 → onError CAPTCHA_TYPE_MISMATCH", async () => {
    const onError = vi.fn();
    const captcha = await import("@/services/captcha");
    vi.mocked(captcha.getCaptcha).mockResolvedValueOnce({
      captchaId: "s1",
      captchaType: "slider",
    } as any);
    render(<TextCaptcha onError={onError} />, { wrapper });
    await waitFor(() => {
      expect(onError).toHaveBeenCalledWith("CAPTCHA_TYPE_MISMATCH");
    });
  });

  it("空 captchaType → 静默返回", async () => {
    const onChange = vi.fn();
    const captcha = await import("@/services/captcha");
    vi.mocked(captcha.getCaptcha).mockResolvedValueOnce({
      captchaId: "",
      captchaType: "",
    } as any);
    render(<TextCaptcha onChange={onChange} />, { wrapper });
    await new Promise((r) => setTimeout(r, 100));
    expect(onChange).not.toHaveBeenCalled();
  });
});
