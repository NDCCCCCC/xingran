/**
 * duty/management 页面行为测试 —— Tab 切换、节假日操作、值班配置、Modal 开关
 *
 * 背景：PR #84 把 duty/holidays、duty/config 两页合并进 duty/management 后，
 * 页面层新增的 Tab 结构与回调（本文件覆盖）缺少测试，导致 pages/duty 语句
 * 覆盖率跌破 GOV-05 门槛。本文件用真实交互 + 断言（而非「渲染不抛错」）
 * 覆盖这些回调与分支。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, act, within } from "@testing-library/react";
import { App as AntdApp, ConfigProvider } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import dayjs from "dayjs";
import type { ReactElement, ReactNode } from "react";

// 页面层的 4 个 hook 全部替换为确定性桩，隔离后端请求，专注验证页面自身交互逻辑。
const scheduleData = {
  currentWeekStart: dayjs("2026-08-24"),
  weeklyDutyData: {},
  fetchWeeklyDuty: vi.fn(async () => undefined),
  prevWeek: vi.fn(),
  todayWeek: vi.fn(),
  nextWeek: vi.fn(),
};
const holidayData = {
  holidays: [
    {
      id: "h1",
      holidayName: "元旦",
      holidayDate: "2026-01-01",
      holidayType: "legal",
      isOffday: true,
      year: 2026,
      remark: "法定节假日",
      createdAt: "2026-01-01",
      createdBy: "admin",
    },
  ],
  loading: false,
  holidayYear: 2026,
  availableYears: [2026, 2025],
  fetchList: vi.fn(async () => undefined),
  fetchYears: vi.fn(async () => undefined),
  setHolidayYear: vi.fn(),
  create: vi.fn(async () => true),
  update: vi.fn(async () => true),
  deleteOne: vi.fn(async () => true),
  batchCreate: vi.fn(async () => true),
};
const dutyConfig = {
  config: {
    id: "cfg1",
    reminderEnabled: true,
    reminderTime: "09:00",
    reminderChannels: "websocket",
    beforeReminderMinutes: 30,
    createdAt: "2026-01-01",
  },
  loading: false,
  saving: false,
  fetch: vi.fn(async () => undefined),
  save: vi.fn(async () => true),
};

vi.mock("../hooks", () => ({
  useScheduleData: () => scheduleData,
  useHolidayData: () => holidayData,
  useDutyConfig: () => dutyConfig,
}));

// useUserOptions / useDict 走 react-query，用空数据桩避免真实请求。
vi.mock("@/hooks/useUserOptions", () => ({ useUserOptions: () => ({ data: [] }) }));
vi.mock("@/hooks/useDict", () => ({ useDict: () => ({ data: [] }) }));

// getDutyPoolList 在页面 mount 时被调用；createHolidayTemplate / handleHolidayImport
// 是被 HolidayManagement 按钮引用的工具函数，桩掉以免触发 xlsx 动态 import。
vi.mock("@/lib/dutyApi", () => ({
  getDutyPoolList: vi.fn(async () => ({ data: { list: [] } })),
}));
vi.mock("../utils", () => ({
  downloadHolidayTemplate: vi.fn(),
  handleHolidayImport: vi.fn(),
}));
vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

// DutyPoolPage 本身是独立大页面（已有自己的测试），此处只需验证 Tab 能挂载它。
vi.mock("../../pools", () => ({
  default: () => <div data-testid="duty-pool-page">值班池页面桩</div>,
}));

import ManagementPage from "../index";
import { downloadHolidayTemplate } from "../utils";

function Wrap({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/duty/management"]}>
      <ConfigProvider>
        <QueryClientProvider client={qc}>
          <AntdApp>{children}</AntdApp>
        </QueryClientProvider>
      </ConfigProvider>
    </MemoryRouter>
  );
}

/** 点击 antd Tabs 的某个 tab 标题（role=tab），等待对应面板内容出现 */
async function switchTab(label: string, expectText: string): Promise<void> {
  const tab = await screen.findByRole("tab", { name: new RegExp(label) });
  await act(async () => {
    fireEvent.click(tab);
  });
  await waitFor(() => {
    expect(screen.getByText(expectText)).toBeInTheDocument();
  });
}

/** 当前可见的 Modal 容器（antd 关闭态 wrap 带 hidden 类） */
function activeModal(): HTMLElement {
  const modal = document.querySelector(
    ".ant-modal-wrap:not(.ant-modal-wrap-hidden)"
  ) as HTMLElement | null;
  expect(modal).toBeTruthy();
  return modal as HTMLElement;
}

