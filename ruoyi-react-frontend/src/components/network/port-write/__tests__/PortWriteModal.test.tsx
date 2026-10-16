/**
 * Phase 53 W4 — PortWriteModal 校验行为单元测试 (UI-02)
 *
 * 锁定行为:
 * - shutdown action: 不选 reason 提交 → 校验失败, wrapper 不被调用, Modal 不关闭
 * - shutdown action: 选预设项 reason + 提交 → writeShutdown 被调用, Modal 关闭 (success path)
 * - description action: 不填 description 提交 → 校验失败 ("请输入新端口描述")
 * - description action: 填 description + 不填 reason → 可空 (D-03 特例) → writeDescription 调用
 * - 不同 action 切换时 Modal 标题随 ACTION_TITLE[action] 变化 (D-01)
 *
 * 实现: 通过 antd Form 行为驱动 (fireEvent submit + waitFor validation), 不直接调 form API。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, act } from "@testing-library/react";
import { screen, fireEvent, waitFor } from "@testing-library/dom";
import { App } from "antd";
import { MemoryRouter } from "react-router-dom";
import type { ReactNode } from "react";
import type { DevicePortStatus } from "@/types/network";

// Polyfill: antd v6 uses ResizeObserver (jsdom lacks it)
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
if (typeof globalThis.ResizeObserver === "undefined") {
  (globalThis as unknown as { ResizeObserver: typeof ResizeObserverStub }).ResizeObserver =
    ResizeObserverStub;
}

// Mock 5 single-port wrappers + showAuditLinkToast 不在测试范围 (内部 navigate, 不测)
const mockWriteShutdown = vi.fn();
const mockWriteUndoShutdown = vi.fn();
const mockWriteDescription = vi.fn();
const mockWriteDot1xEnable = vi.fn();
const mockWriteDot1xDisable = vi.fn();
vi.mock("@/lib/api/networkApi", () => ({
  writeShutdown: (...args: unknown[]) => mockWriteShutdown(...args),
  writeUndoShutdown: (...args: unknown[]) => mockWriteUndoShutdown(...args),
  writeDescription: (...args: unknown[]) => mockWriteDescription(...args),
  writeDot1xEnable: (...args: unknown[]) => mockWriteDot1xEnable(...args),
  writeDot1xDisable: (...args: unknown[]) => mockWriteDot1xDisable(...args),
}));

import { PortWriteModal } from "../PortWriteModal";

function Wrapper({ children }: { children: ReactNode }) {
  return (
    <MemoryRouter>
      <App>{children}</App>
    </MemoryRouter>
  );
}

/** antd Select helper (same as BulkWriteDrawer test) */
async function selectOptionByLabel(labelText: string, optionText: string): Promise<void> {
  const labelEl = screen.getByText(labelText);
  const formItem = labelEl.closest(".ant-form-item");
  if (!formItem) throw new Error(`Form.Item not found for label: ${labelText}`);
  const selectEl = formItem.querySelector(".ant-select");
  if (!selectEl) throw new Error(`ant-select not found for label: ${labelText}`);

  await act(async () => {
    fireEvent.mouseDown(selectEl as HTMLElement);
  });

  await waitFor(() => {
    const options = document.querySelectorAll(".ant-select-item-option-content");
    const matched = Array.from(options).filter((el) => el.textContent === optionText);
    expect(matched.length).toBeGreaterThan(0);
  });

  const options = document.querySelectorAll(".ant-select-item-option-content");
  const target = Array.from(options).find((el) => el.textContent === optionText);
  if (!target) throw new Error(`Option not found: ${optionText}`);

  await act(async () => {
    fireEvent.click(target as HTMLElement);
  });
}

function makePort(): DevicePortStatus {
  return {
    id: "port-1",
    deviceId: "device-1",
    interfaceName: "GigabitEthernet0/1",
    adminStatus: "up",
    operStatus: "down",
    dot1xEnabled: false,
    dot1xPortStatus: "unknown",
    portSecurityEnabled: false,
    portSecurityMode: "",
    collectedAt: "2026-07-07T00:00:00Z",
    createdAt: "2026-07-07T00:00:00Z",
  };
}

