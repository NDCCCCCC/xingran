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
  okButtonProps?: { disabled?: boolean };
}

// Phase 132：递归收集 React 元素树里的所有文本节点，用于断言「本次范围」文案
// 是否挂进 renderContent() / modalRef.current.update() 刷新链路。
function collectScopeWriteTexts(node: unknown): string[] {
  if (node == null || typeof node === "boolean") return [];
  if (typeof node === "string" || typeof node === "number") return [String(node)];
  if (Array.isArray(node)) return node.flatMap(collectScopeWriteTexts);
  if (typeof node === "object") {
    const el = node as { props?: { children?: unknown } };
    if (el.props && "children" in el.props) {
      return collectScopeWriteTexts(el.props.children);
    }
  }
  return [];
}

const scopeWriteRangeText = (cfg: ConfirmConfig): string =>
  collectScopeWriteTexts(cfg.content).find((s) => s.startsWith("本次范围")) ?? "";

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
      props: {
        children: Array<{
          props?: { onChange?: (e: { target: { checked: boolean } }) => Promise<void> };
        }>;
      };
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

  // Phase 150：132 的四类「本次范围：…（共 N 条）」文案整体作废（DECISION-①）；
  // 确认框范围行改为不含全库规模、不含任何数值的可见集内表述。
  const openWithPreview = async (p: BackfillDeptResult) => {
    const onSuccess = vi.fn();
    vi.mocked(workstationApi.backfillDeptFromUser).mockResolvedValueOnce(p);
    const { result } = renderHook(() => useBackfillDept(onSuccess));
    await act(async () => {
      await result.current.open();
    });
  };

  // 范围行通用断言：以「本次范围：」开头、不含任何数字、不含规模量词。
  const expectNoScaleText = (cfg: ConfirmConfig) => {
    const text = scopeWriteRangeText(cfg);
    expect(text.startsWith("本次范围：")).toBe(true);
    expect(text).not.toMatch(/[0-9]/);
    expect(text).not.toContain("共");
    expect(text).not.toContain("条");
    expect(text).not.toContain("等");
    expect(text).not.toContain("个部门");
    return text;
  };

  it("All 档 → 「本次范围：全部工位」且确认按钮可用", async () => {
    await openWithPreview(preview({ scopeTier: "All" }));
    expect(expectNoScaleText(latestConfig)).toBe("本次范围：全部工位");
    expect(latestConfig.okButtonProps?.disabled).toBe(false);
  });

  it("Custom/Dept/DeptChild 档 → 可见集内表述、无任何数值且确认按钮可用", async () => {
    await openWithPreview(
      preview({
        scopeTier: "Custom/Dept/DeptChild",
        scopeDeptNames: ["研发部", "测试部"],
      })
    );
    expect(expectNoScaleText(latestConfig)).toBe("本次范围：当前数据权限可见的部门及子部门");
    expect(latestConfig.okButtonProps?.disabled).toBe(false);
  });

  it("多部门（>3）→ 不回显部门总数、不回显「等 M 个部门」数值", async () => {
    await openWithPreview(
      preview({
        scopeTier: "DeptChild",
        scopeDeptNames: ["A部", "B部", "C部", "D部"],
      })
    );
    const text = expectNoScaleText(latestConfig);
    expect(text).toBe("本次范围：当前数据权限可见的部门及子部门");
    // 确认框范围行不得含任何派生规模数值（旧文案的 9 / 5 一类数字）。
    expect(text).not.toContain("9");
    expect(text).not.toContain("5");
  });

  it("Self 档 → 「本次范围：仅本人关联工位」且确认按钮可用", async () => {
    await openWithPreview(preview({ scopeTier: "Self" }));
    expect(expectNoScaleText(latestConfig)).toBe("本次范围：仅本人关联工位");
    expect(latestConfig.okButtonProps?.disabled).toBe(false);
  });

  it("FailClosed 档 → 「无（数据权限异常，请联系管理员）」且确认按钮 disabled", async () => {
    await openWithPreview(preview({ scopeTier: "FailClosed", scopeFailClosed: true }));
    expect(scopeWriteRangeText(latestConfig)).toBe("本次范围：无（数据权限异常，请联系管理员）");
    expect(latestConfig.okButtonProps?.disabled).toBe(true);
  });

  it("勾选「纠正不一致」触发 update() 刷新链路后，范围文案与 disabled 不丢失", async () => {
    const onSuccess = vi.fn();
    vi.mocked(workstationApi.backfillDeptFromUser)
      .mockResolvedValueOnce(
        preview({
          scopeTier: "Custom/Dept/DeptChild",
          scopeDeptNames: ["研发部"],
        })
      )
      .mockResolvedValueOnce(
        preview({
          scopeTier: "Custom/Dept/DeptChild",
          scopeDeptNames: ["研发部"],
          fixMismatch: true,
        })
      );
    const { result } = renderHook(() => useBackfillDept(onSuccess));
    await act(async () => {
      await result.current.open();
    });
    const content = latestConfig.content as {
      props: {
        children: Array<{
          props?: { onChange?: (e: { target: { checked: boolean } }) => Promise<void> };
        }>;
      };
    };
    const checkbox = content.props.children.find((c) => c?.props?.onChange);
    expect(checkbox).toBeTruthy();
    await act(async () => {
      await checkbox!.props!.onChange!({ target: { checked: true } });
    });
    // update() 的最后一版配置必须仍带范围文案 + 非 disabled，且范围行不含数值。
    const last = updates[updates.length - 1];
    const lastCfg = { content: last.content } as ConfirmConfig;
    expect(expectNoScaleText(lastCfg)).toBe("本次范围：当前数据权限可见的部门及子部门");
    expect(last.okButtonProps?.disabled).toBe(false);
  });
});
