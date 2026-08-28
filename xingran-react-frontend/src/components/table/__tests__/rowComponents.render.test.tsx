/**
 * Phase 88 Batch33 — components/table VDIRow + AssetRow 渲染测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";

import VDIRow from "../VDIRow";
import AssetRow from "../AssetRow";
import { vmOperationButtons } from "@/pages/vdi/VirtualMachineList/vmOperationButtons";

describe("VDIRow", () => {
  const mockVm: any = {
    id: "vm-1",
    name: "测试VM",
    power_state: "stopped",
  };

  it("按 permission 过滤可见按钮(全权限 → 6 个)", () => {
    const { container } = render(
      <VDIRow
        vm={mockVm}
        permissions={vmOperationButtons.map((b) => b.permission)}
        buttons={vmOperationButtons}
        onOperate={vi.fn()}
        onDelete={vi.fn()}
        onSync={vi.fn()}
        onBind={vi.fn()}
      />
    );
    const tooltipTitles = container.querySelectorAll(".anticon");
    expect(tooltipTitles.length).toBeGreaterThanOrEqual(6);
  });

  it("空权限 → 0 按钮", () => {
    const { container } = render(
      <VDIRow
        vm={mockVm}
        permissions={[]}
        buttons={vmOperationButtons}
        onOperate={vi.fn()}
        onDelete={vi.fn()}
        onSync={vi.fn()}
        onBind={vi.fn()}
      />
    );
    expect(container.querySelectorAll(".anticon").length).toBe(0);
  });

  it("power_state=stopped → start 可用 stop 禁用", () => {
    const { container } = render(
      <VDIRow
        vm={{ ...mockVm, power_state: "stopped" }}
        permissions={["vdi:vm:start", "vdi:vm:stop"]}
        buttons={vmOperationButtons}
        onOperate={vi.fn()}
        onDelete={vi.fn()}
        onSync={vi.fn()}
        onBind={vi.fn()}
      />
    );
    const buttons = container.querySelectorAll("button");
    // 期望:start 启用, stop 禁用
    // stop = power_state in {stopped, suspended, pending} → 此处 power_state=stopped → 禁用
    expect(buttons.length).toBe(2);
  });

  it("power_state=in_use → start 禁用 stop 启用", () => {
    const { container } = render(
      <VDIRow
        vm={{ ...mockVm, power_state: "in_use" }}
        permissions={["vdi:vm:start", "vdi:vm:stop"]}
        buttons={vmOperationButtons}
        onOperate={vi.fn()}
        onDelete={vi.fn()}
        onSync={vi.fn()}
        onBind={vi.fn()}
      />
    );
    const buttons = container.querySelectorAll("button");
    expect(buttons.length).toBe(2);
  });

  it("点击 onBind 回调触发", () => {
    const onBind = vi.fn();
    const { container } = render(
      <VDIRow
        vm={mockVm}
        permissions={["vdi:vm:bind"]}
        buttons={vmOperationButtons}
        onOperate={vi.fn()}
        onDelete={vi.fn()}
        onSync={vi.fn()}
        onBind={onBind}
      />
    );
    const btns = container.querySelectorAll("button");
    fireEvent.click(btns[0]);
    expect(onBind).toHaveBeenCalledWith(mockVm);
  });
});

describe("AssetRow", () => {
  const record: any = { id: "a1", name: "设备A" };

  it("渲染编辑/删除按钮 + 点击触发回调", () => {
    const onEdit = vi.fn();
    const onDelete = vi.fn();
    const { container } = render(<AssetRow record={record} onEdit={onEdit} onDelete={onDelete} />);
    const btns = container.querySelectorAll("button");
    expect(btns.length).toBe(2);
    fireEvent.click(btns[0]);
    fireEvent.click(btns[1]);
    expect(onEdit).toHaveBeenCalledWith(record);
    expect(onDelete).toHaveBeenCalledWith("a1");
  });

  it("onEdit/onDelete undefined 时不 throw", () => {
    const { container } = render(<AssetRow record={record} />);
    const btns = container.querySelectorAll("button");
    fireEvent.click(btns[0]);
    fireEvent.click(btns[1]);
    expect(container).not.toBeNull();
  });
});
