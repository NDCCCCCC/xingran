/**
 * v1.52 Coverage 补齐（task-24 后续 / PR #50 GOV-05）—— pages/profile 行为测试
 *
 * 覆盖（真实行为断言，非凑数）：
 * - 成功加载：基本信息 Descriptions、值班摘要统计、今日值班任务 Tag
 * - 加载失败降级：getProfileInfo reject → 停留在加载态且不崩
 * - 值班统计失败降级：getMyDutyStats reject → 页面正常渲染
 * - 编辑提交路径：Modal 预填 → 确定 → updateProfileInfo 调用 + authStore.updateUser 同步
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";

// ---- mocks ----

const mockUpdateUser = vi.fn();
const mockLogout = vi.fn();
const mockAuthState = {
  updateUser: mockUpdateUser,
  logout: mockLogout,
};
vi.mock("@/store/authStore", () => ({
  useAuthStore: Object.assign((sel: (s: unknown) => unknown) => sel(mockAuthState), {
    getState: () => mockAuthState,
  }),
}));

const mockGetProfileInfo = vi.fn();
const mockUpdateProfileInfo = vi.fn();
const mockChangePassword = vi.fn();
const mockUploadAvatar = vi.fn();
vi.mock("@/lib/profileApi", () => ({
  getProfileInfo: (...args: unknown[]) => mockGetProfileInfo(...args),
  updateProfileInfo: (...args: unknown[]) => mockUpdateProfileInfo(...args),
  changePassword: (...args: unknown[]) => mockChangePassword(...args),
  uploadAvatar: (...args: unknown[]) => mockUploadAvatar(...args),
}));

const mockGetMyDutyStats = vi.fn();
vi.mock("@/lib/dutyApi", () => ({
  getMyDutyStats: (...args: unknown[]) => mockGetMyDutyStats(...args),
}));

import ProfilePage from "../index";

// ---- fixtures ----

const PROFILE = {
  id: "u1",
  username: "admin",
  nickname: "管理员",
  gender: 1,
  email: "admin@example.com",
  phone: "13800000000",
  deptName: "运维部",
  status: 0,
  loginIp: "10.0.0.1",
  loginTime: "2026-11-02 08:00:00",
  pwdUpdateTime: "2026-10-01 08:00:00",
  remark: "测试备注",
  avatar: "",
};

const DUTY_STATS = {
  isOnDutyToday: true,
  todayDutyRecords: [
    { scheduleId: "s1", poolId: "pool-1", poolName: "主机池", userId: "u1", dutyType: "weekday" },
  ],
  thisMonthCount: 3,
  totalCount: 42,
  nextDutyDate: "2026-11-05 09:00:00",
  nextDutyPoolName: "主机池",
};

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/profile"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

function renderPage() {
  return render(<ProfilePage />, { wrapper });
}

/** 点击 Modal 底部主按钮（确定/OK，兼容中英 locale） */
async function clickModalOk(): Promise<void> {
  const btn = await waitFor(() => {
    const ok =
      document.querySelector(".ant-modal .ant-btn-primary") ??
      screen.getByRole("button", { name: /OK|确\s*定/ });
    expect(ok).toBeTruthy();
    return ok as HTMLElement;
  });
  await act(async () => {
    fireEvent.click(btn);
  });
}

describe("pages/profile — 行为测试（v1.52 Coverage 补齐）", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetProfileInfo.mockResolvedValue(PROFILE);
    mockGetMyDutyStats.mockResolvedValue({ code: 0, data: DUTY_STATS });
    mockUpdateProfileInfo.mockResolvedValue({ message: "ok" });
  });

  it("成功加载：渲染基本信息、值班统计与今日值班任务", async () => {
    renderPage();

    // profile 加载完成 → Descriptions 渲染
    expect(await screen.findByText("admin")).toBeInTheDocument();
    expect(screen.getByText("管理员")).toBeInTheDocument();
    expect(screen.getByText("admin@example.com")).toBeInTheDocument();
    expect(screen.getByText("运维部")).toBeInTheDocument();

    // 值班摘要
    expect(await screen.findByText("值班中")).toBeInTheDocument();
    expect(screen.getByText("本月次数")).toBeInTheDocument();
    expect(screen.getByText("主机池")).toBeInTheDocument(); // 今日任务 Tag + 下次值班池名
    expect(screen.getByText("今日值班任务")).toBeInTheDocument();
  });

  it("getProfileInfo 失败 → 停留加载态不崩（错误分支）", async () => {
    mockGetProfileInfo.mockRejectedValueOnce(new Error("500"));
    renderPage();

    // profile 永不设置 → 加载态保持（catch 分支执行）
    expect(await screen.findByText("加载中...")).toBeInTheDocument();
    await waitFor(() => {
      expect(mockGetMyDutyStats).toHaveBeenCalled(); // Promise.all 中另一请求照常发出
    });
  });

  it("getMyDutyStats 失败 → 页面正常渲染基本信息（错误分支）", async () => {
    mockGetMyDutyStats.mockRejectedValueOnce(new Error("500"));
    renderPage();

    expect(await screen.findByText("admin")).toBeInTheDocument();
    // 值班摘要区块不渲染统计（dutyStats 为 null）
    expect(screen.queryByText("本月次数")).not.toBeInTheDocument();
  });

  it("编辑提交：Modal 预填 profile → 确定 → updateProfileInfo + updateUser 同步", async () => {
    renderPage();
    expect(await screen.findByText("admin")).toBeInTheDocument();

    // 打开编辑弹窗
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: /编\s*辑|Edit/ }));
    });
    expect(await screen.findByText("编辑个人信息")).toBeInTheDocument();

    // 表单已预填（profile 值）——修改昵称后提交
    const nicknameInput = await waitFor(() => {
      const el = document.querySelector<HTMLInputElement>("#nickname");
      expect(el).not.toBeNull();
      return el!;
    });
    await act(async () => {
      fireEvent.change(nicknameInput, { target: { value: "新昵称" } });
    });

    await clickModalOk();

    await waitFor(() => {
      expect(mockUpdateProfileInfo).toHaveBeenCalledTimes(1);
    });
    const payload = mockUpdateProfileInfo.mock.calls[0][0] as { nickname?: string };
    expect(payload.nickname).toBe("新昵称");
    // 全局用户信息同步更新
    expect(mockUpdateUser).toHaveBeenCalled();
    // 成功后弹窗关闭（重新拉取 profile）
    await waitFor(() => {
      expect(mockGetProfileInfo).toHaveBeenCalledTimes(2);
    });
  });
});