/**
 * 按按钮文本定位并点击。
 * antd Button 对两个汉字的文案会自动插入空格（「今天」→「今 天」、
 * 「新增」→「新 增」），且前置 icon 会拼进 accessible name，所以不能用
 * getByRole(name) 或精确 getByText；统一按去空白后的 textContent 匹配。
 */
function buttonByText(text: string): HTMLElement {
  const btn = Array.from(document.querySelectorAll("button")).find(
    (el) => (el.textContent || "").replace(/\s/g, "") === text
  ) as HTMLElement | undefined;
  expect(btn, `找不到按钮「${text}」`).toBeTruthy();
  return btn as HTMLElement;
}

/** 取 Modal footer 主按钮（确定/OK），兼容中英 locale */
function modalOkButton(): HTMLElement {
  const btn = activeModal().querySelector(
    ".ant-modal-footer .ant-btn-primary"
  ) as HTMLElement | null;
  expect(btn).toBeTruthy();
  return btn as HTMLElement;
}

/** 取 Modal 内取消按钮（取 消 / Cancel），兼容中英 locale */
function modalCancelButton(): HTMLElement {
  const btn = Array.from(activeModal().querySelectorAll(".ant-modal-footer button")).find((el) =>
    /取\s*消|Cancel/i.test(el.textContent || "")
  ) as HTMLElement | undefined;
  expect(btn).toBeTruthy();
  return btn as HTMLElement;
}

beforeEach(() => {
  vi.clearAllMocks();
  window.sessionStorage.clear();
});

afterEach(() => {
  window.sessionStorage.clear();
});

describe("duty/management — Tab 切换", () => {
  it("mount 默认 weekly Tab：渲染周视图按钮并初始化拉取 4 类数据", async () => {
    render(<ManagementPage />, { wrapper: Wrap });

    await waitFor(() => expect(buttonByText("上一周")).toBeInTheDocument());
    expect(buttonByText("今天")).toBeInTheDocument();
    expect(buttonByText("下一周")).toBeInTheDocument();
    await waitFor(() => {
      expect(scheduleData.fetchWeeklyDuty).toHaveBeenCalledWith(scheduleData.currentWeekStart);
      expect(holidayData.fetchYears).toHaveBeenCalled();
      expect(dutyConfig.fetch).toHaveBeenCalled();
    });
  });

  it("切到 节假日管理 Tab：渲染节假日表格行", async () => {
    render(<ManagementPage />, { wrapper: Wrap });
    await switchTab("节假日管理", "元旦");
    expect(document.querySelector(".ant-table-tbody")?.textContent).toContain("法定节假日");
  });

  it("切到 值班配置 Tab：渲染当前配置 + 修改配置表单", async () => {
    render(<ManagementPage />, { wrapper: Wrap });
    await switchTab("值班配置", "当前配置");
    expect(screen.getByText("修改配置")).toBeInTheDocument();
    expect(screen.getByText("已启用")).toBeInTheDocument();
    expect(screen.getByText("09:00")).toBeInTheDocument();
    // 「当前配置」Card 内的描述项把 reminderChannels 拆成 Tag 渲染
    const currentCard = screen.getByText("当前配置").closest(".ant-card") as HTMLElement;
    expect(within(currentCard).getByText("站内通知")).toBeInTheDocument();
    expect(within(currentCard).getByText("30分钟")).toBeInTheDocument();
    expect(screen.getByText("注意事项")).toBeInTheDocument();
  });

  it("切到 值班池管理 Tab：挂载 DutyPoolPage", async () => {
    const { getByTestId } = render(<ManagementPage />, { wrapper: Wrap });
    await switchTab("值班池管理", "值班池页面桩");
    // 说明卡属于值班配置 Tab，切到值班池后应不再显示（面板互斥）
    expect(screen.queryByText("注意事项")).not.toBeInTheDocument();
    expect(getByTestId("duty-pool-page")).toBeInTheDocument();
  });
});

describe("duty/management — 周视图按钮", () => {
  it("上一周 / 今天 / 下一周 三个按钮分别调用 scheduleData 对应方法", async () => {
    render(<ManagementPage />, { wrapper: Wrap });
    await waitFor(() => expect(buttonByText("上一周")).toBeInTheDocument());

    fireEvent.click(buttonByText("上一周"));
    fireEvent.click(buttonByText("今天"));
    fireEvent.click(buttonByText("下一周"));

    expect(scheduleData.prevWeek).toHaveBeenCalledTimes(1);
    expect(scheduleData.todayWeek).toHaveBeenCalledTimes(1);
    expect(scheduleData.nextWeek).toHaveBeenCalledTimes(1);
  });
});

