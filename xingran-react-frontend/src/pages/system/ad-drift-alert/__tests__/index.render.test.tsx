/**
 * Phase 144 ALRT-03 — 同步漂移告警页 index.tsx 渲染/编排测试
 *
 * 覆盖：
 *   - 空数据渲染不抛错、标题/三块子组件/立即检测按钮存在；
 *   - mount 调 getStatus + getConfig（wrapped API 编排）；
 *   - onADDriftAlert 订阅在 mount 时调用、卸载时退订（PLAN Task 4 口径）；
 *   - 收到 ad_sync_drift_alert 时刷新状态 + 日志（refreshToken +1）+ 全局提示；
 *   - 「立即检测」按钮（:trigger 门控）点击走 triggerOnce 并刷新；
 *   - 权限三态门控：无 :trigger 权限时按钮不渲染（useActionPermission mock）。
 *
 * 子组件用轻量占位 mock 隔离 PageTable / Form 的 jsdom 复杂度，专注验证
 * index.tsx 的编排；TriggerLogTable 自身逻辑由 refreshToken prop 契约保证。
 *
 * ⚠️ vi.mock 工厂会被 hoist 到文件顶部，不能引用普通顶层 const；所有 spy /
 * holder 一律经 vi.hoisted() 暴露后再在工厂中引用（vitest 官方约定）。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor, act } from "@testing-library/react";
import { renderWithProviders } from "@/test/utils/renderWithProviders";

const h = vi.hoisted(() => {
  const getStatusMock = vi.fn(() =>
    Promise.resolve({
      enabled: true,
      tolerance: 0,
      throttleMinutes: 60,
      noticeEnabled: true,
      wsEnabled: true,
      webhookEnabled: false,
      baseline: "20,4",
      lastAlertAt: null,
      lastAlertDrifted: false,
      alertCount: 3,
    })
  );
  const getConfigMock = vi.fn(() =>
    Promise.resolve({
      enabled: true,
      tolerance: 0,
      throttleMinutes: 60,
      noticeEnabled: true,
      wsEnabled: true,
      webhookEnabled: false,
      webhookConfigId: "",
      webhookTemplate: "",
      baseline: "20,4",
    })
  );
  const triggerOnceMock = vi.fn(() => Promise.resolve({ drifted: false, alerted: false }));
  const listTriggerLogsMock = vi.fn(() =>
    Promise.resolve({ list: [], total: 0, current: 1, pageSize: 10 })
  );
  const updateConfigMock = vi.fn((p: unknown) => Promise.resolve(p));

  // WS 订阅回调 holder（可变），工厂内赋值，测试体内读取
  const drift = { cb: null as ((e: Record<string, unknown>) => void) | null };
  const unsubscribeMock = vi.fn();
  const onADDriftAlertMock = vi.fn((cb: (e: Record<string, unknown>) => void) => {
    drift.cb = cb;
    return unsubscribeMock;
  });

  const warningMock = vi.fn();
  const infoMock = vi.fn();
  const handleApiErrorMock = vi.fn();
  const handleSuccessMock = vi.fn();
  const decideMock = vi.fn(() => ({ visible: true, disabled: false }));

  return {
    getStatusMock,
    getConfigMock,
    triggerOnceMock,
    listTriggerLogsMock,
    updateConfigMock,
    drift,
    unsubscribeMock,
    onADDriftAlertMock,
    warningMock,
    infoMock,
    handleApiErrorMock,
    handleSuccessMock,
    decideMock,
  };
});

// ---- API 编排 mock ----
vi.mock("@/lib/adDriftAlertApi", () => ({
  getStatus: h.getStatusMock,
  getConfig: h.getConfigMock,
  triggerOnce: h.triggerOnceMock,
  listTriggerLogs: h.listTriggerLogsMock,
  updateConfig: h.updateConfigMock,
}));

// ---- WS 订阅（noticeStore）mock：记录订阅/退订与被注册的回调 ----
vi.mock("@/store/noticeStore", () => ({
  onADDriftAlert: h.onADDriftAlertMock,
}));

// ---- 提示 / 错误处理 mock（避免真实 message 实例与 console.error）----
vi.mock("@/utils/antdMessage", () => ({
  getAppMessage: () => ({
    warning: h.warningMock,
    info: h.infoMock,
    error: vi.fn(),
    success: vi.fn(),
  }),
}));
vi.mock("@/utils/errorHandler", () => ({
  handleApiError: h.handleApiErrorMock,
  handleSuccess: h.handleSuccessMock,
}));

// ---- 权限三态门控 mock：默认 ready 且含所有权限 ----
vi.mock("@/components/shared/useActionPermission", () => ({
  useActionPermission: () => ({
    state: "ready",
    allowed: () => true,
    decide: h.decideMock,
  }),
}));

// ---- 子组件占位 mock（工厂内 JSX 由 .tsx 的 jsx-runtime 提供）----
vi.mock("../components/StatusCards", () => ({
  default: () => <div data-testid="status-cards">状态卡</div>,
}));
vi.mock("../components/TriggerLogTable", () => ({
  default: ({ refreshToken }: { refreshToken: number }) => (
    <div data-testid="trigger-log-table">日志表 refreshToken={refreshToken}</div>
  ),
}));
vi.mock("../components/ConfigForm", () => ({
  default: () => <div data-testid="config-form">配置表单</div>,
}));

import ADDriftAlertPage from "../index";

describe("同步漂移告警页 index.tsx 渲染/编排（Phase 144 ALRT-03）", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    h.drift.cb = null;
    h.decideMock.mockReturnValue({ visible: true, disabled: false });
  });

  it("空数据渲染不抛错：标题 + 三块子组件 + 立即检测按钮", async () => {
    renderWithProviders(<ADDriftAlertPage />);
    expect(screen.getByText("同步漂移告警")).toBeTruthy();
    expect(screen.getByTestId("status-cards")).toBeTruthy();
    expect(screen.getByTestId("trigger-log-table")).toBeTruthy();
    expect(screen.getByTestId("config-form")).toBeTruthy();
    expect(screen.getByRole("button", { name: "立即检测" })).toBeTruthy();
    await waitFor(() => expect(h.getStatusMock).toHaveBeenCalled());
  });

  it("mount 调 getStatus + getConfig，并订阅 onADDriftAlert", async () => {
    renderWithProviders(<ADDriftAlertPage />);
    await waitFor(() => {
      expect(h.getStatusMock).toHaveBeenCalledTimes(1);
      expect(h.getConfigMock).toHaveBeenCalledTimes(1);
      expect(h.onADDriftAlertMock).toHaveBeenCalledTimes(1);
    });
  });

  it("卸载后调用退订函数（cleanup 退订）", async () => {
    const { unmount } = renderWithProviders(<ADDriftAlertPage />);
    expect(h.onADDriftAlertMock).toHaveBeenCalledTimes(1);
    // flush 首屏异步 setState 后再卸载，避免 act() 警告与跨用例微任务泄漏
    await act(async () => {});
    unmount();
    expect(h.unsubscribeMock).toHaveBeenCalledTimes(1);
  });

  it("收到 ad_sync_drift_alert：刷新状态 + 日志 refreshToken +1 + 全局 warning", async () => {
    renderWithProviders(<ADDriftAlertPage />);
    await waitFor(() => expect(h.onADDriftAlertMock).toHaveBeenCalled());
    expect(h.drift.cb).toBeTypeOf("function");

    const before = h.getStatusMock.mock.calls.length;
    // act 包裹：回调内 refreshStatus(async) + bumpLogs(setState) 会触发 React 状态更新
    await act(async () => {
      h.drift.cb?.({
        type: "ad_sync_drift_alert",
        memberCount: 18,
        distinctGroups: 3,
        title: "AD 同步漂移告警",
      });
    });

    await waitFor(() => {
      expect(h.warningMock).toHaveBeenCalled();
      expect(h.getStatusMock.mock.calls.length).toBeGreaterThan(before);
      expect(screen.getByTestId("trigger-log-table").textContent).toContain("refreshToken=1");
    });
  });

  it("「立即检测」点击走 triggerOnce 并刷新状态与日志", async () => {
    renderWithProviders(<ADDriftAlertPage />);
    await waitFor(() => expect(h.getStatusMock).toHaveBeenCalled());
    const before = h.getStatusMock.mock.calls.length;

    fireEvent.click(screen.getByRole("button", { name: "立即检测" }));
    await waitFor(() => {
      expect(h.triggerOnceMock).toHaveBeenCalledTimes(1);
      expect(h.getStatusMock.mock.calls.length).toBeGreaterThan(before);
      expect(screen.getByTestId("trigger-log-table").textContent).toContain("refreshToken=1");
      expect(h.infoMock).toHaveBeenCalled();
    });
  });

  it(":trigger 无权限时「立即检测」按钮不渲染（三态门控 ready 无权限 → 不渲染）", async () => {
    h.decideMock.mockReturnValue({ visible: false, disabled: false });
    renderWithProviders(<ADDriftAlertPage />);
    // flush 首屏 getStatus/getConfig 的异步 setState，避免 act() 警告
    await act(async () => {});
    expect(screen.queryByRole("button", { name: "立即检测" })).toBeNull();
  });
});
