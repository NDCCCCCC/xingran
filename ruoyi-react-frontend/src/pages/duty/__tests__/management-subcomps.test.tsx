/**
 * Phase 88 Batch20b — duty management/holidays 子组件 props 渲染
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ConfigProvider } from "antd";
import dayjs from "dayjs";

import { GenerateScheduleModal } from "../management/components/GenerateScheduleModal";
import { DutyConfig } from "../management/components/DutyConfig";
import { HolidayManagement } from "../management/components/HolidayManagement";
import { ManualScheduleModal } from "../management/modals/ManualScheduleModal";
import { BatchHolidayModal } from "../management/modals/BatchHolidayModal";
import { HolidayEditModal } from "../holidays/modals/EditModal";
import { HolidayBatchModal } from "../holidays/modals/BatchModal";

function wrap(ui: React.ReactElement) {
  return render(
    <ConfigProvider>
      <MemoryRouter>{ui}</MemoryRouter>
    </ConfigProvider>
  );
}

describe("duty — GenerateScheduleModal", () => {
  it("renders closed without crash", () => {
    const { baseElement } = wrap(
      <GenerateScheduleModal visible={false} onCancel={vi.fn()} onOk={vi.fn()} pools={[]} />
    );
    expect(baseElement).not.toBeNull();
  });

  it("renders open form fields", () => {
    const { baseElement } = wrap(
      <GenerateScheduleModal
        visible
        onCancel={vi.fn()}
        onOk={vi.fn()}
        pools={[{ id: "p1", poolName: "一线值班" }]}
      />
    );
    expect(baseElement.innerHTML.length).toBeGreaterThan(100);
  });
});

describe("duty — DutyConfig", () => {
  it("renders null config gracefully", () => {
    const { baseElement } = wrap(
      <DutyConfig config={null} loading={false} saving={false} onSave={vi.fn()} />
    );
    expect(baseElement).not.toBeNull();
  });

  it("renders config form", () => {
    const { baseElement } = wrap(
      <DutyConfig
        config={{
          id: "cfg1",
          reminderEnabled: true,
          reminderTime: "09:00",
          reminderChannels: "websocket,email",
          beforeReminderMinutes: 30,
          createdAt: "2026-01-01",
        }}
        loading={false}
        saving={false}
        onSave={vi.fn()}
      />
    );
    expect(baseElement.innerHTML.length).toBeGreaterThan(100);
  });
});

describe("duty — HolidayManagement", () => {
  it("renders holidays table", () => {
    const { baseElement } = wrap(
      <HolidayManagement
        holidays={[
          {
            id: "h1",
            holidayName: "元旦",
            holidayDate: "2026-01-01",
            holidayType: "legal",
            isOffday: true,
            year: 2026,
            createdAt: "2026-01-01",
            createdBy: "admin",
          },
        ]}
        loading={false}
        holidayYear={2026}
        availableYears={[2025, 2026]}
        onYearChange={vi.fn()}
        onRefresh={vi.fn()}
      />
    );
    expect(baseElement.innerHTML).toContain("元旦");
  });

  it("renders empty holidays", () => {
    const { baseElement } = wrap(
      <HolidayManagement
        holidays={[]}
        loading={false}
        holidayYear={2026}
        availableYears={[2026]}
        onYearChange={vi.fn()}
        onRefresh={vi.fn()}
      />
    );
    expect(baseElement.innerHTML.length).toBeGreaterThan(50);
  });
});

describe("duty — ManualScheduleModal / BatchHolidayModal", () => {
  it("ManualScheduleModal closed no crash", () => {
    const { baseElement } = wrap(
      <ManualScheduleModal
        visible={false}
        pools={[]}
        users={[]}
        onOk={vi.fn()}
        onCancel={vi.fn()}
      />
    );
    expect(baseElement).not.toBeNull();
  });

  it("BatchHolidayModal closed no crash", () => {
    const { baseElement } = wrap(
      <BatchHolidayModal visible={false} onOk={vi.fn()} onCancel={vi.fn()} />
    );
    expect(baseElement).not.toBeNull();
  });
});

describe("duty — HolidayEditModal / HolidayBatchModal", () => {
  it("HolidayEditModal renders for new record", () => {
    const { baseElement } = wrap(
      <HolidayEditModal
        open
        editingRecord={null}
        year={2026}
        availableYears={[2026]}
        holidayTypeDict={[]}
        onOk={vi.fn()}
        onCancel={vi.fn()}
      />
    );
    expect(baseElement.innerHTML.length).toBeGreaterThan(100);
  });

  it("HolidayBatchModal renders batch rows", () => {
    const { baseElement } = wrap(
      <HolidayBatchModal
        open
        batchHolidays={[
          {
            holidayDate: dayjs("2026-10-01"),
            holidayName: "国庆",
            holidayType: "legal",
            isOffday: true,
            year: 2026,
          },
        ]}
        holidayTypeDict={[]}
        onOk={vi.fn()}
        onCancel={vi.fn()}
      />
    );
    expect(baseElement.innerHTML).toContain("国庆");
  });
});
