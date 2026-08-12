import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { App } from "antd";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

const {
  mockLogin,
  mockSubmitLoginPreflight,
  mockGetCaptchaConfig,
  mockFetchMenus,
  mockFetchPermissions,
} = vi.hoisted(() => ({
  mockLogin: vi.fn(),
  mockSubmitLoginPreflight: vi.fn(),
  mockGetCaptchaConfig: vi.fn(),
  mockFetchMenus: vi.fn(),
  mockFetchPermissions: vi.fn(),
}));

vi.mock("@/store/authStore", () => ({
  useAuthStore: () => ({ login: mockLogin }),
}));

vi.mock("@/store/menuStore", () => ({
  useMenuStore: () => ({
    fetchMenus: mockFetchMenus,
    fetchPermissions: mockFetchPermissions,
  }),
}));

vi.mock("@/lib/loginPreflight", () => ({
  submitLoginPreflight: mockSubmitLoginPreflight,
}));

vi.mock("@/services/captcha", () => ({
  getCaptchaConfig: mockGetCaptchaConfig,
}));

vi.mock("@/components/captcha", () => ({
  TextCaptcha: ({ onError }: { onError: (error: string) => void }) => (
    <button type="button" onClick={() => onError("CAPTCHA_TYPE_MISMATCH")}>
      模拟验证码类型不匹配
    </button>
  ),
}));

vi.mock("@/components/captcha/CaptchaModal", () => ({
  default: ({
    visible,
    onError,
  }: {
    visible: boolean;
    onError?: (error: string) => void;
  }) =>
    visible ? (
      <div role="dialog">
        滑动验证码
        <button
          type="button"
          onClick={() => onError?.("CAPTCHA_TYPE_MISMATCH")}
        >
          模拟滑块类型不匹配
        </button>
      </div>
    ) : null,
}));

import Login from "./index";

function renderLogin() {
  return render(
    <MemoryRouter>
      <App>
        <Login />
      </App>
    </MemoryRouter>
  );
}

function submitCredentials() {
  fireEvent.change(screen.getByPlaceholderText("用户名"), {
    target: { value: "admin" },
  });
  fireEvent.change(screen.getByPlaceholderText("密码"), {
    target: { value: "password" },
  });
  fireEvent.click(screen.getByRole("button", { name: /登\s*录/ }));
}

describe("登录页安全配置预检", () => {
  beforeEach(() => {
    mockLogin.mockReset();
    mockSubmitLoginPreflight.mockReset();
    mockGetCaptchaConfig.mockReset();
    mockFetchMenus.mockReset();
    mockFetchPermissions.mockReset();

    mockGetCaptchaConfig.mockResolvedValue({ enabled: "disabled" });
  });

  it("预检失败时阻止登录并显示可操作提示", async () => {
    mockSubmitLoginPreflight.mockResolvedValue({
      ok: false,
      friendlyMessage: "登录安全配置已过期，自动更新失败，请检查网络后重试",
    });
    renderLogin();

    submitCredentials();

    expect(
      await screen.findByText(
        "登录安全配置已过期，自动更新失败，请检查网络后重试"
      )
    ).toBeInTheDocument();
    expect(mockLogin).not.toHaveBeenCalled();
  });
  it("使用预检返回的最新 slider 配置弹出滑动验证码", async () => {
    mockSubmitLoginPreflight.mockResolvedValue({
      ok: true,
      captchaEnabled: "slider",
    });
    renderLogin();

    submitCredentials();

    await waitFor(() => {
      expect(screen.getByRole("dialog")).toHaveTextContent("滑动验证码");
    });
    expect(mockLogin).not.toHaveBeenCalled();
  });

  it("快速重复提交时只启动一次预检", async () => {
    let resolvePreflight!: (value: {
      ok: true;
      captchaEnabled: "disabled";
    }) => void;
    mockSubmitLoginPreflight.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolvePreflight = resolve;
        })
    );
    renderLogin();

    fireEvent.change(screen.getByPlaceholderText("用户名"), {
      target: { value: "admin" },
    });
    fireEvent.change(screen.getByPlaceholderText("密码"), {
      target: { value: "password" },
    });
    const submitButton = screen.getByRole("button", { name: /登\s*录/ });
    fireEvent.click(submitButton);
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(mockSubmitLoginPreflight).toHaveBeenCalledTimes(1);
    });

    resolvePreflight({ ok: true, captchaEnabled: "disabled" });
    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledTimes(1);
    });
  });

  it("滑块验证码类型变化时通知父页面局部刷新", async () => {
    mockSubmitLoginPreflight
      .mockResolvedValueOnce({ ok: true, captchaEnabled: "slider" })
      .mockResolvedValueOnce({ ok: true, captchaEnabled: "normal" });
    renderLogin();
    submitCredentials();

    fireEvent.click(
      await screen.findByRole("button", { name: "模拟滑块类型不匹配" })
    );

    expect(
      await screen.findByText("验证码配置已更新，请重新验证")
    ).toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(mockSubmitLoginPreflight).toHaveBeenCalledTimes(2);
  });

  it("验证码类型变化时局部刷新配置而不是整页刷新", async () => {
    mockGetCaptchaConfig.mockResolvedValue({ enabled: "normal" });
    mockSubmitLoginPreflight.mockResolvedValue({
      ok: true,
      captchaEnabled: "slider",
    });
    renderLogin();

    fireEvent.click(
      await screen.findByRole("button", { name: "模拟验证码类型不匹配" })
    );

    expect(
      await screen.findByText("验证码配置已更新，请重新验证")
    ).toBeInTheDocument();
    expect(mockSubmitLoginPreflight).toHaveBeenCalledTimes(1);
  });
});