describe("duty/management — 节假日操作与 Modal", () => {
  it("新增按钮打开 节假日 Modal，取消按钮关闭", async () => {
    render(<ManagementPage />, { wrapper: Wrap });
    await switchTab("节假日管理", "元旦");

    fireEvent.click(buttonByText("新增"));
    const modal = await waitFor(() => activeModal());
    expect(within(modal).getByText("节假日")).toBeInTheDocument();
    // 表单字段（destroyOnHidden 下打开才挂载）
    expect(within(modal).getByText("日期")).toBeInTheDocument();
    expect(within(modal).getByText("名称")).toBeInTheDocument();

    fireEvent.click(modalCancelButton());
    await waitFor(() => {
      expect(document.querySelector(".ant-modal-wrap:not(.ant-modal-wrap-hidden)")).toBeNull();
    });
  });

  it("批量新增按钮打开 批量添加节假日 Modal，确定校验失败时不提交", async () => {
    render(<ManagementPage />, { wrapper: Wrap });
    await switchTab("节假日管理", "元旦");

    fireEvent.click(buttonByText("批量新增"));
    const modal = await waitFor(() => activeModal());
    expect(within(modal).getByText("批量添加节假日")).toBeInTheDocument();
    expect(within(modal).getByText("日期范围")).toBeInTheDocument();
    expect(within(modal).getByText("最多可选择90天")).toBeInTheDocument();

    await act(async () => {
      fireEvent.click(modalOkButton());
    });
    // 必填项（日期范围 / 名称）未填 → validateFields 抛错 → batchCreate 不被调用
    expect(holidayData.batchCreate).not.toHaveBeenCalled();
  });

  it("刷新 / 下载模板按钮分别调用对应回调", async () => {
    render(<ManagementPage />, { wrapper: Wrap });
    await switchTab("节假日管理", "元旦");

    fireEvent.click(buttonByText("刷新"));
    fireEvent.click(buttonByText("下载模板"));

    expect(holidayData.fetchList).toHaveBeenCalled();
    expect(downloadHolidayTemplate).toHaveBeenCalled();
  });

  it("表格操作列：编辑按钮预填表单并打开 Modal", async () => {
    render(<ManagementPage />, { wrapper: Wrap });
    await switchTab("节假日管理", "元旦");

    const row = await screen.findByText("元旦");
    const tbody = row.closest(".ant-table-tbody") as HTMLElement;
    const editBtn = within(tbody).getByText("编辑");
    fireEvent.click(editBtn);

    // handleHolidayEdit 用 holidayForm.setFieldsValue 预填（dayjs 化日期），再打开 Modal
    const modal = await waitFor(() => activeModal());
    expect(within(modal).getByText("节假日")).toBeInTheDocument();
  });

  it("年份切换下拉：选择 2025 触发 setHolidayYear + fetchList", async () => {
    const { container } = render(<ManagementPage />, { wrapper: Wrap });
    await switchTab("节假日管理", "元旦");

    const yearSelect = container.querySelector(".ant-select") as HTMLElement;
    await act(async () => {
      fireEvent.mouseDown(yearSelect);
    });
    const option = await screen.findByText("2025年");
    await act(async () => {
      fireEvent.click(option);
    });
    expect(holidayData.setHolidayYear).toHaveBeenCalledWith(2025);
    expect(holidayData.fetchList).toHaveBeenCalledWith(2025);
  });
});

describe("duty/management — 值班配置保存", () => {
  it("点击保存配置：表单校验通过后调用 dutyConfig.save", async () => {
    render(<ManagementPage />, { wrapper: Wrap });
    await switchTab("值班配置", "当前配置");

    // config 非空时 DutyConfig 的 useEffect 会把初始值回填到表单（含必填的 reminderTime
    // 与 reminderChannels），因此 validateFields 可通过并走到 handleConfigSave。
    const saveBtn = screen.getByRole("button", { name: /保存配置/ });
    await act(async () => {
      fireEvent.click(saveBtn);
    });
    await waitFor(() => {
      expect(dutyConfig.save).toHaveBeenCalledTimes(1);
    });
    const values = vi.mocked(dutyConfig.save).mock.calls[0][0];
    expect(values.reminderEnabled).toBe(true);
    expect(values.reminderChannels).toEqual(["websocket"]);
    expect(values.beforeReminderMinutes).toBe(30);
  });
});
