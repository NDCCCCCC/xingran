/**
 * Phase 88 Batch231 — components/table/VDIRow 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { VDIRow } from "../VDIRow";
import { vmOperationButtons } from "@/pages/vdi/VirtualMachineList/vmOperationButtons";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const baseVm: any = {
  id: "vm1",
  name: "VM1",
  power_state: "in_use",
};

describe("table/VDIRow", () => {
  it("displayName = VDIRow", () => {
    expect(VDIRow.displayName).toBe("VDIRow");
  });

  it("权限过滤:仅渲染有权限的按钮", () => {
    const perms = ["vdi:vm:start", "vdi:vm:stop"];
    render(
      <VDIRow
        vm={baseVm}
        permissions={perms}
        buttons={vmOperationButtons}
        onOperate={vi.fn()}
        onDelete={vi.fn()}
        onSync={vi.fn()}
        onBind={vi.fn()}
      />,
      { wrapper }
    );
    // start/stop 有权限,其他没有
    expect(screen.getAllByRole("button").length).toBe(2);
  });

  it("start 按钮: power_state=in_use → disabled", () => {
    const onOperate = vi.fn();
    render(
      <VDIRow
        vm={{ ...baseVm, power_state: "in_use" }}
        permissions={["vdi:vm:start"]}
        buttons={[vmOperationButtons[0]]}
        onOperate={onOperate}
        onDelete={vi.fn()}
        onSync={vi.fn()}
        onBind={vi.fn()}
      />,
      { wrapper }
    );
    const btn = screen.getByRole("button");
    expect(btn).toBeDisabled();
  });

  it("start 按钮: power_state=stopped → enabled", () => {
    const onOperate = vi.fn();
    render(
      <VDIRow
        vm={{ ...baseVm, power_state: "stopped" }}
        permissions={["vdi:vm:start"]}
        buttons={[vmOperationButtons[0]]}
        onOperate={onOperate}
        onDelete={vi.fn()}
        onSync={vi.fn()}
        onBind={vi.fn()}
      />,
      { wrapper }
    );
    const btn = screen.getByRole("button");
    expect(btn).not.toBeDisabled();
    fireEvent.click(btn);
    expect(onOperate).toHaveBeenCalledWith("start", ["vm1"]);
  });

  it("stop 按钮: stopped → disabled", () => {
    render(
      <VDIRow
        vm={{ ...baseVm, power_state: "stopped" }}
        permissions={["vdi:vm:stop"]}
        buttons={[vmOperationButtons[1]]}
        onOperate={vi.fn()}
        onDelete={vi.fn()}
        onSync={vi.fn()}
        onBind={vi.fn()}
      />,
      { wrapper }
    );
    expect(screen.getByRole("button")).toBeDisabled();
  });

  it("restart: in_use → enabled", () => {
    render(
      <VDIRow
        vm={{ ...baseVm, power_state: "in_use" }}
        permissions={["vdi:vm:restart"]}
        buttons={[vmOperationButtons[2]]}
        onOperate={vi.fn()}
        onDelete={vi.fn()}
        onSync={vi.fn()}
        onBind={vi.fn()}
      />,
      { wrapper }
    );
    expect(screen.getByRole("button")).not.toBeDisabled();
  });

  it("restart: stopped → disabled", () => {
    render(
      <VDIRow
        vm={{ ...baseVm, power_state: "stopped" }}
        permissions={["vdi:vm:restart"]}
        buttons={[vmOperationButtons[2]]}
        onOperate={vi.fn()}
        onDelete={vi.fn()}
        onSync={vi.fn()}
        onBind={vi.fn()}
      />,
      { wrapper }
    );
    expect(screen.getByRole("button")).toBeDisabled();
  });

  it("sync 按钮调用 onSync", () => {
    const onSync = vi.fn();
    render(
      <VDIRow
        vm={baseVm}
        permissions={["vdi:vm:sync"]}
        buttons={[vmOperationButtons[3]]}
        onOperate={vi.fn()}
        onDelete={vi.fn()}
        onSync={onSync}
        onBind={vi.fn()}
      />,
      { wrapper }
    );
    fireEvent.click(screen.getByRole("button"));
    expect(onSync).toHaveBeenCalledWith("vm1");
  });

  it("bind 按钮调用 onBind", () => {
    const onBind = vi.fn();
    render(
      <VDIRow
        vm={baseVm}
        permissions={["vdi:vm:bind"]}
        buttons={[vmOperationButtons[5]]}
        onOperate={vi.fn()}
        onDelete={vi.fn()}
        onSync={vi.fn()}
        onBind={onBind}
      />,
      { wrapper }
    );
    fireEvent.click(screen.getByRole("button"));
    expect(onBind).toHaveBeenCalledWith(baseVm);
  });

  it("delete 按钮 → Popconfirm", () => {
    const onDelete = vi.fn();
    render(
      <VDIRow
        vm={baseVm}
        permissions={["vdi:vm:remove"]}
        buttons={[vmOperationButtons[4]]}
        onOperate={vi.fn()}
        onDelete={onDelete}
        onSync={vi.fn()}
        onBind={vi.fn()}
      />,
      { wrapper }
    );
    const btn = screen.getByRole("button");
    fireEvent.click(btn);
    // Popconfirm 弹出
    expect(
      screen.getByText("确定要删除这个虚拟机吗？此操作将调用 VDI API 删除虚拟机。")
    ).toBeInTheDocument();
  });
});
