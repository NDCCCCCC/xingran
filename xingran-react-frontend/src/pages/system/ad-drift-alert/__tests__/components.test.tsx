/**
 * v1.55 CI diff-coverage 补测（GOV-05）：同步漂移告警页 3 个子组件真实渲染。
 *
 * v1.54 的 index.render.test.tsx 把 3 个子组件占位 mock 掉，导致 PR diff 门禁里
 * ConfigForm / StatusCards / TriggerLogTable 全部 0% 覆盖。本文件不 mock 组件本身，
 * 覆盖各自主要分支：
 *   - StatusCards：有数据/无数据/loading/通道 Tag/lastAlertAt 与 drifted 分支
 *   - ConfigForm：webhook 下拉加载成功/失败告警/config 回填/保存成功与失败/
 *     权限三态（visible/disabled）/基线只读
 *   - TriggerLogTable：columns 各列 render（时间/来源/漂移/节流/计数/结果）、
 *     request 入参构造（current/pageSize/status/startTime/endTime）、空态与刷新
 *
 * 约定：mock 包装 API（禁止裸 axios），handleApiError 唯一错误出口，就近 __tests__。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { renderWithProviders } from "@/test/utils/renderWithProviders";

const h = vi.hoisted(() => {
  const listTriggerLogs = vi.fn();
  const updateConfig = vi.fn();
  const getAPINotificationConfigList = vi.fn();
  const handleApiError = vi.fn();
  const handleSuccess = vi.fn();
  const decide = vi.fn(() => ({ visible: true, disabled: false, hint: undefined }));
  // PageTable mock 透传：把 columns / request / emptyAction 暴露给测试断言
  const pageTableProps = {
    columns: [] as Array<{ title?: string; render?: (...args: never[]) => ReactNode }>,
    request: null as null | ((p: Record<string, unknown>) => Promise<unknown>),
    emptyAction: null as ReactNode,
    extraFilters: null as Record<string, unknown> | null,
  };
  return {
    listTriggerLogs,
    updateConfig,
    getAPINotificationConfigList,
    handleApiError,
    handleSuccess,
    decide,
    pageTableProps,
  };
});

vi.mock("@/lib/adDriftAlertApi", () => ({
  listTriggerLogs: h.listTriggerLogs,
  updateConfig: h.updateConfig,
}));

vi.mock("@/lib/notificationConfigApi", () => ({
  getAPINotificationConfigList: h.getAPINotificationConfigList,
}));

vi.mock("@/utils/errorHandler", () => ({
  handleApiError: h.handleApiError,
  handleSuccess: h.handleSuccess,
}));

vi.mock("@/components/shared/useActionPermission", () => ({
  useActionPermission: () => ({
    state: "ready",
    allowed: () => true,
    decide: h.decide,
  }),
}));

// PageTable 用轻量透传 mock：保留 columns/render 逻辑真实执行，隔离 jsdom 分页复杂度
vi.mock("@/components/shared/PageTable", () => ({
  default: (props: {
    columns: Array<{ title?: string; render?: (...args: never[]) => ReactNode }>;
    request: (p: Record<string, unknown>) => Promise<unknown>;
    emptyAction: ReactNode;
    extraFilters: Record<string, unknown>;
  }): ReactElement => {
    h.pageTableProps.columns = props.columns;
    h.pageTableProps.request = props.request;
    h.pageTableProps.emptyAction = props.emptyAction;
    h.pageTableProps.extraFilters = props.extraFilters;
    return <div data-testid="page-table">触发记录表</div>;
  },
}));

vi.mock("@/components/shared/EmptyStateWithAction", () => ({
  default: ({ title }: { title?: string }): ReactElement => <div>{title ?? "空态"}</div>,
}));

import StatusCards from "../components/StatusCards";
import ConfigForm from "../components/ConfigForm";
import TriggerLogTable from "../components/TriggerLogTable";

const statusFixture = {
  enabled: true,
  tolerance: 2,
  throttleMinutes: 60,
  noticeEnabled: true,
  wsEnabled: false,
  webhookEnabled: true,
  baseline: "20,4",
  lastAlertAt: "2026-11-09T10:00:00Z",
  lastAlertDrifted: true,
  alertCount: 5,
};

const configFixture = {
  enabled: true,
  tolerance: 1,
  throttleMinutes: 30,
  noticeEnabled: true,
  wsEnabled: true,
  webhookEnabled: false,
  webhookConfigId: "cfg-1",
  webhookTemplate: "tpl",
  baseline: "20,4",
};

describe("AD 漂移告警子组件（v1.55 diff-coverage 补测）", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    h.decide.mockReturnValue({ visible: true, disabled: false, hint: undefined });
    h.getAPINotificationConfigList.mockResolvedValue({
      data: { list: [{ id: "cfg-1", configName: "企微机器人" }] },
    });
    h.updateConfig.mockImplementation(async (p: unknown) => p);
    h.listTriggerLogs.mockResolvedValue({ list: [], total: 0, current: 1, pageSize: 10 });
  });

  // ---------- StatusCards ----------
  it("StatusCards：有数据渲染统计/基线/节流/通道 Tag/上次告警漂移", () => {
    renderWithProviders(<StatusCards status={statusFixture} />);
    expect(screen.getByText("累计告警次数")).toBeTruthy();
    expect(screen.getByText("5")).toBeTruthy();
    expect(screen.getByText("当前基线（成员数,组数）")).toBeTruthy();
    expect(screen.getByText("20,4")).toBeTruthy();
    expect(screen.getByText("容差 / 节流")).toBeTruthy();
    expect(screen.getByText("2 / 60min")).toBeTruthy();
    expect(screen.getByText(/（检测到漂移）/)).toBeTruthy();
    expect(screen.getByText("告警通道")).toBeTruthy();
    expect(screen.getByText("总开关")).toBeTruthy();
    expect(screen.getByText("站内通知")).toBeTruthy();
    expect(screen.getByText("WS 实时")).toBeTruthy();
    expect(screen.getByText("Webhook")).toBeTruthy();
  });

  it("StatusCards：loading 走骨架，不显示漂移标记与统计数值", () => {
    const { container } = renderWithProviders(<StatusCards status={null} loading />);
    // antd Card loading 时以骨架替换内容，故统计数值不渲染
    expect(container.querySelectorAll(".ant-skeleton").length).toBeGreaterThan(0);
    expect(screen.queryByText(/（检测到漂移）/)).toBeNull();
    expect(screen.queryByText("累计告警次数")).toBeNull();
  });

  it("StatusCards：status 为 null 且非 loading 时显示默认零值与 '-'", () => {
    const { container } = renderWithProviders(<StatusCards status={null} />);
    expect(screen.getByText("-")).toBeTruthy();
    expect(screen.getByText("0 / 0min")).toBeTruthy();
    expect(container.querySelectorAll(".ant-skeleton").length).toBe(0);
  });

  // ---------- ConfigForm ----------
  it("ConfigForm：加载 webhook 下拉 + config 回填 + 基线只读 + 表单字段", async () => {
    renderWithProviders(<ConfigForm config={configFixture} />);
    await waitFor(() =>
      expect(h.getAPINotificationConfigList).toHaveBeenCalledWith({ configType: "webhook" })
    );
    expect(screen.getByText("告警配置")).toBeTruthy();
    expect(screen.getByText("启用告警（总开关）")).toBeTruthy();
    expect(screen.getByText("容差（计数变化阈值）")).toBeTruthy();
    expect(screen.getByText("节流窗口（分钟）")).toBeTruthy();
    expect(screen.getByText("站内通知（铃铛）")).toBeTruthy();
    expect(screen.getByText("WS 实时推送")).toBeTruthy();
    expect(screen.getByText("Webhook 推送")).toBeTruthy();
    expect(screen.getByText("Webhook 模板（可选）")).toBeTruthy();
    expect(screen.getByText("当前基线（只读）")).toBeTruthy();
    expect(screen.getByLabelText("当前基线")).toHaveProperty("value", "20,4");
    expect(screen.getByText("保存配置")).toBeTruthy();
  });

  it("ConfigForm：webhook 下拉加载失败显示告警并走 handleApiError", async () => {
    const err = new Error("list failed");
    h.getAPINotificationConfigList.mockRejectedValue(err);
    renderWithProviders(<ConfigForm config={configFixture} />);
    expect(await screen.findByText(/Webhook 配置列表加载失败/)).toBeTruthy();
    await waitFor(() =>
      expect(h.handleApiError).toHaveBeenCalledWith(err, "加载 Webhook 配置", false)
    );
  });

  it("ConfigForm：保存成功走 updateConfig + handleSuccess + onSaved", async () => {
    const onSaved = vi.fn();
    renderWithProviders(<ConfigForm config={configFixture} onSaved={onSaved} />);
    await waitFor(() => expect(h.getAPINotificationConfigList).toHaveBeenCalled());
    fireEvent.click(screen.getByText("保存配置"));
    await waitFor(() => expect(h.updateConfig).toHaveBeenCalledTimes(1));
    const payload = h.updateConfig.mock.calls[0][0];
    expect(payload.baseline).toBe("20,4");
    expect(payload.webhookConfigId).toBe("cfg-1");
    await waitFor(() => expect(h.handleSuccess).toHaveBeenCalledWith("保存配置"));
    expect(onSaved).toHaveBeenCalled();
  });

  it("ConfigForm：保存失败走 handleApiError，不误报成功", async () => {
    const err = new Error("save failed");
    h.updateConfig.mockRejectedValue(err);
    renderWithProviders(<ConfigForm config={configFixture} />);
    await waitFor(() => expect(h.getAPINotificationConfigList).toHaveBeenCalled());
    fireEvent.click(screen.getByText("保存配置"));
    await waitFor(() => expect(h.handleApiError).toHaveBeenCalledWith(err, "保存配置"));
    expect(h.handleSuccess).not.toHaveBeenCalled();
  });

  it("ConfigForm：无 :edit 权限时不渲染保存按钮（三态门控）", async () => {
    h.decide.mockReturnValue({ visible: false, disabled: true, hint: "无权限" });
    renderWithProviders(<ConfigForm config={configFixture} />);
    await waitFor(() => expect(h.getAPINotificationConfigList).toHaveBeenCalled());
    expect(screen.queryByText("保存配置")).toBeNull();
  });

  // ---------- TriggerLogTable ----------
  it("TriggerLogTable：request 构造业务入参（含 status/startTime/endTime）", async () => {
    renderWithProviders(<TriggerLogTable refreshToken={2} />);
    expect(screen.getByTestId("page-table")).toBeTruthy();
    expect(h.pageTableProps.extraFilters).toEqual({ _reload: 2 });
    const req = h.pageTableProps.request!;
    await req({
      current: 2,
      pageSize: 20,
      status: "drifted",
      startTime: "2026-11-01",
      endTime: "2026-11-09",
      _reload: 2,
    });
    expect(h.listTriggerLogs).toHaveBeenCalledWith({
      current: 2,
      pageSize: 20,
      status: "drifted",
      startTime: "2026-11-01",
      endTime: "2026-11-09",
    });
  });

  it("TriggerLogTable：request 缺省值与非合法 status 过滤", async () => {
    renderWithProviders(<TriggerLogTable refreshToken={0} />);
    const req = h.pageTableProps.request!;
    await req({ status: "bogus", _reload: 0 });
    expect(h.listTriggerLogs).toHaveBeenCalledWith({ current: 1, pageSize: 10 });
  });

  it("TriggerLogTable：columns 各列 render 分支（时间/来源/漂移/节流/计数/结果）", () => {
    renderWithProviders(<TriggerLogTable refreshToken={0} />);
    const cols = h.pageTableProps.columns;
    expect(cols.length).toBe(7);
    const byTitle = (t: string) => cols.find((c) => c.title === t)!;
    const render = (t: string, ...args: never[]) => byTitle(t).render!(...args) as ReactNode;

    // formatLogTime：空值 / 合法 / 非法（本地时区由 dayjs 解析，断言日期部分即可）
    expect(render("触发时间", undefined as never)).toBe("-");
    expect(render("触发时间", "2026-11-09T10:00:00Z" as never)).toContain("2026-11-09");
    expect(render("触发时间", "not-a-date" as never)).toBe("not-a-date");
    // 来源
    expect(render("来源", "manual" as never)).toBe("手动");
    expect(render("来源", "cron" as never)).toBe("定时");
    expect(render("来源", undefined as never)).toBe("-");
    // 检测结果：节流优先 / 漂移 / 无漂移
    const throttled = render("检测结果", true as never, { throttled: true } as never);
    expect(throttled).toBeTruthy();
    expect(render("检测结果", true as never, { throttled: false } as never)).toBeTruthy();
    expect(render("检测结果", false as never, { throttled: false } as never)).toBeTruthy();
    // 计数 / 漂移量 / 操作人 / 结果 的缺省兜底
    expect(
      render("成员数 / 组数", undefined as never, { memberCount: 3, distinctGroups: 2 } as never)
    ).toBe("3 / 2");
    expect(render("漂移量", undefined as never)).toBe("-");
    expect(render("操作人", undefined as never)).toBe("-");
    expect(render("结果", undefined as never)).toBe("-");
  });

  it("TriggerLogTable：空态文案渲染", () => {
    renderWithProviders(<TriggerLogTable refreshToken={0} />);
    expect(h.pageTableProps.emptyAction).toBeTruthy();
    // PageTable mock 把 emptyAction 存入 holder；单独渲染它以验证文案契约
    const { getByText } = renderWithProviders(<>{h.pageTableProps.emptyAction}</>);
    expect(getByText("暂无触发记录")).toBeTruthy();
  });
});
