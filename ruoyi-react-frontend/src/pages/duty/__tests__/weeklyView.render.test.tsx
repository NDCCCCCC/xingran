/**
 * Phase 88 Batch27 — duty WeeklyView + SwapScheduleModal + HolidayModal 渲染补测
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { ConfigProvider, App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import dayjs from "dayjs";

import { WeeklyView } from "../management/components/WeeklyView";
import { SwapScheduleModal } from "../management/modals/SwapScheduleModal";
import { HolidayModal } from "../management/modals/HolidayModal";

function wrap(ui: React.ReactElement) {
  return render(
    <ConfigProvider>
      <AntdApp>
        <MemoryRouter>{ui}</MemoryRouter>
      </AntdApp>
    </ConfigProvider>
  );
}

const member = (username: string, dutyType: string, userId = "u1"): any => ({
  userId,
  username,
  nickname: username,
  dutyType,
});

describe("WeeklyView", () => {
  it("空数据渲染 7 天卡片 + 图例", () => {
    const { baseElement, getByText, getAllByText } = wrap(
      <WeeklyView currentWeekStart={dayjs("2026-08-24")} weeklyDutyData={{}} />
    );
    expect(baseElement).not.toBeNull();
    expect(getAllByText("无值班").length).toBe(7);
    // 7 天卡片 + 外层 Card(borderless variant 可能不计 .ant-card-bordered)
    expect(baseElement.querySelectorAll(".ant-card-small").length).toBe(7);
    // 图例
    expect(getByText("工作日")).toBeDefined();
    expect(getByText("周末")).toBeDefined();
    expect(getByText("节假日")).toBeDefined();
  });

  it("带值班数据渲染成员名", () => {
    const { getByText } = wrap(
      <WeeklyView
        currentWeekStart={dayjs("2026-08-24")}
        weeklyDutyData={{
          "2026-08-24": [member("张三", "weekday")],
          "2026-08-29": [member("李四", "weekend")],
        }}
      />
    );
    expect(getByText("张三")).toBeDefined();
    expect(getByText("李四")).toBeDefined();
  });

  it("周范围文本包含起止(startOf week 从周日算)", () => {
    // dayjs().startOf("week") 默认周日为一周起点
    const { baseElement } = wrap(
      <WeeklyView currentWeekStart={dayjs("2026-08-23")} weeklyDutyData={{}} />
    );
    expect(baseElement.innerHTML).toContain("2026年08月23日");
    expect(baseElement.innerHTML).toContain("2026年08月29日");
  });

  it("今天高亮(用真实今天所在周, rgb 色值)", () => {
    const today = dayjs().startOf("week");
    const { baseElement } = wrap(<WeeklyView currentWeekStart={today} weeklyDutyData={{}} />);
    // jsdom 将 hex 转为 rgb
    expect(baseElement.innerHTML).toContain("rgb(230, 247, 255)");
  });

  it("holiday 类型成员渲染", () => {
    const { baseElement } = wrap(
      <WeeklyView
        currentWeekStart={dayjs("2026-08-23")}
        weeklyDutyData={{ "2026-08-24": [member("王五", "holiday")] }}
      />
    );
    expect(baseElement.innerHTML).toContain("王五");
  });
});

describe("SwapScheduleModal", () => {
  it("closed 渲染无 crash", () => {
    const { baseElement } = wrap(
      <SwapScheduleModal
        visible={false}
        allSchedules={[
          { id: "s1", scheduleDate: "2026-08-24", user: { username: "zhang" } } as any,
        ]}
        onCancel={vi.fn()}
        onOk={vi.fn()}
      />
    );
    expect(baseElement).not.toBeNull();
  });
});

describe("HolidayModal", () => {
  it("closed 渲染无 crash", () => {
    const { baseElement } = wrap(
      <HolidayModal visible={false} onCancel={vi.fn()} onOk={vi.fn()} />
    );
    expect(baseElement).not.toBeNull();
  });

  it("open 渲染表单", () => {
    const { baseElement } = wrap(<HolidayModal visible onCancel={vi.fn()} onOk={vi.fn()} />);
    expect(baseElement.innerHTML.length).toBeGreaterThan(100);
  });
});
