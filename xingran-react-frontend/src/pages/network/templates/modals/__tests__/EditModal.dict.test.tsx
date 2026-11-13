/**
 * Phase 88 Batch413 — EditModal 设备类型下拉字典驱动测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";
import { TemplateEditModal } from "../EditModal";
import type { DictItem } from "@/hooks/useDict";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

function renderModal(deviceTypeDict?: DictItem[]) {
  return render(
    <TemplateEditModal
      open={true}
      editingTemplate={null}
      onOk={vi.fn(async () => {})}
      onCancel={vi.fn()}
      deviceTypeDict={deviceTypeDict}
    />,
    { wrapper }
  );
}

describe("TemplateEditModal 设备类型下拉", () => {
  it("字典传入 → 管理员新增类型可在下拉选到", async () => {
    const dict = [
      { id: "1", dictValue: "router", dictLabel: "核心路由器" },
      { id: "2", dictValue: "vpn", dictLabel: "VPN 网关" },
    ] as DictItem[];
    renderModal(dict);
    fireEvent.mouseDown(screen.getAllByRole("combobox")[1]);
    expect(await screen.findByText("VPN 网关")).toBeInTheDocument();
    expect(screen.getByText("核心路由器")).toBeInTheDocument();
  });

  it("字典空态 → 回退静态 5 项", async () => {
    renderModal([]);
    fireEvent.mouseDown(screen.getAllByRole("combobox")[1]);
    expect(await screen.findByText("路由器")).toBeInTheDocument();
    expect(screen.getByText("负载均衡器")).toBeInTheDocument();
  });
});
