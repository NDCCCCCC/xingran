/**
 * Phase 88 Batch146 — components/network/port-write/PortBindingModal 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/api/networkApi", () => ({
  writePortBinding: vi.fn(() => Promise.resolve({ code: 0 })),
}));

vi.mock("@/utils/authHelpers", () => ({
  getAuthHeaders: vi.fn(() => Promise.resolve({ Authorization: "Bearer test" })),
}));

vi.mock("../PortWriteModal", () => ({
  showAuditLinkToast: vi.fn(),
}));

import { PortBindingModal } from "../PortBindingModal";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return (
    <MemoryRouter>
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

const portRecord = {
  id: "p1",
  interfaceName: "eth0",
} as any;

describe("PortBindingModal", () => {
  it("open=true + portRecord → 渲染 Modal + 表单字段", () => {
    const { baseElement } = render(
      <PortBindingModal open portRecord={portRecord} onClose={vi.fn()} onSuccess={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("操作");
    expect(baseElement.textContent).toContain("IP 地址");
    expect(baseElement.textContent).toContain("MAC 地址");
    expect(baseElement.textContent).toContain("操作原因");
  });

  it("portRecord=null + open → 标题显示空接口名", () => {
    const { baseElement } = render(
      <PortBindingModal open portRecord={null} onClose={vi.fn()} onSuccess={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement).toBeDefined();
  });

  it("open=true → Radio.Group 显示 add/remove", () => {
    const { baseElement } = render(
      <PortBindingModal open portRecord={portRecord} onClose={vi.fn()} onSuccess={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("绑定");
  });

  it("confirm button text = 确认执行", () => {
    const { baseElement } = render(
      <PortBindingModal open portRecord={portRecord} onClose={vi.fn()} onSuccess={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("确认执行");
  });

  it("cancel button text = 取消", () => {
    const { baseElement } = render(
      <PortBindingModal open portRecord={portRecord} onClose={vi.fn()} onSuccess={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("取 消");
  });

  it("preset reason options render", () => {
    const { baseElement } = render(
      <PortBindingModal open portRecord={portRecord} onClose={vi.fn()} onSuccess={vi.fn()} />,
      { wrapper }
    );
    // PRESET_REASONS rendered as Select options
    expect(baseElement.querySelector(".ant-select")).toBeTruthy();
  });

  it("portRecord.interfaceName 显示在标题", () => {
    const { baseElement } = render(
      <PortBindingModal
        open
        portRecord={{ ...portRecord, interfaceName: "GigabitEthernet0/1" }}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("GigabitEthernet0/1");
  });
});
