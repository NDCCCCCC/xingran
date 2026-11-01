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
import { render, act } from "@testing-library/react";
import { screen, fireEvent, waitFor } from "@testing-library/dom";
import { App } from "antd";
import { MemoryRouter } from "react-router-dom";
import { useState, type ReactNode } from "react";
import type {
  DevicePortStatus,
  PortResult,
  BatchResult,
  PortWritePreviewResult,
} from "@/types/network";

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

// 捕获 batchWritePorts / previewBatchWritePorts 调用 — 通过 networkApi 模块路径 mock
const mockBatchWritePorts = vi.fn();
const mockPreviewBatchWritePorts = vi.fn();
vi.mock("@/lib/api/networkApi", () => ({
  batchWritePorts: (...args: unknown[]) => mockBatchWritePorts(...args),
  previewBatchWritePorts: (...args: unknown[]) => mockPreviewBatchWritePorts(...args),
}));

// PORTW-03: 可控的 noticeStore 进度事件派发（capture 模式, 测试手动触发回调）
const mockPortWriteListeners = new Set<(e: unknown) => void>();
function emitPortWriteEvent(event: unknown): void {
  mockPortWriteListeners.forEach((cb) => cb(event));
}
vi.mock("@/store/noticeStore", () => ({
  useNoticeStore: {
    getState: () => ({
      onPortWriteProgress: (cb: (e: unknown) => void) => {
        mockPortWriteListeners.add(cb);
        return () => mockPortWriteListeners.delete(cb);
      },
    }),
  },
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
  // eslint-disable-next-line react-hooks/refs -- test harness deliberately exposes setState via ref
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
async function fillForm(): Promise<void> {
  await selectOptionByLabel("操作类型", "关闭端口");
  await selectOptionByLabel("操作原因", "故障排查处理");
}

/**
 * PORTW-02 两段式 helper: 填表 → dry-run 预览 → 确认下发（真实写）。
 * 下游对 mockBatchWritePorts 的参数断言不变，仅提交入口经 preview 中转。
 */
async function fillAndSubmit(): Promise<void> {
  await fillForm();
  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "预览命令（dry-run）" }));
  });
  // preview 视图渲染完成（Alert 常驻标注）
  await waitFor(() => {
    expect(screen.getByText("dry-run 预览，未下发任何设备命令")).toBeInTheDocument();
  });
  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "确认下发（真实写）" }));
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

