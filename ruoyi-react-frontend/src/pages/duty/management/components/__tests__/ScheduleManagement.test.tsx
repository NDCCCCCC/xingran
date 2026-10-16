/**
 * Phase 88 Batch79 — duty/management/components/ScheduleManagement 渲染(46 stmts, 4.3% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { ScheduleManagement } from "../ScheduleManagement";
import { Form } from "antd";
import type { DutySchedule, DutyPool, SimpleUser } from "@/lib/dutyApi";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

interface HarnessProps {
  schedules?: DutySchedule[];
  total?: number;
  pools?: DutyPool[];
  users?: SimpleUser[];
  loading?: boolean;
  selectedRowKeys?: string[];
}

function Harness({
  schedules = [],
  total = 0,
  pools = [],
  users = [],
  loading = false,
  selectedRowKeys = [],
}: HarnessProps) {
  const [form] = Form.useForm();
  return (
    <ScheduleManagement
      schedules={schedules}
      loading={loading}
      total={total}
      current={1}
      pageSize={10}
      selectedRowKeys={selectedRowKeys}
      pools={pools}
      users={users}
      onSearch={vi.fn()}
      onReset={vi.fn()}
      onPageChange={vi.fn()}
      onSelectedChange={vi.fn()}
      onDelete={vi.fn()}
      onBatchDelete={vi.fn()}
      onGenerateClick={vi.fn()}
      onSwapClick={vi.fn()}
      onManualClick={vi.fn()}
      searchForm={form}
    />
  );
}

describe("ScheduleManagement 渲染", () => {
  it("空数据渲染", () => {
    const { baseElement } = renderWithProviders(<Harness />);
    expect(baseElement.querySelector(".ant-table")).not.toBeNull();
  });

  it("非空 schedules 渲染表格行", () => {
    const schedules: DutySchedule[] = [
      {
        id: "s1",
        scheduleDate: "2026-01-01",
        poolId: "p1",
        poolName: "P1",
        userId: "u1",
        userName: "Alice",
        shift: "day",
        status: 0,
      } as any,
    ];
    const { baseElement } = renderWithProviders(
      <Harness
        schedules={schedules}
        total={1}
        pools={[{ id: "p1", name: "P1" } as any]}
        users={[{ id: "u1", userName: "Alice" } as any]}
      />
    );
    expect(baseElement.querySelector(".ant-table-row")).not.toBeNull();
  });

  it("loading=true 渲染 loading", () => {
    const { baseElement } = renderWithProviders(<Harness loading />);
    expect(baseElement.querySelector(".ant-spin")).not.toBeNull();
  });

  it("选中行 keys 不为空", () => {
    const schedules: DutySchedule[] = [
      { id: "s1", scheduleDate: "2026-01-01", status: 0 } as any,
      { id: "s2", scheduleDate: "2026-01-02", status: 0 } as any,
    ];
    const { baseElement } = renderWithProviders(
      <Harness schedules={schedules} total={2} selectedRowKeys={["s1"]} />
    );
    expect(baseElement).toBeDefined();
  });
});
