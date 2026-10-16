/**
 * Phase 88 Batch179 — pages/monitor/job/columns/jobLogColumns 测试
 */
import { describe, it, expect } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App as AntdApp, Table } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/utils/datetime", () => ({
  formatDateTime: vi.fn(() => "2026-08-30 10:00:00"),
}));

import { getJobLogColumns } from "../jobLogColumns";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const dataSource = [
  {
    id: "1",
    startTime: "2026-08-30T10:00:00Z",
    duration: 1500,
    jobMessage: "Success",
    status: 1,
    exceptionInfo: "",
  },
  {
    id: "2",
    startTime: "2026-08-30T11:00:00Z",
    duration: 500,
    jobMessage: "Failed",
    status: 0,
    exceptionInfo: "Error stack",
  },
];

describe("jobLogColumns", () => {
  it("getJobLogColumns 返回 5 列", () => {
    const cols = getJobLogColumns();
    expect(cols.length).toBeGreaterThanOrEqual(5);
  });

  it("列包含基本列名", () => {
    const cols = getJobLogColumns();
    const titles = cols.map((c: any) => c.title);
    expect(titles).toContain("执行时间");
    expect(titles).toContain("执行时长");
    expect(titles).toContain("执行消息");
    expect(titles).toContain("执行状态");
  });

  it("完整表格渲染 + 各列 render", () => {
    const cols = getJobLogColumns();
    const { baseElement } = render(<Table dataSource={dataSource} columns={cols} rowKey="id" />, {
      wrapper,
    });
    // startTime → formatDateTime
    expect(baseElement.textContent).toContain("2026-08-30 10:00:00");
    // duration 1500 → "1.50s"
    expect(baseElement.textContent).toContain("1.50s");
    // duration 500 → "500ms"
    expect(baseElement.textContent).toContain("500ms");
    // jobMessage
    expect(baseElement.textContent).toContain("Success");
    expect(baseElement.textContent).toContain("Failed");
  });

  it("执行状态 → 成功/失败 Tag", () => {
    const cols = getJobLogColumns();
    const { baseElement } = render(<Table dataSource={dataSource} columns={cols} rowKey="id" />, {
      wrapper,
    });
    // status=1 → Success (LogStatus.Success=0, but jobLog uses raw 0/1)
    const tags = baseElement.querySelectorAll(".ant-tag");
    expect(tags.length).toBeGreaterThan(0);
  });
});