/** PORTW-02 dry-run 预览 fixture */
function makePreview(overrides?: Partial<PortWritePreviewResult>): PortWritePreviewResult {
  return {
    dryRun: true,
    deviceId: DEVICE_A,
    deviceName: "SW-01",
    vendor: "huawei",
    model: "S5735",
    action: "shutdown",
    riskLevel: "medium",
    summary: "将向 1 台设备 1 个端口下发 1 条命令",
    ports: [
      {
        portId: "port-a-1",
        interfaceName: "GigabitEthernet0/1",
        status: "preview",
        noOp: false,
        commands: [{ index: 1, command: "interface GigabitEthernet0/1", riskLevel: "low", critical: false }],
      },
    ],
    ...overrides,
  };
}

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
    mockPreviewBatchWritePorts.mockReset();
    mockPortWriteListeners.clear();
    // 默认 dry-run 预览成功（PORTW-02 两段式第一段），各用例可 override
    mockPreviewBatchWritePorts.mockResolvedValue(makePreview());
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

    // select 阶段: 预览提交按钮可见（PORTW-02 两段式入口）
    expect(screen.getByRole("button", { name: "预览命令（dry-run）" })).toBeInTheDocument();

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
      name: "预览命令（dry-run）",
    }) as HTMLButtonElement;
    expect(submitBtn.disabled).toBe(true);

    // batchWritePorts 不应被调用 (按钮 disabled)
    expect(mockBatchWritePorts).not.toHaveBeenCalled();
    expect(mockPreviewBatchWritePorts).not.toHaveBeenCalled();
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

  // ==================== PORTW-02 两段式写确认（v1.51 Phase 121） ====================

  it("PORTW-02: preview renders command sequence with risk tags and skipped noOp annotation, write only fires after confirm", async () => {
    mockPreviewBatchWritePorts.mockResolvedValue(
      makePreview({
        riskLevel: "critical",
        summary: "将向 1 台设备 2 个端口下发命令",
        ports: [
          {
            portId: "port-a-1",
            interfaceName: "GigabitEthernet0/1",
            status: "preview",
            noOp: false,
            commands: [
              { index: 1, command: "interface GigabitEthernet0/1", riskLevel: "low", critical: false },
              { index: 2, command: "shutdown", riskLevel: "critical", critical: true, note: "可能中断链路" },
            ],
          },
          {
            portId: "port-a-2",
            interfaceName: "GigabitEthernet0/2",
            status: "skipped",
            noOp: true,
            commands: [],
          },
        ],
      })
    );

    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1, PORT_A2]}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper: Wrapper }
    );

    await fillForm();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "预览命令（dry-run）" }));
    });

    // dry-run 常驻提示 + 后端 summary
    expect(await screen.findByText("dry-run 预览，未下发任何设备命令")).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByText("将向 1 台设备 2 个端口下发命令")).toBeInTheDocument();
    });

    // 风险色标（整体 critical + 命令级 critical/low）
    expect(screen.getAllByText("高危").length).toBeGreaterThan(0);
    expect(screen.getAllByText("低危").length).toBeGreaterThan(0);

    // 命令序列 + note + skipped noOp 标注
    expect(screen.getByText("shutdown")).toBeInTheDocument();
    expect(screen.getByText("可能中断链路")).toBeInTheDocument();
    expect(screen.getAllByText("跳过 (noOp)").length).toBeGreaterThan(0);

    // 两段式核心: preview 阶段不触发真实写
    expect(mockPreviewBatchWritePorts).toHaveBeenCalledTimes(1);
    expect(mockBatchWritePorts).not.toHaveBeenCalled();

    // 确认后才进入真实写
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "确认下发（真实写）" }));
    });
    await waitFor(() => {
      expect(mockBatchWritePorts).toHaveBeenCalledTimes(1);
    });
    const writeArg = mockBatchWritePorts.mock.calls[0][0] as { deviceId: string; portIds: string[] };
    expect(writeArg.deviceId).toBe(DEVICE_A);
    expect(writeArg.portIds).toEqual(["port-a-1", "port-a-2"]);
  });

  it("PORTW-02: 高危操作确认按钮为 danger（视觉强化），返回修改不触发写", async () => {
    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1]}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper: Wrapper }
    );

    await fillForm();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "预览命令（dry-run）" }));
    });
    await waitFor(() => {
      expect(screen.getByText("dry-run 预览，未下发任何设备命令")).toBeInTheDocument();
    });

    // 默认 fixture riskLevel=medium → 非 danger
    const confirmBtn = screen.getByRole("button", { name: "确认下发（真实写）" }) as HTMLButtonElement;
    expect(confirmBtn.className).not.toContain("danger");

    // 返回修改: 回到 select, 不触发真实写
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "返回修改" }));
    });
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "预览命令（dry-run）" })).toBeInTheDocument();
    });
    expect(mockBatchWritePorts).not.toHaveBeenCalled();
  });

  it("PORTW-02: preview 调用失败（400 参数非法）留在 select 并可重新提交", async () => {
    mockPreviewBatchWritePorts.mockRejectedValueOnce(new Error("vlanId 越界"));

    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1]}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper: Wrapper }
    );

    await fillForm();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "预览命令（dry-run）" }));
    });

    // 失败回 select: 预览按钮重新可见
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "预览命令（dry-run）" })).toBeInTheDocument();
    });
    expect(mockBatchWritePorts).not.toHaveBeenCalled();
  });

  // ==================== Phase 123 错误语义适配（commit 6cfdca54） ====================

  it("Phase 123: 写失败且回滚也失败 → 常驻 error Alert + 引导审计页", async () => {
    mockBatchWritePorts.mockResolvedValueOnce({
      succeeded: [],
      failed: [
        {
          ...makePortResult("port-a-1", "failed", "device timeout"),
          extra: { rollback: "rollback_failed", rollbackError: "undo shutdown 也超时" },
        },
      ],
      skipped: [],
    } as BatchResult);

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

    // 常驻告警出现（不能用一闪而过 message）
    expect(await screen.findByText("严重：部分端口写失败且自动回滚也失败")).toBeInTheDocument();
    expect(screen.getByText(/undo shutdown 也超时/)).toBeInTheDocument();
    expect(screen.getByText("查看端口写审计")).toBeInTheDocument();
  });

  it("Phase 123: rollback=rolled_back（回滚成功）不触发严重告警", async () => {
    mockBatchWritePorts.mockResolvedValueOnce({
      succeeded: [],
      failed: [
        {
          ...makePortResult("port-a-1", "failed", "device timeout"),
          extra: { rollback: "rolled_back" },
        },
      ],
      skipped: [],
    } as BatchResult);

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
      expect(screen.getByText("✗ 失败")).toBeInTheDocument();
    });
    expect(screen.queryByText("严重：部分端口写失败且自动回滚也失败")).not.toBeInTheDocument();
  });

  it("Phase 123: 确认写被拒 409 → 常驻 Alert「端口正在被其他写操作处理」, 留在 preview", async () => {
    const err409 = Object.assign(new Error("端口被 operator-x 占用"), { code: 409 });
    mockBatchWritePorts.mockRejectedValueOnce(err409);

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

    // 409 常驻 Alert + phase 回 preview（预览仍在, 可修正后重试）
    expect(await screen.findByText("端口正在被其他写操作处理")).toBeInTheDocument();
    expect(screen.getByText(/operator-x/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "确认下发（真实写）" })).toBeInTheDocument();
  });

  // ==================== Phase 124 trunk/hybrid（commit 9835291c） ====================

  it("Phase 124: trunk_set_native preview→确认→写, nativeVlan 透传", async () => {
    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1]}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper: Wrapper }
    );

    await selectOptionByLabel("操作类型", "设置 Trunk Native VLAN");
    await act(async () => {
      fireEvent.change(screen.getByLabelText("Native VLAN (PVID)"), { target: { value: "10" } });
    });
    await selectOptionByLabel("操作原因", "故障排查处理");
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "预览命令（dry-run）" }));
    });
    await waitFor(() => {
      expect(screen.getByText("dry-run 预览，未下发任何设备命令")).toBeInTheDocument();
    });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "确认下发（真实写）" }));
    });
    await waitFor(() => {
      expect(mockBatchWritePorts).toHaveBeenCalledTimes(1);
    });

    const previewArg = mockPreviewBatchWritePorts.mock.calls[0][0] as { nativeVlan: number };
    expect(previewArg.nativeVlan).toBe(10);
    const writeArg = mockBatchWritePorts.mock.calls[0][0] as { nativeVlan: number };
    expect(writeArg.nativeVlan).toBe(10);
  });

  it("Phase 124: trunk_set_allowed 去重解析 + mode 透传 + 厂商语义提示", async () => {
    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1]}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper: Wrapper }
    );

    await selectOptionByLabel("操作类型", "配置 Trunk 放行 VLAN");
    await act(async () => {
      fireEvent.change(screen.getByLabelText("放行 VLAN 列表"), {
        target: { value: "20,10 20,30" },
      });
    });
    await selectOptionByLabel("模式", "追加 (add)");
    await selectOptionByLabel("操作原因", "故障排查处理");
    // 厂商语义提示常驻（Form.Item extra, select 阶段可见）
    expect(screen.getByText(/replace（替换）仅锐捷原生支持/)).toBeInTheDocument();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "预览命令（dry-run）" }));
    });
    await waitFor(() => {
      expect(screen.getByText("dry-run 预览，未下发任何设备命令")).toBeInTheDocument();
    });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "确认下发（真实写）" }));
    });
    await waitFor(() => {
      expect(mockBatchWritePorts).toHaveBeenCalledTimes(1);
    });

    const previewArg = mockPreviewBatchWritePorts.mock.calls[0][0] as {
      allowedVlans: number[];
      mode: string;
    };
    expect(previewArg.allowedVlans).toEqual([10, 20, 30]); // 去重升序
    expect(previewArg.mode).toBe("add");
    const writeArg = mockBatchWritePorts.mock.calls[0][0] as { mode: string };
    expect(writeArg.mode).toBe("add");
  });

  it("Phase 124: trunk_set_hybrid 重叠被前端预校验拦截, 修正后可预览", async () => {
    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1]}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper: Wrapper }
    );

    await selectOptionByLabel("操作类型", "配置 Hybrid 端口");
    await act(async () => {
      fireEvent.change(screen.getByLabelText("Tagged VLAN 列表"), { target: { value: "30" } });
      fireEvent.change(screen.getByLabelText("Untagged VLAN 列表"), {
        target: { value: "30,10" },
      });
    });
    await selectOptionByLabel("操作原因", "故障排查处理");
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "预览命令（dry-run）" }));
    });

    // 重叠预校验拦截: 停留 select 且 preview 未被调用
    expect(await screen.findByText(/tagged 与 untagged VLAN 存在重叠/)).toBeInTheDocument();
    expect(mockPreviewBatchWritePorts).not.toHaveBeenCalled();

    // 修正后可正常通过
    await act(async () => {
      fireEvent.change(screen.getByLabelText("Untagged VLAN 列表"), {
        target: { value: "10,20" },
      });
    });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "预览命令（dry-run）" }));
    });
    await waitFor(() => {
      expect(screen.getByText("dry-run 预览，未下发任何设备命令")).toBeInTheDocument();
    });
    const previewArg = mockPreviewBatchWritePorts.mock.calls[0][0] as {
      hybridTagged: number[];
      hybridUntagged: number[];
    };
    expect(previewArg.hybridTagged).toEqual([30]);
    expect(previewArg.hybridUntagged).toEqual([10, 20]);
  });

  // ==================== v1.51 Phase 125 portsec（commit 006ece28） ====================

  it("Phase 125: portsec_max_mac preview→确认→写, maxMacCount 透传", async () => {
    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1]}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper: Wrapper }
    );

    await selectOptionByLabel("操作类型", "设置 MAC 学习上限");
    await act(async () => {
      fireEvent.change(screen.getByLabelText("MAC 学习上限"), { target: { value: "64" } });
    });
    await selectOptionByLabel("操作原因", "故障排查处理");
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "预览命令（dry-run）" }));
    });
    await waitFor(() => {
      expect(screen.getByText("dry-run 预览，未下发任何设备命令")).toBeInTheDocument();
    });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "确认下发（真实写）" }));
    });
    await waitFor(() => {
      expect(mockBatchWritePorts).toHaveBeenCalledTimes(1);
    });

    const writeArg = mockBatchWritePorts.mock.calls[0][0] as { maxMacCount: number };
    expect(writeArg.maxMacCount).toBe(64);
  });

  it("Phase 125: portsec_static_bind critical → 确认按钮 danger, macAddress 归一 + vlanId 透传", async () => {
    // 契约: static_bind preview riskLevel 恒 critical
    mockPreviewBatchWritePorts.mockResolvedValue(makePreview({ riskLevel: "critical" }));

    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1]}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper: Wrapper }
    );

    await selectOptionByLabel("操作类型", "配置静态 MAC 绑定");
    await act(async () => {
      // 点分小写输入 → 归一为大写连字符（三格式归一验证）
      fireEvent.change(screen.getByLabelText("静态 MAC 地址"), {
        target: { value: "aabb.ccdd.eeff" },
      });
      fireEvent.change(screen.getByLabelText("绑定 VLAN"), { target: { value: "100" } });
    });
    await selectOptionByLabel("操作原因", "故障排查处理");
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "预览命令（dry-run）" }));
    });
    await waitFor(() => {
      expect(screen.getByText("dry-run 预览，未下发任何设备命令")).toBeInTheDocument();
    });
    // static_bind=critical → 确认按钮 danger（后端 riskLevel 驱动; mock 返回 critical）
    const confirmBtn = screen.getByRole("button", { name: "确认下发（真实写）" }) as HTMLButtonElement;
    expect(confirmBtn.className).toContain("danger");
    await act(async () => {
      fireEvent.click(confirmBtn);
    });
    await waitFor(() => {
      expect(mockBatchWritePorts).toHaveBeenCalledTimes(1);
    });

    const writeArg = mockBatchWritePorts.mock.calls[0][0] as {
      macAddress: string;
      vlanId: number;
    };
    expect(writeArg.macAddress).toBe("AA-BB-CC-DD-EE-FF");
    expect(writeArg.vlanId).toBe(100);
  });

  it("Phase 125: skipPreflight 开关开启 → 确认写请求携带 skipPreflight, 预览不受影响", async () => {
    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1]}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper: Wrapper }
    );

    await selectOptionByLabel("操作类型", "关闭端口");
    await act(async () => {
      fireEvent.click(screen.getByRole("switch"));
    });
    await selectOptionByLabel("操作原因", "故障排查处理");

    // preview 请求不带 skipPreflight（dry-run 与预检是两回事, 预览流程保留）
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "预览命令（dry-run）" }));
    });
    await waitFor(() => {
      expect(mockPreviewBatchWritePorts).toHaveBeenCalledTimes(1);
    });
    expect(
      (mockPreviewBatchWritePorts.mock.calls[0][0] as { skipPreflight?: boolean }).skipPreflight
    ).toBeUndefined();

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "确认下发（真实写）" }));
    });
    await waitFor(() => {
      expect(mockBatchWritePorts).toHaveBeenCalledTimes(1);
    });
    expect(
      (mockBatchWritePorts.mock.calls[0][0] as { skipPreflight?: boolean }).skipPreflight
    ).toBe(true);
  });

  // ==================== Phase 122 PORTW-03: WS 实时进度（契约 ece40502） ====================

  // PORTW-03 时序确定性改造（返工）: 全部 WS 用例零真实 timer——
  // mock batchWritePorts 返回手动 deferred, 事件用 act 同步派发, 断言后手动 resolve。
  // 原因: 真实 setTimeout(50-800ms) 在目录级并行 fork 下累积悬挂定时器,
  // 触发 worker 崩溃（Phase 88 maxWorkers 同族）; deferred 模式确定性且零泄漏。

  /** 手动 deferred: 测试断言后按需 resolve batch 响应 */
  function makeDeferred(): {
    promise: Promise<BatchResult>;
    resolve: (r: BatchResult) => void;
  } {
    let resolve!: (r: BatchResult) => void;
    const promise = new Promise<BatchResult>((r) => {
      resolve = r;
    });
    return { promise, resolve };
  }

  it("PORTW-03: WS progress 事件驱动真实进度视图（Progress + 端口逐行点亮）", async () => {
    mockPreviewBatchWritePorts.mockResolvedValue(makePreview());
    const deferred = makeDeferred();
    mockBatchWritePorts.mockReturnValueOnce(deferred.promise);

    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1, PORT_A2]}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper: Wrapper }
    );

    await fillAndSubmit();

    // 确认 listener 已随 executing 挂载
    await waitFor(() => {
      expect(mockPortWriteListeners.size).toBeGreaterThan(0);
    });

    // 同步派发事件（act 内, 零真实 timer）
    await act(async () => {
      emitPortWriteEvent({
        type: "port_write_progress",
        batchId: "batch-1",
        deviceId: DEVICE_A,
        action: "shutdown",
        portId: "port-a-1",
        interfaceName: "GigabitEthernet0/1",
        index: 1,
        total: 2,
        status: "succeeded",
      });
    });
    expect(await screen.findByText("GigabitEthernet0/1")).toBeInTheDocument();

    await act(async () => {
      emitPortWriteEvent({
        type: "port_write_progress",
        batchId: "batch-1",
        deviceId: DEVICE_A,
        action: "shutdown",
        portId: "port-a-2",
        interfaceName: "GigabitEthernet0/2",
        index: 2,
        total: 2,
        status: "skipped",
      });
    });
    expect(await screen.findByText("GigabitEthernet0/2")).toBeInTheDocument();
    expect(screen.getByText("跳过")).toBeInTheDocument();

    // 手动 resolve → result 视图（BatchResult 兜底真相源）
    await act(async () => {
      deferred.resolve({
        batchId: "batch-1",
        succeeded: [makePortResult("port-a-1", "succeeded")],
        failed: [],
        skipped: [makePortResult("port-a-2", "skipped")],
      } as BatchResult);
    });
    await waitFor(() => {
      expect(screen.getByText("✓ 成功")).toBeInTheDocument();
    });
  });

  it("PORTW-03: completed 事件渲染 summary 三计数 + fail-fast 中断文案", async () => {
    mockPreviewBatchWritePorts.mockResolvedValue(makePreview());
    const deferred = makeDeferred();
    mockBatchWritePorts.mockReturnValueOnce(deferred.promise);

    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1, PORT_A2]}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper: Wrapper }
    );

    await fillAndSubmit();
    await waitFor(() => {
      expect(mockPortWriteListeners.size).toBeGreaterThan(0);
    });

    await act(async () => {
      emitPortWriteEvent({
        type: "port_write_progress",
        batchId: "batch-2",
        deviceId: DEVICE_A,
        action: "shutdown",
        portId: "port-a-1",
        interfaceName: "GigabitEthernet0/1",
        index: 1,
        total: 2,
        status: "failed",
        message: "device timeout",
      });
    });
    await act(async () => {
      emitPortWriteEvent({
        type: "port_write_completed",
        batchId: "batch-2",
        deviceId: DEVICE_A,
        action: "shutdown",
        portId: null,
        interfaceName: null,
        index: 2,
        total: 2,
        status: "failed",
        summary: '{"succeeded":0,"failed":1,"skipped":0}',
        message: "批次中断（fail-fast）",
      });
    });

    expect(await screen.findByText("批次中断（fail-fast）")).toBeInTheDocument();
    expect(screen.getByText("✗ 失败")).toBeInTheDocument();

    await act(async () => {
      deferred.resolve({
        batchId: "batch-2",
        succeeded: [],
        failed: [makePortResult("port-a-1", "failed", "device timeout")],
        skipped: [],
      } as BatchResult);
    });
    await waitFor(() => {
      expect(screen.getByText(/重试失败端口/)).toBeInTheDocument();
    });
  });

  it("PORTW-03: WS 不可用降级 — 无事件时回退 indeterminate 提示, 同步响应照常进 result", async () => {
    mockPreviewBatchWritePorts.mockResolvedValue(makePreview());
    const deferred = makeDeferred();
    mockBatchWritePorts.mockReturnValueOnce(deferred.promise);

    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1]}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper: Wrapper }
    );

    await fillForm();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "预览命令（dry-run）" }));
    });
    await waitFor(() => {
      expect(screen.getByText("dry-run 预览，未下发任何设备命令")).toBeInTheDocument();
    });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "确认下发（真实写）" }));
    });

    // 无事件 → 降级提示出现（不白屏）
    expect(
      await screen.findByText("实时进度通道未就绪，完成后将直接显示结果")
    ).toBeInTheDocument();

    // 同步响应兜底 → result 视图
    await act(async () => {
      deferred.resolve({
        succeeded: [makePortResult("port-a-1", "succeeded")],
        failed: [],
        skipped: [],
      } as BatchResult);
    });
    await waitFor(() => {
      expect(screen.getByText("✓ 成功")).toBeInTheDocument();
    });
  });

  it("PORTW-03: batchId 不匹配的事件被忽略（批次关联隔离; 用户级隔离由后端会话保证）", async () => {
    mockPreviewBatchWritePorts.mockResolvedValue(makePreview());
    const deferred = makeDeferred();
    mockBatchWritePorts.mockReturnValueOnce(deferred.promise);

    render(
      <BulkWriteDrawer
        open={true}
        selectedPorts={[PORT_A1, PORT_A2]}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />,
      { wrapper: Wrapper }
    );

    await fillAndSubmit();
    await waitFor(() => {
      expect(mockPortWriteListeners.size).toBeGreaterThan(0);
    });

    await act(async () => {
      // 其它批次（batchId 不匹配）→ 应被忽略
      emitPortWriteEvent({
        type: "port_write_progress",
        batchId: "batch-OTHER",
        deviceId: DEVICE_A,
        action: "shutdown",
        portId: "port-a-2",
        interfaceName: "GigabitEthernet0/2",
        index: 2,
        total: 2,
        status: "failed",
      });
      // 本批次 → 接受并锁定 batchId
      emitPortWriteEvent({
        type: "port_write_progress",
        batchId: "batch-mine",
        deviceId: DEVICE_A,
        action: "shutdown",
        portId: "port-a-1",
        interfaceName: "GigabitEthernet0/1",
        index: 1,
        total: 2,
        status: "succeeded",
      });
    });

    expect(await screen.findByText("GigabitEthernet0/1")).toBeInTheDocument();
    expect(screen.queryByText("GigabitEthernet0/2")).not.toBeInTheDocument();

    await act(async () => {
      deferred.resolve({
        batchId: "batch-mine",
        succeeded: [makePortResult("port-a-1", "succeeded")],
        failed: [],
        skipped: [],
      } as BatchResult);
    });
    await waitFor(() => {
      expect(screen.getByText("✓ 成功")).toBeInTheDocument();
    });
  });
});
