/**
 * Phase 88 Batch192 — pages/system/notice/components/RecurrenceConfig 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { App as AntdApp, Form } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/components/CronSelector", () => ({
  default: () => <div data-testid="cron-selector-mock">CronSelector</div>,
}));

import { RecurrenceConfig } from "../RecurrenceConfig";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

function renderConfig() {
  function TestComp() {
    const [form] = Form.useForm();
    return (
      <Form form={form} layout="vertical">
        <RecurrenceConfig />
      </Form>
    );
  }
  return render(<TestComp />, { wrapper });
}

describe("notice/components/RecurrenceConfig", () => {
  it("渲染 Cron 表达式 + 结束时间标签", () => {
    renderConfig();
    expect(screen.getByText("Cron 表达式")).toBeInTheDocument();
    expect(screen.getByText(/结束时间/)).toBeInTheDocument();
  });

  it("必填规则文本", () => {
    renderConfig();
    // required 标识 + rules message
    expect(screen.getByText(/Cron 表达式/)).toBeInTheDocument();
  });

  it("留空说明", () => {
    renderConfig();
    expect(screen.getByText(/留空表示永久执行/)).toBeInTheDocument();
  });

  it("CronSelector mock 渲染", () => {
    renderConfig();
    expect(screen.getByTestId("cron-selector-mock")).toBeInTheDocument();
  });
});
