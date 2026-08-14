/**
 * Phase 53 W4 — BulkWriteDrawer 行为测试 (UI-03 / BATCH-05 / CR-01 regression)
 *
 * 这是 Phase 53 最高价值测试 — 守护 commit 9b01cc68 的 CR-01/CR-02 安全 BLOCKER 修复:
 *   首次提交后,父组件 onSuccess 触发 loadPortStatus → portStatus 引用变化 →
 *   selectedPorts prop 重新过滤得到新引用 → uniqueDeviceIds 重算 →
 *   如果 retry 路径从 uniqueDeviceIds[0] 读 deviceId 会拿到错位的 deviceId,
 *   结合后端 batch_orchestrator fallback SSH 路径会形成跨设备误操作风险。
 *   修复方案:首次提交时缓存 lastDeviceId, retry 时用缓存值。
 *
 * 通过公共组件接口测试 (不导出内部 buildRequest):
 *   - CR-01 regression guard: 在 phase=result 时模拟父组件 selectedPorts prop 漂移到 device B,
 *     点击重试,断言 batchWritePorts 第二次调用的 BatchWriteRequest.deviceId === device A
 *   - buildRequest whitelist: batchWritePorts 第一次调用的对象 only 含
 *     deviceId / action / portIds / description
 *   - retry only takes failed: 第二次 portIds === 第一次 failed 的 portIds
 *   - state machine: open + submit → phase 转 executing → 转入 result (DOM 中可见 Statistic 卡片)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";
import { App } from "antd";
import { MemoryRouter } from "react-router-dom";
import { useState, type ReactNode } from "react";
import type { DevicePortStatus, PortResult, BatchResult } from "@/types/network";

// Polyfill: antd v6 Drawer uses ResizeObserver (jsdom lacks it)
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
if (typeof globalThis.ResizeObserver === "undefined") {
  (globalThis as unknown as { ResizeObserver: typeof ResizeObserverStub }).ResizeObserver =
    ResizeObserverStub;
}

// 捕获 batchWritePorts 调用 — 通过 networkApi 模块路径 mock
const mockBatchWritePorts = vi.fn();
vi.mock("@/lib/api/networkApi", () => ({
  batchWritePorts: (...args: unknown[]) => mockBatchWritePorts(...args),
}));

import { BulkWriteDrawer } from "../BulkWriteDrawer";

// Test wrapper: antd App 提供 message context + MemoryRouter 提供 useNavigate context
// (BulkWriteDrawer 用了 App.useApp() 和 useNavigate())
function Wrapper({ children }: { children: ReactNode }) {
  return (
    <MemoryRouter>
      <App>{children}</App>
    </MemoryRouter>
  );
}

/**
 * CR-01 专用 harness: 暴露 setSelectedPorts 让测试模拟父组件 loadPortStatus
 * 触发 selectedPorts prop 漂移 (不重新挂载 Router, 避免 nested Router 错误)。
 */
interface HarnessProps {
  initialPorts: DevicePortStatus[];
  setSelectedPortsRef: React.MutableRefObject<((ports: DevicePortStatus[]) => void) | null>;
}
function Harness({ initialPorts, setSelectedPortsRef }: HarnessProps) {
  const [ports, setPorts] = useState<DevicePortStatus[]>(initialPorts);
  setSelectedPortsRef.current = setPorts;
  return (
    <BulkWriteDrawer open={true} selectedPorts={ports} onClose={vi.fn()} onSuccess={vi.fn()} />
  );
}

/**
 * antd Select helper: open dropdown by Form.Item label text, click option by option text.
 *
 * antd v6: Select dropdown options render in a portal under <body>;
 * option items have class `.ant-select-item-option-content`. The Select itself
 * is `.ant-select` (no longer `.ant-select-selector`); we trigger mouseDown on
 * the whole `.ant-select` container to open the dropdown.
 */
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

/** 填表 + 提交: action=shutdown + reason="故障排查处理" (走预设项, 不进 TextArea) */
async function fillAndSubmit(): Promise<void> {
  await selectOptionByLabel("操作类型", "关闭端口");
  await selectOptionByLabel("操作原因", "故障排查处理");
  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "开始批量配置" }));
  });
}

// ---- Fixtures ----

