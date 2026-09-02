/**
 * Phase 88 Batch415 — pages/network/executions/modals/ExecuteModal 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { ConfigExecuteModal } from "../ExecuteModal";
import type { ReactElement, ReactNode } from "react";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("ConfigExecuteModal", () => {
  it("open=true 不抛错（空数据）", () => {
    expect(() =>
      render(
        <ConfigExecuteModal
          open={true}
          devices={[]}
          templates={[]}
          selectedTemplate={null}
          selectedRowKeys={[]}
          onOk={vi.fn()}
          onCancel={vi.fn()}
          onTemplateChange={vi.fn()}
          onSelectedRowKeysChange={vi.fn()}
        />,
        { wrapper }
      )
    ).not.toThrow();
  });

  it("有数据不抛错", () => {
    expect(() =>
      render(
        <ConfigExecuteModal
          open={true}
          devices={[
            {
              id: "dev1",
              hostname: "switch1",
              ipAddress: "10.0.0.1",
            } as any,
          ]}
          templates={[
            {
              id: "tpl1",
              templateName: "基础配置",
            } as any,
          ]}
          selectedTemplate={null}
          selectedRowKeys={[]}
          onOk={vi.fn()}
          onCancel={vi.fn()}
          onTemplateChange={vi.fn()}
          onSelectedRowKeysChange={vi.fn()}
        />,
        { wrapper }
      )
    ).not.toThrow();
  });
});