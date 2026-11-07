/**
 * useBackfillDept 交互回归测试（quick-261106-wsb）。
 *
 * 覆盖：
 *  - 点击 → 先 dryRun=true 预览；
 *  - 确认 → dryRun=false 执行，成功后 onSuccess 刷新；
 *  - 「纠正不一致」默认关闭，勾选后重新 dry-run 且 fixMismatch=true；
 *  - 预览/执行异常不崩溃、不刷新。
 *
 * Modal.confirm 是命令式独立 root：这里 mock 其配置对象并直接调用 onOk/update，
 * 不渲染真实 DOM，避免 .ant-modal-confirm 残留。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { Modal } from "antd";
import { workstationApi } from "@/lib/opsApi";
import type { BackfillDeptResult } from "@/types";
import { useBackfillDept } from "../useBackfillDept";

vi.mock("@/lib/opsApi", () => ({
  workstationApi: {
    backfillDeptFromUser: vi.fn(),
  },
}));

vi.mock("@/utils/antdMessage", () => ({
  getAppMessage: () => ({ success: vi.fn(), error: vi.fn() }),
}));

vi.mock("@/utils/errorHandler", () => ({
  handleApiError: vi.fn(),
}));

const preview = (overrides: Partial<BackfillDeptResult> = {}): BackfillDeptResult => ({
  dryRun: true,
  fixMismatch: false,
  candidates: 5,
  toFill: 3,
  toCorrect: 0,
  skippedNoUser: 1,
  skippedBadUserDept: 1,
  skippedAlreadyValid: 1,
  samples: [],
  ...overrides,
});

interface ConfirmConfig {
  onOk?: () => Promise<void> | void;
  onCancel?: () => void;
  afterClose?: () => void;
  content?: unknown;
  okText?: string;
}

describe("useBackfillDept", () => {
  let confirmSpy: ReturnType<typeof vi.spyOn>;
  let latestConfig: ConfirmConfig;
  const updates: Partial<ConfirmConfig>[] = [];

  beforeEach(() => {
    vi.clearAllMocks();
    updates.length = 0;
    latestConfig = {};
    confirmSpy = vi.spyOn(Modal, "confirm").mockImplementation((cfg) => {
      latestConfig = cfg as ConfirmConfig;
      return {
        destroy: vi.fn(),
        update: (next: Partial<ConfirmConfig>) => {
          updates.push(next);
          Object.assign(latestConfig, next);
        },
      } as ReturnType<typeof Modal.confirm>;
    });
  });

  it("预览 + 确认两步：先 dryRun，再执行，成功后刷新", async () => {
    const onSuccess = vi.fn();
    vi.mocked(workstationApi.backfillDeptFromUser)
      .mockResolvedValueOnce(preview())
      .mockResolvedValueOnce(preview({ dryRun: false }));

    const { result } = renderHook(() => useBackfillDept(onSuccess));
    await act(async () => {
      await result.current.open();
    });

    // 1) 第一次调用必须是 dryRun=true，fixMismatch=false。
    expect(workstationApi.backfillDeptFromUser).toHaveBeenNthCalledWith(1, {
      dryRun: true,
      fixMismatch: false,
    });
    expect(confirmSpy).toHaveBeenCalledTimes(1);
    expect(latestConfig.okText).toContain("确认执行");

    // 2) 确认 → 执行 dryRun=false，成功后触发刷新。
    await act(async () => {
      await latestConfig.onOk?.();
    });
    expect(workstationApi.backfillDeptFromUser).toHaveBeenNthCalledWith(2, {
      dryRun: false,
      fixMismatch: false,
    });
    expect(onSuccess).toHaveBeenCalledTimes(1);
  });

  it("勾选「纠正不一致」→ 重新预览且 fixMismatch=true", async () => {
    const onSuccess = vi.fn();
    vi.mocked(workstationApi.backfillDeptFromUser)
      .mockResolvedValueOnce(preview())
      .mockResolvedValueOnce(preview({ fixMismatch: true, toCorrect: 1 }))
      .mockResolvedValueOnce(preview({ dryRun: false, fixMismatch: true, toCorrect: 1 }));

    const { result } = renderHook(() => useBackfillDept(onSuccess));
    await act(async () => {
      await result.current.open();
    });

    // 从 content 里取出 Checkbox 的 onChange 并触发勾选。
    // content 是 React 元素树；用其子级结构定位 Checkbox（Space 的第 4 个 child）。
    const content = latestConfig.content as {
      props: { children: Array<{ props?: { onChange?: (e: { target: { checked: boolean } }) => Promise<void> } }> };
    };
    const checkbox = content.props.children.find((c) => c?.props?.onChange);
    expect(checkbox).toBeTruthy();

    await act(async () => {
      await checkbox!.props!.onChange!({ target: { checked: true } });
    });

    expect(workstationApi.backfillDeptFromUser).toHaveBeenNthCalledWith(2, {
      dryRun: true,
      fixMismatch: true,
    });
    expect(latestConfig.okText).toContain("纠正 1 条");

    await act(async () => {
      await latestConfig.onOk?.();
    });
    expect(workstationApi.backfillDeptFromUser).toHaveBeenNthCalledWith(3, {
      dryRun: false,
      fixMismatch: true,
    });
  });

  it("预览失败 → 不弹确认、不执行", async () => {
    const onSuccess = vi.fn();
    vi.mocked(workstationApi.backfillDeptFromUser).mockRejectedValueOnce(new Error("net"));

    const { result } = renderHook(() => useBackfillDept(onSuccess));
    await act(async () => {
      await result.current.open();
    });

    expect(confirmSpy).not.toHaveBeenCalled();
    expect(onSuccess).not.toHaveBeenCalled();
    expect(workstationApi.backfillDeptFromUser).toHaveBeenCalledTimes(1);
  });

  it("执行失败 → 不刷新", async () => {
    const onSuccess = vi.fn();
    vi.mocked(workstationApi.backfillDeptFromUser)
      .mockResolvedValueOnce(preview())
      .mockRejectedValueOnce(new Error("write failed"));

    const { result } = renderHook(() => useBackfillDept(onSuccess));
    await act(async () => {
      await result.current.open();
    });
    await act(async () => {
      try {
        await latestConfig.onOk?.();
      } catch {
        // onOk 会抛回以阻止 Modal 关闭，属预期
      }
    });

    expect(onSuccess).not.toHaveBeenCalled();
  });
});