/** 触发 Modal OK (antd Modal.okText 按钮) */
function clickOkButton(): void {
  const okBtn = screen.getByRole("button", { name: "确认执行" });
  fireEvent.click(okBtn);
}

describe("PortWriteModal — Phase 53 W4 (UI-02 validation)", () => {
  beforeEach(() => {
    mockWriteShutdown.mockReset();
    mockWriteUndoShutdown.mockReset();
    mockWriteDescription.mockReset();
    mockWriteDot1xEnable.mockReset();
    mockWriteDot1xDisable.mockReset();
  });

  describe("D-01: Modal title reflects ACTION_TITLE[action] + interfaceName", () => {
    it("renders '关闭端口 - <interfaceName>' for action=shutdown", () => {
      render(
        <PortWriteModal
          open={true}
          action="shutdown"
          portRecord={makePort()}
          onClose={vi.fn()}
          onSuccess={vi.fn()}
        />,
        { wrapper: Wrapper }
      );
      expect(screen.getByText(/关闭端口 - GigabitEthernet0\/1/)).toBeInTheDocument();
    });

    it("renders '修改描述' for action=description", () => {
      render(
        <PortWriteModal
          open={true}
          action="description"
          portRecord={makePort()}
          onClose={vi.fn()}
          onSuccess={vi.fn()}
        />,
        { wrapper: Wrapper }
      );
      expect(screen.getByText(/修改描述/)).toBeInTheDocument();
    });
  });

  describe("shutdown action: reason required (D-03 — 非 description action 必填)", () => {
    it("blocks submit when no reason selected — writeShutdown NOT called", async () => {
      const onClose = vi.fn();
      const onSuccess = vi.fn();
      render(
        <PortWriteModal
          open={true}
          action="shutdown"
          portRecord={makePort()}
          onClose={onClose}
          onSuccess={onSuccess}
        />,
        { wrapper: Wrapper }
      );

      await act(async () => {
        clickOkButton();
      });

      // 校验失败: wrapper / onClose / onSuccess 都不应被调用
      await waitFor(() => {
        expect(mockWriteShutdown).not.toHaveBeenCalled();
      });
      expect(onClose).not.toHaveBeenCalled();
      expect(onSuccess).not.toHaveBeenCalled();

      // antd Form 校验错误提示出现
      await waitFor(() => {
        expect(screen.getByText("请选择或输入操作原因")).toBeInTheDocument();
      });
    });

    it("submits writeShutdown when reason selected — wrapper called with portId + reason", async () => {
      mockWriteShutdown.mockResolvedValueOnce({
        portId: "port-1",
        action: "shutdown",
        status: "succeeded",
        noOp: false,
      });
      const onSuccess = vi.fn();
      const onClose = vi.fn();

      render(
        <PortWriteModal
          open={true}
          action="shutdown"
          portRecord={makePort()}
          onClose={onClose}
          onSuccess={onSuccess}
        />,
        { wrapper: Wrapper }
      );

      // 选预设 reason
      await selectOptionByLabel("操作原因", "故障排查处理");

      // 提交
      await act(async () => {
        clickOkButton();
      });

      // writeShutdown 被调用, 参数正确
      await waitFor(() => {
        expect(mockWriteShutdown).toHaveBeenCalledWith("port-1", "故障排查处理");
      });
      // onSuccess + onClose 触发
      await waitFor(() => {
        expect(onSuccess).toHaveBeenCalledTimes(1);
        expect(onClose).toHaveBeenCalledTimes(1);
      });
    });
  });

  describe("description action: description field required + reason optional (D-03)", () => {
    it("blocks submit when description empty — shows '请输入新端口描述'", async () => {
      const onClose = vi.fn();
      render(
        <PortWriteModal
          open={true}
          action="description"
          portRecord={makePort()}
          onClose={onClose}
          onSuccess={vi.fn()}
        />,
        { wrapper: Wrapper }
      );

      await act(async () => {
        clickOkButton();
      });

      // wrapper 不被调用
      expect(mockWriteDescription).not.toHaveBeenCalled();
      expect(onClose).not.toHaveBeenCalled();

      // "请输入新端口描述" 校验提示出现 (description 必填)
      await waitFor(() => {
        expect(screen.getByText("请输入新端口描述")).toBeInTheDocument();
      });
    });

    it("submits writeDescription when description filled and reason empty (D-03 optional reason)", async () => {
      mockWriteDescription.mockResolvedValueOnce({
        portId: "port-1",
        action: "description",
        status: "succeeded",
        noOp: false,
      });
      const onSuccess = vi.fn();

      render(
        <PortWriteModal
          open={true}
          action="description"
          portRecord={makePort()}
          onClose={vi.fn()}
          onSuccess={onSuccess}
        />,
        { wrapper: Wrapper }
      );

      // 填 description, 不填 reason (D-03: reason 可空)
      const descInput = screen.getByPlaceholderText("请输入新端口描述");
      await act(async () => {
        fireEvent.change(descInput, { target: { value: "uplink-to-core" } });
      });

      await act(async () => {
        clickOkButton();
      });

      // writeDescription 被调用 — reason 为 undefined (D-03 optional)
      await waitFor(() => {
        expect(mockWriteDescription).toHaveBeenCalledWith("port-1", "uplink-to-core", undefined);
      });
      await waitFor(() => {
        expect(onSuccess).toHaveBeenCalledTimes(1);
      });
    });
  });

  describe("all 5 actions reach correct wrappers (D-01 single Modal covers 5 actions)", () => {
    it("undo_shutdown action calls writeUndoShutdown", async () => {
      mockWriteUndoShutdown.mockResolvedValueOnce({
        portId: "port-1",
        action: "undo_shutdown",
        status: "succeeded",
        noOp: false,
      });

      render(
        <PortWriteModal
          open={true}
          action="undo_shutdown"
          portRecord={makePort()}
          onClose={vi.fn()}
          onSuccess={vi.fn()}
        />,
        { wrapper: Wrapper }
      );

      await selectOptionByLabel("操作原因", "故障排查处理");
      await act(async () => {
        clickOkButton();
      });

      await waitFor(() => {
        expect(mockWriteUndoShutdown).toHaveBeenCalledWith("port-1", "故障排查处理");
      });
    });

    it("dot1x_enable action calls writeDot1xEnable", async () => {
      mockWriteDot1xEnable.mockResolvedValueOnce({
        portId: "port-1",
        action: "dot1x_enable",
        status: "succeeded",
        noOp: false,
      });

      render(
        <PortWriteModal
          open={true}
          action="dot1x_enable"
          portRecord={makePort()}
          onClose={vi.fn()}
          onSuccess={vi.fn()}
        />,
        { wrapper: Wrapper }
      );

      await selectOptionByLabel("操作原因", "故障排查处理");
      await act(async () => {
        clickOkButton();
      });

      await waitFor(() => {
        expect(mockWriteDot1xEnable).toHaveBeenCalledWith("port-1", "故障排查处理");
      });
    });

    it("dot1x_disable action calls writeDot1xDisable", async () => {
      mockWriteDot1xDisable.mockResolvedValueOnce({
        portId: "port-1",
        action: "dot1x_disable",
        status: "succeeded",
        noOp: false,
      });

      render(
        <PortWriteModal
          open={true}
          action="dot1x_disable"
          portRecord={makePort()}
          onClose={vi.fn()}
          onSuccess={vi.fn()}
        />,
        { wrapper: Wrapper }
      );

      await selectOptionByLabel("操作原因", "故障排查处理");
      await act(async () => {
        clickOkButton();
      });

      await waitFor(() => {
        expect(mockWriteDot1xDisable).toHaveBeenCalledWith("port-1", "故障排查处理");
      });
    });
  });
});