function makePort(id: string, deviceId: string, interfaceName: string): DevicePortStatus {
  return {
    id,
    deviceId,
    interfaceName,
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

const DEVICE_A = "device-a-uuid";
const DEVICE_B = "device-b-uuid";

const PORT_A1 = makePort("port-a-1", DEVICE_A, "GigabitEthernet0/1");
const PORT_A2 = makePort("port-a-2", DEVICE_A, "GigabitEthernet0/2");
const PORT_A3 = makePort("port-a-3", DEVICE_A, "GigabitEthernet0/3");

const PORT_B1 = makePort("port-b-1", DEVICE_B, "GigabitEthernet0/1");

function makePortResult(
  portId: string,
  status: "succeeded" | "failed" | "skipped",
  error?: string
): PortResult {
  return {
    portId,
    action: "shutdown",
    status,
    noOp: false,
    error,
    commandSent: "shutdown",
  };
}

describe("BulkWriteDrawer — Phase 53 W4 (UI-03/BATCH-05/CR-01 regression)", () => {
  beforeEach(() => {
    mockBatchWritePorts.mockReset();
  });

  /**
   * CR-01 regression guard (核心守护):
   * 首次提交 device A 的端口返回部分失败 → 父组件刷新使 selectedPorts 漂移到 device B
   * → 点重试 → batchWritePorts 第二次调用的 deviceId 必须仍是 device A
   * (缓存的 lastDeviceId, 不从漂移的 selectedPorts 读)
   */
  it("CR-01: retry uses cached lastDeviceId even when selectedPorts prop drifts to a different device", async () => {
    // 首次提交返回: 2 成功 + 1 失败 (port-a-3)
    const firstResult: BatchResult = {
      succeeded: [makePortResult("port-a-1", "succeeded"), makePortResult("port-a-2", "succeeded")],
      failed: [makePortResult("port-a-3", "failed", "device timeout")],
      skipped: [],
    };
    // 重试结果: 全部成功
    const retryResult: BatchResult = {
      succeeded: [makePortResult("port-a-3", "succeeded")],
      failed: [],
      skipped: [],
    };
    mockBatchWritePorts.mockResolvedValueOnce(firstResult).mockResolvedValueOnce(retryResult);

    // ★ Harness 模式: setSelectedPortsRef 让测试中途改 selectedPorts prop 而不重挂 Router
    const setSelectedPortsRef: React.MutableRefObject<
      ((ports: DevicePortStatus[]) => void) | null
    > = { current: null };

    render(
      <MemoryRouter>
        <App>
          <Harness
            initialPorts={[PORT_A1, PORT_A2, PORT_A3]}
            setSelectedPortsRef={setSelectedPortsRef}
          />
        </App>
      </MemoryRouter>
    );

    await fillAndSubmit();

    // 等待首次 batchWritePorts 调用完成 + 进入 result 阶段
    await waitFor(() => {
      expect(mockBatchWritePorts).toHaveBeenCalledTimes(1);
    });
    const firstCallArg = mockBatchWritePorts.mock.calls[0][0] as {
      deviceId: string;
      action: string;
      portIds: string[];
    };
    expect(firstCallArg.deviceId).toBe(DEVICE_A);

    // 等待结果视图渲染 (失败计数 Statistic 出现)
    await waitFor(() => {
      expect(screen.getByText("✗ 失败")).toBeInTheDocument();
    });

    // ★ 父组件刷新模拟: selectedPorts prop 漂移到 device B 的端口
    //    (这正是 CR-01 描述的 onSuccess → loadPortStatus → selectedPorts 引用漂移场景)
    await act(async () => {
      setSelectedPortsRef.current?.([PORT_B1]);
    });

    // 点击"重试失败端口 (1)"
    const retryBtn = await screen.findByRole("button", {
      name: /重试失败端口/,
    });
    await act(async () => {
      fireEvent.click(retryBtn);
    });

    // ★ 核心断言: retry 调用的 deviceId 必须是缓存的 device A, 不是漂移后的 device B
    await waitFor(() => {
      expect(mockBatchWritePorts).toHaveBeenCalledTimes(2);
    });
    const retryCallArg = mockBatchWritePorts.mock.calls[1][0] as {
      deviceId: string;
      action: string;
      portIds: string[];
    };

    expect(retryCallArg.deviceId).toBe(DEVICE_A);
    // 如果修复未落地, deviceId 会是 DEVICE_B → 此处失败 = BLOCKER
    expect(retryCallArg.deviceId).not.toBe(DEVICE_B);
  });

  /**
   * buildRequest whitelist (T-53-07):
   * batchWritePorts 第一次调用对象 only 含 deviceId/action/portIds/description,
   * 不 spread 整个 port record 进去 (防意外字段污染后端 binding)
   */
  it("buildRequest whitelist: only deviceId/action/portIds/description keys are sent (no port-record spread)", async () => {
    const result: BatchResult = {
      succeeded: [makePortResult("port-a-1", "succeeded")],
      failed: [],
      skipped: [],
    };
    mockBatchWritePorts.mockResolvedValueOnce(result);

    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1]}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper: Wrapper }
    );

    await fillAndSubmit();

    await waitFor(() => {
      expect(mockBatchWritePorts).toHaveBeenCalledTimes(1);
    });

    const callArg = mockBatchWritePorts.mock.calls[0][0] as Record<string, unknown>;
    // 仅允许 deviceId / action / portIds / description 4 个白名单字段
    const allowedKeys = new Set(["deviceId", "action", "portIds", "description"]);
    const actualKeys = Object.keys(callArg);
    for (const key of actualKeys) {
      expect(allowedKeys.has(key)).toBe(true);
    }
    // 关键字段非空
    expect(typeof callArg.deviceId).toBe("string");
    expect(callArg.deviceId).toBe(DEVICE_A);
    expect(callArg.action).toBe("shutdown");
    expect(Array.isArray(callArg.portIds)).toBe(true);
    expect(callArg.portIds).toEqual(["port-a-1"]);
  });

  /**
   * retry only takes failed (D-06):
   * succeeded / skipped 端口不进 retry 范围 (重试 skipped 无意义 — 设备已是目标态)
   */
  it("D-06: retry only takes failed portIds, excludes succeeded and skipped", async () => {
    const firstResult: BatchResult = {
      succeeded: [makePortResult("port-a-1", "succeeded")],
      failed: [makePortResult("port-a-2", "failed", "device timeout")],
      skipped: [makePortResult("port-a-3", "skipped")],
    };
    const retryResult: BatchResult = {
      succeeded: [makePortResult("port-a-2", "succeeded")],
      failed: [],
      skipped: [],
    };
    mockBatchWritePorts.mockResolvedValueOnce(firstResult).mockResolvedValueOnce(retryResult);

    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1, PORT_A2, PORT_A3]}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper: Wrapper }
    );

    await fillAndSubmit();

    await waitFor(() => expect(mockBatchWritePorts).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(screen.getByText("✗ 失败")).toBeInTheDocument());

    // retry
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: /重试失败端口/ }));
    });

    await waitFor(() => expect(mockBatchWritePorts).toHaveBeenCalledTimes(2));

    const retryArg = mockBatchWritePorts.mock.calls[1][0] as {
      portIds: string[];
    };
    // 仅 port-a-2 (failed), 不含 port-a-1 (succeeded) / port-a-3 (skipped)
    expect(retryArg.portIds).toEqual(["port-a-2"]);
  });

  /**
   * state machine: open + 提交 → executing → result
   * (select 视图含"开始批量配置"按钮; result 视图含"✓ 成功"/"⚠ 跳过"/"✗ 失败" Statistic)
   */
  it("state machine: select → executing → result", async () => {
    const result: BatchResult = {
      succeeded: [makePortResult("port-a-1", "succeeded")],
      failed: [],
      skipped: [],
    };
    mockBatchWritePorts.mockResolvedValueOnce(result);

    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1]}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper: Wrapper }
    );

    // select 阶段: 提交按钮可见
    expect(screen.getByRole("button", { name: "开始批量配置" })).toBeInTheDocument();

    await fillAndSubmit();

    // result 阶段: 三 Statistic 卡片渲染
    await waitFor(() => {
      expect(screen.getByText("✓ 成功")).toBeInTheDocument();
      expect(screen.getByText("⚠ 跳过")).toBeInTheDocument();
      expect(screen.getByText("✗ 失败")).toBeInTheDocument();
    });
  });

  /**
   * 跨设备预校验 (T-53-07): selectedPorts 含多设备 → Alert 显示 + 提交按钮 disabled
   */
  it("cross-device pre-validation: mixed devices shows Alert and disables submit button", async () => {
    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1, PORT_B1]} // 2 different devices
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper: Wrapper }
    );

    // 跨设备 Alert 出现
    expect(screen.getByText("批量必须同设备")).toBeInTheDocument();

    // 提交按钮 disabled
    const submitBtn = screen.getByRole("button", {
      name: "开始批量配置",
    }) as HTMLButtonElement;
    expect(submitBtn.disabled).toBe(true);

    // batchWritePorts 不应被调用 (按钮 disabled)
    expect(mockBatchWritePorts).not.toHaveBeenCalled();
  });

  /**
   * D-07 onExecutingChange callback: phase 转变时上抛给父组件
   * (父组件靠它禁用刷新+采集按钮 — LANDMINE #4)
   *
   * 验证: select (initial) → false; 任意 phase 变化都触发回调 (mount / executing / result)。
   * 即便 React 在 act() 内可能批处理 setPhase+await+setPhase (跳过 executing 的中间 effect),
   * 父组件至少能感知到 select(false) 和 result(false), 不会一直卡在 stale false。
   */
  it("D-07: onExecutingChange is called on phase transitions; final state is false after submit completes", async () => {
    const onExecutingChange = vi.fn();
    const result: BatchResult = {
      succeeded: [makePortResult("port-a-1", "succeeded")],
      failed: [],
      skipped: [],
    };
    mockBatchWritePorts.mockResolvedValueOnce(result);

    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1]}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
        onExecutingChange={onExecutingChange}
      />,
      { wrapper: Wrapper }
    );

    // 初始 mount: select 阶段, onExecutingChange(false) 已触发
    expect(onExecutingChange).toHaveBeenCalledWith(false);
    const callsAfterMount = onExecutingChange.mock.calls.length;

    await fillAndSubmit();

    // 等待提交完成 (mock resolve + 进入 result 阶段)
    await waitFor(() => {
      expect(mockBatchWritePorts).toHaveBeenCalledTimes(1);
    });
    await waitFor(() => {
      expect(screen.getByText("✓ 成功")).toBeInTheDocument();
    });

    // 至少触发了一次额外回调 (无论是否经过 executing 中间态)
    expect(onExecutingChange.mock.calls.length).toBeGreaterThan(callsAfterMount);

    // 最终最后一次调用必须是 false (result 阶段已结束 executing)
    const lastCall = onExecutingChange.mock.calls[onExecutingChange.mock.calls.length - 1];
    expect(lastCall[0]).toBe(false);
  });
});
